# Security Headers & CORS Reference Policy

Production Go services serving REST APIs must attach security headers to all HTTP responses to protect clients against Cross-Site Scripting (XSS), Clickjacking, MIME-sniffing, and MITM attacks.

---

## 1. Mandatory HTTP Security Headers

| Header | Production Recommended Value | Purpose |
| :--- | :--- | :--- |
| `X-Content-Type-Options` | `nosniff` | Prevents browser MIME-type sniffing |
| `X-Frame-Options` | `DENY` (or `SAMEORIGIN`) | Prevents clickjacking by blocking iframe embedding |
| `X-XSS-Protection` | `1; mode=block` | Legacy filter activation for older browsers |
| `Strict-Transport-Security` | `max-age=63072000; includeSubDomains; preload` | Enforces HTTPS strictly (2 years) |
| `Content-Security-Policy` | `default-src 'self'; frame-ancestors 'none';` | Restricts resource origins and framing |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | Strips path/query from referrer across origins |
| `Permissions-Policy` | `geolocation=(), camera=(), microphone=()` | Disables sensitive browser hardware APIs |

---

## 2. Cross-Origin Resource Sharing (CORS) Policy

CORS must be strictly configured based on environment:

### Forbidden in Production

- `Access-Control-Allow-Origin: *` when credentials (`cookies`, `Authorization`) are allowed.
- Dynamic reflection of `Origin` header without whitelist verification.

### Recommended Configuration

```go
// CORSConfig defines production CORS parameters.
type CORSConfig struct {
 AllowedOrigins   []string
 AllowedMethods   []string
 AllowedHeaders   []string
 ExposedHeaders   []string
 AllowCredentials bool
 MaxAgeSeconds    int
}

// Production Defaults
var DefaultCORS = CORSConfig{
 AllowedOrigins: []string{
  "https://app.example.com",
  "https://admin.example.com",
 },
 AllowedMethods: []string{
  "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS",
 },
 AllowedHeaders: []string{
  "Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID",
 },
 ExposedHeaders: []string{
  "X-Request-ID", "Retry-After",
 },
 AllowCredentials: true,
 MaxAgeSeconds:    86400, // 24 hours preflight cache
}
```
