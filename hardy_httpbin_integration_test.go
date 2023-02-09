package hardy_test

/*type TestService struct {
	Client *hardy.Client
}

func (s *TestService) GetHelloMessage(ctx context.Context, name string) (string, error) {
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

func (s *TestService) PostHelloMessage(ctx context.Context, name string) (string, error) {
	body := struct {
		Name string `json:"name"`
	}{
		Name: name,
	}
	b, err := json.Marshal(&body)
	if err != nil {
		return "", err
	}
	var helloMessage string
	request, err := http.NewRequest(http.MethodPost, "https://httpbin.org/status/500,200,300", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	readerFunc := func(receiver *string) hardy.ReaderFunc {
		return func(response *http.Response) error {
			if response.StatusCode == http.StatusOK {
				*receiver = fmt.Sprintf("hello to requester, %s!", name)
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
	t.Parallel()

	client, err := hardy.NewClient(
		hardy.WithDebugDisabled(),
		hardy.WithMaxRetries(4),
		hardy.WithWaitInterval(3*time.Millisecond),
		hardy.WithMaxInterval(3*time.Second),
	)

	if err != nil {
		t.Error(err)
	}

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
			msg, err := testService.PostHelloMessage(ctx, tt.args.name)
			if err != nil {
				t.Logf("GetHelloMessage() error = %v\n", err)
			} else {
				t.Logf("GetHelloMessage() message = %s\n", msg)
			}

		})
	}
}*/
