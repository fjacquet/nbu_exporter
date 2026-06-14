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

## Quickstart demo stack (Docker Compose)

The repository ships a self-contained demo stack — the exporter plus Prometheus and Grafana — so you
can see metrics, alerts, and dashboards end to end with a single command.

### Start the stack

```bash
# Build the exporter locally, then start exporter + Prometheus + Grafana
docker compose up -d

# Or pull the published image instead of building
docker compose -f docker-compose.ghcr.yml up -d
```

The build variant uses `docker-compose.yml` and builds the exporter from the local source. The ghcr
variant uses `docker-compose.ghcr.yml`, which pulls `ghcr.io/fjacquet/nbu_exporter:latest`; the two
files are otherwise identical.

### Service URLs

| Service | URL | Notes |
|---|---|---|
| Exporter metrics | <http://localhost:2112/metrics> | Scraped by the in-stack Prometheus |
| Prometheus | <http://localhost:9090> | Loads alerting rules from `deploy/prometheus/nbu.rules.yml` |
| Grafana | <http://localhost:3000> | Default login `admin` / `admin` |

Override the Grafana credentials with the `GF_ADMIN_USER` and `GF_ADMIN_PASSWORD` environment
variables (for example in a gitignored `.env` file).

### Host ports

If a host port is already in use (for example another Prometheus on `9090`), override the published
port without editing the compose files:

```bash
NBU_EXPORTER_PORT=12112 PROMETHEUS_PORT=19090 GRAFANA_PORT=13000 docker compose up -d
```

The defaults are `2112` (exporter), `9090` (Prometheus), and `3000` (Grafana).

### Logs

The exporter logs to stdout (`logName: "stdout"` in `config.yaml`), so view logs with
`docker compose logs -f nbu_exporter`. No log directory or bind mount is required; set `logName` to a
file path in `config.yaml` if you also want a log file.

### Auto-provisioning

The Grafana datasource and the NetBackup dashboards are provisioned automatically — no manual import
is needed. See [Dashboards](dashboards.md) for what is included. Prometheus loads the alerting rules
from `deploy/prometheus/nbu.rules.yml`.

### Configuration

`config.yaml` is the source of truth for the exporter; it is mounted read-only into the container. For
the single-target quickstart, set `NBU1_HOSTNAME` and `NBU1_APIKEY` (for example in a gitignored
`.env` file) so the exporter knows which NetBackup master to query.

### Tear down

```bash
# Stop the stack and remove the named volumes (Prometheus + Grafana data)
docker compose down -v
```

## Verify Container

```bash
docker logs nbu_exporter
curl http://localhost:2112/metrics
curl http://localhost:2112/health
```
