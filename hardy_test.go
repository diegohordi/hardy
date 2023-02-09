package hardy_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/diegohordi/hardy"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type BuggyReaderCloser struct {
}

func (b BuggyReaderCloser) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("some reading error")
}

func (b BuggyReaderCloser) Close() error {
	return fmt.Errorf("some close error")
}

func TestClient_Try(t *testing.T) {
	t.Parallel()
	type fields struct {
		Client func() (*hardy.Client, error)
	}
	type args struct {
		ctx          func() (context.Context, context.CancelFunc)
		req          func() *http.Request
		readerFunc   hardy.ReaderFunc
		fallbackFunc hardy.FallbackFunc
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		wantClientErr bool
		wantErr       bool
		errWant       error
	}{
		{
			name: "should perform the request successfully",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with debugger enabled",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					var buf bytes.Buffer
					logger := log.Default()
					logger.SetOutput(&buf)
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugEnabled(logger),
						hardy.WithDebugDisabled(),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully closing the response body from the given ReaderFunc",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							_, _ = resp.WriteString("test body")
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					_, err := io.ReadAll(response.Body)
					if err != nil {
						return err
					}
					return response.Body.Close()
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with some error occur while closing the response body",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							_, _ = resp.WriteString("test body")
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}

					var buf bytes.Buffer
					logger := log.Default()
					logger.SetOutput(&buf)
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugEnabled(logger),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					buggyReader := BuggyReaderCloser{}
					response.Body = buggyReader
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with no client User-Agent header",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					var buf bytes.Buffer
					logger := log.Default()
					logger.SetOutput(&buf)
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugEnabled(logger),
						hardy.WithNoUserAgentHeader(),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with custom User-Agent header",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithUserAgentHeader("my-own-user-agent-header"),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with a custom backoff multiplier",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithBackoffMultiplier(3),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should perform the request successfully with a not acceptable backoff multiplier given",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithBackoffMultiplier(1),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "should try only one since the error got doesnt allow retries",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							return nil, fmt.Errorf("not retriable error")
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrUnexpected,
		},
		{
			name: "should reach out four failure retries",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					var buf bytes.Buffer
					logger := log.Default()
					logger.SetOutput(&buf)
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugEnabled(logger),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					if response.StatusCode == http.StatusServiceUnavailable {
						return fmt.Errorf("%s", response.Status)
					}
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrMaxRetriesReached,
		},
		{
			name: "should reach out four failure retries and call the given fallback function",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					if response.StatusCode == http.StatusServiceUnavailable {
						return fmt.Errorf("%s", response.Status)
					}
					return nil
				},
				fallbackFunc: func() error {
					return hardy.ErrUnexpected
				},
			},
			wantErr: true,
			errWant: hardy.ErrUnexpected,
		},
		{
			name: "should reach out four failure retries without max timeout",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithMaxRetries(4),
						hardy.WithDebugDisabled(),
						hardy.WithWaitInterval(1*time.Millisecond),
						hardy.WithMaxInterval(0),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					if response.StatusCode == http.StatusServiceUnavailable {
						return fmt.Errorf("%s", response.Status)
					}
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrMaxRetriesReached,
		},
		{
			name: "should reach out four failure retries respecting the max timeout defined",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithMaxRetries(4),
						hardy.WithDebugDisabled(),
						hardy.WithWaitInterval(3*time.Millisecond),
						hardy.WithMaxInterval(1*time.Millisecond),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					if response.StatusCode == http.StatusServiceUnavailable {
						return fmt.Errorf("%s", response.Status)
					}
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrMaxRetriesReached,
		},
		{
			name: "should fail due to the try context deadline",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							time.Sleep(10 * time.Second)
							return &http.Response{}, nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithMaxRetries(4),
						hardy.WithDebugDisabled(),
						hardy.WithWaitInterval(10*time.Second),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.TODO(), 3*time.Millisecond)
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: true,
			errWant: context.DeadlineExceeded,
		},
		{
			name: "should fail due to a nil http client given",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					return hardy.NewClient(
						hardy.WithHttpClient(nil),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.TODO(), 3*time.Millisecond)
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantClientErr: true,
			errWant:       hardy.ErrInvalidClientConfiguration,
		},
		{
			name: "should fail due to a nil debugger given",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					return hardy.NewClient(
						hardy.WithDebugEnabled(nil),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.TODO(), 3*time.Millisecond)
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantClientErr: true,
			errWant:       hardy.ErrInvalidClientConfiguration,
		},
		{
			name: "should fail due to a invalid request body while debugging the request",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}

					var buf bytes.Buffer
					logger := log.Default()
					logger.SetOutput(&buf)
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugEnabled(logger),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					buggyReader := BuggyReaderCloser{}
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", buggyReader)
					return req
				},
				readerFunc: func(response *http.Response) error {
					buggyReader := BuggyReaderCloser{}
					response.Body = buggyReader
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrUnexpected,
		},
		{
			name: "should fail due to a invalid request body while cloning the request",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				req: func() *http.Request {
					buggyReader := BuggyReaderCloser{}
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", buggyReader)
					req.GetBody = func() (io.ReadCloser, error) {
						return req.Body, hardy.ErrUnexpected
					}
					return req
				},
				readerFunc: func(response *http.Response) error {
					return nil
				},
			},
			wantErr: true,
			errWant: hardy.ErrUnexpected,
		},
		{
			name: "should fail due to the empty reader func given",
			fields: fields{
				Client: func() (*hardy.Client, error) {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							time.Sleep(10 * time.Second)
							return &http.Response{}, nil
						}),
					}
					return hardy.NewClient(
						hardy.WithHttpClient(httpClient),
						hardy.WithDebugDisabled(),
						hardy.WithMaxRetries(4),
						hardy.WithWaitInterval(10*time.Second),
					)
				},
			},
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.WithTimeout(context.TODO(), 3*time.Millisecond)
				},
				req: func() *http.Request {
					req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
					return req
				},
				readerFunc: nil,
			},
			wantErr: true,
			errWant: hardy.ErrNoReaderFuncFound,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the client
			client, err := tt.fields.Client()
			if err != nil != tt.wantClientErr {
				t.Errorf("Client() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantClientErr && !errors.Is(err, tt.errWant) {
				t.Errorf("Client() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && errors.Is(err, tt.errWant) {
				return
			}

			// Create the context
			ctx, cancelFunc := tt.args.ctx()
			if cancelFunc != nil {
				defer cancelFunc()
			}

			// Call try
			err = client.Try(ctx, tt.args.req(), tt.args.readerFunc, tt.args.fallbackFunc)
			if err != nil != tt.wantErr {
				t.Errorf("Try() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, tt.errWant) {
				t.Errorf("Try() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
