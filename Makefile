format:
	docker run --rm -v $(shell pwd):/data cytopia/gofmt -l -w .

lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:v1.42.1 golangci-lint run -v ./...

test:
	docker run --rm -v $(shell pwd):/app -w /app golang:1.17 go test -count=1 -v -race -cover ./...

benchmark:
	docker run --rm -v $(shell pwd):/app -w /app golang:1.17 go test -count=1 -bench=. -run=$Benchmark -benchmem -memprofile mem.prof -cpuprofile cpu.prof
