# Define the output binaries
CLI_BIN = nbu_exporter

# Default target: build both binaries
all: cli test docker

test:
	go test

# Build the CLI binary
cli:
	go build -o bin/$(CLI_BIN) .


# Build the Docker image
docker:
	@if [ -n "$(shell docker images -q $(CLI_BIN) 2> /dev/null)" ]; then \
		docker image rm -f $(CLI_BIN); \
	fi
	docker build -t $(CLI_BIN) .

.PHONY: docker clean run-cli run-web run-docker

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
	docker run -d -p 8080:8080 --name $(CLI_BIN) $(CLI_BIN)