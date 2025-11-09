# Trace Analysis Guide

This guide provides detailed instructions for querying and analyzing traces from the NBU Exporter to diagnose performance issues and optimize your monitoring setup.

## Table of Contents

- [Understanding Trace Structure](#understanding-trace-structure)
- [Common Queries](#common-queries)
- [Performance Analysis](#performance-analysis)
- [Troubleshooting Scenarios](#troubleshooting-scenarios)
- [Best Practices](#best-practices)

## Understanding Trace Structure

### Span Hierarchy

Each Prometheus scrape creates a trace with the following structure:

```
prometheus.scrape (root span)
├── netbackup.fetch_storage
│   └── http.request (GET /storage/storage-units)
└── netbackup.fetch_jobs
    ├── netbackup.fetch_job_page (offset=0)
    │   └── http.request (GET /admin/jobs?offset=0)
    ├── netbackup.fetch_job_page (offset=1)
    │   └── http.request (GET /admin/jobs?offset=1)
    └── netbackup.fetch_job_page (offset=N)
        └── http.request (GET /admin/jobs?offset=N)
```

### Span Attributes

#### Root Span (prometheus.scrape)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `scrape.duration_ms` | int | Total scrape duration in milliseconds | 45230 |
| `scrape.storage_metrics_count` | int | Number of storage metrics collected | 12 |
| `scrape.job_metrics_count` | int | Number of job metrics collected | 156 |
| `scrape.status` | string | Overall scrape status | "success", "partial_failure" |

#### Storage Fetch (netbackup.fetch_storage)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `netbackup.endpoint` | string | API endpoint path | "/storage/storage-units" |
| `netbackup.storage_units` | int | Number of storage units retrieved | 6 |
| `netbackup.api_version` | string | API version used | "13.0" |

#### Job Fetch (netbackup.fetch_jobs)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `netbackup.endpoint` | string | API endpoint path | "/admin/jobs" |
| `netbackup.time_window` | string | Scraping interval | "1h" |
| `netbackup.total_jobs` | int | Total jobs retrieved | 1523 |
| `netbackup.total_pages` | int | Number of pages fetched | 16 |

#### HTTP Request (http.request)

| Attribute | Type | Description | Example |
|-----------|------|-------------|---------|
| `http.method` | string | HTTP method | "GET" |
| `http.url` | string | Full request URL | "<https://nbu:1556/netbackup/admin/jobs>" |
| `http.status_code` | int | HTTP response status code | 200 |
| `http.duration_ms` | int | Request duration in milliseconds | 2341 |

## Common Queries

### Jaeger UI Queries

#### Find All Traces

```
Service: nbu-exporter
Operation: prometheus.scrape
```

#### Find Slow Scrapes (> 30 seconds)

```
Service: nbu-exporter
Operation: prometheus.scrape
Min Duration: 30s
```

#### Find Failed Scrapes

```
Service: nbu-exporter
Tags: scrape.status=partial_failure
```

#### Find Specific Time Range

```
Service: nbu-exporter
Lookback: Custom
Start: 2024-01-15 10:00
End: 2024-01-15 11:00
```

#### Find High Pagination

```
Service: nbu-exporter
Tags: netbackup.total_pages>10
```

### TraceQL Queries (Tempo/Grafana)

#### Find Slow API Calls

```traceql
{ span.http.duration_ms > 5000 }
```

#### Find Failed HTTP Requests

```traceql
{ span.http.status_code >= 400 }
```

#### Find Scrapes with Many Jobs

```traceql
{ span.netbackup.total_jobs > 1000 }
```

#### Find Storage Fetch Issues

```traceql
{ name = "netbackup.fetch_storage" && duration > 10s }
```

#### Aggregate by Status Code

```traceql
{ span.http.status_code } | count() by (span.http.status_code)
```

## Performance Analysis

### Identifying Bottlenecks

#### Step 1: Find Slow Traces

1. Open Jaeger UI
2. Select service: `nbu-exporter`
3. Set "Min Duration" to 30s
4. Click "Find Traces"
5. Sort by duration (longest first)

#### Step 2: Analyze Span Timeline

Click on a slow trace and examine the waterfall view:

**Example 1: Slow Storage Fetch**

```
prometheus.scrape: 35.2s
├── netbackup.fetch_storage: 30.1s ⚠️ BOTTLENECK
│   └── http.request: 30.0s
└── netbackup.fetch_jobs: 5.1s
```

**Diagnosis**: Storage API is slow
**Action**: Check NetBackup server performance, verify network latency

**Example 2: High Pagination**

```
prometheus.scrape: 48.3s
├── netbackup.fetch_storage: 2.1s
└── netbackup.fetch_jobs: 46.2s ⚠️ BOTTLENECK
    ├── netbackup.fetch_job_page: 15.4s
    ├── netbackup.fetch_job_page: 15.3s
    └── netbackup.fetch_job_page: 15.5s
```

**Diagnosis**: Too many job pages (high job volume)
**Action**: Reduce `scrapingInterval` from 1h to 30m

**Example 3: API Errors**

```
prometheus.scrape: 25.2s
├── netbackup.fetch_storage: 2.1s
└── netbackup.fetch_jobs: 23.1s
    ├── netbackup.fetch_job_page: 5.2s (http.status_code=500) ⚠️ ERROR
    ├── netbackup.fetch_job_page: 5.3s (http.status_code=500) ⚠️ ERROR
    └── netbackup.fetch_job_page: 5.4s (http.status_code=500) ⚠️ ERROR
```

**Diagnosis**: NetBackup API errors
**Action**: Check NetBackup server logs, verify API key permissions

#### Step 3: Examine Span Attributes

Click on a span to view its attributes:

**Key metrics to check:**

- `http.duration_ms`: Request latency
- `http.status_code`: Success/failure status
- `netbackup.total_pages`: Pagination count
- `netbackup.total_jobs`: Job volume

### Performance Metrics

#### Baseline Performance

**Normal scrape (< 1000 jobs, 6 storage units):**

```
prometheus.scrape: 8-12s
├── netbackup.fetch_storage: 1-2s
└── netbackup.fetch_jobs: 6-10s
    └── 1-2 pages @ 3-5s each
```

**High-volume scrape (> 5000 jobs):**

```
prometheus.scrape: 45-60s
├── netbackup.fetch_storage: 1-2s
└── netbackup.fetch_jobs: 43-58s
    └── 10-15 pages @ 3-5s each
```

#### Performance Targets

| Metric | Target | Warning | Critical |
|--------|--------|---------|----------|
| Total scrape duration | < 30s | 30-60s | > 60s |
| Storage fetch | < 5s | 5-10s | > 10s |
| Job page fetch | < 5s | 5-10s | > 10s |
| HTTP status codes | 200 | 4xx | 5xx |
| Total pages | < 5 | 5-10 | > 10 |

## Troubleshooting Scenarios

### Scenario 1: Scrapes Timing Out

**Symptoms:**

- Prometheus scrape timeout errors
- Incomplete metrics
- Traces show > 60s duration

**Analysis:**

```
Query: Min Duration: 60s
Look for: High netbackup.total_pages
```

**Solutions:**

1. Reduce `scrapingInterval` from 1h to 30m
2. Increase Prometheus scrape timeout
3. Optimize NetBackup server performance

### Scenario 2: Intermittent Failures

**Symptoms:**

- Some scrapes succeed, others fail
- HTTP 500 errors in traces
- `scrape.status=partial_failure`

**Analysis:**

```
Query: Tags: scrape.status=partial_failure
Look for: http.status_code >= 500
```

**Solutions:**

1. Check NetBackup server logs for errors
2. Verify API key has correct permissions
3. Check network connectivity
4. Review NetBackup server resource usage

### Scenario 3: Slow Storage Fetch

**Symptoms:**

- Storage metrics delayed
- `netbackup.fetch_storage` > 10s

**Analysis:**

```
Query: Operation: netbackup.fetch_storage, Min Duration: 10s
Look for: http.duration_ms in child spans
```

**Solutions:**

1. Check NetBackup storage service status
2. Verify network latency to NetBackup server
3. Review NetBackup server disk I/O
4. Check for storage unit issues

### Scenario 4: High Pagination

**Symptoms:**

- Long scrape durations
- Many `netbackup.fetch_job_page` spans
- `netbackup.total_pages` > 10

**Analysis:**

```
Query: Tags: netbackup.total_pages>10
Look for: Pattern of multiple page fetches
```

**Solutions:**

1. Reduce `scrapingInterval` to fetch fewer jobs
2. Consider filtering jobs by policy or status
3. Optimize NetBackup job retention policies

### Scenario 5: API Version Issues

**Symptoms:**

- HTTP 406 errors
- Version detection failures
- Inconsistent API responses

**Analysis:**

```
Query: Tags: http.status_code=406
Look for: netbackup.api_version attribute
```

**Solutions:**

1. Verify NetBackup version compatibility
2. Check API version configuration
3. Enable automatic version detection
4. Review NetBackup API logs

## Best Practices

### Query Optimization

**Use specific time ranges:**

```
Lookback: Last 1 hour (instead of Last 24 hours)
```

**Filter by operation:**

```
Operation: netbackup.fetch_jobs (instead of all operations)
```

**Use tags for precision:**

```
Tags: netbackup.api_version=13.0
```

### Sampling Strategy

**Development:**

```yaml
samplingRate: 1.0  # Trace everything
```

**Production (normal load):**

```yaml
samplingRate: 0.1  # 10% sampling
```

**Production (high load):**

```yaml
samplingRate: 0.01  # 1% sampling
```

### Alerting on Traces

**Create alerts for:**

- Scrape duration > 60s
- HTTP status codes >= 500
- High pagination (> 10 pages)
- Partial failures

**Example Prometheus alert:**

```yaml
- alert: SlowNBUScrape
  expr: histogram_quantile(0.95, rate(scrape_duration_seconds_bucket[5m])) > 60
  annotations:
    summary: "NBU scrapes are slow"
    description: "95th percentile scrape duration is {{ $value }}s"
```

### Regular Analysis

**Daily:**

- Check for failed scrapes
- Review slow traces (> 30s)
- Monitor HTTP error rates

**Weekly:**

- Analyze pagination trends
- Review API version distribution
- Check sampling effectiveness

**Monthly:**

- Optimize scraping intervals
- Review trace retention policies
- Update performance baselines

## Advanced Analysis

### Comparing Traces

**Compare before/after optimization:**

1. Find baseline trace (before changes)
2. Note key metrics (duration, pages, etc.)
3. Make configuration changes
4. Find new trace (after changes)
5. Compare metrics side-by-side

**Example comparison:**

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Total duration | 48.3s | 12.1s | 75% faster |
| Total pages | 15 | 3 | 80% reduction |
| Job count | 5234 | 1156 | Reduced scope |

### Correlation with Metrics

**Link traces to Prometheus metrics:**

1. Note trace timestamp
2. Query Prometheus for same time range
3. Correlate trace spans with metric values
4. Identify patterns

**Example:**

```
Trace shows: 15 pages fetched
Prometheus shows: nbu_jobs_count{} = 5234
Calculation: 5234 / 15 ≈ 349 jobs per page
```

### Exporting Trace Data

**Export for analysis:**

1. In Jaeger UI, click "JSON" on trace view
2. Save trace JSON
3. Analyze with custom tools
4. Generate reports

**Example Python analysis:**

```python
import json

with open('trace.json') as f:
    trace = json.load(f)

for span in trace['spans']:
    if span['operationName'] == 'http.request':
        duration = span['duration'] / 1000  # Convert to ms
        print(f"HTTP request: {duration}ms")
```

## Span Attributes Reference

### Complete Attribute List

| Span | Attribute | Type | Description |
|------|-----------|------|-------------|
| prometheus.scrape | scrape.duration_ms | int | Total scrape duration |
| prometheus.scrape | scrape.storage_metrics_count | int | Storage metrics collected |
| prometheus.scrape | scrape.job_metrics_count | int | Job metrics collected |
| prometheus.scrape | scrape.status | string | Overall status |
| netbackup.fetch_storage | netbackup.endpoint | string | API endpoint |
| netbackup.fetch_storage | netbackup.storage_units | int | Storage units retrieved |
| netbackup.fetch_storage | netbackup.api_version | string | API version |
| netbackup.fetch_jobs | netbackup.endpoint | string | API endpoint |
| netbackup.fetch_jobs | netbackup.time_window | string | Scraping interval |
| netbackup.fetch_jobs | netbackup.total_jobs | int | Total jobs retrieved |
| netbackup.fetch_jobs | netbackup.total_pages | int | Pages fetched |
| netbackup.fetch_job_page | netbackup.page_offset | int | Page offset |
| netbackup.fetch_job_page | netbackup.jobs_in_page | int | Jobs in this page |
| http.request | http.method | string | HTTP method |
| http.request | http.url | string | Request URL |
| http.request | http.status_code | int | Response status |
| http.request | http.duration_ms | int | Request duration |

## Further Reading

- [OpenTelemetry Tracing Specification](https://opentelemetry.io/docs/specs/otel/trace/)
- [Jaeger Query Documentation](https://www.jaegertracing.io/docs/latest/frontend-ui/)
- [TraceQL Documentation](https://grafana.com/docs/tempo/latest/traceql/)
- [Distributed Tracing Best Practices](https://opentelemetry.io/docs/concepts/observability-primer/#distributed-tracing)
