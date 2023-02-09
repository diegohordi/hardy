format:
	docker run -v $(shell pwd):/data cytopia/gofmt -l -w .

mod:
	go mod tidy
	go mod vendor

lint: mod
	docker run -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v ./...

start_dev_env:
	docker-compose -f ./deployments/docker-compose.yml up -d

test: start_dev_env
	go test -count=1 ./... -race -cover -v -coverprofile cover.out
	go tool cover -func cover.out

benchmark:
	go test -count=1 -bench=. -run=$Benchmark -benchmem -memprofile mem.prof -cpuprofile cpu.prof
