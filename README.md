# Hardy

Hardy is a very simple wrapper around http.Client that enables you to 
add more resilience and reliability for your HTTP calls through retries. As retry 
strategy, Hardy will use the exponential algorithm and jitter to ensure that 
our services doesn't cause total outage to their dependencies.

You can read more:
* https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter

## Usage

### Install

`go get github.com/diegohordi/hardy`

### Creating the client

In order to create the Hardy client you will need to provide a http.Client and a 
log.Logger instances. If no log.Logger instance was given, the log will be disabled.

Additional parameters:
* **WithMaxRetries** - will determine how many retries should be attempted.
* **WithWaitInterval** - will define the base duration between each retry.
* **WithMultiplier** - the multiplier that should be used to calculate the backoff interval. Should be greater than the hardy.DefaultMultiplier.
* **WithMaxInterval** - the max interval between each retry. If no one was given, the interval between each retry will grow exponentially.

```
httpClient := &http.Client{Timeout: 3 * time.Second}
client := hardy.NewClient(httpClient, log.Default()).
		    WithMaxRetries(4).
		    WithWaitInterval(3 * time.Millisecond).
		    WithMultiplier(hardy.DefaultMultiplier).
		    WithMaxInterval(3 * time.Second)
```

### Using the client

The wrapper adds the method Try(context.Context, *http.Request, hardy.ReaderFunc, hardy.FallbackFunc),
which receives:

* **context.Context** a proper context to the request, mandatory. Hardy is also enabled to deal with context deadline/cancellation.
* ***http.Request** an instance of the request that should be performed, mandatory.
* **hardy.ReaderFunc** a reader function, mandatory, that will be responsible to handle each request result.
* **hardy.FallbackFunc** a fallback function that will be called if all retries fail, optional.

#### hardy.ReaderFunc

The ReaderFunc defines the function responsible to read the HTTP response and also determines if a new retry
must be performed returning an error or not, returning nil.

Keep in mind while writing your reader function that we shouldn't perform a retry if the response contains
an error due to a client error (400-499 HTTP error codes), but consider only the ones not caused by them instead,
as 500 and 503 HTTP error codes, for instance.

#### Example

```

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
	readerFunc := func(message *string) hardy.ReaderFunc {
		return func(response *http.Response) error {
			if response.StatusCode == http.StatusOK {
				*message = fmt.Sprintf("hello from reader, %s!", name)
				return nil
			}
			return fmt.Errorf(response.Status)
		}
	}
	fallbackFunc := func(message *string) hardy.FallbackFunc {
		return func() error {
			*message = fmt.Sprintf("hello from fallback, %s!", name)
			return nil
		}
	}
	err = s.Client.Try(ctx, request, readerFunc(&helloMessage), fallbackFunc(&helloMessage))
	if err != nil {
		return "", err
	}
	return helloMessage, nil
}

...

httpClient := &http.Client{Timeout: 3 * time.Second}
client := hardy.NewClient(httpClient, log.Default()).
		WithMaxRetries(4).
		WithWaitInterval(3 * time.Millisecond).
		WithMultiplier(hardy.DefaultMultiplier).
		WithMaxInterval(3 * time.Second)
testService := &TestService{Client: client}

msg, err := testService.GetHelloMessage(ctx, tt.args.name)
```

## Tests

The coverage so far is greater than 90%, covering also failure scenarios. Also, there are no 
race conditions detected in the -race tests.

You can run the tests and the benchmark from Makefile, as below:

## Test
`make test`

## Benchmarks
`make benchmark`

# TODO

- [ ] Improve logs
