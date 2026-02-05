.PHONY: build install test lint clean run fmt vet

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/rbansal42/bb/internal/cmd.Version=$(VERSION) -X github.com/rbansal42/bb/internal/cmd.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o bin/bb ./cmd/bb

install:
	go install $(LDFLAGS) ./cmd/bb

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
	go clean

run:
	go run ./cmd/bb $(ARGS)

fmt:
	go fmt ./...

vet:
	go vet ./...
