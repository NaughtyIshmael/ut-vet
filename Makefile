.PHONY: all build test test-unit test-e2e clean lint

# Default target
all: build test

# Build the ut-vet binary
build: fmt
	go build -o bin/ut-vet ./cmd/ut-vet/

# Run all tests (unit + e2e)
test: test-unit test-e2e

# Run unit tests only
test-unit:
	go test ./pkg/... -count=1 -v

# Run e2e tests (requires build first)
test-e2e: build
	UT_VET_BIN=$(CURDIR)/bin/ut-vet go test ./e2e/... -count=1 -v

# Run tests with race detector
test-race:
	go test ./pkg/... -race -count=1
	go build -race -o bin/ut-vet-race ./cmd/ut-vet/
	UT_VET_BIN=$(CURDIR)/bin/ut-vet-race go test ./e2e/... -race -count=1
	rm -f bin/ut-vet-race

# Run ut-vet against its own test files (dogfooding)
dogfood: build
	./bin/ut-vet -v ./pkg/ ./e2e/ || true

# Run go fmt
fmt:
	gofmt -w -s .

# Check formatting (CI-friendly — fails if files need formatting)
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Clean build artifacts
clean:
	rm -rf bin/

# Install the binary
install:
	go install ./cmd/ut-vet/

# Show available rules
rules: build
	./bin/ut-vet --list-rules
