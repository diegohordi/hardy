package hardy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/diegohordi/hardy"
	"net/http"
	"os"
	"testing"
	"time"
)

const HTTPBinURL = "http://localhost:80"

func getHTTPBinURL() string {
	if v := os.Getenv("HTTBIN_API"); v != "" {
		return v
	}
	return HTTPBinURL
}

// MessageService is a service used to call HTTP Bin API
type MessageService struct {
	Client *hardy.Client
}

// Message is the message that will be sent
type Message struct {
	Message string `json:"message"`
}

// PostMessageWithFallback sends a message with some possible HTTP status code responses, comma separated.
func (s *MessageService) PostMessage(ctx context.Context, message Message, statuses string, withFallback bool) (string, error) {
	// Marshal the given message to send to API
	b, err := json.Marshal(&message)
	if err != nil {
		return "", err
	}

	// Create a new API request
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/status/%s", getHTTPBinURL(), statuses), bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	// helloMessage will hold the message returned by the API
	var helloMessage string
	var reqErr error

	readerFunc := func(response *http.Response) error {
		if response.StatusCode == http.StatusOK {
			helloMessage = message.Message // Use the same message as the status endpoint has en empty response
			return nil
		}
		if response.StatusCode == http.StatusBadRequest {
			reqErr = fmt.Errorf("error while posting message: %d - %s", response.StatusCode, response.Status)
			return nil
		}
		return fmt.Errorf(response.Status)
	}

	// Create a fallback function with the API doesn't respond properly
	fallbackFunc := func() error {
		helloMessage = fmt.Sprintf("Hello from fallback!")
		return nil
	}

	// Try to execute the request
	if withFallback {
		err = s.Client.Try(ctx, request, readerFunc, fallbackFunc)
	} else {
		err = s.Client.Try(ctx, request, readerFunc, nil)
	}
	if err != nil {
		return "", err
	}

	if reqErr != nil {
		return "", reqErr
	}

	return helloMessage, nil
}

func TestClient_Integration_Try(t *testing.T) {
	t.Parallel()

	client, err := hardy.NewClient(
		hardy.WithDebugDisabled(),
		hardy.WithMaxRetries(3),
		hardy.WithWaitInterval(3*time.Millisecond),
		hardy.WithMaxInterval(3*time.Second),
	)

	if err != nil {
		t.Error(err)
	}

	messageService := &MessageService{
		Client: client,
	}

	type args struct {
		message      Message
		statuses     string
		withFallback bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		errWant error
		want    string
	}{
		{
			name: "should receive a message from fallback",
			args: args{
				message:      Message{Message: "Hello John Doe"},
				statuses:     "500",
				withFallback: true,
			},
			wantErr: false,
			errWant: nil,
			want:    "Hello from fallback!",
		},
		{
			name: "should receive a message from the reader",
			args: args{
				message:      Message{Message: "Hello John Doe"},
				statuses:     "200",
				withFallback: true,
			},
			wantErr: false,
			errWant: nil,
			want:    "Hello John Doe",
		},
		{
			name: "should receive an error since it not allow retries",
			args: args{
				message:      Message{Message: "Hello John Doe"},
				statuses:     "400",
				withFallback: true,
			},
			wantErr: true,
			errWant: errors.New("error while posting message: 400 - 400 BAD REQUEST"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			//t.Parallel()
			got, err := messageService.PostMessage(context.Background(), tt.args.message, tt.args.statuses, tt.args.withFallback)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("Try() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr && err.Error() != tt.errWant.Error() {
					t.Errorf("Try() error = %v, errWant %v", err, tt.errWant)
				}
			}
			if got != tt.want {
				t.Errorf("Try() got = %v, want %v", got, tt.want)
			}
		})
	}

}
