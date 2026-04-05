# Get the current commit hash for versioning
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean test fmt deps lint run install build-all

all: build

build:
	go build $(LDFLAGS) -o ./build/syncer ./main.go

clean:
	rm -rf ./build

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

deps:
	go mod download
	go mod tidy

lint:
	golangci-lint run ./...

run:
	go run $(LDFLAGS) ./main.go

install:
	go install $(LDFLAGS) ./main.go

build-all:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ./build/syncer-darwin-amd64 ./main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ./build/syncer-darwin-arm64 ./main.go
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ./build/syncer-linux-amd64 ./main.go
