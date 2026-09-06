package auth

import "github.com/golang-jwt/jwt/v5"

// UserClaims defines standard ERP user JWT payload with multi-tenancy and RBAC roles.
type UserClaims struct {
	UserID    int64    `json:"uid"`
	CompanyID int64    `json:"cid"`
	Roles     []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// HasRole checks if the claims contain a specified role.
func (c *UserClaims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r == role {
			return true
		}
	}
	return false
}
