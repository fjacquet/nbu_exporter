# Deployment Quick Reference

## Pre-Flight Checklist

```bash
# Verify NetBackup version (must be 10.5+)
ssh netbackup-master "bpgetconfig -g | grep VERSION"

# Backup current setup
mkdir -p backups/$(date +%Y%m%d)
cp config.yaml backups/$(date +%Y%m%d)/
cp bin/nbu_exporter backups/$(date +%Y%m%d)/
```

## Quick Deployment

```bash
# 1. Stop exporter
sudo systemctl stop nbu_exporter

# 2. Update binary
make cli  # or download pre-built binary

# 3. Add API version to config (if not present)
grep -q "apiVersion:" config.yaml || \
  sed -i '/nbuserver:/a\    apiVersion: "12.0"' config.yaml

# 4. Start exporter
sudo systemctl start nbu_exporter

# 5. Verify
curl http://localhost:2112/health
curl http://localhost:2112/metrics | grep "^nbu_" | head -5
```

## Quick Rollback

```bash
# 1. Stop exporter
sudo systemctl stop nbu_exporter

# 2. Restore from backup
BACKUP_DIR=$(ls -td backups/*/ | head -1)
cp "${BACKUP_DIR}config.yaml" config.yaml
cp "${BACKUP_DIR}nbu_exporter" bin/nbu_exporter

# 3. Start exporter
sudo systemctl start nbu_exporter

# 4. Verify
curl http://localhost:2112/health
```

## Verification Commands

```bash
# Health check
curl http://localhost:2112/health

# Metrics check
curl -s http://localhost:2112/metrics | grep "^nbu_" | wc -l

# Log check
tail -50 log/nbu-exporter.log | grep -E "ERROR|Successfully"

# Prometheus check
curl -s 'http://prometheus:9090/api/v1/query?query=up{job="netbackup"}'
```

## Common Issues

### Exporter won't start
```bash
# Check logs
./bin/nbu_exporter --config config.yaml --debug

# Verify config syntax
cat config.yaml | python -m yaml
```

### API version errors
```bash
# Verify NetBackup version
ssh netbackup-master "bpgetconfig -g | grep VERSION"

# Check API version format (must be "X.Y")
grep apiVersion config.yaml
```

### Metrics not updating
```bash
# Test API connectivity
curl -k -H "Authorization: YOUR_API_KEY" \
  https://netbackup-master:1556/netbackup/storage/storage-units

# Check logs
tail -100 log/nbu-exporter.log | grep ERROR
```

## Automated Scripts

```bash
# Run deployment verification
./scripts/verify-deployment.sh

# Automated deployment (if available)
./scripts/deploy-nbu-exporter.sh

# Automated rollback (if available)
./scripts/rollback-nbu-exporter.sh backups/YYYYMMDD-HHMMSS/
```

## Support

- Full Guide: [docs/deployment-verification.md](deployment-verification.md)
- Migration Guide: [docs/api-10.5-migration.md](api-10.5-migration.md)
- Issues: <https://github.com/fjacquet/nbu_exporter/issues>
