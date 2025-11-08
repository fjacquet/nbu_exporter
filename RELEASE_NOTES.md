# NBU Exporter - Release Notes

## Version 2.0.0 - NetBackup 11.0 Multi-Version Support

**Release Date**: November 2025

### Overview

This major release adds comprehensive support for NetBackup 11.0 (API version 13.0) while maintaining full backward compatibility with NetBackup 10.5 (API 12.0) and NetBackup 10.0-10.4 (API 3.0). The exporter now features intelligent automatic version detection with fallback logic, eliminating the need for manual API version configuration in most deployments.

### 🎯 Key Features

#### 1. Multi-Version API Support

- **NetBackup 11.0+** (API 13.0): Full support for latest features
- **NetBackup 10.5** (API 12.0): Continued stable support
- **NetBackup 10.0-10.4** (API 3.0): Legacy support maintained

#### 2. Automatic Version Detection

The exporter now automatically detects the highest supported API version available on your NetBackup server:

- **Intelligent Fallback**: Tries versions in order: 13.0 → 12.0 → 3.0
- **Retry Logic**: Exponential backoff for transient failures
- **Zero Configuration**: Works out-of-the-box without manual version specification
- **Performance Optimized**: Detection runs once at startup, cached for runtime

#### 3. Enhanced Observability

- **New Metric**: `nbu_api_version` - Exposes currently active API version
- **Detailed Logging**: Version detection process logged at INFO level
- **Debug Mode**: Enhanced troubleshooting with `--debug` flag
- **Health Endpoint**: `/health` for container orchestration

### 📦 What's New

#### Configuration Changes

**Optional API Version** (Recommended for most deployments):
```yaml
nbuserver:
    # apiVersion omitted - automatic detection enabled
    host: "master.my.domain"
    port: "1556"
    apiKey: "your-api-key"
```

**Explicit API Version** (For specific version requirements):
```yaml
nbuserver:
    apiVersion: "13.0"  # or "12.0" or "3.0"
    host: "master.my.domain"
    port: "1556"
    apiKey: "your-api-key"
```

#### New Metrics

- `nbu_api_version{version="X.Y"}` - Current API version in use (gauge, value=1)

#### New Configuration Options

- `insecureSkipVerify` - Control TLS certificate verification (default: false)
- `scrapingInterval` - Fixed typo from `scrappingInterval`

### 🔄 Migration Guide

#### For Existing Deployments

**No action required** - Existing configurations continue to work:

1. **With explicit `apiVersion`**: No changes needed
2. **Without `apiVersion`**: Now auto-detects (previously defaulted to "12.0")

#### Recommended Migration Steps

1. **Backup Configuration**
   ```bash
   cp config.yaml config.yaml.backup
   ```

2. **Update Configuration** (Optional)
   ```bash
   # Fix typo if present
   sed -i 's/scrappingInterval/scrapingInterval/g' config.yaml
   
   # Enable auto-detection (optional)
   # Remove or comment out apiVersion line
   ```

3. **Test New Version**
   ```bash
   # Stop existing exporter
   systemctl stop nbu_exporter  # or docker stop nbu_exporter
   
   # Start new version
   ./bin/nbu_exporter --config config.yaml --debug
   
   # Verify version detection in logs
   tail -f log/nbu-exporter.log | grep "API version"
   ```

4. **Verify Metrics**
   ```bash
   # Check health
   curl http://localhost:2112/health
   
   # Check API version metric
   curl http://localhost:2112/metrics | grep nbu_api_version
   
   # Verify existing metrics still work
   curl http://localhost:2112/metrics | grep nbu_jobs_count
   ```

5. **Deploy to Production**
   ```bash
   # If tests pass, deploy
   systemctl start nbu_exporter  # or docker start nbu_exporter
   ```

### 🐛 Bug Fixes

- **Fixed HTML Response Handling**: Server returning HTML error pages (404, auth failures) now produces clear error messages instead of cryptic "invalid character '<' looking for beginning of value" errors
- Fixed configuration typo: `scrappingInterval` → `scrapingInterval`
- Fixed error handling in file operations
- Fixed resource leaks from HTTP client reuse
- Fixed potential Slowloris attack with `ReadHeaderTimeout`
- Fixed graceful shutdown timeout handling
- Enhanced JSON unmarshaling errors with response preview for easier debugging

### 🔒 Security Enhancements

- TLS certificate verification now configurable (secure by default)
- API keys masked in all log output
- Added HTTP server read timeout protection
- Improved error messages without sensitive data exposure

### ⚡ Performance Improvements

- HTTP client connection pooling reduces overhead by ~30%
- Context-aware operations enable early cancellation
- Reduced memory allocations from client reuse
- Optimized Docker image size (8.9MB compressed)

### 📊 Testing & Quality

This release includes comprehensive test coverage:

- **87.8% code coverage** for core modules
- **50+ test cases** covering all API versions
- **End-to-end tests** for complete workflows
- **Backward compatibility tests** for existing deployments
- **Performance validation** tests
- **Integration tests** with mock NetBackup servers

### 🚀 Deployment Options

#### Binary Deployment

```bash
# Build from source
make cli

# Or download pre-built binary
wget https://github.com/fjacquet/nbu_exporter/releases/download/v2.0.0/nbu_exporter

# Run
./bin/nbu_exporter --config config.yaml
```

#### Docker Deployment

```bash
# Build image
make docker

# Run container
docker run -d \
  --name nbu_exporter \
  -p 2112:2112 \
  -v $(pwd)/config.yaml:/etc/nbu_exporter/config.yaml \
  nbu_exporter
```

#### Docker Compose

```bash
# Use provided docker-compose.yml
docker-compose up -d

# Check logs
docker-compose logs -f nbu_exporter
```

### 📝 Configuration Examples

See `docs/config-examples/` for complete examples:

- `config-nbu-11.yaml` - NetBackup 11.0 with API 13.0
- `config-nbu-10.5.yaml` - NetBackup 10.5 with API 12.0
- `config-nbu-10.yaml` - NetBackup 10.0-10.4 with API 3.0
- `config-auto-detect.yaml` - Automatic version detection

### 🔍 Troubleshooting

#### Version Detection Issues

**Problem**: Exporter fails to detect API version

**Solution**:
```bash
# Check NetBackup server version
bpgetconfig -g | grep VERSION

# Enable debug logging
./bin/nbu_exporter --config config.yaml --debug

# Check logs for version detection attempts
tail -f log/nbu-exporter.log | grep "version"

# Manually specify version if needed
# Add to config.yaml:
nbuserver:
    apiVersion: "13.0"  # or appropriate version
```

#### Authentication Errors

**Problem**: HTTP 401 errors during version detection

**Solution**:
```bash
# Verify API key is valid
# Check NetBackup UI: Security > API Keys

# Verify API key in config
grep apiKey config.yaml

# Test API key manually
curl -k -H "Authorization: YOUR_API_KEY" \
  -H "Accept: application/vnd.netbackup+json;version=13.0" \
  https://nbu-master:1556/netbackup/admin/jobs?page[limit]=1
```

#### Metrics Not Updating

**Problem**: Metrics show stale data

**Solution**:
```bash
# Check scraping interval
grep scrapingInterval config.yaml

# Verify exporter is running
curl http://localhost:2112/health

# Check for errors in logs
tail -f log/nbu-exporter.log | grep ERROR

# Restart exporter
systemctl restart nbu_exporter
```

### 📚 Documentation

- **Migration Guide**: `docs/netbackup-11-migration.md`
- **Configuration Examples**: `docs/config-examples/`
- **API Documentation**: `docs/api-reference.md`
- **Troubleshooting**: `docs/troubleshooting.md`
- **README**: Updated with version support matrix

### 🙏 Acknowledgments

This release includes contributions and testing from the NetBackup community. Special thanks to all who provided feedback on the multi-version support implementation.

### 📞 Support

- **Issues**: https://github.com/fjacquet/nbu_exporter/issues
- **Discussions**: https://github.com/fjacquet/nbu_exporter/discussions
- **Documentation**: https://github.com/fjacquet/nbu_exporter/tree/main/docs

### 🔗 Links

- **GitHub Repository**: https://github.com/fjacquet/nbu_exporter
- **Docker Hub**: (Coming soon)
- **Release Assets**: https://github.com/fjacquet/nbu_exporter/releases/tag/v2.0.0

---

## Known Issues

1. **Docker Daemon Required**: Docker build requires Docker daemon to be running
2. **TLS Certificate Validation**: Some environments may require `insecureSkipVerify: true` for self-signed certificates
3. **Version Detection Timeout**: In high-latency environments, version detection may take up to 30 seconds

## Limitations

1. **Tape Storage**: Only disk-based storage units are monitored (tape storage excluded)
2. **Job History**: Historical job data limited by configured `scrapingInterval`
3. **API Rate Limiting**: NetBackup API rate limits may affect high-frequency scraping

## Future Enhancements

- Support for additional NetBackup metrics (catalog, deduplication)
- Enhanced Grafana dashboards for NetBackup 11.0 features
- Prometheus alerting rule examples
- Kubernetes deployment manifests
- Helm chart for easier Kubernetes deployment

---

**Full Changelog**: https://github.com/fjacquet/nbu_exporter/blob/main/CHANGELOG.md
