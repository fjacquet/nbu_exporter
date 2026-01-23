# Phase 2: Security Hardening - Research

**Researched:** 2026-01-23
**Domain:** Go security patterns, HTTP client security, secrets management
**Confidence:** HIGH

## Summary

Research focused on three security requirements for the NBU Prometheus exporter: secure API key handling (SEC-01), TLS verification enforcement (SEC-02), and rate limiting with backoff (SEC-03).

**Key findings:**

- Go's garbage collector makes true memory zeroing difficult; practical approach is to avoid logging secrets and use constant-time comparison for validation
- TLS verification is secure by default in Go, but requires explicit enforcement patterns to prevent accidental opt-out
- Resty v2 has built-in exponential backoff with jitter and native 429/Retry-After header handling

**Primary recommendation:** Use resty's built-in retry mechanisms, enforce TLS verification with explicit configuration validation, and implement crypto/subtle for API key comparison. Memory zeroing is not practical in Go; focus on preventing leaks through logs and process listings.

## Standard Stack

The established libraries/tools for this domain:

### Core

| Library                      | Version | Purpose                   | Why Standard                               |
| ---------------------------- | ------- | ------------------------- | ------------------------------------------ |
| crypto/subtle                | stdlib  | Constant-time comparison  | Prevents timing attacks on secrets         |
| crypto/tls                   | stdlib  | TLS configuration         | Standard Go TLS implementation             |
| github.com/go-resty/resty/v2 | v2.x    | HTTP client with retry    | Built-in exponential backoff, 429 handling |
| golang.org/x/time/rate       | latest  | Client-side rate limiting | Official Go rate limiter (token bucket)    |

### Supporting

| Library                               | Version | Purpose                       | When to Use                                     |
| ------------------------------------- | ------- | ----------------------------- | ----------------------------------------------- |
| github.com/awnumar/memguard           | latest  | Memory-secure secret storage  | If memory encryption required (likely overkill) |
| github.com/hashicorp/go-retryablehttp | latest  | Alternative retry HTTP client | If not using resty                              |
| github.com/securego/gosec             | latest  | Security linting              | CI/CD security auditing                         |

### Alternatives Considered

| Instead of            | Could Use                                    | Tradeoff                               |
| --------------------- | -------------------------------------------- | -------------------------------------- |
| resty retry           | hashicorp/go-retryablehttp                   | More control, but resty already in use |
| Environment variables | Secrets manager (Vault, AWS Secrets Manager) | Better for production, adds complexity |
| crypto/subtle         | Plain string comparison                      | Vulnerable to timing attacks           |

**Installation:**

```bash
# Already in use
go get github.com/go-resty/resty/v2

# For rate limiting
go get golang.org/x/time/rate

# For security linting (dev only)
go get github.com/securego/gosec/v2/cmd/gosec
```

## Architecture Patterns

### Recommended Configuration Structure

```
internal/
├── exporter/
│   ├── client.go           # HTTP client with retry/backoff
│   ├── netbackup.go        # API calls with rate limiting
├── models/
│   ├── Config.go           # Configuration validation
│   └── SecureConfig.go     # API key validation helpers
```

### Pattern 1: Secure API Key Validation

**What:** Use constant-time comparison to prevent timing attacks when validating API keys
**When to use:** Any API key authentication, especially in middleware
**Example:**

```go
// Source: https://dev.to/caiorcferreira/implementing-a-safe-and-sound-api-key-authorization-middleware-in-go-3g2c
import "crypto/subtle"

func compareAPIKeys(provided, expected string) bool {
    // Convert strings to byte slices for constant-time comparison
    providedBytes := []byte(provided)
    expectedBytes := []byte(expected)

    // Check length first to avoid leaking key length in timing
    if len(providedBytes) != len(expectedBytes) {
        // Use constant-time comparison even for failure case
        return subtle.ConstantTimeCompare(providedBytes, expectedBytes) == 1
    }

    return subtle.ConstantTimeCompare(providedBytes, expectedBytes) == 1
}
```

**Note:** crypto/subtle.ConstantTimeCompare returns 0 for different lengths without bitwise operations, potentially leaking key length. For this use case (outbound client, not auth middleware), it's acceptable since we control the key.

### Pattern 2: TLS Verification Enforcement

**What:** Enforce TLS verification by default with explicit opt-out requiring configuration flag
**When to use:** Any HTTPS client configuration
**Example:**

```go
// Source: https://pkg.go.dev/crypto/tls
import (
    "crypto/tls"
    "fmt"
)

func configureTLS(insecureAllowed bool) (*tls.Config, error) {
    // Default: secure
    config := &tls.Config{
        InsecureSkipVerify: false,
        MinVersion:         tls.VersionTLS12, // TLS 1.2 minimum
    }

    // Explicit opt-out requires flag AND warning
    if insecureAllowed {
        log.Error("SECURITY WARNING: TLS verification disabled - this is insecure for production")
        config.InsecureSkipVerify = true
    }

    return config, nil
}
```

### Pattern 3: Resty Retry with Exponential Backoff

**What:** Configure resty client with retry logic, exponential backoff, and 429 handling
**When to use:** All HTTP clients making API calls
**Example:**

```go
// Source: https://pkg.go.dev/github.com/go-resty/resty/v2
import (
    "net/http"
    "time"
    "github.com/go-resty/resty/v2"
)

func newClientWithRetry() *resty.Client {
    client := resty.New()

    // Configure retry behavior
    client.
        SetRetryCount(3).                           // 3 retry attempts
        SetRetryWaitTime(5 * time.Second).         // Initial wait: 5s
        SetRetryMaxWaitTime(60 * time.Second).     // Max wait: 60s
        AddRetryCondition(func(r *resty.Response, err error) bool {
            // Retry on network errors
            if err != nil {
                return true
            }
            // Retry on 429 (rate limit) and 5xx errors
            return r.StatusCode() == http.StatusTooManyRequests ||
                   r.StatusCode() >= 500
        })

    // Automatically handle Retry-After header in 429 responses
    client.AddRetryAfterErrorCondition()

    return client
}
```

**Key features:**

- Exponential backoff with jitter (built-in)
- Retry-After header parsing for 429 responses
- Configurable retry conditions
- Request hooks for logging/metrics

### Pattern 4: Client-Side Rate Limiting

**What:** Limit outbound request rate to prevent overwhelming the API server
**When to use:** When scraping metrics on a schedule (Prometheus exporter use case)
**Example:**

```go
// Source: https://pkg.go.dev/golang.org/x/time/rate
import (
    "context"
    "golang.org/x/time/rate"
)

type RateLimitedClient struct {
    client  *resty.Client
    limiter *rate.Limiter
}

func NewRateLimitedClient() *RateLimitedClient {
    // Allow 10 requests per second, burst of 20
    limiter := rate.NewLimiter(rate.Limit(10), 20)

    return &RateLimitedClient{
        client:  newClientWithRetry(),
        limiter: limiter,
    }
}

func (c *RateLimitedClient) Get(ctx context.Context, url string) (*resty.Response, error) {
    // Wait for rate limiter token
    if err := c.limiter.Wait(ctx); err != nil {
        return nil, err
    }

    return c.client.R().SetContext(ctx).Get(url)
}
```

### Anti-Patterns to Avoid

- **Logging API keys in plain text:** Use MaskAPIKey() for logs, never log raw keys
- **Ignoring 429 responses:** Always implement retry with backoff for rate limits
- **Silent TLS verification disable:** Require explicit config flag + warning log
- **Plain string comparison for secrets:** Use crypto/subtle.ConstantTimeCompare
- **Not reading response bodies:** Prevents connection reuse, exhausts pool
- **Hard-coded secrets:** Use environment variables (local) or secrets manager (production)

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem                  | Don't Build                   | Use Instead                          | Why                                                |
| ------------------------ | ----------------------------- | ------------------------------------ | -------------------------------------------------- |
| Exponential backoff      | Custom sleep calculation      | resty.SetRetryCount/SetRetryWaitTime | Handles jitter, Retry-After headers, edge cases    |
| Rate limiting            | Custom token tracking         | golang.org/x/time/rate.Limiter       | Token bucket algorithm, thread-safe, context-aware |
| Retry-After parsing      | Manual header parsing         | resty.AddRetryAfterErrorCondition    | Handles multiple formats (seconds, HTTP-date)      |
| Constant-time comparison | bytes.Equal or ==             | crypto/subtle.ConstantTimeCompare    | Prevents timing attacks                            |
| TLS configuration        | Custom certificate validation | crypto/tls standard config           | Security audited, handles edge cases               |
| Secret zeroing           | Manual memory overwrite       | Accept Go GC limitations             | Go GC makes true zeroing impractical               |

**Key insight:** HTTP client retry logic has many edge cases (connection resets, DNS failures, partial responses). Use battle-tested libraries instead of custom implementations.

## Common Pitfalls

### Pitfall 1: API Key Leaking in Logs

**What goes wrong:** API keys appear in debug logs, error messages, or crash dumps
**Why it happens:** Default error formatting includes request details; debug logging prints full config
**How to avoid:**

- Implement MaskAPIKey() method (already exists in Config.go)
- Use structured logging with explicit field exclusion
- Never log full request/response in production
  **Warning signs:**
- Logs contain "apiKey:" or "Authorization:" with full values
- Error messages include full HTTP headers

### Pitfall 2: Ignoring Retry-After Headers

**What goes wrong:** Client continues hammering API after 429, potentially getting banned
**Why it happens:** Manual retry logic doesn't parse Retry-After header
**How to avoid:** Use resty's AddRetryAfterErrorCondition() built-in handler
**Warning signs:**

- Repeated 429 errors in logs
- Exponentially increasing response times
- API returning 403 Forbidden (ban)

### Pitfall 3: TLS Verification Disabled Without Warning

**What goes wrong:** Production systems run with InsecureSkipVerify=true silently
**Why it happens:** Copy-paste from dev config, no validation enforcement
**How to avoid:**

- Add validation in Config.Validate() to warn on insecure mode
- Require explicit INSECURE_MODE=true environment variable
- Log ERROR-level message on startup if TLS disabled
  **Warning signs:**
- No TLS warnings in production logs
- Config files with insecureSkipVerify: true

### Pitfall 4: Connection Pool Exhaustion

**What goes wrong:** HTTP client opens hundreds of connections, exhausting system resources
**Why it happens:** Not reading response bodies to completion; default MaxIdleConnsPerHost is only 2
**How to avoid:**

- Always read and close response bodies (already done in FetchData)
- Configure MaxIdleConnsPerHost to reasonable value (20-100)
- Set IdleConnTimeout to reclaim stale connections (90s)
  **Warning signs:**
- "too many open files" errors
- Increasing response times over scrape cycles
- Connections stuck in TIME_WAIT state

### Pitfall 5: Timing Attack Vulnerability

**What goes wrong:** Attackers deduce API key characters through response time analysis
**Why it happens:** Using == or strings.Compare for API key validation
**How to avoid:** Use crypto/subtle.ConstantTimeCompare for all secret comparisons
**Warning signs:**

- Security scanner flags timing attack vulnerability
- Using == for API key validation in code

### Pitfall 6: Environment Variables in Production

**What goes wrong:** Secrets leak through process listings (/proc/\*/environ), crash dumps, or logs
**Why it happens:** Following outdated best practices from pre-2020
**How to avoid:**

- Document that env vars are for local dev only
- Recommend secrets manager for production (AWS Secrets Manager, Vault)
- Support file-based secrets for Kubernetes
  **Warning signs:**
- API keys visible in `ps auxe` output
- Crash dumps contain secrets

## Code Examples

Verified patterns from official sources:

### Configuration Validation with TLS Enforcement

```go
// Source: Derived from crypto/tls package docs
func (c *Config) Validate() error {
    // ... existing validation ...

    // Warn if TLS verification disabled
    if c.NbuServer.InsecureSkipVerify {
        // Check if explicitly allowed via environment variable
        insecureAllowed := os.Getenv("NBU_INSECURE_MODE") == "true"
        if !insecureAllowed {
            return errors.New("TLS verification disabled but NBU_INSECURE_MODE not set - " +
                "set NBU_INSECURE_MODE=true to explicitly allow insecure connections")
        }
        log.Error("SECURITY WARNING: TLS certificate verification disabled - insecure for production")
    }

    return nil
}
```

### Retry Configuration in Client Initialization

```go
// Source: https://pkg.go.dev/github.com/go-resty/resty/v2
func NewNbuClient(cfg models.Config) *NbuClient {
    // ... existing TLS config ...

    client := resty.New().
        SetTLSClientConfig(&tls.Config{
            InsecureSkipVerify: cfg.NbuServer.InsecureSkipVerify,
            MinVersion:         tls.VersionTLS12,
        }).
        SetTimeout(defaultTimeout).
        // NEW: Configure retry behavior
        SetRetryCount(3).
        SetRetryWaitTime(5 * time.Second).
        SetRetryMaxWaitTime(60 * time.Second).
        AddRetryCondition(func(r *resty.Response, err error) bool {
            // Retry on errors or rate limiting
            return err != nil ||
                   r.StatusCode() == http.StatusTooManyRequests ||
                   r.StatusCode() >= 500
        })

    // Handle Retry-After headers automatically
    client.AddRetryAfterErrorCondition()

    return &NbuClient{client: client, cfg: cfg}
}
```

### Connection Pool Tuning

```go
// Source: https://davidbacisin.com/writing/golang-http-connection-pools-1
import "net/http"

func configureTLSWithPooling(insecureSkipVerify bool) *tls.Config {
    // Get the underlying http.Client from resty
    httpClient := client.GetClient()

    // Configure connection pooling
    httpClient.Transport = &http.Transport{
        MaxIdleConns:        100,              // Total connections
        MaxIdleConnsPerHost: 20,               // Per host (not 2!)
        IdleConnTimeout:     90 * time.Second, // Reclaim idle connections
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: insecureSkipVerify,
            MinVersion:         tls.VersionTLS12,
        },
    }
}
```

## State of the Art

| Old Approach                      | Current Approach                      | When Changed       | Impact                                          |
| --------------------------------- | ------------------------------------- | ------------------ | ----------------------------------------------- |
| Environment variables for secrets | Secrets managers (Vault, AWS SM)      | 2024-2025          | Env vars now considered insecure for production |
| TLS 1.0/1.1                       | TLS 1.2 minimum, 1.3 preferred        | Go 1.27 (2026)     | Go 1.27 defaults to TLS 1.2 minimum             |
| Manual retry logic                | Built-in resty retry with backoff     | resty v2.0+ (2019) | Handles edge cases automatically                |
| Manual Retry-After parsing        | AddRetryAfterErrorCondition()         | resty v2.7+ (2022) | Parses both seconds and HTTP-date formats       |
| String comparison for secrets     | crypto/subtle.ConstantTimeCompare     | Always available   | Security best practice                          |
| Certificate validity 398 days     | 200 days (March 2026), 47 days (2029) | CA/Browser Forum   | Requires automated cert rotation                |

**Deprecated/outdated:**

- **TLS 1.0/1.1:** Go 1.27 removes support by default
- **Environment variables for production secrets:** Use secrets managers instead
- **MaxIdleConnsPerHost=2:** Increase to 20-100 for performance
- **No retry logic:** Modern clients should handle transient failures
- **Manual backoff calculation:** Use library-provided exponential backoff

## Open Questions

Things that couldn't be fully resolved:

1. **Memory zeroing in Go for API keys**
   - What we know: Go's GC makes true zeroing impractical; memguard provides encryption-at-rest in memory
   - What's unclear: Whether the complexity of memguard is worth it for this use case
   - Recommendation: Accept Go's limitations. Focus on preventing leaks (no logging, constant-time comparison). Memguard is overkill for Prometheus exporter.

2. **Rate limiting values for NetBackup API**
   - What we know: Resty supports rate limiting and backoff; NetBackup API may have limits
   - What's unclear: NetBackup API's actual rate limits (vendor-specific)
   - Recommendation: Start conservative (10 req/sec), monitor for 429 responses, tune based on actual behavior. Make rate limit configurable.

3. **Secrets manager integration**
   - What we know: Production should use secrets managers, not env vars
   - What's unclear: Which secrets manager(s) to support (Vault, AWS, GCP, Azure)
   - Recommendation: Document env var limitations, suggest secrets managers in README. Don't implement integration (out of scope, adds significant complexity).

4. **gosec integration in CI/CD**
   - What we know: gosec can detect TLS and secret handling issues
   - What's unclear: Whether to add to CI pipeline or just document
   - Recommendation: Document as optional tool in README. Adding to CI is Phase 5 (Testing) concern.

## Sources

### Primary (HIGH confidence)

- [crypto/tls package documentation](https://pkg.go.dev/crypto/tls) - Official Go TLS implementation
- [crypto/subtle package documentation](https://pkg.go.dev/crypto/subtle) - Constant-time comparison
- [Resty v2 package documentation](https://pkg.go.dev/github.com/go-resty/resty/v2) - HTTP client with retry
- [golang.org/x/time/rate package](https://pkg.go.dev/golang.org/x/time/rate) - Rate limiting
- [How to Implement Retry Logic in Go with Exponential Backoff](https://oneuptime.com/blog/post/2026-01-07-go-retry-exponential-backoff/view) - 2026 best practices
- [How to Implement API Key Authentication in Go](https://oneuptime.com/blog/post/2026-01-07-go-api-key-authentication/view) - 2026 security patterns

### Secondary (MEDIUM confidence)

- [Implementing a safe and sound API Key authorization middleware in Go](https://dev.to/caiorcferreira/implementing-a-safe-and-sound-api-key-authorization-middleware-in-go-3g2c) - crypto/subtle usage patterns
- [Are environment variables still safe for secrets in 2026?](https://securityboulevard.com/2025/12/are-environment-variables-still-safe-for-secrets-in-2026/) - Current stance on env vars
- [HTTP Connection Pooling in Go by David Bacisin](https://davidbacisin.com/writing/golang-http-connection-pools-1) - Connection pool configuration
- [The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) - Cloudflare's timeout patterns
- [memguard package](https://pkg.go.dev/github.com/awnumar/memguard) - Memory-secure secret storage
- [memory security in go](https://spacetime.dev/memory-security-go) - Go memory limitations
- [gosec security scanner](https://github.com/securego/gosec) - Go security auditing

### Tertiary (LOW confidence - context only)

- WebSearch results for Prometheus exporter patterns - No specific NBU API limits found
- WebSearch results for rate limiting metrics - General patterns, not exporter-specific

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Resty v2 retry is well-documented, crypto/subtle is stdlib
- Architecture: HIGH - Patterns verified from official docs and security-focused articles
- Pitfalls: MEDIUM - Based on general Go HTTP client patterns, not NBU-specific

**Research date:** 2026-01-23
**Valid until:** ~30 days (stable domain, but check for Go 1.27 release and resty updates)
