# NetBackup 11.0 API Migration Guide

## Overview

This guide provides step-by-step instructions for upgrading the NBU Exporter to support NetBackup 11.0 (API version 13.0) while maintaining backward compatibility with NetBackup 10.5 (API 12.0) and NetBackup 10.0-10.4 (API 3.0).

## What's New

### Multi-Version API Support

The exporter now supports three NetBackup API versions:

| NetBackup Version | API Version | Support Status |
|-------------------|-------------|----------------|
| 11.0+             | 13.0        | ✅ Fully Supported |
| 10.5              | 12.0        | ✅ Fully Supported |
| 10.0 - 10.4       | 3.0         | ✅ Legacy Support |

### Automatic Version Detection

The exporter can automatically detect the highest supported API version available on your NetBackup server. This eliminates the need for manual version configuration in most scenarios.

**Detection Process:**
1. Attempts API version 13.0 (NetBackup 11.0+)
2. Falls back to API version 12.0 (NetBackup 10.5)
3. Falls back to API version 3.0 (NetBackup 10.0-10.4)
4. Uses the first version that responds successfully

### Key Benefits

- **Single Binary**: Deploy the same exporter across all NetBackup versions
- **Zero Configuration**: Automatic detection works out of the box
- **Backward Compatible**: Existing configurations continue to work
- **Future Proof**: Ready for NetBackup upgrades without exporter changes

## Migration Scenarios

### Scenario 1: New Deployment (NetBackup 11.0)

**Recommended for:** Fresh installations on NetBackup 11.0 environments.

**Steps:**

1. **Download the latest exporter binary**
   ```bash
   # Clone or download the latest release
   git clone https://github.com/fjacquet/nbu_exporter.git
   cd nbu_exporter
   make cli
   ```

2. **Create configuration file** (config.yaml)
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
       host: "nbu-master.my.domain"
       port: "1556"
       # apiVersion not specified - will auto-detect 13.0
       apiKey: "your-api-key-here"
       contentType: "application/vnd.netbackup+json; version=3.0"
       insecureSkipVerify: false
   ```

3. **Start the exporter**
   ```bash
   ./bin/nbu_exporter --config config.yaml
   ```

4. **Verify version detection**
   ```
   INFO[0000] Starting NBU Exporter
   INFO[0001] Attempting API version detection
   INFO[0002] Detected NetBackup API version: 13.0
   INFO[0002] Successfully connected to NetBackup API
   ```

5. **Test metrics endpoint**
   ```bash
   curl http://localhost:2112/metrics | grep nbu_
   ```

**Expected Result:** Exporter automatically detects API version 13.0 and collects metrics successfully.

---

### Scenario 2: Existing Deployment (NetBackup 10.5 → 11.0 Upgrade)

**Recommended for:** Environments upgrading from NetBackup 10.5 to 11.0.

**Before Upgrade:**
- Exporter configured with `apiVersion: "12.0"`
- NetBackup 10.5 running with API 12.0

**After NetBackup Upgrade:**

**Option A: Enable Automatic Detection (Recommended)**

1. **Stop the exporter**
   ```bash
   # If running as systemd service
   sudo systemctl stop nbu_exporter

   # If running manually
   pkill nbu_exporter
   ```

2. **Update configuration** - Remove explicit version
   ```yaml
   nbuserver:
       # apiVersion: "12.0"  # Comment out or remove
       host: "nbu-master.my.domain"
       # ... other settings remain unchanged
   ```

3. **Restart the exporter**
   ```bash
   sudo systemctl start nbu_exporter
   # or
   ./bin/nbu_exporter --config config.yaml
   ```

4. **Verify new version detected**
   ```bash
   # Check logs
   tail -f log/nbu-exporter.log | grep "Detected NetBackup API version"
   # Expected: Detected NetBackup API version: 13.0
   ```

**Option B: Explicit Version Update**

1. **Stop the exporter**

2. **Update configuration** - Change version to 13.0
   ```yaml
   nbuserver:
       apiVersion: "13.0"  # Updated from 12.0
       host: "nbu-master.my.domain"
       # ... other settings remain unchanged
   ```

3. **Restart the exporter**

4. **Verify metrics collection**
   ```bash
   curl http://localhost:2112/metrics | grep nbu_api_version
   # Expected: nbu_api_version{version="13.0"} 1
   ```

**Option C: No Changes (Continue with API 12.0)**

NetBackup 11.0 maintains backward compatibility with API 12.0. You can continue using your existing configuration without changes:

```yaml
nbuserver:
    apiVersion: "12.0"  # Still works with NetBackup 11.0
    # ... other settings
```

**Note:** This option works but doesn't take advantage of new API 13.0 features.

---

### Scenario 3: Mixed Environment (Multiple NetBackup Versions)

**Recommended for:** Organizations with multiple NetBackup servers running different versions.

**Challenge:** Different NetBackup servers require different API versions.

**Solution:** Use automatic version detection for each exporter instance.

**Deployment:**

1. **Deploy exporter per NetBackup server**
   ```bash
   # Server 1: NetBackup 11.0
   ./bin/nbu_exporter --config config-nbu11.yaml

   # Server 2: NetBackup 10.5
   ./bin/nbu_exporter --config config-nbu10.yaml

   # Server 3: NetBackup 10.0
   ./bin/nbu_exporter --config config-nbu10-legacy.yaml
   ```

2. **Configuration for each server** (no apiVersion specified)
   ```yaml
   # config-nbu11.yaml
   nbuserver:
       host: "nbu11-master.my.domain"
       # apiVersion omitted - will detect 13.0

   # config-nbu10.yaml
   nbuserver:
       host: "nbu10-master.my.domain"
       # apiVersion omitted - will detect 12.0

   # config-nbu10-legacy.yaml
   nbuserver:
       host: "nbu10-legacy.my.domain"
       # apiVersion omitted - will detect 3.0
   ```

3. **Each exporter automatically detects the correct version**

**Alternative:** Use explicit versions if you prefer predictable behavior:

```yaml
# config-nbu11.yaml
nbuserver:
    apiVersion: "13.0"
    host: "nbu11-master.my.domain"

# config-nbu10.yaml
nbuserver:
    apiVersion: "12.0"
    host: "nbu10-master.my.domain"
```

---

### Scenario 4: Docker Deployment

**Recommended for:** Containerized deployments.

**Steps:**

1. **Build updated Docker image**
   ```bash
   make docker
   # or
   docker build -t nbu_exporter:latest .
   ```

2. **Update docker-compose.yml** (if using)
   ```yaml
   version: '3.8'
   services:
     nbu_exporter:
       image: nbu_exporter:latest
       ports:
         - "2112:2112"
       volumes:
         - ./config.yaml:/config.yaml:ro
       command: ["--config", "/config.yaml"]
       restart: unless-stopped
   ```

3. **Configuration** (config.yaml)
   ```yaml
   nbuserver:
       # apiVersion omitted for auto-detection
       host: "nbu-master.my.domain"
       # ... other settings
   ```

4. **Deploy container**
   ```bash
   docker-compose up -d
   ```

5. **Verify logs**
   ```bash
   docker-compose logs -f nbu_exporter | grep "Detected NetBackup API version"
   ```

---

### Scenario 5: Kubernetes Deployment

**Recommended for:** Kubernetes/OpenShift environments.

**Steps:**

1. **Update ConfigMap** (remove explicit version)
   ```yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: nbu-exporter-config
   data:
     config.yaml: |
       server:
         host: "0.0.0.0"
         port: "2112"
         uri: "/metrics"
         scrapingInterval: "1h"
         logName: "/var/log/nbu-exporter.log"
       nbuserver:
         scheme: "https"
         uri: "/netbackup"
         domain: "my.domain"
         domainType: "NT"
         host: "nbu-master.my.domain"
         port: "1556"
         # apiVersion omitted for auto-detection
         apiKey: "${NBU_API_KEY}"
         contentType: "application/vnd.netbackup+json; version=3.0"
         insecureSkipVerify: false
   ```

2. **Update Deployment** (use latest image)
   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: nbu-exporter
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: nbu-exporter
     template:
       metadata:
         labels:
           app: nbu-exporter
       spec:
         containers:
         - name: nbu-exporter
           image: nbu_exporter:latest
           ports:
           - containerPort: 2112
           volumeMounts:
           - name: config
             mountPath: /config.yaml
             subPath: config.yaml
           env:
           - name: NBU_API_KEY
             valueFrom:
               secretKeyRef:
                 name: nbu-credentials
                 key: api-key
         volumes:
         - name: config
           configMap:
             name: nbu-exporter-config
   ```

3. **Apply changes**
   ```bash
   kubectl apply -f configmap.yaml
   kubectl apply -f deployment.yaml
   kubectl rollout restart deployment/nbu-exporter
   ```

4. **Verify deployment**
   ```bash
   kubectl logs -f deployment/nbu-exporter | grep "Detected NetBackup API version"
   ```

---

## Rollback Procedures

### Rollback Scenario 1: Issues After Enabling Auto-Detection

**Problem:** Auto-detection is not working as expected.

**Solution:** Revert to explicit version configuration.

1. **Stop the exporter**
   ```bash
   sudo systemctl stop nbu_exporter
   ```

2. **Restore explicit version** in config.yaml
   ```yaml
   nbuserver:
       apiVersion: "12.0"  # Restore previous version
       # ... other settings
   ```

3. **Restart the exporter**
   ```bash
   sudo systemctl start nbu_exporter
   ```

4. **Verify metrics collection**
   ```bash
   curl http://localhost:2112/metrics | grep nbu_
   ```

---

### Rollback Scenario 2: Issues After NetBackup 11.0 Upgrade

**Problem:** Exporter not working correctly after NetBackup upgrade.

**Solution:** Temporarily revert to API 12.0 (backward compatible).

1. **Stop the exporter**

2. **Configure explicit API 12.0**
   ```yaml
   nbuserver:
       apiVersion: "12.0"  # Use backward-compatible version
       # ... other settings
   ```

3. **Restart and verify**

**Note:** NetBackup 11.0 supports API 12.0 for backward compatibility.

---

### Rollback Scenario 3: Complete Rollback to Previous Exporter Version

**Problem:** New exporter version causing issues.

**Solution:** Restore previous exporter binary and configuration.

1. **Stop current exporter**
   ```bash
   sudo systemctl stop nbu_exporter
   ```

2. **Restore previous binary**
   ```bash
   cp bin/nbu_exporter.backup bin/nbu_exporter
   # or download previous release
   ```

3. **Restore previous configuration**
   ```bash
   cp config.yaml.backup config.yaml
   ```

4. **Restart exporter**
   ```bash
   sudo systemctl start nbu_exporter
   ```

5. **Verify functionality**
   ```bash
   curl http://localhost:2112/metrics | grep nbu_
   ```

---

## Troubleshooting Common Migration Issues

### Issue 1: Version Detection Fails

**Symptoms:**
```
ERROR: Failed to detect compatible NetBackup API version.
Attempted versions: 13.0, 12.0, 3.0
```

**Diagnosis:**

1. **Check NetBackup version**
   ```bash
   # On NetBackup master server
   bpgetconfig -g | grep VERSION
   ```

2. **Test API connectivity**
   ```bash
   curl -k -H "Authorization: YOUR_API_KEY" \
        -H "Accept: application/vnd.netbackup+json;version=13.0" \
        https://nbu-master:1556/netbackup/admin/jobs?page[limit]=1
   ```

**Solutions:**

- **If NetBackup < 10.0:** Upgrade NetBackup or use an older exporter version
- **If network issue:** Check firewall rules, DNS resolution, and network connectivity
- **If authentication issue:** Verify API key is valid and has correct permissions
- **Workaround:** Configure explicit version based on your NetBackup version

---

### Issue 2: Slow Startup with Auto-Detection

**Symptoms:**
- Exporter takes 3-5 seconds to start
- Multiple version attempts in logs

**Diagnosis:**
```
DEBUG[0001] Trying API version 13.0
WARN[0002] API version 13.0 not supported (HTTP 406), trying next version
DEBUG[0002] Trying API version 12.0
INFO[0003] Successfully detected API version: 12.0
```

**Solution:** Configure explicit version to skip detection

```yaml
nbuserver:
    apiVersion: "12.0"  # Skip detection, connect immediately
    # ... other settings
```

**Result:** Startup time reduced to < 1 second.

---

### Issue 3: Metrics Missing After Upgrade

**Symptoms:**
- Exporter starts successfully
- Some metrics are missing or zero

**Diagnosis:**

1. **Check API version metric**
   ```bash
   curl http://localhost:2112/metrics | grep nbu_api_version
   ```

2. **Verify NetBackup API responses**
   ```bash
   # Test jobs endpoint
   curl -k -H "Authorization: YOUR_API_KEY" \
        -H "Accept: application/vnd.netbackup+json;version=13.0" \
        https://nbu-master:1556/netbackup/admin/jobs

   # Test storage endpoint
   curl -k -H "Authorization: YOUR_API_KEY" \
        -H "Accept: application/vnd.netbackup+json;version=13.0" \
        https://nbu-master:1556/netbackup/storage/storage-units
   ```

**Solutions:**

- **If API returns empty data:** Check NetBackup has jobs/storage to report
- **If API returns errors:** Verify API key permissions
- **If specific metrics missing:** Check exporter logs for parsing errors

---

### Issue 4: Authentication Errors After Migration

**Symptoms:**
```
ERROR: failed to fetch storage data: HTTP 401 Unauthorized
```

**Diagnosis:**

1. **Verify API key is still valid**
   ```bash
   # Test authentication
   curl -k -H "Authorization: YOUR_API_KEY" \
        https://nbu-master:1556/netbackup/admin/jobs?page[limit]=1
   ```

2. **Check API key permissions** in NetBackup UI

**Solutions:**

- **If key expired:** Generate new API key in NetBackup UI
- **If permissions changed:** Update API key permissions
- **If key format changed:** Verify key format in configuration

---

## Validation Checklist

After migration, verify the following:

### ✅ Exporter Startup

- [ ] Exporter starts without errors
- [ ] Version detection completes successfully (if using auto-detection)
- [ ] Correct API version is logged
- [ ] No authentication errors in logs

### ✅ Metrics Collection

- [ ] Metrics endpoint responds: `curl http://localhost:2112/metrics`
- [ ] Job metrics are present: `grep nbu_jobs_count`
- [ ] Storage metrics are present: `grep nbu_storage_bytes`
- [ ] API version metric is present: `grep nbu_api_version`

### ✅ Prometheus Integration

- [ ] Prometheus successfully scrapes the exporter
- [ ] No scrape errors in Prometheus logs
- [ ] Metrics appear in Prometheus UI
- [ ] Existing alerts still work

### ✅ Grafana Dashboards

- [ ] Existing dashboards display data correctly
- [ ] No broken panels or queries
- [ ] Time series data is continuous (no gaps)
- [ ] All visualizations render properly

### ✅ Performance

- [ ] Scrape duration is acceptable (< 30 seconds)
- [ ] No timeout errors
- [ ] CPU and memory usage is normal
- [ ] No resource leaks over time

---

## Best Practices

### 1. Test in Non-Production First

Always test the migration in a development or staging environment before production:

```bash
# Test environment
./bin/nbu_exporter --config config-test.yaml --debug
```

### 2. Enable Debug Logging During Migration

Use debug mode to troubleshoot issues:

```bash
./bin/nbu_exporter --config config.yaml --debug
```

### 3. Monitor Logs During Migration

Watch logs in real-time during the migration:

```bash
tail -f log/nbu-exporter.log
```

### 4. Backup Configuration Before Changes

Always backup your configuration:

```bash
cp config.yaml config.yaml.backup
```

### 5. Use Explicit Versions for Production

For production environments, consider using explicit versions for predictable behavior:

```yaml
nbuserver:
    apiVersion: "13.0"  # Explicit version for production
```

### 6. Document Your Configuration

Add comments to your configuration file:

```yaml
nbuserver:
    apiVersion: "13.0"  # NetBackup 11.0 - Updated 2025-01-15
    host: "nbu-master.my.domain"
```

### 7. Plan for Rollback

Always have a rollback plan:
- Keep previous exporter binary
- Backup configuration files
- Document rollback steps
- Test rollback procedure

---

## Support and Resources

### Documentation

- [README.md](../README.md) - Main documentation
- [API 10.5 Migration Guide](api-10.5-migration.md) - Previous migration guide
- [CHANGELOG.md](../CHANGELOG.md) - Version history

### NetBackup API Documentation

- [NetBackup 11.0 API Documentation](https://sort.veritas.com/public/documents/nbu/11.0/)
- [NetBackup 10.5 API Documentation](https://sort.veritas.com/public/documents/nbu/10.5/)

### Getting Help

- **GitHub Issues:** https://github.com/fjacquet/nbu_exporter/issues
- **Discussions:** https://github.com/fjacquet/nbu_exporter/discussions

---

## Appendix: API Version Comparison

### API 13.0 (NetBackup 11.0) vs API 12.0 (NetBackup 10.5)

**Jobs API:**
- Same endpoint structure
- Same core attributes
- Potential new optional fields in 13.0

**Storage API:**
- Same endpoint structure
- Same capacity fields
- Consistent metric names

**Authentication:**
- Same API key mechanism
- Same header format
- No changes required

**Backward Compatibility:**
- NetBackup 11.0 supports API 12.0
- NetBackup 11.0 supports API 3.0
- Smooth upgrade path

---

**Document Version:** 1.0  
**Last Updated:** 2025-01-15  
**Exporter Version:** Latest (with multi-version support)
