package hardy_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/diegohordi/hardy"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClient_Try(t *testing.T) {
	type fields struct {
		Client func() *hardy.Client
	}
	type args struct {
		ctx          func() (context.Context, context.CancelFunc)
		req          func() *http.Request
		readerFunc   hardy.ReaderFunc
		fallbackFunc hardy.FallbackFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "should perform the request successfully",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusOK)
							return resp.Result(), nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(1 * time.Millisecond)
					return client
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
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							return nil, fmt.Errorf("not retriable error")
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(1 * time.Millisecond)
					return client
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
		},
		{
			name: "should reach out four failure retries",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(1 * time.Millisecond)
					return client
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
		},
		{
			name: "should reach out four failure retries and call the given fallback function",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(1 * time.Millisecond)
					return client
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
					return fmt.Errorf("error from fallback function")
				},
			},
			wantErr: true,
		},
		{
			name: "should reach out four failure retries without max timeout",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(1 * time.Millisecond).
						WithMaxInterval(0)
					return client
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
		},
		{
			name: "should reach out four failure retries respecting the max timeout defined",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							resp := httptest.NewRecorder()
							resp.WriteHeader(http.StatusServiceUnavailable)
							return resp.Result(), nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(3 * time.Millisecond).
						WithMaxInterval(1 * time.Millisecond)
					return client
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
		},
		{
			name: "should fail due to the try context deadline",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							time.Sleep(10 * time.Second)
							return &http.Response{}, nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(10 * time.Second)
					return client
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
		},
		{
			name: "should fail due to the empty reader func given",
			fields: fields{
				Client: func() *hardy.Client {
					httpClient := &http.Client{
						Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
							time.Sleep(10 * time.Second)
							return &http.Response{}, nil
						}),
					}
					client := hardy.NewClient(httpClient, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(10 * time.Second)
					return client
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
		},
		{
			name: "should fail due to the empty http client given",
			fields: fields{
				Client: func() *hardy.Client {
					client := hardy.NewClient(nil, log.Default()).
						WithMaxRetries(4).
						WithWaitInterval(10 * time.Second)
					return client
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.SetFlags(log.LstdFlags | log.Lmicroseconds)
			client := tt.fields.Client()
			ctx, cancelFunc := tt.args.ctx()
			if cancelFunc != nil {
				defer cancelFunc()
			}
			if err := client.Try(ctx, tt.args.req(), tt.args.readerFunc, tt.args.fallbackFunc); (err != nil) != tt.wantErr {
				t.Errorf("Try() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type TestService struct {
	Client *hardy.Client
}

func (s *TestService) GetHelloMessage(ctx context.Context, name string) (string, error) {
	if s.Client == nil {
		return "", fmt.Errorf("no client was given")
	}
	var helloMessage string
	request, err := http.NewRequest(http.MethodGet, "https://httpbin.org/status/500,200,300", nil)
	if err != nil {
		return "", err
	}
	readerFunc := func(receiver *string) hardy.ReaderFunc {
		return func(response *http.Response) error {
			if response.StatusCode == http.StatusOK {
				*receiver = fmt.Sprintf("hello from reader, %s!", name)
				return nil
			}
			return fmt.Errorf(response.Status)
		}
	}
	fallbackFunc := func(receiver *string) hardy.FallbackFunc {
		return func() error {
			*receiver = fmt.Sprintf("hello from fallback, %s!", name)
			return nil
		}
	}
	err = s.Client.Try(ctx, request, readerFunc(&helloMessage), fallbackFunc(&helloMessage))
	if err != nil {
		return "", err
	}
	return helloMessage, nil
}

func TestClient_TryParallel(t *testing.T) {

	httpClient := &http.Client{Timeout: 3 * time.Second}
	client := hardy.NewClient(httpClient, nil).
		WithMaxRetries(4).
		WithWaitInterval(3 * time.Millisecond).
		WithMultiplier(hardy.DefaultMultiplier).
		WithMaxInterval(3 * time.Second)
	testService := &TestService{Client: client}

	type args struct {
		ctx  func() (context.Context, context.CancelFunc)
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "should say hello to John",
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				name: "John",
			},
		},
		{
			name: "should say hello to Doe",
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				name: "Doe",
			},
		},
		{
			name: "should say hello to John Doe",
			args: args{
				ctx: func() (context.Context, context.CancelFunc) {
					return context.TODO(), nil
				},
				name: "John Doe",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			log.SetFlags(log.LstdFlags | log.Lmicroseconds)
			ctx, cancelFunc := tt.args.ctx()
			if cancelFunc != nil {
				defer cancelFunc()
			}
			msg, err := testService.GetHelloMessage(ctx, tt.args.name)
			if err != nil {
				t.Logf("GetHelloMessage() error = %v\n", err)
			} else {
				t.Logf("GetHelloMessage() message = %s\n", msg)
			}

		})
	}
}

func BenchmarkClient_TryWithNoRetries(b *testing.B) {
	httpClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := httptest.NewRecorder()
			resp.WriteHeader(http.StatusOK)
			return resp.Result(), nil
		}),
	}
	client := hardy.NewClient(httpClient, log.Default()).
		WithMaxRetries(4).
		WithWaitInterval(1 * time.Millisecond)

	readerFunc := func(response *http.Response) error {
		return nil
	}
	for n := 0; n < b.N; n++ {
		req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
		err := client.Try(context.TODO(), req, readerFunc, nil)
		if err != nil {
			b.Error(err)
		}
	}
}
func BenchmarkClient_TryWithRandomRetries(b *testing.B) {
	httpClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			resp := httptest.NewRecorder()
			resp.WriteHeader(http.StatusOK)
			return resp.Result(), nil
		}),
	}
	client := hardy.NewClient(httpClient, nil).
		WithMaxRetries(4).
		WithWaitInterval(1 * time.Millisecond).
		WithMaxInterval(1 * time.Millisecond)

	readerFunc := func(execution int) func(response *http.Response) error {
		return func(response *http.Response) error {
			random := rand.Intn(4)
			if random%2 == 0 {
				return fmt.Errorf("retry")
			}
			return nil
		}
	}
	for n := 0; n < b.N; n++ {
		req, _ := http.NewRequest(http.MethodPost, "http://localhost:80", bytes.NewReader(nil))
		_ = client.Try(context.TODO(), req, readerFunc(n), nil)
	}
}
