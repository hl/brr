.PHONY: build test lint fmt vet check clean

# Build the brr binary
build:
	mise exec -- go build -o brr ./cmd/brr

# Run all tests
test:
	mise exec -- go test ./...

# Run linter
lint:
	mise exec -- golangci-lint run ./...

# Check formatting
fmt:
	@test -z "$$(mise exec -- gofmt -l .)" || (mise exec -- gofmt -l . && echo "Run 'make fmt-fix' to fix" && exit 1)

# Fix formatting
fmt-fix:
	mise exec -- gofmt -w .

# Run go vet
vet:
	mise exec -- go vet ./...

# Run all quality gates (use this before committing)
check: fmt vet lint test build

# Clean build artifacts
clean:
	rm -f brr
