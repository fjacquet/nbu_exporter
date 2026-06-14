# Define the output binary
CLI_BIN = nbu_exporter
DIST    = dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w

# Pinned tool versions (installed by `make tools`).
GOLANGCI_LINT_VERSION   ?= v2.12.2
CYCLONEDX_GOMOD_VERSION ?= latest
GOVULNCHECK_VERSION     ?= latest

# Default target: build, test, docker
all: cli test docker

# Install pinned dev/CI tooling into $(GOBIN)/$GOPATH/bin.
tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_GOMOD_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# --- quality gates (used by CI) ---

fmt-check:
	@out=$$(gofmt -l $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	test -z "$$out" || { echo "gofmt needed in:"; echo "$$out"; exit 1; }

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

test:
	go test ./...

test-race:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

vuln:
	govulncheck ./...

# Aggregate gate run by CI.
ci: fmt-check vet lint test-race vuln

# Local convenience: format, vet, test, build, lint.
sure: fmt vet test
	go build ./...
	golangci-lint run

# --- artifacts ---

# Build the CLI binary
cli:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# Build with stripped symbols (alias kept for compatibility)
build-release:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# CycloneDX SBOM for the Go module (source/dependency SBOM).
sbom:
	@mkdir -p $(DIST)
	cyclonedx-gomod mod -licenses -json -output $(DIST)/sbom.cdx.json
	@echo "wrote $(DIST)/sbom.cdx.json"

# Tests with HTML coverage report
test-coverage: test-race
	go tool cover -html=coverage.out -o coverage.html

# Build the Docker image
docker:
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi
	docker build -t $(CLI_BIN) .

# Run the CLI binary
run-cli: cli
	./bin/$(CLI_BIN) --config config.yaml

# Run the Docker container
run-docker: docker
	docker run -d -p 9440:9440 --name $(CLI_BIN) $(CLI_BIN)

# Clean up build artifacts
clean:
	rm -f bin/$(CLI_BIN) coverage.out coverage.html
	rm -rf $(DIST)
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi

.PHONY: all tools fmt-check fmt vet lint test test-race vuln ci sure \
        cli build-release sbom test-coverage docker run-cli run-docker clean
