# Configuration

Create a `config.yaml` file:

```yaml
server:
    host: "localhost"
    port: "9440"
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
    insecureSkipVerify: false  # or "${NBU1_SKIP_CERTIFICATE}" — see Environment Variables below

# Optional: OpenTelemetry distributed tracing
# opentelemetry:
#     enabled: true
#     endpoint: "localhost:4317"
#     insecure: true
#     samplingRate: 0.1
```

!!! warning
    Never commit API keys to version control. Use environment variables or secure secret management.

## Multiple Sites

To monitor more than one NetBackup primary server from a single exporter, replace the single
`nbuserver:` block with a `nbuservers:` list — one entry per site, each with a **required, unique
`site`**. Every metric is then labelled with its `site`, and a background loop collects all sites
on `server.collectionInterval` (default `5m`), so backend API load is decoupled from Prometheus
scrape frequency.

```yaml
server:
    host: "0.0.0.0"
    port: "9440"
    uri: "/metrics"
    scrapingInterval: "1h"
    collectionInterval: "5m"   # how often every site is polled (default 5m)
    logName: "log/nbu-exporter.log"

nbuservers:
    - site: "paris"
      scheme: "https"
      uri: "/netbackup"
      host: "nbu-paris.my.domain"
      port: "1556"
      apiVersion: "13.0"        # optional; omit to auto-detect
      apiKey: "your-paris-api-key"
      insecureSkipVerify: false
    - site: "lyon"
      scheme: "https"
      uri: "/netbackup"
      host: "nbu-lyon.my.domain"
      port: "1556"
      apiKey: "your-lyon-api-key"
```

An unreachable site reports only `nbu_up{site="..."}=0` and never affects the others. A legacy
single `nbuserver:` block is automatically mapped to a one-entry list (with `site` defaulting to
the host), so existing single-site configurations keep working unchanged. See
[`config-multisite.yaml`](../config-examples/config-multisite.yaml).

## Environment Variables / .env loading

The `host` and `apiKey` fields in the `nbuserver` section support `${VAR}` interpolation.
At startup the exporter expands every `${VAR}` reference and fails loudly if a variable is not set.

`insecureSkipVerify` accepts either a native boolean (`insecureSkipVerify: true`) or the same
`${VAR}` interpolation (`insecureSkipVerify: "${NBU1_SKIP_CERTIFICATE}"`), resolved at startup
alongside `host`/`apiKey`. This lets you toggle TLS verification per environment (e.g. a lab
override) without editing `config.yaml`.

`nbu_exporter` loads a `.env` file natively at startup — you do not need a shell wrapper or
`export` statements. It looks for `.env` in the working directory first, then in the same
directory as `config.yaml`. Already-set environment variables always win: values in `.env`
never shadow real env/secret injection.

**One-server quickstart** — copy `.env.example` to `.env`, fill in your values, and reference
them from `config.yaml`:

```bash
cp .env.example .env
# edit .env — set NBU1_HOSTNAME and NBU1_APIKEY
```

```yaml
nbuserver:
  host: "${NBU1_HOSTNAME}"
  apiKey: "${NBU1_APIKEY}"
```

The `NBU1_*` variables are passed into the container by `docker-compose.yml` from the `.env`
file automatically.

**Multi-site** — use the `nbuservers:` list (see [Multiple Sites](#multiple-sites) above). Each
entry's `host` and `apiKey` support the same `${VAR}` interpolation:

```yaml
nbuservers:
  - site: "paris"
    host: "${NBU_PARIS_HOST}"
    apiKey: "${NBU_PARIS_KEY}"
    # ...
  - site: "lyon"
    host: "${NBU_LYON_HOST}"
    apiKey: "${NBU_LYON_KEY}"
    # ...
```

The `.env` / `NBU1_*` naming is a single-server convenience; there is no limit on variable names.

### API keys with special characters

The API key is sent verbatim as the value of the HTTP `Authorization` header — it is
never URL-encoded and never placed in a request body, so any character is safe end to
end. The only place quoting matters is **parsing at load time**, and it differs by where
you put the key:

| Source | Rule |
|---|---|
| `.env`, single-quoted `'…'` | Fully literal — no `$` expansion, no `\` escapes, no `#` comment. Best default. Cannot contain a literal `'`. |
| `.env`, double-quoted `"…"` | Expands `$VAR`/`${VAR}` and processes `\` escapes. `$`, `\`, `"` are special — write `\$`, `\\`, `\"`. |
| `.env`, unquoted | `$VAR` expands; a ` #` (space-hash) starts a comment; a value **starting** with `'`/`"` is treated as quoted. |
| `config.yaml` inline | Only the exact `${NAME}` token is interpolated (`os.LookupEnv`), so a literal key containing `${NAME}` is treated as an env ref. Prefer referencing an env var. |

For quotes inside the key specifically: use double quotes to include a `'`, single
quotes to include a `"`. NetBackup API keys are token strings (dot-separated,
base64url-style segments), so in practice they contain none of these characters and an
unquoted value works fine. When referencing an env var from `config.yaml`
(`apiKey: "${NBU1_APIKEY}"`) the value is inserted verbatim and never re-scanned, so the
env var itself may contain `$`, `${…}`, or any character.

## Server Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `host` | string | Yes | Server bind address |
| `port` | string | Yes | Server port (1-65535) |
| `uri` | string | Yes | Metrics endpoint path |
| `scrapingInterval` | duration | Yes | Job lookback window per collection (e.g., "1h", "30m") |
| `collectionInterval` | duration | No | Background poll interval for every site (default "5m"). Effective job window = max(scrapingInterval, collectionInterval). |
| `logName` | string | Yes | Log file path |

## NBU Server Section

For multiple servers, use a `nbuservers:` list instead of this single block — each entry takes
the same fields plus a required, unique `site` (see [Multiple Sites](#multiple-sites)).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scheme` | string | Yes | Protocol (http/https) |
| `uri` | string | Yes | API base path |
| `domain` | string | Yes | NetBackup domain |
| `domainType` | string | Yes | Domain type (NT, vx, etc.) |
| `host` | string | Yes | NetBackup master server hostname |
| `port` | string | Yes | API port (typically 1556) |
| `apiVersion` | string | No | API version (14.0, 13.0, 12.0, or 10.0). Auto-detects if omitted. |
| `apiKey` | string | Yes | NetBackup API key |
| `contentType` | string | Yes | API content type header |
| `insecureSkipVerify` | bool or `${VAR}` | No | Skip TLS certificate verification. Native bool or a `${VAR}` reference (e.g. `${NBU1_SKIP_CERTIFICATE}`) resolved at startup; defaults to `false`. |

## OpenTelemetry Section (Optional)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | bool | No | Enable OpenTelemetry tracing (default: false) |
| `endpoint` | string | Yes* | OTLP gRPC endpoint (e.g., "localhost:4317") |
| `insecure` | bool | No | Use insecure connection (default: false) |
| `samplingRate` | float | No | Sampling rate 0.0-1.0 (default: 1.0) |

*Required when `enabled` is `true`

See [Configuration Examples](../config-examples/README.md) for complete sample configs.
