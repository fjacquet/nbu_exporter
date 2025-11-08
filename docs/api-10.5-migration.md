# NetBackup API 10.5 Migration Guide

This guide provides detailed instructions for upgrading the NBU Exporter to support Veritas NetBackup API version 10.5 (API version 12.0).

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Version Compatibility Matrix](#version-compatibility-matrix)
- [What's Changed](#whats-changed)
- [Upgrade Steps](#upgrade-steps)
- [Configuration Changes](#configuration-changes)
- [Verification](#verification)
- [Rollback Procedure](#rollback-procedure)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

## Overview

The NBU Exporter has been updated to support NetBackup 10.5, which introduces API version 12.0. This upgrade maintains full backward compatibility with existing Prometheus metrics while adapting to the new API version requirements.

### Key Changes

- **API Version Header**: All API requests now include version information in the Accept header
- **Configuration Field**: New `apiVersion` field in configuration (defaults to "12.0")
- **Data Model Updates**: Support for new optional fields in storage and job responses
- **Enhanced Error Handling**: Better error messages for version-related issues

### Breaking Changes

**None** - This is a backward-compatible upgrade. Existing configurations will work with default values.

## Prerequisites

Before upgrading, ensure you have:

1. **NetBackup Server**: Version 10.5 or later
2. **API Access**: Valid API key with appropriate permissions
3. **Current Exporter**: Backup your current configuration and binary
4. **Go Environment** (if building from source): Go 1.23.4 or later

### Checking Your NetBackup Version

```bash
# SSH to your NetBackup master server
bpgetconfig -g | grep VERSION

# Or check via API (if accessible)
curl -k -H "Authorization: YOUR_API_KEY" \
  https://netbackup-master:1556/netbackup/config/servers/master
```

## Version Compatibility Matrix

| NBU Exporter Version | NetBackup API Version | NetBackup Version | Status |
|----------------------|----------------------|-------------------|--------|
| 1.x.x (legacy)       | 10.x                 | 10.0 - 10.4       | Deprecated |
| 2.0.0+               | 12.0                 | 10.5+             | Current |
| 2.0.0+               | 10.x (fallback)      | 10.0 - 10.4       | Limited Support* |

*Limited Support: The exporter may work with older NetBackup versions if the API endpoints remain compatible, but this is not officially tested or supported.

## What's Changed

### API Request Headers

**Before (API 10.x):**
```http
GET /netbackup/storage/storage-units
Accept: application/json
Authorization: YOUR_API_KEY
```

**After (API 12.0):**
```http
GET /netbackup/storage/storage-units
Accept: application/vnd.netbackup+json;version=12.0
Authorization: YOUR_API_KEY
```

### Configuration Schema

**New Field Added:**
```yaml
nbuserver:
    # ... existing fields ...
    apiVersion: "12.0"  # NEW: API version for NetBackup 10.5+
```

### Data Models

**Storage Model** - New optional fields available (automatically handled):
- `storageCategory` - Storage categorization
- `replicationCapable`, `replicationSourceCapable`, `replicationTargetCapable` - Replication capabilities
- `snapshot`, `mirror`, `independent`, `primary` - Snapshot-related flags
- `scaleOutEnabled`, `wormCapable`, `useWorm` - Advanced storage features

**Jobs Model** - New optional field available:
- `kilobytesDataTransferred` - Actual data transferred (vs. estimated `kilobytesTransferred`)

**Note**: These new fields are optional and do not affect existing metric collection. They are parsed if present but not currently exposed as Prometheus metrics.

## Upgrade Steps

### Step 1: Backup Current Setup

```bash
# Backup configuration
cp config.yaml config.yaml.backup

# Backup binary (if applicable)
cp bin/nbu_exporter bin/nbu_exporter.backup

# Note current version
./bin/nbu_exporter --version  # If version flag exists
```

### Step 2: Stop the Exporter

```bash
# If running as systemd service
sudo systemctl stop nbu_exporter

# If running in Docker
docker stop nbu_exporter

# If running manually
# Use Ctrl+C or send SIGTERM to the process
kill -TERM $(pgrep nbu_exporter)
```

### Step 3: Update the Binary

#### Option A: Download Pre-built Binary

```bash
# Download latest release
wget https://github.com/fjacquet/nbu_exporter/releases/download/v2.0.0/nbu_exporter-linux-amd64

# Make executable
chmod +x nbu_exporter-linux-amd64

# Replace existing binary
mv nbu_exporter-linux-amd64 bin/nbu_exporter
```

#### Option B: Build from Source

```bash
# Pull latest code
git pull origin main

# Or clone fresh
git clone https://github.com/fjacquet/nbu_exporter.git
cd nbu_exporter

# Build
make cli

# Binary will be in bin/nbu_exporter
```

### Step 4: Update Configuration

Edit your `config.yaml` to add the API version:

```yaml
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    domain: "my.domain"
    domainType: "NT"
    host: "master.my.domain"
    port: "1556"
    apiVersion: "12.0"  # ADD THIS LINE
    apiKey: "your-api-key-here"
    contentType: "application/vnd.netbackup+json; version=3.0"
    insecureSkipVerify: false
```

**Note**: If you omit `apiVersion`, it will default to "12.0", but explicitly setting it is recommended for clarity.

### Step 5: Validate Configuration

```bash
# Test configuration with debug mode
./bin/nbu_exporter --config config.yaml --debug

# Look for log messages like:
# INFO[0000] Using NetBackup API version: 12.0
# INFO[0001] Successfully connected to NetBackup API
```

Press Ctrl+C after verifying the startup logs.

### Step 6: Start the Exporter

```bash
# If running as systemd service
sudo systemctl start nbu_exporter
sudo systemctl status nbu_exporter

# If running in Docker
docker start nbu_exporter

# If running manually
./bin/nbu_exporter --config config.yaml &
```

### Step 7: Verify Operation

See [Verification](#verification) section below.

## Configuration Changes

### Required Changes

**None** - The exporter will work with existing configurations using default values.

### Recommended Changes

Add the `apiVersion` field for explicit version control:

```yaml
nbuserver:
    apiVersion: "12.0"
```

### Optional Changes

If you want to prepare for future API versions:

```yaml
nbuserver:
    apiVersion: "12.0"  # Can be updated when newer versions are available
```

### Complete Example Configuration

```yaml
---
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/nbu-exporter.log"

nbuserver:
    scheme: "https"
    uri: "/netbackup"
    domain: "production"
    domainType: "NT"
    host: "netbackup-master.example.com"
    port: "1556"
    apiVersion: "12.0"  # NetBackup 10.5 API version
    apiKey: "your-api-key-here"
    contentType: "application/vnd.netbackup+json; version=3.0"
    insecureSkipVerify: false
```

## Verification

### 1. Check Health Endpoint

```bash
curl http://localhost:2112/health

# Expected output:
# {"status":"healthy"}
```

### 2. Check Metrics Endpoint

```bash
curl http://localhost:2112/metrics | grep nbu_

# Expected output should include:
# nbu_storage_bytes{name="...",type="...",size="free"} ...
# nbu_storage_bytes{name="...",type="...",size="used"} ...
# nbu_jobs_count{action="...",policy_type="...",status="..."} ...
# nbu_jobs_size_bytes{action="...",policy_type="...",status="..."} ...
```

### 3. Check Logs

```bash
# View recent logs
tail -f log/nbu-exporter.log

# Look for successful API calls:
# INFO[...] Successfully fetched storage data
# INFO[...] Successfully fetched job data
# INFO[...] Collected X storage units
# INFO[...] Collected Y jobs
```

### 4. Verify in Prometheus

```bash
# Query Prometheus to ensure metrics are being scraped
curl 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}'

# Expected: "value": [timestamp, "1"]
```

### 5. Check Grafana Dashboard

1. Open your Grafana dashboard for NetBackup
2. Verify that panels are displaying data
3. Check that data is current (within scraping interval)

## Rollback Procedure

If you encounter issues after upgrading, follow these steps to rollback:

### Step 1: Stop the New Version

```bash
# Stop the service
sudo systemctl stop nbu_exporter
# Or
docker stop nbu_exporter
# Or
kill -TERM $(pgrep nbu_exporter)
```

### Step 2: Restore Previous Binary

```bash
# Restore from backup
cp bin/nbu_exporter.backup bin/nbu_exporter
```

### Step 3: Restore Previous Configuration

```bash
# Restore configuration
cp config.yaml.backup config.yaml
```

### Step 4: Restart Service

```bash
# Start the service
sudo systemctl start nbu_exporter
sudo systemctl status nbu_exporter
```

### Step 5: Verify Rollback

```bash
# Check health
curl http://localhost:2112/health

# Check metrics
curl http://localhost:2112/metrics | grep nbu_
```

### Step 6: Report Issue

If rollback was necessary, please report the issue:

1. Collect logs: `cat log/nbu-exporter.log > issue-logs.txt`
2. Note your environment details (NetBackup version, OS, etc.)
3. Create an issue: https://github.com/fjacquet/nbu_exporter/issues

## Troubleshooting

### Issue: "406 Not Acceptable" Error

**Symptom:**
```
ERROR Failed to fetch storage data: HTTP 406 Not Acceptable
```

**Cause:** NetBackup server doesn't support API version 12.0

**Solution:**
1. Verify NetBackup version: Must be 10.5 or later
2. If using older NetBackup, try setting `apiVersion: "10.0"` (unsupported)
3. Consider upgrading NetBackup to 10.5+

### Issue: "Invalid API Version Format"

**Symptom:**
```
ERROR Invalid configuration: apiVersion must be in format X.Y
```

**Cause:** API version format is incorrect

**Solution:**
```yaml
# Correct format:
apiVersion: "12.0"

# Incorrect formats:
apiVersion: "12"      # Missing minor version
apiVersion: "v12.0"   # Don't include 'v' prefix
apiVersion: 12.0      # Must be a string, not a number
```

### Issue: Metrics Not Updating

**Symptom:** Prometheus shows stale data or no data

**Diagnosis:**
```bash
# Check exporter logs
tail -f log/nbu-exporter.log

# Check if exporter is running
ps aux | grep nbu_exporter

# Check if port is accessible
curl http://localhost:2112/health
```

**Solutions:**
1. Verify exporter is running
2. Check firewall rules allow access to port 2112
3. Verify NetBackup API is accessible
4. Check API key is valid and not expired

### Issue: TLS Certificate Errors

**Symptom:**
```
ERROR x509: certificate signed by unknown authority
```

**Solutions:**

**Option 1: Install CA Certificate (Recommended)**
```bash
# Copy NetBackup CA cert to system trust store
sudo cp netbackup-ca.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates
```

**Option 2: Disable Verification (Testing Only)**
```yaml
nbuserver:
    insecureSkipVerify: true  # NOT recommended for production
```

### Issue: Authentication Failures

**Symptom:**
```
ERROR Failed to fetch data: HTTP 401 Unauthorized
```

**Solutions:**
1. Verify API key is correct
2. Check API key hasn't expired
3. Verify API key has required permissions:
   - Read access to `/storage/storage-units`
   - Read access to `/admin/jobs`
4. Generate new API key if necessary

### Issue: High Memory Usage

**Symptom:** Exporter consuming excessive memory

**Diagnosis:**
```bash
# Check memory usage
ps aux | grep nbu_exporter

# Check job count
curl http://localhost:2112/metrics | grep nbu_jobs_count
```

**Solutions:**
1. Reduce `scrapingInterval` to collect fewer jobs:
   ```yaml
   server:
       scrapingInterval: "30m"  # Instead of "1h"
   ```
2. Verify pagination is working correctly in logs
3. Check for memory leaks (report if found)

### Issue: Slow Metric Collection

**Symptom:** Scrapes taking longer than expected

**Diagnosis:**
```bash
# Enable debug mode to see timing
./bin/nbu_exporter --config config.yaml --debug

# Look for timing information in logs
```

**Solutions:**
1. Check network latency to NetBackup server
2. Reduce `scrapingInterval` to fetch fewer jobs
3. Verify NetBackup API performance
4. Check if pagination is working (should see multiple API calls for large datasets)

### Issue: Missing Metrics

**Symptom:** Some expected metrics are not present

**Diagnosis:**
```bash
# Check what metrics are available
curl http://localhost:2112/metrics | grep nbu_

# Check logs for errors
grep ERROR log/nbu-exporter.log
```

**Solutions:**
1. Verify NetBackup has data to export (storage units, jobs)
2. Check time range - jobs older than `scrapingInterval` won't be collected
3. Verify API permissions allow access to required endpoints
4. Check if tape storage is being filtered (expected behavior)

## FAQ

### Q: Do I need to upgrade if I'm using NetBackup 10.4 or earlier?

**A:** No, but it's recommended. The new exporter version defaults to API version 12.0, which requires NetBackup 10.5+. If you must use an older NetBackup version, you can try setting `apiVersion: "10.0"`, but this is not officially supported.

### Q: Will my existing Prometheus queries and Grafana dashboards still work?

**A:** Yes, all metric names and labels remain unchanged. Your existing queries and dashboards will continue to work without modification.

### Q: What happens if I don't add the `apiVersion` field to my config?

**A:** The exporter will use the default value of "12.0", which is correct for NetBackup 10.5+. However, explicitly setting it is recommended for clarity.

### Q: Can I use different API versions for different NetBackup servers?

**A:** Currently, the exporter supports one API version per instance. If you need to monitor multiple NetBackup servers with different versions, run separate exporter instances with different configurations.

### Q: Are there any new metrics available with API 10.5?

**A:** The API 10.5 response includes new optional fields (storage categories, replication capabilities, etc.), but these are not currently exposed as Prometheus metrics. They may be added in future versions based on user feedback.

### Q: How do I know which API version my NetBackup server supports?

**A:** NetBackup 10.5 and later support API version 12.0. You can also check the NetBackup API documentation or contact your NetBackup administrator.

### Q: Will this upgrade affect my historical metrics in Prometheus?

**A:** No, historical metrics are preserved. The upgrade only affects how new metrics are collected going forward.

### Q: Do I need to restart Prometheus after upgrading the exporter?

**A:** No, Prometheus will automatically detect the updated exporter on the next scrape. However, you may want to reload Prometheus configuration if you changed any scrape settings.

### Q: Can I test the upgrade without affecting production?

**A:** Yes, recommended approach:
1. Deploy the new exporter to a test environment
2. Point it at your production NetBackup server (read-only operations)
3. Verify metrics collection works correctly
4. Then upgrade production exporter

### Q: What's the performance impact of this upgrade?

**A:** Minimal to none. The only change is the Accept header in API requests, which adds negligible overhead.

### Q: How often should I update the exporter?

**A:** Follow these guidelines:
- **Security updates**: Apply immediately
- **Bug fixes**: Apply within 1-2 weeks
- **Feature updates**: Apply during planned maintenance windows
- **Major versions**: Test thoroughly before production deployment

## Automated Verification

A verification script is available to test deployment procedures:

```bash
# Run deployment verification tests
./scripts/verify-deployment.sh
```

This script verifies:
- Binary and configuration files
- Backward compatibility with configs missing API version
- Default API version behavior
- Unit tests for configuration validation

## Additional Resources

- [Deployment Verification Guide](deployment-verification.md) - Comprehensive deployment and rollback procedures
- [NetBackup 10.5 API Documentation](https://sort.veritas.com/public/documents/nbu/10.5/)
- [NBU Exporter GitHub Repository](https://github.com/fjacquet/nbu_exporter)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)

## Support

If you encounter issues not covered in this guide:

1. **Check Logs**: Enable debug mode for detailed information
2. **Search Issues**: Check existing GitHub issues for similar problems
3. **Create Issue**: Report new issues with logs and environment details
4. **Community**: Ask questions in GitHub Discussions

**GitHub Issues**: https://github.com/fjacquet/nbu_exporter/issues

## Changelog

See [CHANGELOG.md](../CHANGELOG.md) for detailed version history and all changes.
