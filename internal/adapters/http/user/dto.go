package userhttp

import (
	"time"

	"cashflow_backend/internal/domain/user"
	userusecase "cashflow_backend/internal/usecase/user"
)

// CreateUserRequest represents the incoming JSON payload to register a new user.
type CreateUserRequest struct {
	Login                     string `json:"login"`
	Email                     string `json:"email"`
	EmailNotificationsEnabled *bool  `json:"email_notifications_enabled"`
	Name                      string `json:"name"`
	Password                  string `json:"password"`
	PartnerID                 *int64 `json:"partner_id"`
	PartnerName               string `json:"partner_name"`
	CompanyID                 int64  `json:"company_id"`
	IsSuperuser               bool   `json:"is_superuser"`
}

// ToInput maps the HTTP request DTO to the application usecase input.
func (r CreateUserRequest) ToInput() userusecase.CreateUserInput {
	return userusecase.CreateUserInput{
		Login:                     r.Login,
		Email:                     r.Email,
		EmailNotificationsEnabled: r.EmailNotificationsEnabled,
		Name:                      r.Name,
		Password:                  r.Password,
		PartnerID:                 r.PartnerID,
		PartnerName:               r.PartnerName,
		CompanyID:                 r.CompanyID,
		IsSuperuser:               r.IsSuperuser,
	}
}

// UpdateUserRequest represents partial or full fields to update an existing user.
type UpdateUserRequest struct {
	Login                     *string `json:"login"`
	Email                     *string `json:"email"`
	EmailNotificationsEnabled *bool   `json:"email_notifications_enabled"`
	Name                      *string `json:"name"`
	Password                  *string `json:"password"`
	CompanyID                 *int64  `json:"company_id"`
	IsSuperuser               *bool   `json:"is_superuser"`
}

// ToInput maps the update request DTO to the application usecase input.
func (r UpdateUserRequest) ToInput() userusecase.UpdateUserInput {
	return userusecase.UpdateUserInput{
		Login:                     r.Login,
		Email:                     r.Email,
		EmailNotificationsEnabled: r.EmailNotificationsEnabled,
		Name:                      r.Name,
		Password:                  r.Password,
		CompanyID:                 r.CompanyID,
		IsSuperuser:               r.IsSuperuser,
	}
}

// LoginRequest represents the login credentials payload.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginResponse formats the authenticated session output.
type LoginResponse struct {
	Token string        `json:"token"`
	User  *UserResponse `json:"user"`
}

// UserResponse formats user data for HTTP JSON client output.
type UserResponse struct {
	ID                        int64   `json:"id"`
	Login                     string  `json:"login"`
	Email                     string  `json:"email,omitempty"`
	EmailNotificationsEnabled bool    `json:"email_notifications_enabled"`
	Name                      string  `json:"name"`
	PartnerID                 int64   `json:"partner_id"`
	CompanyID                 int64   `json:"company_id"`
	Active                    bool    `json:"active"`
	IsSuperuser               bool    `json:"is_superuser"`
	LastLoginAt               *string `json:"last_login_at,omitempty"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
}

// ToUserResponse converts a domain User entity into UserResponse DTO.
func ToUserResponse(u *user.User) UserResponse {
	if u == nil {
		return UserResponse{}
	}
	resp := UserResponse{
		ID:                        u.ID,
		Login:                     u.Login,
		Email:                     u.Email,
		EmailNotificationsEnabled: u.EmailNotificationsEnabled,
		Name:                      u.Name,
		PartnerID:                 u.PartnerID,
		CompanyID:                 u.CompanyID,
		Active:                    u.Active,
		IsSuperuser:               u.IsSuperuser,
		CreatedAt:                 u.Audit.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                 u.Audit.UpdatedAt.Format(time.RFC3339),
	}
	if u.LastLoginAt != nil {
		last := u.LastLoginAt.Format(time.RFC3339)
		resp.LastLoginAt = &last
	}
	return resp
}

// ToUserResponseList converts a slice of domain users into response DTOs.
func ToUserResponseList(items []user.User) []UserResponse {
	result := make([]UserResponse, len(items))
	for i, item := range items {
		result[i] = ToUserResponse(&item)
	}
	return result
}
