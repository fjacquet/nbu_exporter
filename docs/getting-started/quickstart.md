# Quick Start

## 1. Build

```bash
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter
make cli
```

## 2. Configure

Create `config.yaml` with your NetBackup server details. See [Configuration](configuration.md) for all options.

## 3. Run

```bash
./bin/nbu_exporter --config config.yaml
```

The exporter exposes:

- **Metrics**: `http://localhost:9440/metrics`
- **Health**: `http://localhost:9440/health`

## 4. Add to Prometheus

```yaml
scrape_configs:
  - job_name: 'netbackup'
    static_configs:
      - targets: ['localhost:9440']
    scrape_interval: 60s
    scrape_timeout: 30s
```

## Command Line Options

```
Flags:
  -c, --config string   Path to configuration file (required)
  -d, --debug           Enable debug mode
  -h, --help            Help for nbu_exporter
      --trace           Log every NetBackup API response body (live-appliance
                        payload validation; bodies only, never headers; very verbose)
```

### Validating against a live appliance

`--trace` logs the method, URL, status, and body of every NetBackup API
response so payload shapes can be checked against a real master server.
Request and response headers are never logged (the `Authorization` API key
cannot leak), and credential-bearing endpoints (login/token/API-key) are
skipped entirely:

```bash
./bin/nbu_exporter --config config.yaml --trace
curl -s localhost:9440/metrics > /dev/null   # trigger a scrape, then read the log
```
