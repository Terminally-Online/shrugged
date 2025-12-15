.PHONY: build build-api test docs clean run-api examples clean-examples fmt

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

fmt:
	go fmt ./...

EXAMPLES_DB_PORT := 5499
EXAMPLES_DB_URL := postgres://shrugged:shrugged@localhost:$(EXAMPLES_DB_PORT)/shrugged?sslmode=disable

examples: build
	@echo "Regenerating examples..."
	@echo "Starting temporary database..."
	@docker rm -f shrugged-examples-db 2>/dev/null || true
	@docker run -d --name shrugged-examples-db \
		-e POSTGRES_USER=shrugged \
		-e POSTGRES_PASSWORD=shrugged \
		-e POSTGRES_DB=shrugged \
		-p $(EXAMPLES_DB_PORT):5432 \
		postgres:16 >/dev/null
	@echo "Waiting for database to be ready..."
	@sleep 3
	@for dir in examples/*/; do \
		name=$$(basename $$dir); \
		echo "  Processing: $$name"; \
		rm -rf $$dir/migrations $$dir/models $$dir/queries; \
		mkdir -p $$dir/migrations $$dir/models; \
		./bin/shrugged migrate --schema $$dir/schema.sql --migrations-dir $$dir/migrations || { docker rm -f shrugged-examples-db >/dev/null; exit 1; }; \
		docker exec shrugged-examples-db psql -U shrugged -d shrugged -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;" >/dev/null 2>&1; \
		./bin/shrugged apply --url "$(EXAMPLES_DB_URL)" --migrations-dir $$dir/migrations || { docker rm -f shrugged-examples-db >/dev/null; exit 1; }; \
		if [ -f $$dir/queries.sql ]; then \
			./bin/shrugged generate --url "$(EXAMPLES_DB_URL)" --out $$dir/models --language go --queries $$dir/queries.sql --queries-out $$dir/queries || { docker rm -f shrugged-examples-db >/dev/null; exit 1; }; \
		else \
			./bin/shrugged generate --url "$(EXAMPLES_DB_URL)" --out $$dir/models --language go || { docker rm -f shrugged-examples-db >/dev/null; exit 1; }; \
		fi; \
	done
	@docker rm -f shrugged-examples-db >/dev/null
	@echo "Formatting generated code..."
	@for dir in examples/*/; do \
		(cd $$dir && go fmt ./... >/dev/null 2>&1) || true; \
	done
	@echo "Examples regenerated successfully"

clean-examples:
	rm -rf examples/*/migrations examples/*/models examples/*/queries
	@docker rm -f shrugged-examples-db 2>/dev/null || true
