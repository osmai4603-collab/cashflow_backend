package user

import (
	"net/mail"
	"strings"
	"time"

	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 10

// User represents a system user (res.users in Odoo).
type User struct {
	ID                        int64        `json:"id"`
	Login                     string       `json:"login"`
	Email                     string       `json:"email,omitempty"`
	EmailNotificationsEnabled bool         `json:"email_notifications_enabled"`
	Name                      string       `json:"name"`
	PasswordHash              string       `json:"-"`
	PartnerID                 int64        `json:"partner_id"`
	CompanyID                 int64        `json:"company_id"`
	Active                    bool         `json:"active"`
	IsSuperuser               bool         `json:"is_superuser"`
	LastLoginAt               *time.Time   `json:"last_login_at,omitempty"`
	Audit                     audit.Fields `json:"audit"`
}

// Validate ensures the user entity satisfies all domain invariants.
func (u *User) Validate() error {
	trimmedLogin := strings.TrimSpace(u.Login)
	if trimmedLogin == "" {
		return platformerrors.Validation("login is required", map[string]string{
			"login": "cannot be empty",
		})
	}
	if len(trimmedLogin) > 255 {
		return platformerrors.Validation("login exceeds maximum length", map[string]string{
			"login": "must not exceed 255 characters",
		})
	}
	u.Login = strings.ToLower(trimmedLogin)

	trimmedName := strings.TrimSpace(u.Name)
	if trimmedName == "" {
		return platformerrors.Validation("user name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	u.Name = trimmedName

	if u.Email != "" {
		trimmedEmail := strings.TrimSpace(u.Email)
		if _, err := mail.ParseAddress(trimmedEmail); err != nil {
			return platformerrors.Validation("invalid email address format", map[string]string{
				"email": "invalid email address",
			})
		}
		u.Email = strings.ToLower(trimmedEmail)
	}

	if u.PartnerID <= 0 {
		return platformerrors.Validation("partner is required", map[string]string{
			"partner_id": "must be a valid partner",
		})
	}

	if u.CompanyID <= 0 {
		return platformerrors.Validation("company is required", map[string]string{
			"company_id": "must be a valid company",
		})
	}

	return nil
}

// SetPassword hashes the plaintext password and stores it in PasswordHash.
func (u *User) SetPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return platformerrors.Internal("failed to hash password", err)
	}
	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword compares a plaintext password against the stored hash.
func (u *User) CheckPassword(plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plain))
	return err == nil
}
