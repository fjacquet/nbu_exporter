# Configuration

Create a `config.yaml` file:

```yaml
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    uri: "/netbackup"
    domain: "my.domain"
    domainType: "NT"
    host: "master.my.domain"
    port: "1556"
    apiVersion: "13.0"  # Optional: auto-detects if omitted
    apiKey: "your-api-key-here"
    contentType: "application/vnd.netbackup+json; version=13.0"
    insecureSkipVerify: false

# Optional: OpenTelemetry distributed tracing
# opentelemetry:
#     enabled: true
#     endpoint: "localhost:4317"
#     insecure: true
#     samplingRate: 0.1
```

!!! warning
    Never commit API keys to version control. Use environment variables or secure secret management.

## Server Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | Server bind address |
| `port` | string | Yes | Server port (1-65535) |
| `uri` | string | Yes | Metrics endpoint path |
| `scrapingInterval` | duration | Yes | Time window for job collection (e.g., "1h", "30m") |
| `logName` | string | Yes | Log file path |

## NBU Server Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scheme` | string | Yes | Protocol (http/https) |
| `uri` | string | Yes | API base path |
| `domain` | string | Yes | NetBackup domain |
| `domainType` | string | Yes | Domain type (NT, vx, etc.) |
| `host` | string | Yes | NetBackup master server hostname |
| `port` | string | Yes | API port (typically 1556) |
| `apiVersion` | string | No | API version (13.0, 12.0, or 3.0). Auto-detects if omitted. |
| `apiKey` | string | Yes | NetBackup API key |
| `contentType` | string | Yes | API content type header |
| `insecureSkipVerify` | bool | No | Skip TLS certificate verification |

## OpenTelemetry Section (Optional)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable OpenTelemetry tracing (default: false) |
| `endpoint` | string | Yes* | OTLP gRPC endpoint (e.g., "localhost:4317") |
| `insecure` | bool | No | Use insecure connection (default: false) |
| `samplingRate` | float | No | Sampling rate 0.0-1.0 (default: 1.0) |

*Required when `enabled` is `true`

See [Configuration Examples](../config-examples/README.md) for complete sample configs.
