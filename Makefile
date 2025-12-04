.PHONY: build test docs clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/shrugged ./cmd/shrugged

test:
	go test ./...

test-verbose:
	go test -v ./...

docs:
	go run ./cmd/gendocs ./docs

clean:
	rm -rf bin/ docs/
