// Package hardy contains a wrapper for http.Client with some extra features, like common headers, debugger, fallback
// and exponential backoff as retry mechanism.
package hardy

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"runtime"
	"time"
)

const (

	// ClientVersion is the client version
	ClientVersion = "0.2.0"

	// DefaultWaitIntervalMilliseconds is the default wait interval in milliseconds between each retry.
	DefaultWaitIntervalMilliseconds = 500

	// DefaultMaxIntervalInMilliseconds is the default maximum wait interval in milliseconds between each retry.
	DefaultMaxIntervalInMilliseconds = 5000

	// DefaultMaxRetries is the default maximum allowed retries.
	DefaultMaxRetries = 3

	// DefaultBackoffMultiplier is the default backoff multiplier used to get next intervals.
	DefaultBackoffMultiplier = 2

	// DefaultTimeoutInSeconds is the maximum timeout for each attempt in seconds.
	DefaultTimeoutInSeconds = 10

	// userAgentHeader is the default User-Agent header.
	userAgentHeader = "User-Agent"

	// clientName is the client name used in as part of the User-Agent header.
	clientName = "go-hardy-http-client"
)

// ReaderFunc defines the function responsible to read the HTTP response and also determines if a new retry
// must be performed returning an error or not, returning nil.
//
// Keep in mind while writing your reader function that we shouldn't perform a retry if the response contains
// an error due to a client error (400-499 HTTP error codes), but consider only the ones not caused by them instead,
// as 500 and 503 HTTP error codes, for instance.
type ReaderFunc func(response *http.Response) error

// Debugger declares the methods that the debuggers should implement.
type Debugger interface {
	Println(v ...any)
}

// FallbackFunc defines the function that should be used as fallback when max retries was reached out.
type FallbackFunc func() error

type Client struct {

	// httpClient is the HTTP Client used to make the calls.
	httpClient *http.Client

	// waitInterval determines the base duration between each fail request
	waitInterval time.Duration

	// maxRetries determines how many retries should be attempted
	maxRetries int

	// maxInterval determines the max interval between each fail request
	maxInterval time.Duration

	// multiplier determines the multiplier that should be used to calculate the backoff interval
	multiplier float64

	// debug determines if each request should be dumped to the output. Default true.
	debug bool

	// Debugger that should be used to display request and response dumps. Default standard logger.
	debugger Debugger

	// withUserAgentHeader determines if it should add the User-Agent header for all requests. Default true.
	withUserAgentHeader bool

	// userAgent holds the user agent that will be added as header.
	userAgent string
}

// NewClient creates a new Hardy wrapper with the defaults or an error if it was misconfigured by some given option.
func NewClient(options ...Option) (*Client, error) {

	// Clone the default transport in order to add new configurations if needed.
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// Create the client with default configuration
	c := &Client{
		httpClient: &http.Client{
			Timeout:   DefaultTimeoutInSeconds * time.Second,
			Transport: transport,
		},
		waitInterval:        DefaultWaitIntervalMilliseconds * time.Millisecond,
		maxInterval:         DefaultMaxIntervalInMilliseconds * time.Millisecond,
		maxRetries:          DefaultMaxRetries,
		multiplier:          DefaultBackoffMultiplier,
		withUserAgentHeader: true,
		debug:               true,
		debugger:            log.Default(),
	}

	// Apply the given configurations
	for i := range options {
		if err := options[i](c); err != nil {
			return nil, newError(ErrInvalidClientConfiguration, withCause(err))
		}
	}

	// build User-Agent header
	c.setUserAgentHeader()
	return c, nil
}

// Option defines the optional configurations for the Client.
type Option func(c *Client) error

// WithDebugEnabled enables the debug mode, dumping the requests to output using the client logger.
func WithDebugEnabled(debugger Debugger) Option {
	return func(c *Client) error {
		if debugger == nil {
			return ErrNoDebuggerFound
		}
		c.debug = true
		c.debugger = debugger
		return nil
	}
}

// WithDebugDisabled disables the debug mode.
func WithDebugDisabled() Option {
	return func(c *Client) error {
		c.debug = false
		return nil
	}
}

// WithNoUserAgentHeader disables adding the User-Agent header in the request.
func WithNoUserAgentHeader() Option {
	return func(c *Client) error {
		c.withUserAgentHeader = false
		return nil
	}
}

// WithHttpClient overrides the default HTTP Client used by the one given.
func WithHttpClient(httpClient *http.Client) Option {
	return func(c *Client) error {
		if httpClient == nil {
			return ErrNoHTTPClientFound
		}
		c.httpClient = httpClient
		return nil
	}
}

// WithUserAgentHeader enables adding the User-Agent header in the request and overrides the default one.
func WithUserAgentHeader(userAgent string) Option {
	return func(c *Client) error {
		if userAgent != "" {
			c.userAgent = userAgent
		}
		c.withUserAgentHeader = true
		return nil
	}
}

// WithWaitInterval determines the base duration between each fail request.
func WithWaitInterval(interval time.Duration) Option {
	return func(c *Client) error {
		c.waitInterval = interval
		return nil
	}
}

// WithMaxRetries determines how many retries should be attempted.
func WithMaxRetries(maxRetries int) Option {
	return func(c *Client) error {
		c.maxRetries = maxRetries
		return nil
	}
}

// WithMaxInterval determines the max interval between each fail request.
func WithMaxInterval(interval time.Duration) Option {
	return func(c *Client) error {
		c.maxInterval = interval
		return nil
	}
}

// WithBackoffMultiplier Determines the multiplier that should be used to calculate the backoff interval.
func WithBackoffMultiplier(multiplier float64) Option {
	return func(c *Client) error {
		if multiplier < DefaultBackoffMultiplier {
			return nil
		}
		c.multiplier = multiplier
		return nil
	}
}

// setUserAgentHeader sets the User-Agent information that will be sent as header, accordingly to RFC7231.
func (c *Client) setUserAgentHeader() {
	userAgentFormatString := "%s/%s (%s)"
	c.userAgent = fmt.Sprintf(userAgentFormatString, clientName, ClientVersion, runtime.Version())
}

// getInterval calculates the interval between each retry based on the given attempt and the client configuration.
func (c *Client) getInterval(waitInterval, maxInterval time.Duration, attempt int, multiplier float64) time.Duration {
	rand.Seed(time.Now().UnixNano())
	backoff := waitInterval.Milliseconds() * int64(math.Pow(multiplier, float64(attempt)))
	random := int64(rand.Intn(1000))
	totalInterval := time.Duration(backoff+random) * time.Millisecond
	if maxInterval == 0 {
		return totalInterval
	}
	if totalInterval > maxInterval {
		return maxInterval
	}
	return totalInterval
}

// Try tries to perform the given request as per configurations. If some FallbackFunc is given,
// after max retries were reached, it will be called. It might return the following errors:
//
// - ErrNoReaderFuncFound - when no reader function was provided.
//
// - ErrMaxRetriesReached - if max retries were reached.
//
// - context.DeadlineExceeded or context.Canceled - if the given context was gone.
//
// - ErrUnexpected is the error returned when no one of the previous errors match.
func (c *Client) Try(ctx context.Context, req *http.Request, readerFunc ReaderFunc, fallbackFunc FallbackFunc) error {

	// Checks if a reader function was given
	if readerFunc == nil {
		return ErrNoReaderFuncFound
	}

	// Sets the User-Agent header if asked
	if c.withUserAgentHeader {
		req.Header.Add(userAgentHeader, c.userAgent)
	} else {
		if c.debug {
			if v := req.Header.Get(userAgentHeader); v == "" {
				c.debugger.Println("no User-Agent was given")
			}
		}
	}

	// Create channels to receive some error or the signal that the request was successfully performed.
	errChan := make(chan error, 1)
	resultChan := make(chan struct{}, 1)

	// Sends the request
	go c.sendRequest(ctx, req, readerFunc, errChan, resultChan)

	// Listen to the channels previously created or some signaling from the given context.
	select {
	case err := <-errChan:
		if fallbackFunc != nil {
			return fallbackFunc()
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-resultChan:
		return nil
	}
}

// sendRequest Sends the given request calling the given ReaderFunc to parse and analyse its return. Both, errors
// results are communicated via channels.
func (c *Client) sendRequest(ctx context.Context, req *http.Request, readerFunc ReaderFunc, errChan chan<- error, resultChan chan<- struct{}) {

	// Attempts counter
	attempt := 0

	// Will iterate until max retries were reached or the request was successfully performed.
	for {

		// Dumps the request if the debug is enabled
		if c.debug {
			b, err := httputil.DumpRequest(req, true)
			if err != nil {
				errChan <- newError(ErrUnexpected, withCause(err))
				return
			}
			c.debugger.Println(string(b))
		}

		// Clone the request to avoid reading twice
		clonedReq := req.Clone(ctx)
		if req.Body != nil {
			clonedBody, err := req.GetBody()
			if err != nil {
				errChan <- newError(ErrUnexpected, withCause(err))
			}
			clonedReq.Body = clonedBody
		}

		// Perform the request
		resp, err := c.httpClient.Do(clonedReq)

		// If some unexpected error occurred
		if err != nil {
			errChan <- newError(ErrUnexpected, withCause(fmt.Errorf("unexpected error during attempt %d: %w", attempt+1, err)))
			return
		}

		// Dumps the response if the debug is enabled
		if c.debug {
			b, err := httputil.DumpResponse(resp, true)
			if err != nil {
				errChan <- newError(ErrUnexpected, withCause(err))
			}
			c.debugger.Println(string(b))
		}

		// Call provided ReaderFunc and if some error was returned, will allow a new attempt.
		err = readerFunc(resp)

		// Closes the response body just in case the reader function forgot to do so.
		func(Body io.ReadCloser) {
			if closeErr := Body.Close(); closeErr != nil {
				if c.debug {
					c.debugger.Println(fmt.Errorf("error while closing response body: %w", closeErr))
				}
			}
		}(resp.Body)

		// If no error, send out the result.
		if err == nil {
			resultChan <- struct{}{}
			return
		}

		// Print the given error from the ReaderFunc if the debug is enabled.
		if c.debug {
			c.debugger.Println(fmt.Errorf("attempt %d: %w", attempt+1, err))
		}

		// Increase the attempts counter and check its limit.
		attempt++
		if attempt == c.maxRetries {
			errChan <- ErrMaxRetriesReached
			return
		}

		// Wait for the next iteration using exponential backoff and jitter
		retryTimer := time.NewTimer(c.getInterval(c.waitInterval, c.maxInterval, attempt+1, c.multiplier))
		<-retryTimer.C
	}
}
