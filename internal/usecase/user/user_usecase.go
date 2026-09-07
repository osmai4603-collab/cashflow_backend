package userusecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/domain/user"
	"cashflow_backend/internal/platform/audit"
	"cashflow_backend/internal/platform/auth"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// CreateUserInput defines input parameters for creating a new user.
type CreateUserInput struct {
	Login       string `json:"login"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Password    string `json:"password"`
	PartnerID   *int64 `json:"partner_id"`
	PartnerName string `json:"partner_name"`
	CompanyID   int64  `json:"company_id"`
	IsSuperuser bool   `json:"is_superuser"`
}

// UpdateUserInput defines input parameters for modifying an existing user.
type UpdateUserInput struct {
	Login       *string `json:"login"`
	Email       *string `json:"email"`
	Name        *string `json:"name"`
	Password    *string `json:"password"`
	CompanyID   *int64  `json:"company_id"`
	IsSuperuser *bool   `json:"is_superuser"`
}

// LoginResult is returned after successful authentication.
type LoginResult struct {
	Token string     `json:"token"`
	User  *user.User `json:"user"`
}

// UseCase defines the application interface for User domain operations.
type UseCase interface {
	CreateUser(ctx context.Context, in CreateUserInput) (*user.User, error)
	GetUser(ctx context.Context, id int64) (*user.User, error)
	UpdateUser(ctx context.Context, id int64, in UpdateUserInput) (*user.User, error)
	DeleteUser(ctx context.Context, id int64) error
	ListUsers(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[user.User], error)
	Login(ctx context.Context, login, password string) (*LoginResult, error)
}

// UserUseCase implements the UseCase interface.
type UserUseCase struct {
	repo      user.Repository
	partner   partner.Repository
	logger    *slog.Logger
	jwtSecret string
	tokenTTL  time.Duration
}

// New constructs a new UserUseCase.
func New(repo user.Repository, partnerRepo partner.Repository, logger *slog.Logger, jwtSecret string, tokenTTL time.Duration) *UserUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	if jwtSecret == "" {
		jwtSecret = "odoo-go-insecure-dev-secret-key-change-in-production"
	}
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &UserUseCase{
		repo:      repo,
		partner:   partnerRepo,
		logger:    logger,
		jwtSecret: jwtSecret,
		tokenTTL:  tokenTTL,
	}
}

func (uc *UserUseCase) resolvePartner(ctx context.Context, in CreateUserInput) (int64, error) {
	if in.PartnerID != nil && *in.PartnerID > 0 {
		if _, err := uc.partner.GetByID(ctx, *in.PartnerID); err != nil {
			return 0, platformerrors.Validation("partner does not exist", map[string]string{
				"partner_id": fmt.Sprintf("partner %d not found", *in.PartnerID),
			})
		}
		return *in.PartnerID, nil
	}

	if in.PartnerName != "" {
		p := &partner.Partner{
			Name:   in.PartnerName,
			Email:  in.Email,
			Type:   partner.PartnerTypeIndividual,
			Active: true,
			Audit:  audit.NewFields(ctx),
		}
		if err := p.Validate(); err != nil {
			return 0, err
		}
		if err := uc.partner.Create(ctx, p); err != nil {
			return 0, err
		}
		return p.ID, nil
	}

	return 0, platformerrors.Validation("partner is required for user", map[string]string{
		"partner_id":   "provide partner_id or partner_name",
		"partner_name": "provide partner_id or partner_name",
	})
}

func (uc *UserUseCase) CreateUser(ctx context.Context, in CreateUserInput) (*user.User, error) {
	if in.CompanyID <= 0 {
		return nil, platformerrors.Validation("company is required for user", map[string]string{
			"company_id": "must be a valid company",
		})
	}

	if in.Password == "" {
		return nil, platformerrors.Validation("password is required", map[string]string{
			"password": "cannot be empty",
		})
	}

	partnerID, err := uc.resolvePartner(ctx, in)
	if err != nil {
		return nil, err
	}

	u := &user.User{
		Login:       in.Login,
		Email:       in.Email,
		Name:        in.Name,
		PartnerID:   partnerID,
		CompanyID:   in.CompanyID,
		IsSuperuser: in.IsSuperuser,
		Active:      true,
		Audit:       audit.NewFields(ctx),
	}

	if err := u.SetPassword(in.Password); err != nil {
		return nil, err
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, u); err != nil {
		uc.logger.Error("failed to create user", "error", err, "login", u.Login)
		return nil, err
	}

	uc.logger.Info("user created successfully", "id", u.ID, "login", u.Login)
	return u, nil
}

func (uc *UserUseCase) GetUser(ctx context.Context, id int64) (*user.User, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid user id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, id int64, in UpdateUserInput) (*user.User, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid user id")
	}

	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Login != nil {
		u.Login = *in.Login
	}
	if in.Email != nil {
		u.Email = *in.Email
	}
	if in.Name != nil {
		u.Name = *in.Name
	}
	if in.Password != nil && *in.Password != "" {
		if err := u.SetPassword(*in.Password); err != nil {
			return nil, err
		}
	}
	if in.CompanyID != nil {
		u.CompanyID = *in.CompanyID
	}
	if in.IsSuperuser != nil {
		u.IsSuperuser = *in.IsSuperuser
	}

	u.Audit.Touch(ctx)

	if err := u.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, u); err != nil {
		uc.logger.Error("failed to update user", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("user updated successfully", "id", u.ID)
	return u, nil
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid user id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("user soft-deleted successfully", "id", id)
	return nil
}

func (uc *UserUseCase) ListUsers(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[user.User], error) {
	return uc.repo.List(ctx, f, page)
}

func (uc *UserUseCase) Login(ctx context.Context, login, password string) (*LoginResult, error) {
	u, err := uc.repo.GetByLogin(ctx, login)
	if err != nil {
		return nil, platformerrors.Unauthorized("invalid credentials")
	}

	if !u.CheckPassword(password) {
		return nil, platformerrors.Unauthorized("invalid credentials")
	}

	if err := uc.repo.UpdateLastLogin(ctx, u.ID); err != nil {
		uc.logger.Warn("failed to update last login time", "error", err, "user_id", u.ID)
	}

	roles := []string{"user"}
	if u.IsSuperuser {
		roles = append(roles, "admin")
	}

	token, err := auth.GenerateToken(u.ID, u.CompanyID, roles, uc.jwtSecret, uc.tokenTTL)
	if err != nil {
		uc.logger.Error("failed to generate auth token", "error", err)
		return nil, platformerrors.Internal("failed to generate authentication token", err)
	}

	uc.logger.Info("user authenticated successfully", "id", u.ID, "login", u.Login)
	return &LoginResult{Token: token, User: u}, nil
}
