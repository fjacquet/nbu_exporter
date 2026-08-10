# Stage 1: Build
FROM docker.io/library/golang:1.26 AS builder

WORKDIR /app

# Copy and download dependencies using go mod
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Static build
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o nbu_exporter .

# Stage 2: Runtime
FROM docker.io/library/alpine:latest

# Create the runtime user and log dir. These are busybox builtins (no network).
RUN adduser -D -u 10001 nbu && \
    mkdir -p /var/log/nbu_exporter && \
    chown nbu:nbu /var/log/nbu_exporter

# Copy the CA bundle from the builder stage instead of `apk add ca-certificates`.
# The latter fetches from the Alpine CDN over TLS, which fails behind a corporate
# MITM proxy: the bare alpine image has no CA bundle yet to validate the proxy
# cert (chicken-and-egg). The Debian-based golang builder already ships the bundle.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Copy the binary and default config
COPY --from=builder /app/nbu_exporter /usr/bin/nbu_exporter
COPY config.yaml /etc/nbu_exporter/config.yaml

# Expose the default port (configurable via config.yaml)
EXPOSE 9440

# /livez never depends on target reachability or the collection cycle, so it
# can never flag a healthy process as down over an unreachable NetBackup
# master (see ADR-0005 / architecture.md "Health probes").
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1:9440/livez || exit 1

USER nbu

ENTRYPOINT ["/usr/bin/nbu_exporter"]
CMD ["--config", "/etc/nbu_exporter/config.yaml"]
