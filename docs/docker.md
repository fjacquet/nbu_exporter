# Docker Deployment

## Build Image

```bash
# Using Makefile
make docker

# Or manually
docker build -t nbu_exporter .
```

The Docker image uses a multi-stage build with Alpine Linux for minimal footprint.

## Run Container

```bash
# Using Makefile (port 2112)
make run-docker

# Manual with custom configuration
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config.yaml:/etc/nbu_exporter/config.yaml \
  -v $(pwd)/log:/var/log/nbu_exporter \
  nbu_exporter
```

## Docker Compose

```yaml
version: '3.8'
services:
  nbu_exporter:
    image: nbu_exporter:latest
    container_name: nbu_exporter
    ports:
      - "2112:2112"
    volumes:
      - ./config.yaml:/etc/nbu_exporter/config.yaml:ro
      - ./log:/var/log/nbu_exporter
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:2112/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

## Docker Compose with OpenTelemetry

See the [OpenTelemetry Setup Guide](opentelemetry-example.md) for a complete Docker Compose stack with OpenTelemetry Collector and Jaeger.

## Verify Container

```bash
docker logs nbu_exporter
curl http://localhost:2112/metrics
curl http://localhost:2112/health
```
