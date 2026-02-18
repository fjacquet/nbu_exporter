# Installation

## Prerequisites

- Go 1.25 or later
- Veritas NetBackup 10.0 or later
- Access to NetBackup REST API
- NetBackup API key (generated from NBU UI)

## From Source

```bash
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter
make cli
```

The binary will be output to `bin/nbu_exporter`.

## Docker

```bash
# Build the image
make docker

# Run
make run-docker
```

See the [Docker deployment guide](../docker.md) for advanced usage.

## GitHub Releases

Download pre-built binaries from the [Releases page](https://github.com/fjacquet/nbu_exporter/releases).
