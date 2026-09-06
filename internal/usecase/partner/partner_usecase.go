package partnerusecase

import (
	"context"
	"fmt"
	"log/slog"

	"cashflow_backend/internal/domain/partner"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// CreatePartnerInput defines input parameters for creating a new partner.
type CreatePartnerInput struct {
	Name       string              `json:"name"`
	Email      string              `json:"email"`
	Phone      string              `json:"phone"`
	Mobile     string              `json:"mobile"`
	Type       partner.PartnerType `json:"type"`
	IsCustomer *bool               `json:"is_customer"`
	IsSupplier *bool               `json:"is_supplier"`
	VATNumber  string              `json:"vat_number"`
	Website    string              `json:"website"`
	CompanyID  *int64              `json:"company_id"`
	ParentID   *int64              `json:"parent_id"`
	Street     string              `json:"street"`
	Street2    string              `json:"street2"`
	City       string              `json:"city"`
	State      string              `json:"state"`
	Country    string              `json:"country"`
	ZipCode    string              `json:"zip_code"`
}

// UpdatePartnerInput defines input parameters for modifying an existing partner.
type UpdatePartnerInput struct {
	Name       *string              `json:"name"`
	Email      *string              `json:"email"`
	Phone      *string              `json:"phone"`
	Mobile     *string              `json:"mobile"`
	Type       *partner.PartnerType `json:"type"`
	IsCustomer *bool                `json:"is_customer"`
	IsSupplier *bool                `json:"is_supplier"`
	VATNumber  *string              `json:"vat_number"`
	Website    *string              `json:"website"`
	CompanyID  *int64               `json:"company_id"`
	ParentID   *int64               `json:"parent_id"`
	Street     *string              `json:"street"`
	Street2    *string              `json:"street2"`
	City       *string              `json:"city"`
	State      *string              `json:"state"`
	Country    *string              `json:"country"`
	ZipCode    *string              `json:"zip_code"`
}

// UseCase defines the application interface for Partner domain operations.
type UseCase interface {
	CreatePartner(ctx context.Context, in CreatePartnerInput) (*partner.Partner, error)
	GetPartner(ctx context.Context, id int64) (*partner.Partner, error)
	UpdatePartner(ctx context.Context, id int64, in UpdatePartnerInput) (*partner.Partner, error)
	DeletePartner(ctx context.Context, id int64) error
	ListPartners(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error)
	ListCustomers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error)
	ListSuppliers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error)
}

// PartnerUseCase implements the UseCase interface.
type PartnerUseCase struct {
	repo   partner.Repository
	logger *slog.Logger
}

// New constructs a new PartnerUseCase.
func New(repo partner.Repository, logger *slog.Logger) *PartnerUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &PartnerUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *PartnerUseCase) CreatePartner(ctx context.Context, in CreatePartnerInput) (*partner.Partner, error) {
	pType := in.Type
	if pType == "" {
		pType = partner.PartnerTypeIndividual
	}

	isCustomer := true
	if in.IsCustomer != nil {
		isCustomer = *in.IsCustomer
	}

	isSupplier := false
	if in.IsSupplier != nil {
		isSupplier = *in.IsSupplier
	}

	// If parent ID specified, ensure parent exists
	if in.ParentID != nil && *in.ParentID > 0 {
		if _, err := uc.repo.GetByID(ctx, *in.ParentID); err != nil {
			return nil, platformerrors.Validation("parent partner does not exist", map[string]string{
				"parent_id": fmt.Sprintf("partner %d not found", *in.ParentID),
			})
		}
	}

	p := &partner.Partner{
		Name:       in.Name,
		Email:      in.Email,
		Phone:      in.Phone,
		Mobile:     in.Mobile,
		Type:       pType,
		IsCustomer: isCustomer,
		IsSupplier: isSupplier,
		VATNumber:  in.VATNumber,
		Website:    in.Website,
		CompanyID:  in.CompanyID,
		ParentID:   in.ParentID,
		Street:     in.Street,
		Street2:    in.Street2,
		City:       in.City,
		State:      in.State,
		Country:    in.Country,
		ZipCode:    in.ZipCode,
		Active:     true,
		Audit:      audit.NewFields(ctx),
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, p); err != nil {
		uc.logger.Error("failed to create partner", "error", err, "name", p.Name)
		return nil, err
	}

	uc.logger.Info("partner created successfully", "id", p.ID, "name", p.Name, "type", p.Type)
	return p, nil
}

func (uc *PartnerUseCase) GetPartner(ctx context.Context, id int64) (*partner.Partner, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid partner id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *PartnerUseCase) UpdatePartner(ctx context.Context, id int64, in UpdatePartnerInput) (*partner.Partner, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid partner id")
	}

	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		p.Name = *in.Name
	}
	if in.Email != nil {
		p.Email = *in.Email
	}
	if in.Phone != nil {
		p.Phone = *in.Phone
	}
	if in.Mobile != nil {
		p.Mobile = *in.Mobile
	}
	if in.Type != nil {
		p.Type = *in.Type
	}
	if in.IsCustomer != nil {
		p.IsCustomer = *in.IsCustomer
	}
	if in.IsSupplier != nil {
		p.IsSupplier = *in.IsSupplier
	}
	if in.VATNumber != nil {
		p.VATNumber = *in.VATNumber
	}
	if in.Website != nil {
		p.Website = *in.Website
	}
	if in.CompanyID != nil {
		p.CompanyID = in.CompanyID
	}
	if in.ParentID != nil {
		if *in.ParentID == id {
			return nil, platformerrors.Validation("circular reference", map[string]string{
				"parent_id": "partner cannot be its own parent",
			})
		}
		if *in.ParentID > 0 {
			if _, err := uc.repo.GetByID(ctx, *in.ParentID); err != nil {
				return nil, platformerrors.Validation("parent partner does not exist", map[string]string{
					"parent_id": fmt.Sprintf("partner %d not found", *in.ParentID),
				})
			}
		}
		p.ParentID = in.ParentID
	}
	if in.Street != nil {
		p.Street = *in.Street
	}
	if in.Street2 != nil {
		p.Street2 = *in.Street2
	}
	if in.City != nil {
		p.City = *in.City
	}
	if in.State != nil {
		p.State = *in.State
	}
	if in.Country != nil {
		p.Country = *in.Country
	}
	if in.ZipCode != nil {
		p.ZipCode = *in.ZipCode
	}

	p.Audit.Touch(ctx)

	if err := p.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, p); err != nil {
		uc.logger.Error("failed to update partner", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("partner updated successfully", "id", p.ID)
	return p, nil
}

func (uc *PartnerUseCase) DeletePartner(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid partner id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("partner soft-deleted successfully", "id", id)
	return nil
}

func (uc *PartnerUseCase) ListPartners(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	return uc.repo.List(ctx, f, page)
}

func (uc *PartnerUseCase) ListCustomers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	return uc.repo.ListCustomers(ctx, page)
}

func (uc *PartnerUseCase) ListSuppliers(ctx context.Context, page pagination.PageRequest) (pagination.PageResult[partner.Partner], error) {
	return uc.repo.ListSuppliers(ctx, page)
}
