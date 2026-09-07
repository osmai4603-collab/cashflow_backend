package config

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// DefaultPBKDF2Rounds matches Odoo's passlib pbkdf2_sha512 configuration.
const DefaultPBKDF2Rounds = 600_000

// SetAdminPassword hashes the given plaintext (using pbkdf2_sha512) and stores
// the result in Security.AdminHash, mirroring Odoo's admin_passwd handling.
func (c *Configuration) SetAdminPassword(plaintext string, rounds int) error {
	if rounds <= 0 {
		rounds = DefaultPBKDF2Rounds
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generate salt: %w", err)
	}
	digest := pbkdf2.Key([]byte(plaintext), salt, rounds, sha512.Size, sha512.New)
	c.Security.AdminHash = encodePBKDF2(rounds, salt, digest)
	return nil
}

// VerifyAdminPassword reports whether the plaintext matches the stored
// pbkdf2_sha512 hash. An empty stored hash always returns false.
func (c *Configuration) VerifyAdminPassword(plaintext string) bool {
	if plaintext == "" || c.Security.AdminHash == "" {
		return false
	}
	rounds, salt, digest, err := decodePBKDF2(c.Security.AdminHash)
	if err != nil {
		return false
	}
	computed := pbkdf2.Key([]byte(plaintext), salt, rounds, sha512.Size, sha512.New)
	return fixedTimeEqual(digest, computed)
}

func encodePBKDF2(rounds int, salt, digest []byte) string {
	return fmt.Sprintf("$pbkdf2-sha512$%d$%s$%s",
		rounds,
		base64.StdEncoding.EncodeToString(salt),
		base64.StdEncoding.EncodeToString(digest))
}

func decodePBKDF2(encoded string) (rounds int, salt, digest []byte, err error) {
	parts := strings.Split(encoded, "$")
	// $pbkdf2-sha512$600000$salt$hash
	if len(parts) != 5 || parts[0] != "" || parts[1] != "pbkdf2-sha512" {
		return 0, nil, nil, fmt.Errorf("invalid pbkdf2 format")
	}
	rounds, err = strconv.Atoi(parts[2])
	if err != nil || rounds <= 0 {
		return 0, nil, nil, fmt.Errorf("invalid pbkdf2 rounds")
	}
	salt, err = base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return 0, nil, nil, fmt.Errorf("invalid pbkdf2 salt")
	}
	digest, err = base64.StdEncoding.DecodeString(parts[4])
	if err != nil {
		return 0, nil, nil, fmt.Errorf("invalid pbkdf2 digest")
	}
	return rounds, salt, digest, nil
}

func fixedTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}