# JWT Lifecycle, Cryptographic Validation & Refresh Tokens

JSON Web Tokens (JWT) are a compact, URL-safe means of representing claims to be transferred between parties. In Go services, improper token parsing or signature handling is a frequent vector for authentication bypass.

---

## 1. Cryptographic Security Standards

1. **Explicit Algorithm Enforcement**:
   Always whitelist acceptable algorithms (`HS256`, `RS256`, `EdDSA`). Reject tokens using the `none` algorithm or mismatched keys (e.g. attempting to verify an HMAC signature using an RSA public key).
2. **Signature Verification**:
   Signatures must be validated using cryptographic primitives with sufficient entropy:
   - For HMAC: Secret key length $\ge 256$ bits (32 bytes).
   - For RSA: Key size $\ge 2048$ bits.
   - For ECDSA: P-256 or P-384 curves.
3. **Claims Verification**:
   - `exp` (Expiration Time): Enforce strict expiration. Do not allow indefinitely valid tokens.
   - `nbf` (Not Before) & `iat` (Issued At): Reject future-dated tokens allowing for small clock skew ($\le 60$ seconds).
   - `iss` (Issuer): Ensure token was minted by the authorized identity authority.
   - `aud` (Audience): Ensure token was intended for this specific service.

---

## 2. Token Lifecycle: Access vs Refresh Tokens

| Attribute | Access Token | Refresh Token |
| :--- | :--- | :--- |
| **Lifespan** | Short (10 to 15 minutes) | Long (7 to 30 days) |
| **Storage** | Memory / In-flight Authorization Header | Secure HttpOnly, SameSite=Strict Cookie |
| **Payload** | Rich claims (User ID, Company ID, Roles) | Opaque identifier or minimal cryptographically signed ID |
| **Revocation** | Difficult without distributed blocklist | Checked against database/cache on every refresh |
| **Rotation** | Generated upon refresh | Rotated on every use (Refresh Token Rotation - RTR) |

---

## 3. JWT Claims Structure

Standardized claim structure for multi-tenant Go services:

```go
package auth

import (
	"time"
)

// AppClaims represents the standard token claims structure.
type AppClaims struct {
	Subject   string   `json:"sub"`        // User ID
	CompanyID string   `json:"company_id"` // Multi-tenant company/tenant ID
	Email     string   `json:"email"`      // User email
	Roles     []string `json:"roles"`      // Assigned roles (e.g. ["admin", "accountant"])
	IssuedAt  int64    `json:"iat"`        // Unix timestamp
	ExpiresAt int64    `json:"exp"`        // Unix timestamp
	Issuer    string   `json:"iss"`        // Originating auth service
}

// Valid performs standard validation checks.
func (c *AppClaims) Valid(now time.Time) bool {
	if c.ExpiresAt <= now.Unix() {
		return false
	}
	if c.CompanyID == "" || c.Subject == "" {
		return false
	}
	return true
}
```

---

## 4. Refresh Token Rotation (RTR) & Reuse Detection

To prevent compromised refresh tokens from granting infinite access:
1. When a client presents `RefreshToken_A`, the server validates it.
2. The server issues a new pair: `AccessToken_2` and `RefreshToken_B`.
3. `RefreshToken_A` is marked as revoked/consumed.
4. **Reuse Detection**: If `RefreshToken_A` is ever presented again, the system detects a token reuse breach and **invalidates all active sessions** for that user immediately.
