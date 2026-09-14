package examples

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidToken       = errors.New("invalid token format")
	ErrUnsupportedAlg     = errors.New("unsupported signing algorithm: only HS256 permitted")
	ErrSignatureMismatch  = errors.New("token signature verification failed")
	ErrTokenExpired       = errors.New("token has expired")
	ErrWeakSigningKey     = errors.New("signing secret must be at least 32 bytes")
)

// JWTHeader represents the header of a JWT.
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// TokenClaims holds standard and application-specific claims.
type TokenClaims struct {
	Subject   string   `json:"sub"`
	CompanyID string   `json:"company_id"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	Issuer    string   `json:"iss"`
}

// JWTAuthenticator validates incoming Bearer tokens using HMAC-SHA256.
type JWTAuthenticator struct {
	secret        []byte
	expectedIssuer string
	clockSkew     time.Duration
}

// NewJWTAuthenticator instantiates a validator with a validated secret key.
func NewJWTAuthenticator(secret string, expectedIssuer string) (*JWTAuthenticator, error) {
	key := []byte(secret)
	if len(key) < 32 {
		return nil, ErrWeakSigningKey
	}
	return &JWTAuthenticator{
		secret:         key,
		expectedIssuer: expectedIssuer,
		clockSkew:      60 * time.Second,
	}, nil
}

// Verify parses and cryptographically validates a JWT token string.
func (a *JWTAuthenticator) Verify(tokenStr string) (*TokenClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid header encoding", ErrInvalidToken)
	}

	var header JWTHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("%w: invalid header json", ErrInvalidToken)
	}

	if header.Alg != "HS256" {
		return nil, ErrUnsupportedAlg
	}

	// Verify HMAC-SHA256 signature
	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid signature encoding", ErrInvalidToken)
	}

	if !hmac.Equal(actualSig, expectedSig) {
		return nil, ErrSignatureMismatch
	}

	// Parse payload claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: invalid payload encoding", ErrInvalidToken)
	}

	var claims TokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("%w: invalid payload json", ErrInvalidToken)
	}

	now := time.Now()
	// Check expiration with clock skew allowance
	if now.Unix() > claims.ExpiresAt+int64(a.clockSkew.Seconds()) {
		return nil, ErrTokenExpired
	}

	if a.expectedIssuer != "" && claims.Issuer != a.expectedIssuer {
		return nil, errors.New("token issuer mismatch")
	}

	return &claims, nil
}

// Middleware creates an HTTP authentication middleware that populates the TenantContext.
func (a *JWTAuthenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"unauthorized","message":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			http.Error(w, `{"error":"unauthorized","message":"invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		claims, err := a.Verify(parts[1])
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":%q}`, err.Error()), http.StatusUnauthorized)
			return
		}

		// Inject TenantContext into request context
		tenant := TenantContext{
			UserID:    claims.Subject,
			CompanyID: claims.CompanyID,
			Email:     claims.Email,
			Roles:     claims.Roles,
		}
		ctx := WithTenant(r.Context(), tenant)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
