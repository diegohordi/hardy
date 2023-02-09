format:
	docker run -v $(shell pwd):/data cytopia/gofmt -l -w .

mod:
	go mod tidy
	go mod vendor

lint: mod
	docker run -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v ./...

start_dev_env:
	docker-compose up -d

test:
	go test -short -count=1 ./... -race -cover -v -coverprofile cover.out
	go tool cover -func cover.out

integration_test:
	docker-compose up --exit-code-from integration-tests

benchmark:
	go test -count=1 -bench=. -run=$Benchmark -benchmem -memprofile mem.prof -cpuprofile cpu.prof
