.PHONY: build test lint fmt fmt-fix vet vuln check clean cross-build release-check release-snapshot

# Build the brr binary
build:
	mise exec -- go build -o brr ./cmd/brr

# Run all tests with race detector
test:
	mise exec -- go test -race ./...

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

# Check reachable dependency vulnerabilities
vuln:
	mise exec -- go run golang.org/x/vuln/cmd/govulncheck@v1.3.0 ./...

# Cross-compile for all release targets (mirrors .goreleaser.yaml: CGO_ENABLED=0)
cross-build:
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "  building $$os/$$arch..."; \
			CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch mise exec -- go build ./... || exit 1; \
		done; \
	done
	@echo "  all targets OK"

# Run all quality gates (use this before committing)
check: fmt vet lint test vuln build cross-build

# Clean build artifacts
clean:
	rm -f brr

# Validate goreleaser config
release-check:
	mise exec -- goreleaser check

# Build a local snapshot (no publish, no tag required)
release-snapshot:
	mise exec -- goreleaser release --snapshot --clean
