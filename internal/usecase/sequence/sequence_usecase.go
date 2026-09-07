package sequenceusecase

import (
	"context"
	"log/slog"
	"time"

	"cashflow_backend/internal/domain/sequence"
	platformerrors "cashflow_backend/internal/platform/errors"
	platformsequence "cashflow_backend/internal/platform/sequence"
)

// CreateSequenceInput defines input parameters for creating a new sequence.
type CreateSequenceInput struct {
	Name         string                        `json:"name"`
	Code         string                        `json:"code"`
	Prefix       string                        `json:"prefix"`
	Suffix       string                        `json:"suffix"`
	Padding      int16                         `json:"padding"`
	IncrementBy  int                           `json:"increment_by"`
	StartNumber  int                           `json:"start_number"`
	SequenceType sequence.SequenceType         `json:"sequence_type"`
	DateRange    sequence.DateRangeGranularity `json:"date_range"`
	CompanyID    *int64                        `json:"company_id"`
}

// UpdateSequenceInput defines input parameters for modifying an existing sequence.
type UpdateSequenceInput struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Prefix      *string `json:"prefix"`
	Suffix      *string `json:"suffix"`
	Padding     *int16  `json:"padding"`
	IncrementBy *int    `json:"increment_by"`
	StartNumber *int    `json:"start_number"`
	Active      *bool   `json:"active"`
}

// UseCase defines the application interface for Sequence domain operations.
type UseCase interface {
	CreateSequence(ctx context.Context, in CreateSequenceInput) (*sequence.Sequence, error)
	GetSequence(ctx context.Context, id int64) (*sequence.Sequence, error)
	UpdateSequence(ctx context.Context, id int64, in UpdateSequenceInput) (*sequence.Sequence, error)
	DeleteSequence(ctx context.Context, id int64) error
	ListSequences(ctx context.Context) ([]sequence.Sequence, error)
	GenerateNext(ctx context.Context, code string, date time.Time) (string, error)
	GenerateNextByID(ctx context.Context, id int64, date time.Time) (string, error)
}

// SequenceUseCase implements the UseCase interface.
type SequenceUseCase struct {
	repo      sequence.Repository
	generator *platformsequence.Generator
	logger    *slog.Logger
}

// New constructs a new SequenceUseCase.
func New(repo sequence.Repository, logger *slog.Logger) *SequenceUseCase {
	if logger == nil {
		logger = slog.Default()
	}
	return &SequenceUseCase{
		repo:      repo,
		generator: platformsequence.NewGenerator(repo),
		logger:    logger,
	}
}

func (uc *SequenceUseCase) CreateSequence(ctx context.Context, in CreateSequenceInput) (*sequence.Sequence, error) {
	seqType := in.SequenceType
	if seqType == "" {
		seqType = sequence.SequenceTypeNormal
	}

	padding := in.Padding
	if padding == 0 {
		padding = 5
	}

	incrementBy := in.IncrementBy
	if incrementBy == 0 {
		incrementBy = 1
	}

	startNumber := in.StartNumber
	if startNumber == 0 {
		startNumber = 1
	}

	s := &sequence.Sequence{
		Name:          in.Name,
		Code:          in.Code,
		Prefix:        in.Prefix,
		Suffix:        in.Suffix,
		Padding:       padding,
		IncrementBy:   incrementBy,
		StartNumber:   startNumber,
		CurrentNumber: 0,
		SequenceType:  seqType,
		DateRange:     in.DateRange,
		CompanyID:     in.CompanyID,
		Active:        true,
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Create(ctx, s); err != nil {
		uc.logger.Error("failed to create sequence", "error", err, "code", s.Code)
		return nil, err
	}

	uc.logger.Info("sequence created successfully", "id", s.ID, "code", s.Code)
	return s, nil
}

func (uc *SequenceUseCase) GetSequence(ctx context.Context, id int64) (*sequence.Sequence, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid sequence id")
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *SequenceUseCase) UpdateSequence(ctx context.Context, id int64, in UpdateSequenceInput) (*sequence.Sequence, error) {
	if id <= 0 {
		return nil, platformerrors.BadRequest("invalid sequence id")
	}

	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Name != nil {
		s.Name = *in.Name
	}
	if in.Code != nil {
		s.Code = *in.Code
	}
	if in.Prefix != nil {
		s.Prefix = *in.Prefix
	}
	if in.Suffix != nil {
		s.Suffix = *in.Suffix
	}
	if in.Padding != nil {
		s.Padding = *in.Padding
	}
	if in.IncrementBy != nil {
		s.IncrementBy = *in.IncrementBy
	}
	if in.StartNumber != nil {
		s.StartNumber = *in.StartNumber
	}
	if in.Active != nil {
		s.Active = *in.Active
	}

	if err := s.Validate(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(ctx, s); err != nil {
		uc.logger.Error("failed to update sequence", "error", err, "id", id)
		return nil, err
	}

	uc.logger.Info("sequence updated successfully", "id", s.ID)
	return s, nil
}

func (uc *SequenceUseCase) DeleteSequence(ctx context.Context, id int64) error {
	if id <= 0 {
		return platformerrors.BadRequest("invalid sequence id")
	}

	if err := uc.repo.Delete(ctx, id); err != nil {
		return err
	}

	uc.logger.Info("sequence soft-deleted successfully", "id", id)
	return nil
}

func (uc *SequenceUseCase) ListSequences(ctx context.Context) ([]sequence.Sequence, error) {
	return uc.repo.List(ctx)
}

func (uc *SequenceUseCase) GenerateNext(ctx context.Context, code string, date time.Time) (string, error) {
	s, err := uc.repo.GetByCode(ctx, code)
	if err != nil {
		return "", err
	}

	return uc.generator.Next(ctx, s, date)
}

func (uc *SequenceUseCase) GenerateNextByID(ctx context.Context, id int64, date time.Time) (string, error) {
	s, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return uc.generator.Next(ctx, s, date)
}
