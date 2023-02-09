[![codecov](https://codecov.io/gh/diegohordi/hardy/branch/master/graph/badge.svg?token=2BPZ9TX105)](https://codecov.io/gh/diegohordi/hardy)

# Hardy

Hardy is wrapper around http.Client that enables you to add more resilience and reliability for your HTTP calls 
through retries. As retry strategy, it uses the exponential backoff algorithm and jitter to ensure that 
our services doesn't cause total outage to their dependencies. Besides that, it also provides some useful 
features like debug mode and default User-Agent headers.

* [Backoff and Jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter)

## Usage

### Install

`go get github.com/diegohordi/hardy`

### Creating the client

Hardy client already creates a http.Client by default, but you can override it, and configure it the way
you want it, as follows:

Optional parameters:
- **WithHttpClient** - will use the given `http.Client` to perform the requests.
- **WithDebugger** - will use the given debugger to print out the debug output.
- **WithDebugDisabled** - will disable the debug mode, which is enabled by default.
- **WithNoUserAgentHeader** - will use not User-Agent header.
- **WithUserAgentHeader** - will use a custom User-Agent header.
- **WithMaxRetries** - will determine how many retries should be attempted.
- **WithWaitInterval** - will define the base duration between each retry.
- **WithMultiplier** - the multiplier that should be used to calculate the backoff interval. Should be greater than the hardy.DefaultMultiplier.
- **WithMaxInterval** - the max interval between each retry. If no one was given, the interval between each retry will grow exponentially.

```go
httpClient := &http.Client{Timeout: 3 * time.Second}
client, err := hardy.NewClient(
    WithHttpClient(httpClient),
    WithMaxRetries(4),
	WithWaitInterval(3 * time.Millisecond),
	WithMultiplier(hardy.DefaultMultiplier),
	WithMaxInterval(3 * time.Second), 
)		   
```

### Using the client

The wrapper adds the method Try(context.Context, *http.Request, hardy.ReaderFunc, hardy.FallbackFunc),
which receives:

- **context.Context** a proper context to the request, mandatory. Hardy is also enabled to deal with context deadline/cancellation.
-***http.Request** an instance of the request that should be performed, mandatory.
- **hardy.ReaderFunc** a reader function, mandatory, that will be responsible to handle each request result.
-**hardy.FallbackFunc** a fallback function that will be called if all retries fail, optional.

#### hardy.ReaderFunc

The ReaderFunc defines the function responsible to read the HTTP response and also determines if a new retry
must be performed returning an error or not, returning nil.

Keep in mind while writing your reader function that we shouldn't perform a retry if the response contains
an error due to a client error (400-499 HTTP error codes), but consider only the ones not caused by them instead,
as 500 and 503 HTTP error codes, for instance.

#### Example

```go

// MessageService is a service used to call HTTP Bin API
type MessageService struct {
    Client *hardy.Client
}

// Message is the message that will be sent
type Message struct {
    Message string `json:"message"`
}

// PostMessageWithFallback sends a message with some possible HTTP status code responses, comma separated.
func (s *MessageService) PostMessage(ctx context.Context, message Message) (string, error) {
    
    // Marshal the given message to send to API
    b, err := json.Marshal(&message)
    if err != nil {
        return "", err
    }

    // Create a new API request
    request, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://someapi", bytes.NewReader(b))
    if err != nil {
        return "", err
    }

    // helloMessage will hold the message returned by the API
    var helloMessage string
    var reqErr error

    readerFunc := func(response *http.Response) error {
        if response.StatusCode == http.StatusBadRequest {
            reqErr = fmt.Errorf("error while posting message: %d - %s", response.StatusCode, response.Status)
            return nil
        }
        if response.StatusCode >= 500 {
            return fmt.Errorf(response.Status) // Will retry    	
        }
        var responseStruct ResponseMessageStruct
        if err := json.NewDecoder(response.Body).Decode(&responseStruct); err != nil {
            reqErr = fmt.Errorf("error while parsing response body: %w", err)
            return nil
        }
        helloMessage = responseStruct.Message
        return nil
    }

    // Create a fallback function with the API doesn't respond properly
	fallbackFunc := func() error {
        helloMessage = fmt.Sprintf("Hello from fallback!")
        return nil
    }

    // Try to execute the request
	err = s.Client.Try(ctx, request, readerFunc, fallbackFunc)
    if err != nil {
        return "", err
	}

    if reqErr != nil {
        return "", reqErr
    }

    return helloMessage, nil
}

// Create the Hardy client
client, err := hardy.NewClient(
    hardy.WithDebugDisabled(),
    hardy.WithMaxRetries(3),
    hardy.WithWaitInterval(3*time.Millisecond),
    hardy.WithMaxInterval(3*time.Second),
)
if err != nil {
    panic(err)
}

// Create the message service
messageService := &MessageService{
    Client: client,
}

// Post the message
message, err := messageService.PostMessage(context.Background(), Message{Message: "Hello John Doe"})
if err != nil {
    panic(err)
}

```

## Tests

The coverage so far is greater than 90%, covering also failure scenarios, and also, there are no 
race conditions detected in the -race tests. For integration tests, it uses [HTTP BIN](https://httpbin.org).

You can run the tests from Makefile, as below:

## Unit tests
```
make tests
```

## Integration tests
```
make integration_tests
```