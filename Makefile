# Define the output binaries
CLI_BIN = nbu_exporter

# Default target: build both binaries
all: cli test docker

# Ensure code quality: format, test, and build
sure:
	go fmt ./...
	go test ./...
	go build ./...
	golangci-lint run 

test:
	go test ./...

# Build the CLI binary
cli:
	go build -o bin/$(CLI_BIN) .

# Build with version information
build-release:
	go build -ldflags="-s -w" -o bin/$(CLI_BIN) .


# Build the Docker image
docker:
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi
	docker build -t $(CLI_BIN) .

.PHONY: all cli test docker clean run-cli run-docker sure build-release test-coverage

# Clean up build artifacts
clean:
	rm -f bin/$(CLI_BIN)
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi

# Run the CLI binary
run-cli: $(CLI_BIN)
	./bin/$(CLI_BIN)  config config.yaml

# Run the Docker container
run-docker: docker
	docker run -d -p 2112:2112 --name $(CLI_BIN) $(CLI_BIN)

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html