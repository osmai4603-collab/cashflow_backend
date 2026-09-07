package currencyusecase

import (
	"context"
	"fmt"
	"log/slog"

	"cashflow_backend/internal/domain/currency"
	"cashflow_backend/internal/platform/audit"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// CreateCurrencyInput defines input parameters for creating a new currency.
type CreateCurrencyInput struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	Symbol        string `json:"symbol"`
	DecimalPlaces int16  `json:"decimal_places"`
}

// UpdateCurrencyInput defines input parameters for modifying an existing currency.
type UpdateCurrencyInput struct {
	Name          *string `json:"name"`
	FullName      *string `json:"full_name"`
	Symbol        *string `json:"symbol"`
	DecimalPlaces *int16  `json:"decimal_places"`
}

// CreateRateInput defines input parameters for registering an exchange rate.
type CreateRateInput struct {
	CurrencyID int64   `json:"currency_id"`
	Rate       float64 `json:"rate"`
	Date       string  `json:"date"`
	CompanyID  *int64  `json:"company_id"`
}

// ConvertInput defines input parameters for a currency conversion request.
type ConvertInput struct {
	Amount       float64 `json:"amount"`
	FromCurrency int64   `json:"from_currency_id"`
	ToCurrency   int64   `json:"to_currency_id"`
	Date         string  `json:"date"`
}

// ConvertResult is returned after a successful conversion.
type ConvertResult struct {
	Amount       float64 `json:"amount"`
	Converted    float64 `json:"converted"`
	FromCurrency int64   `json:"from_currency_id"`
	ToCurrency   int64   `json:"to_currency_id"`
	Rate         float64 `json:"rate"`
	Date         string  `json:"date"`
}

// UseCase defines the application interface for Currency domain operations.
type UseCase interface {
	CreateCurrency(ctx context.Context, in CreateCurrencyInput) (*currency.Currency, error)
	GetCurrency(ctx context.Context, id int64) (*currency.Currency, error)
	UpdateCurrency(ctx context.Context, id int64, in UpdateCurrencyInput) (*currency.Currency, error)
	DeleteCurrency(ctx context.Context, id int64) error
	ListCurrencies(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[currency.Currency], error)
	CreateRate(ctx context.Context, in CreateRateInput) (*currency.CurrencyRate, error)
	ListRates(ctx context.Context, currencyID int64, page pagination.PageRequest) (pagination.PageResult[currency.CurrencyRate], error)
	Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error)
}

// CurrencyUseCase implements the UseCase interface.
type CurrencyUseCase struct {
	repo      currency.Repository
	rateRepo  currency.RateRepository
	converter ConverterService
	logger    *slog.Logger
}

// ConverterService performs currency conversion given historical rates.
type ConverterService interface {
	Convert(ctx context.Context, amount float64, fromCurrencyID, toCurrencyID int64, date string, companyID *int64) (float64, error)
}

// New constructs a new CurrencyUseCase.
func New(repo currency.Repository, rateRepo currency.RateRepository, converter ConverterService, logger *slog.Logger) *CurrencyUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CurrencyUseCase{
		repo:      repo,
		rateRepo:  rateRepo,
		converter: converter,
		logger:    logger,
	}
}

func (uc *CurrencyUseCase) CreateCurrency(ctx context.Context, in CreateCurrencyInput) (*currency.Currency, error) {
	c := &currency.Currency{
		Name:          in.Name,
		FullName:      in.FullName,
		Symbol:        in.Symbol,
		DecimalPlaces: in.DecimalPlaces,
		Active:        true,
	}

	if c.DecimalPlaces == 0 && c.Name != "" {
		c.DecimalPlaces = 2
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, c); err != nil {
		uc.logger.Error("failed to create currency", "error", err, "name", c.Name)
		return nil, err
	}

	uc.logger.Info("currency created successfully", "id", c.ID, "name", c.Name)
	return c, nil
}

func (uc *CurrencyUseCase) GetCurrency(ctx context.Context, id int64) (*currency.Currency, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid currency id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *CurrencyUseCase) UpdateCurrency(ctx context.Context, id int64, in UpdateCurrencyInput) (*currency.Currency, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid currency id")
	}

	c, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		c.Name = *in.Name
	}
	if in.FullName != nil {
		c.FullName = *in.FullName
	}
	if in.Symbol != nil {
		c.Symbol = *in.Symbol
	}
	if in.DecimalPlaces != nil {
		c.DecimalPlaces = *in.DecimalPlaces
	}

	if err := c.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, c); err != nil {
		uc.logger.Error("failed to update currency", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("currency updated successfully", "id", c.ID)
	return c, nil
}

func (uc *CurrencyUseCase) DeleteCurrency(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid currency id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("currency soft-deleted successfully", "id", id)
	return nil
}

func (uc *CurrencyUseCase) ListCurrencies(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[currency.Currency], error) {
	return uc.repo.List(ctx, f, page)
}

func (uc *CurrencyUseCase) CreateRate(ctx context.Context, in CreateRateInput) (*currency.CurrencyRate, error) {
	if in.CurrencyID <= 0 {
		return nil, platformerrors.Validation("currency is required", map[string]string{
			"currency_id": "must be a valid currency",
		})
	}

	if _, err := uc.repo.GetByID(ctx, in.CurrencyID); err != nil {
		return nil, platformerrors.Validation("currency does not exist", map[string]string{
			"currency_id": fmt.Sprintf("currency %d not found", in.CurrencyID),
		})
	}

	rate := &currency.CurrencyRate{
		CurrencyID: in.CurrencyID,
		Rate:       in.Rate,
		Date:       in.Date,
		CompanyID:  in.CompanyID,
		Audit:      audit.NewFields(ctx),
	}

	if err := rate.Validate(); err != nil {
		return nil, err
	}

	if err := uc.rateRepo.Create(ctx, rate); err != nil {
		uc.logger.Error("failed to create currency rate", "error", err)
		return nil, err
	}

	uc.logger.Info("currency rate created successfully", "id", rate.ID, "currency_id", rate.CurrencyID)
	return rate, nil
}

func (uc *CurrencyUseCase) ListRates(ctx context.Context, currencyID int64, page pagination.PageRequest) (pagination.PageResult[currency.CurrencyRate], error) {
	return uc.rateRepo.ListByCurrency(ctx, currencyID, page)
}

func (uc *CurrencyUseCase) Convert(ctx context.Context, in ConvertInput) (*ConvertResult, error) {
	if in.Amount < 0 {
		return nil, platformerrors.Validation("amount cannot be negative", map[string]string{
			"amount": "must be >= 0",
		})
	}
	if in.FromCurrency <= 0 || in.ToCurrency <= 0 {
		return nil, platformerrors.Validation("both currencies are required", nil)
	}

	companyID := audit.CompanyIDFromContext(ctx)

	converted, err := uc.converter.Convert(ctx, in.Amount, in.FromCurrency, in.ToCurrency, in.Date, companyID)
	if err != nil {
		return nil, err
	}

	uc.logger.Info("currency converted",
		"from", in.FromCurrency, "to", in.ToCurrency,
		"amount", in.Amount, "converted", converted, "date", in.Date,
	)

	return &ConvertResult{
		Amount:       in.Amount,
		Converted:    converted,
		FromCurrency: in.FromCurrency,
		ToCurrency:   in.ToCurrency,
		Date:         in.Date,
	}, nil
}
