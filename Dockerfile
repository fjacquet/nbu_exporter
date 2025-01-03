# Stage 1: Build
FROM golang:latest AS builder

# Set the working directory
WORKDIR /app

# Copy and download dependency using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o nbu_exporter

# Stage 2: Runtime
FROM alpine:latest

# Set up the working directory
WORKDIR /root/

# Copy the binary from the builder
COPY --from=builder /app/nbu_exporter /usr/bin/nbu_exporter

# Copy the configuration file (if needed)
COPY config.yaml /etc/config.yaml

# Expose the default port
EXPOSE 8080

# Run the exporter with the default configuration
# ENTRYPOINT ["/usr/bin/nbu_exporter"]
ENTRYPOINT ["/bin/sh"]
# CMD ["/usr/bin/nbu_exporter", "--config", "/etc/config.yaml"]
