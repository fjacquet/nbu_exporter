# Define the output binary
CLI_BIN = nbu_exporter
DIST    ?= dist
COVER   ?= coverage.out
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS = -s -w -X main.version=$(VERSION)

# Pinned tool versions (installed by `make tools`).
GOLANGCI_VERSION    ?= v2.12.2
GORELEASER_VERSION  ?= v2.16.0
CYCLONEDX_GOMOD_VERSION ?= latest
GOVULNCHECK_VERSION     ?= latest

.PHONY: all clean install tools lint format test build vuln sbom security docs coverage-upload release ci \
        fmt-check fmt vet test-race check-rules sure cli build-release test-coverage docker run-cli run-docker

.DEFAULT_GOAL := all

all: clean lint test build

# --- tooling ---

install:
	go mod download

tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
	go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@$(CYCLONEDX_GOMOD_VERSION)

# --- quality gates ---

fmt-check:
	@out=$$(gofmt -l $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	test -z "$$out" || { echo "gofmt needed in:"; echo "$$out"; exit 1; }

fmt:
	go fmt ./...

format:
	golangci-lint fmt

vet:
	go vet ./...

lint:
	golangci-lint run --timeout=5m

test:
	go test -race -coverprofile=$(COVER) -covermode=atomic ./...

test-race:
	go test -race -coverprofile=$(COVER) -covermode=atomic ./...

build:
	go build -v ./...

vuln:
	govulncheck ./...

# Validate + unit-test the Prometheus alerting rules (requires promtool).
check-rules:
	promtool check rules deploy/prometheus/nbu.rules.yml deploy/prometheus/rules-perclient.yml deploy/prometheus/rules-tape.yml deploy/prometheus/rules-multisite.yml
	promtool test rules deploy/prometheus/rules-perclient_test.yml deploy/prometheus/rules-tape_test.yml deploy/prometheus/rules-multisite_test.yml

# Aggregate gate run by CI.
ci: lint test build vuln

# --- artifacts ---

sbom:
	@mkdir -p $(DIST)
	go run github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest mod -json -output $(DIST)/sbom.cdx.json

security:  # advisory: reports findings but never blocks the build (CodeQL/osv are the blocking gates)
	uvx semgrep scan --config auto --skip-unknown-extensions || true

docs:
	uvx --with mkdocs-material --with pymdown-extensions mkdocs build --strict --site-dir site

coverage-upload:
	uvx --from codecov-cli codecov upload-process --file $(COVER) || true

release:
	goreleaser release --clean

# --- local convenience ---

# Build the CLI binary
cli:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# Build with stripped symbols (alias kept for compatibility)
build-release:
	go build -ldflags="$(LDFLAGS)" -o bin/$(CLI_BIN) .

# Tests with HTML coverage report
test-coverage: test-race
	go tool cover -html=$(COVER) -o coverage.html

# Local convenience: format, vet, test, build, lint.
sure: fmt vet test
	go build ./...
	golangci-lint run

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
	rm -rf $(DIST) site $(COVER) *.sarif
	rm -f bin/$(CLI_BIN) coverage.html
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi
