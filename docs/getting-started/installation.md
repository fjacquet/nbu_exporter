# Installation

## Prerequisites

- Go 1.25 or later
- Veritas NetBackup 10.0 or later
- Access to NetBackup REST API
- NetBackup API key (generated from NBU UI)

## Homebrew

The quickest way to install on macOS or Linux:

```bash
brew install fjacquet/tap/nbu_exporter
```

Or tap the repository first, then install:

```bash
brew tap fjacquet/tap
brew install nbu_exporter
```

Upgrade to the latest release:

```bash
brew upgrade nbu_exporter
```

Verify the installation:

```bash
nbu_exporter --version
```

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
