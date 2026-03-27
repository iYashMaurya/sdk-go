.PHONY: test test-integration test-coverage test-coverage-html build vet fmt lint clean help

# Default target
help:
	@echo "Available commands:"
	@echo "  make test               Run all unit tests"
	@echo "  make test-integration   Run integration tests (requires LINGODOTDEV_API_KEY)"
	@echo "  make test-coverage      Run unit tests with coverage summary"
	@echo "  make test-coverage-html Run unit tests and open HTML coverage report"
	@echo "  make build              Build the SDK"
	@echo "  make vet                Run go vet"
	@echo "  make fmt                Format code with gofmt"
	@echo "  make lint               Run fmt + vet"
	@echo "  make clean              Remove coverage files"

test:
	go test ./tests/... -skip "TestRealAPI" -v -timeout 60s

test-integration:
	@if [ -z "$$LINGODOTDEV_API_KEY" ]; then \
		echo "Error: LINGODOTDEV_API_KEY is not set"; \
		exit 1; \
	fi
	go test ./tests/... -run "TestRealAPI" -v -timeout 120s

test-coverage:
	go test ./tests/... -skip "TestRealAPI" -timeout 60s -coverpkg=github.com/lingodotdev/sdk-go -coverprofile=coverage.out
	go tool cover -func=coverage.out

test-coverage-html:
	go test ./tests/... -skip "TestRealAPI" -timeout 60s -coverpkg=github.com/lingodotdev/sdk-go -coverprofile=coverage.out
	go tool cover -html=coverage.out

build:
	go build ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

lint: fmt vet

clean:
	rm -f coverage.out
