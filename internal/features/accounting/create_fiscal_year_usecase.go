package accounting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type CreateFiscalYearUseCase struct {
	repo   AccountingRepository
	logger *slog.Logger
}

func NewCreateFiscalYearUseCase(repo AccountingRepository, logger *slog.Logger) *CreateFiscalYearUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &CreateFiscalYearUseCase{repo: repo, logger: logger}
}

func (uc *CreateFiscalYearUseCase) Execute(ctx context.Context, request CreateFiscalYearRequest) (*FiscalYearResponse, error) {
	logger := useCaseLogger(ctx, uc.logger, "create_fiscal_year")
	if err := request.Validate(); err != nil {
		return nil, err
	}
	startDate, err := parseDate(request.StartDate, "start_date")
	if err != nil {
		return nil, err
	}
	endDate, err := parseDate(request.EndDate, "end_date")
	if err != nil {
		return nil, err
	}
	if startDate.Year() != request.Year {
		return nil, errors.New("year must match start_date year")
	}
	if startDate.Day() != 1 || !endDate.Equal(startDate.AddDate(1, 0, -1)) {
		return nil, errors.New("fiscal year must span 12 complete calendar months")
	}

	now := time.Now().UTC()
	year := &FiscalYear{
		ID:        uuid.New(),
		Year:      request.Year,
		StartDate: startDate,
		EndDate:   endDate,
		State:     string(PeriodStateOpen),
		CreatedAt: now,
	}
	periods := make([]*FiscalPeriod, 12)
	for i := range periods {
		periodStart := startDate.AddDate(0, i, 0)
		periodEnd := periodStart.AddDate(0, 1, -1)
		periods[i] = &FiscalPeriod{
			ID:           uuid.New(),
			FiscalYearID: year.ID,
			PeriodNumber: i + 1,
			Name:         periodStart.Format("January 2006"),
			StartDate:    periodStart,
			EndDate:      periodEnd,
			State:        PeriodStateOpen,
			CreatedAt:    now,
		}
	}
	if err := uc.repo.CreateFiscalYear(ctx, year, periods); err != nil {
		logger.ErrorContext(ctx, "fiscal year creation failed", slog.Int("year", year.Year), slog.Any("error", err))
		return nil, fmt.Errorf("create fiscal year: %w", err)
	}

	logger.InfoContext(ctx, "fiscal year created", slog.String("fiscal_year_id", year.ID.String()))
	response := ToFiscalYearResponse(year, periods)
	return &response, nil
}
