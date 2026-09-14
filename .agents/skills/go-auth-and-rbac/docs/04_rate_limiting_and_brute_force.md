# Rate Limiting & Brute Force Prevention

Authentication endpoints (`/api/v1/auth/login`, `/api/v1/auth/reset-password`) are prime targets for automated credential stuffing and brute-force dictionary attacks. A production Go service must throttle requests before expensive cryptographic operations (e.g. `bcrypt` password hashing) consume server CPU.

---

## 1. Multi-Dimensional Rate Limiting

Effective defense requires limiting along multiple dimensions simultaneously:

1. **IP-Based Limiting**: Limits requests per remote IP address (e.g., maximum 10 attempts per minute). Protects against single-host brute-force attacks.
2. **Account/Identity-Based Limiting**: Limits attempts against a specific account/email (e.g., maximum 5 failed attempts per 15 minutes before locking). Protects against distributed credential stuffing across botnets.
3. **Global Endpoint Limiting**: Overall ceiling for authentication traffic to protect server capacity.

---

## 2. In-Memory Token Bucket Algorithm

For single-instance or localized rate limiting, the Token Bucket algorithm provides smooth burst handling and constant refilling:

```text
Tokens in bucket: [● ● ● ● ●]  (Capacity: 5)
Requests consume:  ▼
Refill rate:       1 token per 5 seconds
Empty bucket:      Returns 429 Too Many Requests
```

### Key Headers to Return on Throttling

When a request is rate-limited, return HTTP 429 and standard rate-limiting headers:

- `Retry-After`: Number of seconds to wait before retrying.
- `X-RateLimit-Limit`: Maximum allowed attempts in window.
- `X-RateLimit-Remaining`: Remaining attempts.
- `X-RateLimit-Reset`: Unix timestamp when bucket refills.

---

## 3. Account Lockout & Graduated Delays

- **Exponential Delay**: After 3 failed password attempts, introduce a progressive artificial delay (e.g., 500ms, 1s, 2s) before returning the response.
- **Account Lockout**: After 5 failed attempts within 15 minutes, temporarily lock the account for 30 minutes. Send a security notification email to the account owner.
- **Constant Time Password Verification**: Always execute password hashing checks in constant time (`bcrypt.CompareHashAndPassword` or `subtle.ConstantTimeCompare`) even if the user does not exist (using a dummy hash) to prevent user enumeration via timing attacks.
