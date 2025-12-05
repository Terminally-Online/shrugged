.PHONY: build build-api test docs clean run-api

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/shrugged ./cmd/shrugged

build-api:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/shrugged-api ./cmd/api

run-api: build-api
	./bin/shrugged-api

test:
	go test ./...

test-verbose:
	go test -v ./...

docs:
	go run ./cmd/gendocs ./docs

clean:
	rm -rf bin/ docs/
