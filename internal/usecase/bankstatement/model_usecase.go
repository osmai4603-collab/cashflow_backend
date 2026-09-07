package bankstatementusecase

import (
	"context"
	"errors"

	"cashflow_backend/internal/domain/bankstatement"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ─────────────────────────────────────────────────────────────────────────────
// Reconcile Models
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateReconcileModel(ctx context.Context, m *bankstatement.ReconcileModel) (*bankstatement.ReconcileModel, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if m.Sequence <= 0 {
		m.Sequence = 1
	}
	if err := uc.repo.CreateReconcileModel(ctx, m); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "reconcile model created", "id", m.ID, "name", m.Name)
	return uc.repo.GetReconcileModelByID(ctx, m.ID)
}

func (uc *UseCase) GetReconcileModel(ctx context.Context, id int64) (*bankstatement.ReconcileModel, error) {
	return uc.repo.GetReconcileModelByID(ctx, id)
}

func (uc *UseCase) UpdateReconcileModel(ctx context.Context, id int64, m *bankstatement.ReconcileModel) (*bankstatement.ReconcileModel, error) {
	existing, err := uc.repo.GetReconcileModelByID(ctx, id)
	if err != nil {
		return nil, err
	}
	m.ID = existing.ID
	m.CreatedAt = existing.CreatedAt
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateReconcileModel(ctx, m); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "reconcile model updated", "id", m.ID)
	return uc.repo.GetReconcileModelByID(ctx, m.ID)
}

func (uc *UseCase) DeleteReconcileModel(ctx context.Context, id int64) error {
	return uc.repo.DeleteReconcileModel(ctx, id)
}

func (uc *UseCase) ListReconcileModels(ctx context.Context) ([]bankstatement.ReconcileModel, error) {
	return uc.repo.ListReconcileModels(ctx)
}

// ─────────────────────────────────────────────────────────────────────────────
// Auto reconciliation
// ─────────────────────────────────────────────────────────────────────────────

// AutoReconcile applies the auto-reconcile models to every open line of a statement.
// It returns the number of successfully reconciled lines.
func (uc *UseCase) AutoReconcile(ctx context.Context, statementID int64) (int, error) {
	st, err := uc.repo.GetStatementWithLines(ctx, statementID)
	if err != nil {
		return 0, err
	}
	models, err := uc.repo.ListAutoReconcileModels(ctx)
	if err != nil {
		return 0, err
	}
	if len(models) == 0 {
		return 0, platformerrors.NotFound("no auto reconcile model configured")
	}

	reconciledCount := 0
	for i := range st.Lines {
		line := &st.Lines[i]
		if line.Reconciled {
			continue
		}
		model := firstMatchingModel(models, line)
		if model == nil {
			continue
		}
		candidates, err := uc.FindCandidates(ctx, line.ID, 10)
		if err != nil {
			var appErr *platformerrors.AppError
			if errors.As(err, &appErr) && appErr.Code == platformerrors.CodeConflict {
				continue
			}
			return reconciledCount, err
		}
		for _, candidate := range candidates {
			if !candidate.ExactMatch {
				continue
			}
			if model.MatchPartnerIDs == nil || len(model.MatchPartnerIDs) == 0 ||
				(line.PartnerID != nil && containsID(model.MatchPartnerIDs, *line.PartnerID)) {
				if _, err := uc.ReconcileLine(ctx, line.ID, candidate.MoveLine.ID); err == nil {
					reconciledCount++
					break
				}
			}
		}
	}
	uc.logger.InfoContext(ctx, "auto reconcile finished", "statement_id", statementID, "reconciled", reconciledCount)
	return reconciledCount, nil
}

func firstMatchingModel(models []bankstatement.ReconcileModel, line *bankstatement.BankStatementLine) *bankstatement.ReconcileModel {
	for i := range models {
		if models[i].Matches(line) {
			return &models[i]
		}
	}
	return nil
}

func containsID(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────────────────────────────────────
// Cash Rounding
// ─────────────────────────────────────────────────────────────────────────────

func (uc *UseCase) CreateCashRounding(ctx context.Context, cr *bankstatement.CashRounding) (*bankstatement.CashRounding, error) {
	if err := cr.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.CreateCashRounding(ctx, cr); err != nil {
		return nil, err
	}
	uc.logger.InfoContext(ctx, "cash rounding created", "id", cr.ID, "name", cr.Name)
	return cr, nil
}

func (uc *UseCase) GetCashRounding(ctx context.Context, id int64) (*bankstatement.CashRounding, error) {
	return uc.repo.GetCashRoundingByID(ctx, id)
}

func (uc *UseCase) UpdateCashRounding(ctx context.Context, id int64, cr *bankstatement.CashRounding) (*bankstatement.CashRounding, error) {
	existing, err := uc.repo.GetCashRoundingByID(ctx, id)
	if err != nil {
		return nil, err
	}
	cr.ID = existing.ID
	cr.CreatedAt = existing.CreatedAt
	if err := cr.Validate(); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateCashRounding(ctx, cr); err != nil {
		return nil, err
	}
	return cr, nil
}

func (uc *UseCase) DeleteCashRounding(ctx context.Context, id int64) error {
	return uc.repo.DeleteCashRounding(ctx, id)
}

func (uc *UseCase) ListCashRoundings(ctx context.Context) ([]bankstatement.CashRounding, error) {
	return uc.repo.ListCashRoundings(ctx)
}

// RoundLine is the rounding application result for a single statement line.
type RoundLine struct {
	LineID         int64   `json:"line_id"`
	OriginalAmount float64 `json:"original_amount"`
	RoundedAmount  float64 `json:"rounded_amount"`
	Difference     float64 `json:"difference"`
}

// ApplyCashRounding indicates the rounding difference for each unreconciled line.
// The domain Round only; the write-back follows the Odoo add_invoice_line strategy.
func (uc *UseCase) ApplyCashRounding(ctx context.Context, statementID int64, roundingID int64) ([]RoundLine, error) {
	cr, err := uc.repo.GetCashRoundingByID(ctx, roundingID)
	if err != nil {
		return nil, err
	}
	st, err := uc.repo.GetStatementWithLines(ctx, statementID)
	if err != nil {
		return nil, err
	}

	var results []RoundLine
	for _, l := range st.Lines {
		if l.Reconciled {
			continue
		}
		difference := cr.RoundDiff(l.Amount)
		if mathAbs(difference) <= residualTolerance {
			continue
		}
		results = append(results, RoundLine{
			LineID:         l.ID,
			OriginalAmount: l.Amount,
			RoundedAmount:  cr.Round(l.Amount),
			Difference:     difference,
		})
	}
	uc.logger.InfoContext(ctx, "cash rounding previewed",
		"statement_id", statementID, "rounding_id", roundingID, "lines", len(results))
	return results, nil
}
