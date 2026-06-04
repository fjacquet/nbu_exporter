# Stage 1: Build
FROM golang:1.26 AS builder

WORKDIR /app

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Static build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o nbu_exporter .

# Stage 2: Runtime
FROM alpine:latest

# ca-certificates for HTTPS, plus a non-root user and writable log dir
RUN apk --no-cache add ca-certificates && \
    adduser -D -u 10001 nbu && \
    mkdir -p /var/log/nbu_exporter && \
    chown nbu:nbu /var/log/nbu_exporter

# Copy the binary and default config
COPY --from=builder /app/nbu_exporter /usr/bin/nbu_exporter
COPY config.yaml /etc/nbu_exporter/config.yaml

# Expose the default port (configurable via config.yaml)
EXPOSE 2112

USER nbu

ENTRYPOINT ["/usr/bin/nbu_exporter"]
CMD ["--config", "/etc/nbu_exporter/config.yaml"]
