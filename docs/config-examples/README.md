# Configuration Examples

This directory contains example configuration files for different NetBackup deployment scenarios.

## Available Examples

### 1. Automatic Version Detection (Recommended)

**File:** `config-auto-detect.yaml`

**Use when:**
- Deploying across multiple NetBackup versions
- You want a single configuration that adapts automatically
- Upgrading NetBackup and want the exporter to detect the new version

**Features:**
- No `apiVersion` field specified
- Automatically detects highest supported version (14.0 → 13.0 → 12.0 → 10.0)
- Works with NetBackup 10.0, 10.5, 11.0, and 11.2 without changes
- Adds 1-3 seconds to startup for version detection

**Best for:** Mixed environments, future-proof deployments

---

### 2. NetBackup 11.0 (Explicit Version)

**File:** `config-netbackup-11.yaml`

**Use when:**
- Running NetBackup 11.0 or later
- You want to explicitly use API version 13.0
- You want faster startup (skips version detection)

**Features:**
- `apiVersion: "13.0"` explicitly configured
- Immediate connection without detection
- Takes advantage of latest API features

**Best for:** NetBackup 11.0 production environments

---

### 3. NetBackup 10.5 (Explicit Version)

**File:** `config-netbackup-10.5.yaml`

**Use when:**
- Running NetBackup 10.5
- You want to explicitly use API version 12.0
- Maintaining backward compatibility with existing configurations

**Features:**
- `apiVersion: "12.0"` explicitly configured
- Compatible with previous exporter versions
- Also works with NetBackup 11.0 (backward compatible)

**Best for:** NetBackup 10.5 environments, backward compatibility

---

### 4. NetBackup 10.0-10.4 (Legacy Support)

**File:** `config-netbackup-10.0.yaml`

**Use when:**
- Running NetBackup 10.0, 10.1, 10.2, 10.3, or 10.4
- You want to explicitly use API version 10.0
- Maintaining legacy NetBackup environments

**Features:**
- `apiVersion: "10.0"` explicitly configured
- Legacy API support
- Core metrics (jobs, storage) fully functional

**Best for:** Legacy NetBackup 10.0-10.4 environments

---

### 5. Multi-Site (multiple primary servers)

**File:** `config-multisite.yaml`

**Use when:**
- You run more than one NetBackup primary server (typically one per site)
- You want a single exporter instance to scrape them all
- You want every metric labelled with its `site`

**Features:**
- A `nbuservers:` list with one entry per server, each with a unique `site`
- A background collection loop polls every site on `collectionInterval`
  (default 5m); scrapes read the latest snapshot, so backend API load is
  decoupled from scrape frequency
- Every metric series carries a `site` label (first label)
- A down site shows only `nbu_up{site="..."}=0` and never affects the others
- The legacy single `nbuserver:` block still works (auto-mapped to a one-entry
  list whose `site` defaults to the host)

**Best for:** Two-or-more-primary deployments; per-site dashboards and alerts

---

## Quick Start

### Step 1: Choose Your Configuration

Select the configuration file that matches your NetBackup version:

| NetBackup Version | Recommended File | API Version |
|-------------------|------------------|-------------|
| 11.0+             | `config-netbackup-11.yaml` | 13.0 |
| 10.5              | `config-netbackup-10.5.yaml` | 12.0 |
| 10.0-10.4         | `config-netbackup-10.0.yaml` | 10.0 |
| Any/Mixed         | `config-auto-detect.yaml` | Auto |
| Multiple servers  | `config-multisite.yaml` | Per-site |

### Step 2: Copy and Customize

```bash
# Copy the example to your workspace
cp docs/config-examples/config-auto-detect.yaml config.yaml

# Edit the configuration
nano config.yaml
```

### Step 3: Update Required Fields

Replace these placeholders with your actual values:

```yaml
nbuserver:
    host: "nbu-master.my.domain"  # Your NetBackup master server hostname
    apiKey: "your-api-key-here"   # Your NetBackup API key
```

### Step 4: Run the Exporter

```bash
./bin/nbu_exporter --config config.yaml
```

---

## Configuration Field Reference

### Server Section

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `host` | string | Yes | - | Server bind address (e.g., "localhost", "0.0.0.0") |
| `port` | string | Yes | - | Server port (1-65535, typically "9440") |
| `uri` | string | Yes | - | Metrics endpoint path (typically "/metrics") |
| `scrapingInterval` | duration | Yes | - | Job lookback window per collection (e.g., "30m", "1h", "2h") |
| `collectionInterval` | duration | No | "5m" | How often the background loop polls every site. Backend API load is driven by this, not by scrape frequency. The effective job lookback window is `max(scrapingInterval, collectionInterval)`. |
| `logName` | string | Yes | - | Log file path (e.g., "log/nbu-exporter.log") |

### NBU Server Section

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `scheme` | string | Yes | - | Protocol ("http" or "https") |
| `uri` | string | Yes | - | API base path (typically "/netbackup") |
| `domain` | string | Yes | - | NetBackup domain name |
| `domainType` | string | Yes | - | Domain type (e.g., "NT", "vx") |
| `host` | string | Yes | - | NetBackup master server hostname |
| `port` | string | Yes | - | API port (typically "1556") |
| `apiVersion` | string | No | Auto-detect | API version ("13.0", "12.0", "10.0", or omit for auto-detection) |
| `apiKey` | string | Yes | - | NetBackup API key (generate from NetBackup UI) |
| `contentType` | string | Yes | - | API content type header |
| `insecureSkipVerify` | bool | No | false | Skip TLS verification (not recommended for production) |

### NBU Servers Section (multi-site)

For multiple primary servers, use a `nbuservers:` list **instead of** the single
`nbuserver:` block. Each entry takes the same fields as the NBU Server section
above, plus a required, unique `site`:

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `site` | string | Yes | - | Unique site identifier; emitted as the `site` label on every metric from this server |
| *(all NBU Server fields)* | - | - | - | Same as the single-server section above |

A legacy single `nbuserver:` block is automatically promoted to a one-entry
`nbuservers:` list whose `site` defaults to the host, so existing single-site
configurations keep working unchanged. See `config-multisite.yaml`.

---

## Version Detection Behavior

When `apiVersion` is **not specified**, the exporter performs automatic detection:

### Detection Process

The exporter tries supported API versions in descending order and uses the first
that responds successfully:

1. **Try API 14.0** (NetBackup 11.2+)
   - Makes lightweight API call with version 14.0
   - If successful (HTTP 200), uses version 14.0
   - If not supported (HTTP 406), tries next version

2. **Try API 13.0** (NetBackup 11.0+)
   - Makes lightweight API call with version 13.0
   - If successful (HTTP 200), uses version 13.0
   - If not supported (HTTP 406), tries next version

3. **Try API 12.0** (NetBackup 10.5)
   - Makes lightweight API call with version 12.0
   - If successful (HTTP 200), uses version 12.0
   - If not supported (HTTP 406), tries next version

4. **Try API 10.0** (NetBackup 10.0-10.4)
   - Makes lightweight API call with version 10.0
   - If successful (HTTP 200), uses version 10.0
   - If not supported, reports error

### Detection Characteristics

- **Startup Time:** Adds 1-3 seconds (one API call per version attempt)
- **Retry Logic:** Automatically retries transient failures with exponential backoff
- **Error Handling:** Distinguishes version incompatibility from network/auth errors
- **Logging:** Each attempt is logged for troubleshooting
- **Reachability:** Detection runs at startup and contacts the appliance. When
  `apiVersion` is omitted and the appliance is unreachable, startup **fails fast**
  rather than starting and failing on every scrape. Set an explicit `apiVersion` to
  start without contacting the appliance first.

### Startup Log Example

```
INFO[0000] Starting NBU Exporter
INFO[0001] Attempting API version detection
DEBUG[0001] Trying API version 13.0
WARN[0002] API version 13.0 not supported (HTTP 406), trying next version
DEBUG[0002] Trying API version 12.0
INFO[0003] Detected NetBackup API version: 12.0
INFO[0003] Successfully connected to NetBackup API
```

---

## Choosing Between Auto-Detection and Explicit Version

### Use Auto-Detection When:

✅ Deploying across multiple NetBackup versions  
✅ You want a single configuration for all environments  
✅ You're upgrading NetBackup and want automatic adaptation  
✅ You don't mind 1-3 seconds additional startup time  
✅ You want future-proof configuration  

### Use Explicit Version When:

✅ Running a single NetBackup version  
✅ You want faster startup (< 1 second)  
✅ You want predictable, consistent behavior  
✅ You're in a production environment with strict requirements  
✅ You want to lock to a specific API version  

---

## Environment-Specific Recommendations

### Development/Testing

**Recommended:** Auto-detection (`config-auto-detect.yaml`)

**Rationale:**
- Flexibility to test against different NetBackup versions
- Single configuration for all test environments
- Easy to switch between NetBackup versions

### Staging

**Recommended:** Explicit version matching production

**Rationale:**
- Staging should mirror production configuration
- Predictable behavior for testing
- Faster startup for performance testing

### Production

**Recommended:** Explicit version (`config-netbackup-11.yaml`, `config-netbackup-10.5.yaml`, etc.)

**Rationale:**
- Faster startup (no detection overhead)
- Predictable, consistent behavior
- Explicit configuration is easier to audit
- Reduces potential for unexpected behavior

### Mixed Environments

**Recommended:** Auto-detection (`config-auto-detect.yaml`)

**Rationale:**
- Single configuration works across all servers
- Automatically adapts to each NetBackup version
- Simplifies deployment and maintenance

---

## Security Best Practices

### API Key Management

❌ **Don't:**
- Commit API keys to version control
- Share API keys between environments
- Use the same API key for all exporters

✅ **Do:**
- Use environment variables for API keys
- Generate separate API keys per environment
- Rotate API keys regularly
- Use secret management systems (Vault, AWS Secrets Manager, etc.)

### TLS Configuration

❌ **Don't:**
- Set `insecureSkipVerify: true` in production
- Disable TLS certificate verification

✅ **Do:**
- Keep `insecureSkipVerify: false` (default)
- Install proper CA certificates
- Use valid TLS certificates on NetBackup servers

### Network Security

✅ **Best Practices:**
- Restrict exporter access to NetBackup API (firewall rules)
- Use network segmentation
- Monitor exporter access logs
- Implement rate limiting if needed

---

## Troubleshooting

### Configuration Validation

Test your configuration before deployment:

```bash
# Test configuration syntax
./bin/nbu_exporter --config config.yaml --debug

# Check logs for errors
tail -f log/nbu-exporter.log
```

### Common Issues

**Issue:** Configuration file not found

```bash
Error: config file not found: config.yaml
```

**Solution:** Ensure the file path is correct and the file exists.

---

**Issue:** Invalid API version

```bash
Error: invalid configuration: apiVersion must match format X.Y
```

**Solution:** Use valid API versions: "10.0", "12.0", or "13.0"

---

**Issue:** Version detection fails

```bash
ERROR: Failed to detect compatible NetBackup API version
```

**Solution:** 
1. Verify NetBackup version is 10.0 or later
2. Check network connectivity to NetBackup server
3. Verify API key is valid
4. Try explicit version configuration

---

## Additional Resources

- [Main README](../../README.md) - Complete documentation
- [Migration Guide](../netbackup-11-migration.md) - Upgrade instructions
- [API 10.5 Migration](../api-10.5-migration.md) - Previous migration guide
- [NetBackup API Documentation](https://sort.veritas.com/public/documents/nbu/)

---

## Support

For issues and questions:
- **GitHub Issues:** https://github.com/fjacquet/nbu_exporter/issues
- **Discussions:** https://github.com/fjacquet/nbu_exporter/discussions

---

**Last Updated:** 2025-01-15  
**Exporter Version:** Latest (with multi-version support)
