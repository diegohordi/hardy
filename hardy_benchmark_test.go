package hardy

/*
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
*/
