FROM golang:1.19.5-alpine3.17 as test
ENV GOOS=linux
ENV CGO_ENABLED=0
RUN mkdir /app
COPY . /app
WORKDIR /app
CMD go test -count=1 -v ./...