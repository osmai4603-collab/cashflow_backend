package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or malformed token")
	ErrExpiredToken = errors.New("token has expired")
)

// GenerateToken generates a signed HMAC-SHA256 JWT containing user ID, company ID, and roles.
func GenerateToken(userID, companyID int64, roles []string, secret string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret cannot be empty")
	}

	now := time.Now().UTC()
	claims := UserClaims{
		UserID:    userID,
		CompanyID: companyID,
		Roles:     roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign jwt token: %w", err)
	}

	return tokenStr, nil
}

// ValidateToken parses and validates a signed JWT string against the secret key.
func ValidateToken(tokenString, secret string) (*UserClaims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret cannot be empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*UserClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
