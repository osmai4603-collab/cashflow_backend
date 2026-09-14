# Infrastructure Resilience & Timeout Matrix

Every external outbound connection in the infrastructure layer must have strict timeouts, retry bounds, and circuit breaker policies to prevent cascading network failures from consuming server resources.

---

## 1. Resilience Parameters by External Integration Type

| Integration Type | Connect Timeout | Request Timeout | Retry Policy | Circuit Breaker (Trip / Reset) |
| :--- | :--- | :--- | :--- | :--- |
| **Synchronous Payment Gateways** | 2 seconds | 8 seconds | **0 retries** on charge (use idempotency); 2 retries on read | 5 failures / 30s reset |
| **Email Delivery (SMTP / API)** | 3 seconds | 10 seconds | 3 retries with exponential backoff & jitter | 10 failures / 60s reset |
| **Regulatory / Tax EDI** | 3 seconds | 15 seconds | 2 retries on network timeout | 5 failures / 60s reset |
| **SMS / Push Notification** | 2 seconds | 5 seconds | 2 retries with jitter | 10 failures / 30s reset |
| **External ERP / CRM Sync** | 5 seconds | 30 seconds | Async background worker retry (DLQ after 5 attempts) | 5 failures / 120s reset |

---

## 2. HTTP Transport Best Practices

Configure dedicated `*http.Transport` instances for external infrastructure clients rather than relying on `http.DefaultTransport`:

```go
func NewResilientHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		DisableKeepAlives:   false,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}
```

---

## 3. Idempotency on Mutating Infrastructure Calls

- **Never blindly retry mutating POST calls** (e.g. `POST /v1/charges`) without an `Idempotency-Key` or unique client mutation token.
- If a mutating call times out without a response, the client must treat the transaction as **in-doubt / pending verification**, rather than assuming it failed.
