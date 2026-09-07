package companyusecase

import (
	"context"
	"log/slog"

	"cashflow_backend/internal/domain/company"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// CreateCompanyInput defines input parameters for creating a new company.
type CreateCompanyInput struct {
	Name       string `json:"name"`
	PartnerID  *int64 `json:"partner_id"`
	CurrencyID int64  `json:"currency_id"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Website    string `json:"website"`
	VAT        string `json:"vat"`
	Street     string `json:"street"`
	Street2    string `json:"street2"`
	City       string `json:"city"`
	State      string `json:"state"`
	Country    string `json:"country"`
	ZipCode    string `json:"zip_code"`
}

// UpdateCompanyInput defines input parameters for modifying an existing company.
type UpdateCompanyInput struct {
	Name       *string `json:"name"`
	PartnerID  *int64  `json:"partner_id"`
	CurrencyID *int64  `json:"currency_id"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Website    *string `json:"website"`
	VAT        *string `json:"vat"`
	Street     *string `json:"street"`
	Street2    *string `json:"street2"`
	City       *string `json:"city"`
	State      *string `json:"state"`
	Country    *string `json:"country"`
	ZipCode    *string `json:"zip_code"`
}

// UseCase defines the application interface for Company domain operations.
type UseCase interface {
	CreateCompany(ctx context.Context, in CreateCompanyInput) (*company.Company, error)
	GetCompany(ctx context.Context, id int64) (*company.Company, error)
	UpdateCompany(ctx context.Context, id int64, in UpdateCompanyInput) (*company.Company, error)
	DeleteCompany(ctx context.Context, id int64) error
	ListCompanies(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[company.Company], error)
	GetDefaultCompany(ctx context.Context) (*company.Company, error)
}

// CompanyUseCase implements the UseCase interface.
type CompanyUseCase struct {
	repo   company.Repository
	logger *slog.Logger
}

// New constructs a new CompanyUseCase.
func New(repo company.Repository, logger *slog.Logger) *CompanyUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CompanyUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *CompanyUseCase) CreateCompany(ctx context.Context, in CreateCompanyInput) (*company.Company, error) {
	currencyID := in.CurrencyID
	if currencyID == 0 {
		return nil, platformerrors.Validation("currency is required for company", map[string]string{
			"currency_id": "must be a valid currency",
		})
	}

	c := &company.Company{
		Name:       in.Name,
		PartnerID:  in.PartnerID,
		CurrencyID: currencyID,
		Phone:      in.Phone,
		Email:      in.Email,
		Website:    in.Website,
		VAT:        in.VAT,
		Street:     in.Street,
		Street2:    in.Street2,
		City:       in.City,
		State:      in.State,
		Country:    in.Country,
		ZipCode:    in.ZipCode,
		Active:     true,
		Audit:      audit.NewFields(ctx),
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, c); err != nil {
		uc.logger.Error("failed to create company", "error", err, "name", c.Name)
		return nil, err
	}

	uc.logger.Info("company created successfully", "id", c.ID, "name", c.Name)
	return c, nil
}

func (uc *CompanyUseCase) GetCompany(ctx context.Context, id int64) (*company.Company, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid company id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *CompanyUseCase) UpdateCompany(ctx context.Context, id int64, in UpdateCompanyInput) (*company.Company, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid company id")
	}

	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.PartnerID != nil {
		c.PartnerID = in.PartnerID
	}
	if in.CurrencyID != nil {
		c.CurrencyID = *in.CurrencyID
	}
	if in.Phone != nil {
		c.Phone = *in.Phone
	}
	if in.Email != nil {
		c.Email = *in.Email
	}
	if in.Website != nil {
		c.Website = *in.Website
	}
	if in.VAT != nil {
		c.VAT = *in.VAT
	}
	if in.Street != nil {
		c.Street = *in.Street
	}
	if in.Street2 != nil {
		c.Street2 = *in.Street2
	}
	if in.City != nil {
		c.City = *in.City
	}
	if in.State != nil {
		c.State = *in.State
	}
	if in.Country != nil {
		c.Country = *in.Country
	}
	if in.ZipCode != nil {
		c.ZipCode = *in.ZipCode
	}

	c.Audit.Touch(ctx)

	if err := c.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, c); err != nil {
		uc.logger.Error("failed to update company", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("company updated successfully", "id", c.ID)
	return c, nil
}

func (uc *CompanyUseCase) DeleteCompany(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid company id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("company soft-deleted successfully", "id", id)
	return nil
}

func (uc *CompanyUseCase) ListCompanies(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[company.Company], error) {
	return uc.repo.List(ctx, f, page)
}

func (uc *CompanyUseCase) GetDefaultCompany(ctx context.Context) (*company.Company, error) {
	return uc.repo.GetDefaultCompany(ctx)
}
