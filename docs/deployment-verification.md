# Deployment Verification and Rollback Procedures

## Overview

This document provides comprehensive procedures for deploying the NBU Exporter with NetBackup API 10.5 support, verifying the deployment, and rolling back if issues occur.

## Pre-Deployment Checklist

### Environment Requirements

- [ ] NetBackup Server version 10.5 or later installed
- [ ] Valid NetBackup API key with required permissions
- [ ] Network connectivity from exporter host to NetBackup master server (port 1556)
- [ ] Prometheus instance configured to scrape the exporter
- [ ] Backup of current configuration and binary

### Verification Commands

```bash
# Check NetBackup version
ssh netbackup-master "bpgetconfig -g | grep VERSION"

# Test API connectivity
curl -k -H "Authorization: YOUR_API_KEY" \
  https://netbackup-master:1556/netbackup/config/servers/master

# Verify current exporter is running
ps aux | grep nbu_exporter
curl http://localhost:2112/health
```

## Deployment Procedures

### Method 1: Binary Upgrade (Recommended)

#### Step 1: Backup Current Setup

```bash
# Create backup directory
mkdir -p backups/$(date +%Y%m%d)

# Backup configuration
cp config.yaml backups/$(date +%Y%m%d)/config.yaml.backup

# Backup binary
cp bin/nbu_exporter backups/$(date +%Y%m%d)/nbu_exporter.backup

# Backup logs (optional)
cp -r log backups/$(date +%Y%m%d)/log.backup
```

#### Step 2: Stop Current Exporter

```bash
# If running as systemd service
sudo systemctl stop nbu_exporter
sudo systemctl status nbu_exporter  # Verify stopped

# If running in Docker
docker stop nbu_exporter
docker ps | grep nbu_exporter  # Verify stopped

# If running manually
pkill -TERM nbu_exporter
# Wait 10 seconds for graceful shutdown
sleep 10
ps aux | grep nbu_exporter  # Verify stopped
```

#### Step 3: Update Binary

```bash
# Option A: Download pre-built binary
wget https://github.com/fjacquet/nbu_exporter/releases/download/v2.0.0/nbu_exporter-linux-amd64
chmod +x nbu_exporter-linux-amd64
mv nbu_exporter-linux-amd64 bin/nbu_exporter

# Option B: Build from source
git pull origin main
make cli
# Binary will be in bin/nbu_exporter
```

#### Step 4: Update Configuration

```bash
# Edit config.yaml to add API version
cat >> config.yaml << 'EOF'
# Add under nbuserver section:
    apiVersion: "12.0"  # NetBackup API version for 10.5+
EOF

# Or use sed to add it programmatically
sed -i '/nbuserver:/a\    apiVersion: "12.0"' config.yaml
```

#### Step 5: Validate Configuration

```bash
# Test configuration with debug mode (don't start service yet)
./bin/nbu_exporter --config config.yaml --debug &
EXPORTER_PID=$!

# Wait for startup
sleep 5

# Check logs for successful initialization
tail -20 log/nbu-exporter.log | grep -E "Using NetBackup API version|Successfully"

# Stop test instance
kill -TERM $EXPORTER_PID
```

#### Step 6: Start Exporter

```bash
# If running as systemd service
sudo systemctl start nbu_exporter
sudo systemctl status nbu_exporter

# If running in Docker
docker start nbu_exporter
docker logs -f nbu_exporter

# If running manually
nohup ./bin/nbu_exporter --config config.yaml > /dev/null 2>&1 &
echo $! > nbu_exporter.pid
```

### Method 2: Docker Deployment

#### Step 1: Backup and Stop

```bash
# Backup configuration
docker cp nbu_exporter:/config.yaml config.yaml.backup

# Stop and remove container
docker stop nbu_exporter
docker rm nbu_exporter
```

#### Step 2: Update Configuration

```bash
# Edit local config.yaml to add apiVersion
vi config.yaml
# Add: apiVersion: "12.0" under nbuserver section
```

#### Step 3: Pull New Image and Start

```bash
# Pull latest image
docker pull fjacquet/nbu_exporter:latest

# Start new container
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config.yaml:/config.yaml \
  --restart unless-stopped \
  fjacquet/nbu_exporter:latest --config /config.yaml

# Check logs
docker logs -f nbu_exporter
```

## Verification Procedures

### Level 1: Basic Health Check

```bash
# Check process is running
ps aux | grep nbu_exporter
# Or for Docker
docker ps | grep nbu_exporter

# Check health endpoint
curl http://localhost:2112/health
# Expected: {"status":"healthy"}
```

### Level 2: Metrics Availability

```bash
# Check metrics endpoint responds
curl -s http://localhost:2112/metrics | head -20

# Verify NBU-specific metrics are present
curl -s http://localhost:2112/metrics | grep -E "^nbu_" | head -10

# Expected metrics:
# nbu_storage_bytes{...}
# nbu_jobs_count{...}
# nbu_jobs_size_bytes{...}
```

### Level 3: API Communication

```bash
# Check logs for successful API calls
tail -50 log/nbu-exporter.log | grep -E "Successfully fetched|API version"

# Expected log entries:
# INFO[...] Using NetBackup API version: 12.0
# INFO[...] Successfully fetched storage data
# INFO[...] Successfully fetched job data
```

### Level 4: Data Validation

```bash
# Verify storage metrics have data
curl -s http://localhost:2112/metrics | grep "nbu_storage_bytes" | wc -l
# Should be > 0 if storage units exist

# Verify job metrics have data
curl -s http://localhost:2112/metrics | grep "nbu_jobs_count" | wc -l
# Should be > 0 if jobs exist in time window

# Check metric values are reasonable
curl -s http://localhost:2112/metrics | grep "nbu_storage_bytes" | grep "size=\"free\""
curl -s http://localhost:2112/metrics | grep "nbu_jobs_count"
```

### Level 5: Prometheus Integration

```bash
# Query Prometheus to verify scraping
curl -s 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}' | jq .

# Expected: "value": [timestamp, "1"]

# Verify recent data
curl -s 'http://prometheus:9090/api/v1/query?query=nbu_storage_bytes' | jq .

# Check scrape duration
curl -s 'http://prometheus:9090/api/v1/query?query=scrape_duration_seconds{job="netbackup"}' | jq .
```

### Level 6: Grafana Dashboard

```bash
# Access Grafana dashboard
# Navigate to: http://grafana:3000/d/netbackup-stats

# Verify panels show data:
# - Storage capacity charts
# - Job success rate graphs
# - Data transfer volumes
# - Job status distribution
```

## Backward Compatibility Verification

### Test 1: Configuration Without API Version

```bash
# Create test config without apiVersion field
cat > test-config-no-version.yaml << 'EOF'
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "log/test.log"
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "master.my.domain"
    port: "1556"
    apiKey: "test-key"
    contentType: "application/json"
    insecureSkipVerify: false
EOF

# Test that it works with default version
./bin/nbu_exporter --config test-config-no-version.yaml --debug &
TEST_PID=$!
sleep 5

# Check logs for default version
tail -20 log/test.log | grep "Using NetBackup API version: 12.0"

# Cleanup
kill -TERM $TEST_PID
rm test-config-no-version.yaml
```

### Test 2: Legacy Configuration Structure

```bash
# Test with old-style config (pre-10.5)
# Should work with default API version
./bin/nbu_exporter --config backups/*/config.yaml.backup --debug &
LEGACY_PID=$!
sleep 5

# Verify it starts and defaults to 12.0
tail -20 log/nbu-exporter.log | grep "API version"

# Cleanup
kill -TERM $LEGACY_PID
```

### Test 3: Metrics Consistency

```bash
# Capture metrics before upgrade
curl -s http://localhost:2112/metrics | grep "^nbu_" > metrics-before.txt

# After upgrade, capture again
curl -s http://localhost:2112/metrics | grep "^nbu_" > metrics-after.txt

# Compare metric names (should be identical)
diff <(cut -d'{' -f1 metrics-before.txt | sort -u) \
     <(cut -d'{' -f1 metrics-after.txt | sort -u)

# Expected: No differences in metric names
```

## Rollback Procedures

### When to Rollback

Rollback if you encounter:

- Exporter fails to start after upgrade
- HTTP 406 (Not Acceptable) errors from NetBackup API
- Metrics collection failures
- Prometheus scraping failures
- Data inconsistencies or missing metrics

### Quick Rollback (< 5 minutes)

```bash
# Stop new version
sudo systemctl stop nbu_exporter
# Or: docker stop nbu_exporter
# Or: pkill -TERM nbu_exporter

# Restore binary
cp backups/$(ls -t backups/ | head -1)/nbu_exporter.backup bin/nbu_exporter

# Restore configuration
cp backups/$(ls -t backups/ | head -1)/config.yaml.backup config.yaml

# Start old version
sudo systemctl start nbu_exporter
# Or: docker start nbu_exporter
# Or: ./bin/nbu_exporter --config config.yaml &

# Verify rollback
curl http://localhost:2112/health
curl http://localhost:2112/metrics | grep "^nbu_" | head -5
```

### Detailed Rollback Steps

#### Step 1: Stop New Version

```bash
# Systemd
sudo systemctl stop nbu_exporter
sudo systemctl status nbu_exporter  # Verify stopped

# Docker
docker stop nbu_exporter
docker rm nbu_exporter

# Manual
pkill -TERM nbu_exporter
sleep 10
pkill -9 nbu_exporter  # Force kill if needed
```

#### Step 2: Restore Previous Binary

```bash
# Find most recent backup
BACKUP_DIR=$(ls -td backups/*/ | head -1)
echo "Restoring from: $BACKUP_DIR"

# Restore binary
cp "${BACKUP_DIR}nbu_exporter.backup" bin/nbu_exporter
chmod +x bin/nbu_exporter

# Verify binary
ls -lh bin/nbu_exporter
```

#### Step 3: Restore Previous Configuration

```bash
# Restore config
cp "${BACKUP_DIR}config.yaml.backup" config.yaml

# Verify config
cat config.yaml | grep -A 15 "nbuserver:"
```

#### Step 4: Restart Service

```bash
# Systemd
sudo systemctl start nbu_exporter
sudo systemctl status nbu_exporter

# Docker
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config.yaml:/config.yaml \
  --restart unless-stopped \
  fjacquet/nbu_exporter:previous-version --config /config.yaml

# Manual
./bin/nbu_exporter --config config.yaml &
echo $! > nbu_exporter.pid
```

#### Step 5: Verify Rollback Success

```bash
# Check health
curl http://localhost:2112/health
# Expected: {"status":"healthy"}

# Check metrics
curl -s http://localhost:2112/metrics | grep "^nbu_" | wc -l
# Should be > 0

# Check logs
tail -50 log/nbu-exporter.log | grep -E "ERROR|FATAL"
# Should be empty or minimal

# Verify Prometheus scraping
curl -s 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}' | jq '.data.result[0].value[1]'
# Expected: "1"
```

#### Step 6: Document Rollback Reason

```bash
# Create incident report
cat > rollback-report-$(date +%Y%m%d-%H%M%S).txt << EOF
Rollback Report
===============
Date: $(date)
Reason: [Describe why rollback was necessary]
Error Messages: [Copy relevant error messages]
NetBackup Version: [Version number]
Exporter Version Attempted: [Version]
Rolled Back To: [Previous version]

Logs:
$(tail -100 log/nbu-exporter.log)

Environment:
- OS: $(uname -a)
- Go Version: $(go version)
- NetBackup API Accessible: [yes/no]

Next Steps:
- [ ] Report issue to GitHub
- [ ] Investigate root cause
- [ ] Plan retry with fixes
EOF

# Report issue
echo "Please report this issue at: https://github.com/fjacquet/nbu_exporter/issues"
```

## Breaking Changes Verification

### Verification Test Suite

Run this test suite to ensure no breaking changes:

```bash
#!/bin/bash
# save as: verify-no-breaking-changes.sh

set -e

echo "=== NBU Exporter Breaking Changes Verification ==="
echo

# Test 1: Configuration without API version
echo "Test 1: Configuration without API version field"
cat > /tmp/test-config-1.yaml << 'EOF'
server:
    host: "localhost"
    port: "2112"
    uri: "/metrics"
    scrapingInterval: "5m"
    logName: "/tmp/test1.log"
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    host: "test.example.com"
    port: "1556"
    apiKey: "test-key"
    contentType: "application/json"
EOF

./bin/nbu_exporter --config /tmp/test-config-1.yaml --debug &
PID1=$!
sleep 3
if ps -p $PID1 > /dev/null; then
    echo "✓ PASS: Exporter starts without API version field"
    kill -TERM $PID1
else
    echo "✗ FAIL: Exporter failed to start without API version field"
    exit 1
fi
echo

# Test 2: Legacy configuration structure
echo "Test 2: Legacy configuration structure"
cat > /tmp/test-config-2.yaml << 'EOF'
server:
    host: "localhost"
    port: "2113"
    uri: "/metrics"
    scrapingInterval: "1h"
    logName: "/tmp/test2.log"
nbuserver:
    scheme: "https"
    uri: "/netbackup"
    domain: "example.com"
    domainType: "NT"
    host: "master.example.com"
    port: "1556"
    apiKey: "legacy-key"
    contentType: "application/json"
    insecureSkipVerify: false
EOF

./bin/nbu_exporter --config /tmp/test-config-2.yaml --debug &
PID2=$!
sleep 3
if ps -p $PID2 > /dev/null; then
    echo "✓ PASS: Exporter works with legacy config structure"
    kill -TERM $PID2
else
    echo "✗ FAIL: Exporter failed with legacy config"
    exit 1
fi
echo

# Test 3: Metric names unchanged
echo "Test 3: Metric names consistency"
EXPECTED_METRICS=(
    "nbu_storage_bytes"
    "nbu_jobs_count"
    "nbu_jobs_size_bytes"
)

for metric in "${EXPECTED_METRICS[@]}"; do
    if curl -s http://localhost:2112/metrics | grep -q "^# HELP $metric"; then
        echo "✓ PASS: Metric $metric exists"
    else
        echo "✗ FAIL: Metric $metric missing"
        exit 1
    fi
done
echo

# Test 4: Default API version
echo "Test 4: Default API version is 12.0"
if grep -q "Using NetBackup API version: 12.0" /tmp/test1.log; then
    echo "✓ PASS: Default API version is 12.0"
else
    echo "✗ FAIL: Default API version is not 12.0"
    exit 1
fi
echo

# Cleanup
rm -f /tmp/test-config-*.yaml /tmp/test*.log

echo "=== All Breaking Changes Tests Passed ==="
```

## Post-Deployment Monitoring

### First 24 Hours

Monitor these metrics closely:

```bash
# Check error rate
curl -s http://localhost:2112/metrics | grep "nbu_scrape_errors_total"

# Check scrape duration
curl -s http://localhost:2112/metrics | grep "nbu_scrape_duration_seconds"

# Monitor logs for errors
tail -f log/nbu-exporter.log | grep -E "ERROR|WARN"

# Check Prometheus scrape success
curl -s 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}' | jq .
```

### Alerting Rules

Add these Prometheus alerting rules:

```yaml
groups:
  - name: nbu_exporter
    interval: 60s
    rules:
      - alert: NBUExporterDown
        expr: up{job="netbackup"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "NBU Exporter is down"
          description: "NBU Exporter has been down for more than 5 minutes"

      - alert: NBUExporterScrapeFailing
        expr: increase(nbu_scrape_errors_total[5m]) > 3
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "NBU Exporter scrape errors"
          description: "NBU Exporter has had {{ $value }} scrape errors in the last 5 minutes"

      - alert: NBUExporterSlowScrape
        expr: nbu_scrape_duration_seconds > 30
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "NBU Exporter slow scrapes"
          description: "NBU Exporter scrapes are taking longer than 30 seconds"
```

### Performance Baseline

Establish performance baselines:

```bash
# Scrape duration baseline
curl -s http://localhost:2112/metrics | grep "nbu_scrape_duration_seconds"

# Memory usage baseline
ps aux | grep nbu_exporter | awk '{print $6}'

# API response times (from logs)
grep "Successfully fetched" log/nbu-exporter.log | tail -20
```

## Troubleshooting Deployment Issues

### Issue: Exporter Won't Start

**Symptoms:**
- Process exits immediately
- No logs generated
- Health endpoint not accessible

**Diagnosis:**

```bash
# Run in foreground with debug
./bin/nbu_exporter --config config.yaml --debug

# Check for configuration errors
./bin/nbu_exporter --config config.yaml 2>&1 | grep -i error

# Verify config file syntax
cat config.yaml | python -m yaml
```

**Solutions:**
1. Check configuration file path is correct
2. Verify YAML syntax is valid
3. Ensure all required fields are present
4. Check file permissions on config and log directory

### Issue: API Version Errors

**Symptoms:**
- HTTP 406 (Not Acceptable) errors
- "Invalid API version format" errors

**Diagnosis:**

```bash
# Check NetBackup version
ssh netbackup-master "bpgetconfig -g | grep VERSION"

# Check configured API version
grep apiVersion config.yaml

# Test API manually
curl -k -H "Authorization: YOUR_API_KEY" \
  -H "Accept: application/vnd.netbackup+json;version=12.0" \
  https://netbackup-master:1556/netbackup/config/servers/master
```

**Solutions:**
1. Verify NetBackup is version 10.5 or later
2. Ensure apiVersion is set to "12.0" (string format)
3. Check API version format: must be "X.Y" (e.g., "12.0", not "12" or "v12.0")

### Issue: Metrics Not Updating

**Symptoms:**
- Stale data in Prometheus
- No new metrics after upgrade
- Empty metric values

**Diagnosis:**

```bash
# Check exporter logs
tail -100 log/nbu-exporter.log | grep -E "ERROR|Successfully"

# Test metrics endpoint
curl -s http://localhost:2112/metrics | grep "^nbu_" | wc -l

# Check Prometheus scraping
curl -s 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}'

# Verify NetBackup API access
curl -k -H "Authorization: YOUR_API_KEY" \
  https://netbackup-master:1556/netbackup/storage/storage-units
```

**Solutions:**
1. Verify API key is valid and not expired
2. Check network connectivity to NetBackup server
3. Ensure firewall allows traffic on port 1556
4. Verify scraping interval is appropriate
5. Check NetBackup has data to export (storage units, jobs)

## Deployment Checklist

Use this checklist for each deployment:

### Pre-Deployment

- [ ] NetBackup version verified (10.5+)
- [ ] API key validated and has required permissions
- [ ] Network connectivity tested
- [ ] Current configuration backed up
- [ ] Current binary backed up
- [ ] Prometheus configuration reviewed
- [ ] Maintenance window scheduled (if required)
- [ ] Rollback plan reviewed

### Deployment

- [ ] Exporter stopped gracefully
- [ ] New binary deployed
- [ ] Configuration updated with apiVersion
- [ ] Configuration validated
- [ ] Exporter started
- [ ] Health check passed
- [ ] Metrics endpoint responding
- [ ] Logs show successful API calls

### Post-Deployment

- [ ] Prometheus scraping successfully
- [ ] Grafana dashboard showing data
- [ ] No errors in logs (first 15 minutes)
- [ ] Metric values are reasonable
- [ ] Backward compatibility verified
- [ ] Performance baseline established
- [ ] Monitoring alerts configured
- [ ] Documentation updated
- [ ] Team notified of successful deployment

### Rollback (If Needed)

- [ ] Issue documented
- [ ] Exporter stopped
- [ ] Previous binary restored
- [ ] Previous configuration restored
- [ ] Exporter restarted
- [ ] Rollback verified
- [ ] Incident report created
- [ ] GitHub issue created

## Automation Scripts

### Deployment Script

```bash
#!/bin/bash
# deploy-nbu-exporter.sh

set -e

BACKUP_DIR="backups/$(date +%Y%m%d-%H%M%S)"
CONFIG_FILE="config.yaml"
BINARY_PATH="bin/nbu_exporter"

echo "=== NBU Exporter Deployment Script ==="
echo "Backup directory: $BACKUP_DIR"
echo

# Create backup
echo "Creating backup..."
mkdir -p "$BACKUP_DIR"
cp "$CONFIG_FILE" "$BACKUP_DIR/config.yaml.backup"
cp "$BINARY_PATH" "$BACKUP_DIR/nbu_exporter.backup"
echo "✓ Backup created"
echo

# Stop exporter
echo "Stopping exporter..."
if systemctl is-active --quiet nbu_exporter; then
    sudo systemctl stop nbu_exporter
    echo "✓ Systemd service stopped"
elif pgrep -x nbu_exporter > /dev/null; then
    pkill -TERM nbu_exporter
    sleep 5
    echo "✓ Process stopped"
else
    echo "✓ Exporter not running"
fi
echo

# Update binary
echo "Updating binary..."
if [ -f "nbu_exporter-new" ]; then
    cp nbu_exporter-new "$BINARY_PATH"
    chmod +x "$BINARY_PATH"
    echo "✓ Binary updated"
else
    echo "✗ New binary not found: nbu_exporter-new"
    exit 1
fi
echo

# Update configuration
echo "Updating configuration..."
if ! grep -q "apiVersion:" "$CONFIG_FILE"; then
    sed -i '/nbuserver:/a\    apiVersion: "12.0"' "$CONFIG_FILE"
    echo "✓ Added apiVersion to config"
else
    echo "✓ apiVersion already present"
fi
echo

# Validate configuration
echo "Validating configuration..."
timeout 10s "$BINARY_PATH" --config "$CONFIG_FILE" --debug &
VALIDATE_PID=$!
sleep 5
if ps -p $VALIDATE_PID > /dev/null; then
    kill -TERM $VALIDATE_PID
    echo "✓ Configuration valid"
else
    echo "✗ Configuration validation failed"
    echo "Rolling back..."
    cp "$BACKUP_DIR/config.yaml.backup" "$CONFIG_FILE"
    cp "$BACKUP_DIR/nbu_exporter.backup" "$BINARY_PATH"
    exit 1
fi
echo

# Start exporter
echo "Starting exporter..."
if systemctl list-unit-files | grep -q nbu_exporter.service; then
    sudo systemctl start nbu_exporter
    sleep 3
    if systemctl is-active --quiet nbu_exporter; then
        echo "✓ Systemd service started"
    else
        echo "✗ Failed to start systemd service"
        exit 1
    fi
else
    nohup "$BINARY_PATH" --config "$CONFIG_FILE" > /dev/null 2>&1 &
    echo $! > nbu_exporter.pid
    sleep 3
    if ps -p $(cat nbu_exporter.pid) > /dev/null; then
        echo "✓ Process started"
    else
        echo "✗ Failed to start process"
        exit 1
    fi
fi
echo

# Verify deployment
echo "Verifying deployment..."
if curl -sf http://localhost:2112/health > /dev/null; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    exit 1
fi

if curl -s http://localhost:2112/metrics | grep -q "^nbu_"; then
    echo "✓ Metrics available"
else
    echo "✗ Metrics not available"
    exit 1
fi

echo
echo "=== Deployment Successful ==="
echo "Backup location: $BACKUP_DIR"
echo "Monitor logs: tail -f log/nbu-exporter.log"
```

### Rollback Script

```bash
#!/bin/bash
# rollback-nbu-exporter.sh

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <backup-directory>"
    echo "Available backups:"
    ls -1d backups/*/
    exit 1
fi

BACKUP_DIR="$1"
CONFIG_FILE="config.yaml"
BINARY_PATH="bin/nbu_exporter"

echo "=== NBU Exporter Rollback Script ==="
echo "Rolling back from: $BACKUP_DIR"
echo

# Verify backup exists
if [ ! -d "$BACKUP_DIR" ]; then
    echo "✗ Backup directory not found: $BACKUP_DIR"
    exit 1
fi

if [ ! -f "$BACKUP_DIR/config.yaml.backup" ] || [ ! -f "$BACKUP_DIR/nbu_exporter.backup" ]; then
    echo "✗ Backup files not found in $BACKUP_DIR"
    exit 1
fi

# Stop exporter
echo "Stopping exporter..."
if systemctl is-active --quiet nbu_exporter; then
    sudo systemctl stop nbu_exporter
    echo "✓ Systemd service stopped"
elif pgrep -x nbu_exporter > /dev/null; then
    pkill -TERM nbu_exporter
    sleep 5
    pkill -9 nbu_exporter 2>/dev/null || true
    echo "✓ Process stopped"
else
    echo "✓ Exporter not running"
fi
echo

# Restore files
echo "Restoring files..."
cp "$BACKUP_DIR/config.yaml.backup" "$CONFIG_FILE"
cp "$BACKUP_DIR/nbu_exporter.backup" "$BINARY_PATH"
chmod +x "$BINARY_PATH"
echo "✓ Files restored"
echo

# Start exporter
echo "Starting exporter..."
if systemctl list-unit-files | grep -q nbu_exporter.service; then
    sudo systemctl start nbu_exporter
    sleep 3
    if systemctl is-active --quiet nbu_exporter; then
        echo "✓ Systemd service started"
    else
        echo "✗ Failed to start systemd service"
        exit 1
    fi
else
    nohup "$BINARY_PATH" --config "$CONFIG_FILE" > /dev/null 2>&1 &
    echo $! > nbu_exporter.pid
    sleep 3
    if ps -p $(cat nbu_exporter.pid) > /dev/null; then
        echo "✓ Process started"
    else
        echo "✗ Failed to start process"
        exit 1
    fi
fi
echo

# Verify rollback
echo "Verifying rollback..."
if curl -sf http://localhost:2112/health > /dev/null; then
    echo "✓ Health check passed"
else
    echo "✗ Health check failed"
    exit 1
fi

if curl -s http://localhost:2112/metrics | grep -q "^nbu_"; then
    echo "✓ Metrics available"
else
    echo "✗ Metrics not available"
    exit 1
fi

echo
echo "=== Rollback Successful ==="
echo "Restored from: $BACKUP_DIR"
echo "Monitor logs: tail -f log/nbu-exporter.log"
```

## Summary

This document provides comprehensive procedures for:

1. **Deployment**: Step-by-step instructions for upgrading to API 10.5 support
2. **Verification**: Multi-level verification from basic health to Grafana dashboards
3. **Backward Compatibility**: Tests to ensure existing configurations work
4. **Rollback**: Quick and detailed rollback procedures
5. **Troubleshooting**: Common issues and solutions
6. **Automation**: Scripts for deployment and rollback

### Key Takeaways

- **No Breaking Changes**: Existing configurations work with default API version
- **Backward Compatible**: Legacy configs without apiVersion field work correctly
- **Safe Rollback**: Complete rollback procedures with verification steps
- **Comprehensive Testing**: Multiple verification levels ensure deployment success
- **Automated Scripts**: Deployment and rollback scripts for consistency

### Support

For issues during deployment:

1. Check logs: `tail -f log/nbu-exporter.log`
2. Review troubleshooting section above
3. Consult [Migration Guide](api-10.5-migration.md)
4. Report issues: <https://github.com/fjacquet/nbu_exporter/issues>

## Related Documentation

- [API 10.5 Migration Guide](api-10.5-migration.md) - Detailed upgrade instructions
- [README.md](../README.md) - General usage and configuration
- [CHANGELOG.md](../CHANGELOG.md) - Version history and changes
