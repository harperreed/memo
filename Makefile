# ABOUTME: Build and test targets for memo CLI
# ABOUTME: Provides standard targets for development, CI, and release

.PHONY: build test test-race test-coverage install clean lint fmt

# Build binary to current directory (for CI compatibility)
build:
	CGO_ENABLED=1 go build -o memo ./cmd/memo

# Build to bin directory (for local development)
build-dev:
	CGO_ENABLED=1 go build -o bin/memo ./cmd/memo

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	CGO_ENABLED=1 go test -race -v ./...

# Run tests with coverage
test-coverage:
	CGO_ENABLED=1 go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Install to GOPATH/bin
install:
	go install ./cmd/memo

# Run linter
lint:
	golangci-lint run --timeout=10m

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	rm -f memo
	rm -rf bin/
	rm -f coverage.out

# Run all checks (useful before committing)
check: fmt lint test-race
