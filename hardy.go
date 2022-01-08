// Package hardy contains a resilient wrapper for http.Client that retries requests as per configurations adding jitter
// to calculate the interval between each try.
//
// Read more: https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/
package hardy

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

const (
	ErrMaxRetriesReached Error = "max retries reached"
)

const (
	DefaultWaitIntervalMs = 500
	DefaultMaxIntervalMs  = 5000
	DefaultMaxRetries     = 3
	DefaultMultiplier     = 2
)

// ReaderFunc defines the function responsible to read the HTTP response and also determines if a new retry
// must be performed returning an error or not, returning nil.
//
// Keep in mind while writing your reader function that we shouldn't perform a retry if the response contains
// an error due to a client error (400-499 HTTP error codes), but consider only the ones not caused by them instead,
// as 500 and 503 HTTP error codes, for instance.
type ReaderFunc func(response *http.Response) error

// FallbackFunc defines the function that should be used as fallback when max retries was reached out.
type FallbackFunc func() error

type Client struct {
	*http.Client
	logger       *log.Logger
	waitInterval time.Duration // Determines the base duration between each fail request
	maxRetries   int           // Determines how many retries should be attempted
	maxInterval  time.Duration // Determines the max interval between each fail request
	multiplier   float64       // Determines the multiplier that should be used to calculate the backoff interval
}

// NewClient creates a new Hardy wrapper around the given client that uses the given logger to log error messages.
//
// If no logger instance was given, no messages will be logged.
func NewClient(client *http.Client, logger *log.Logger) *Client {
	return &Client{
		Client:       client,
		logger:       logger,
		waitInterval: DefaultWaitIntervalMs * time.Millisecond,
		maxInterval:  DefaultMaxIntervalMs * time.Millisecond,
		maxRetries:   DefaultMaxRetries,
		multiplier:   DefaultMultiplier,
	}
}

// WithWaitInterval determines the base duration between each fail request.
func (c *Client) WithWaitInterval(interval time.Duration) *Client {
	c.waitInterval = interval
	return c
}

// WithMaxRetries determines how many retries should be attempted.
func (c *Client) WithMaxRetries(maxRetries int) *Client {
	c.maxRetries = maxRetries
	return c
}

// WithMaxInterval determines the max interval between each fail request.
func (c *Client) WithMaxInterval(interval time.Duration) *Client {
	c.maxInterval = interval
	return c
}

// WithMultiplier Determines the multiplier that should be used to calculate the backoff interval.
func (c *Client) WithMultiplier(multiplier float64) *Client {
	if multiplier < DefaultMultiplier {
		return c
	}
	c.multiplier = multiplier
	return c
}

func (c *Client) log(v ...interface{}) {
	if c.logger == nil {
		return
	}
	c.logger.Println(v...)
}

// getInterval calculates the interval between each retry based on the given attempt and the client configuration.
func getInterval(waitInterval, maxInterval time.Duration, attempt int, multiplier float64) time.Duration {
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

// Try tries to perform the given request as per configurations.
//
// An error is returned if no http.Client or ReaderFunc was given, or if the given context.Context was done, or if
// max retries was reached out, or if the given FallbackFunc returns some.
func (c *Client) Try(ctx context.Context, req *http.Request, readerFunc ReaderFunc, fallbackFunc FallbackFunc) error {
	if c.Client == nil {
		return fmt.Errorf("no http client was given")
	}
	if readerFunc == nil {
		return fmt.Errorf("no reader function was given")
	}
	errChan := make(chan error, 1)
	resultChan := make(chan struct{}, 1)
	go c.sendRequest(req, readerFunc, errChan, resultChan)
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

func (c *Client) sendRequest(req *http.Request, readerFunc ReaderFunc, errChan chan<- error, resultChan chan<- struct{}) {
	attempt := 0
	for {
		reqClone := req.Clone(req.Context())
		skipReaderFunc := false
		retriable := true
		resp, err := c.Do(reqClone)
		if err != nil {
			c.log(fmt.Errorf("attempt %d: %w", attempt+1, err))
			skipReaderFunc = true
			retriable = false
		}
		if !skipReaderFunc {
			err = readerFunc(resp)
			_ = resp.Body.Close()
			if err == nil {
				resultChan <- struct{}{}
				return
			}
			c.log(fmt.Errorf("attempt %d: %w", attempt+1, err))
		}
		if !retriable {
			errChan <- err
			return
		}
		attempt++
		if attempt == c.maxRetries {
			c.log(fmt.Sprintf("max retries reached %d", c.maxRetries))
			errChan <- ErrMaxRetriesReached
			return
		}
		retryTimer := time.NewTimer(getInterval(c.waitInterval, c.maxInterval, attempt+1, c.multiplier))
		<-retryTimer.C
	}
}
