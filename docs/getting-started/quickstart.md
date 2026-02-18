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

- **Metrics**: `http://localhost:2112/metrics`
- **Health**: `http://localhost:2112/health`

## 4. Add to Prometheus

```yaml
scrape_configs:
  - job_name: 'netbackup'
    static_configs:
      - targets: ['localhost:2112']
    scrape_interval: 60s
    scrape_timeout: 30s
```

## Command Line Options

```
Flags:
  -c, --config string   Path to configuration file (required)
  -d, --debug           Enable debug mode
  -h, --help            Help for nbu_exporter
```
