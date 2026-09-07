package bankstatementusecase

import (
	"context"
	"fmt"
	"time"

	"cashflow_backend/internal/domain/accounting"
	"cashflow_backend/internal/domain/bankstatement"
	platformerrors "cashflow_backend/internal/platform/errors"
)

// ─────────────────────────────────────────────────────────────────────────────
// Candidate surfacing
// ─────────────────────────────────────────────────────────────────────────────

// Candidate is a reconciliation candidate for a statement line.
type Candidate struct {
	MoveLine        accounting.AccountMoveLine
	SuggestedAmount float64 `json:"suggested_amount"`
	ExactMatch      bool    `json:"exact_match"`
}

// FindCandidates returns posted, reconcilable move lines that can be settled with
// the given statement line (same money direction, outstanding amount within tolerance).
func (uc *UseCase) FindCandidates(ctx context.Context, lineID int64, limit int) ([]Candidate, error) {
	line, err := uc.repo.GetStatementLineByID(ctx, lineID)
	if err != nil {
		return nil, err
	}
	if line.Reconciled {
		return nil, platformerrors.Conflict("statement line is already reconciled")
	}

	raw, err := uc.accountingSvc.ListReconcilableMoveLines(ctx, line.PartnerID, nil, limit)
	if err != nil {
		return nil, err
	}

	want := mathAbs(line.Amount)
	var candidates []Candidate
	for _, l := range raw {
		// The statement side and the candidate side must balance out: same net direction.
		if (line.Amount > 0 && l.Balance <= 0) || (line.Amount < 0 && l.Balance >= 0) {
			continue
		}
		// A candidate whose residual exceeds the statement line cannot be fully settled by it.
		if l.AmountResidual > want+residualTolerance {
			continue
		}
		amount := l.AmountResidual
		if amount > want {
			amount = want
		}
		exact := mathAbs(l.AmountResidual-want) <= residualTolerance
		candidates = append(candidates, Candidate{
			MoveLine:        l,
			SuggestedAmount: roundTo4(amount),
			ExactMatch:      exact,
		})
	}

	// Exact matches first, then by distance from the target amount.
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			c := candidates[i]
			d := candidates[j]
			less := false
			if c.ExactMatch != d.ExactMatch {
				less = c.ExactMatch
			} else {
				less = mathAbs(c.MoveLine.AmountResidual-want) < mathAbs(d.MoveLine.AmountResidual-want)
			}
			if less {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	return candidates, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Manual reconciliation
// ─────────────────────────────────────────────────────────────────────────────

// ReconcileLine reconciles a statement line against an account move line candidate.
func (uc *UseCase) ReconcileLine(ctx context.Context, lineID, candidateLineID int64) (*ReconcileResult, error) {
	line, err := uc.repo.GetStatementLineByID(ctx, lineID)
	if err != nil {
		return nil, err
	}
	if line.Reconciled {
		return nil, platformerrors.Conflict("statement line is already reconciled")
	}
	if line.MoveID == nil || *line.MoveID <= 0 {
		return nil, platformerrors.Validation("statement line is not booked", map[string]string{
			"statement_line_id": "the line must be booked before it can be reconciled",
		})
	}

	move, err := uc.accountingSvc.GetMove(ctx, *line.MoveID)
	if err != nil {
		return nil, err
	}
	counterpartID, err := findCounterpartLine(move, line.ID)
	if err != nil {
		return nil, err
	}

	candidate, err := uc.accountingSvc.GetMoveLine(ctx, candidateLineID)
	if err != nil {
		return nil, err
	}
	if candidate.Reconciled || candidate.AmountResidual <= residualTolerance {
		return nil, platformerrors.Conflict("candidate move line is already reconciled")
	}
	if (line.Amount > 0 && candidate.Balance <= 0) || (line.Amount < 0 && candidate.Balance >= 0) {
		return nil, platformerrors.Validation("cannot reconcile opposite money directions", map[string]string{
			"candidate_line_id": "candidate must have the same net direction as the statement line",
		})
	}

	// The settlement amount is limited by the smallest outstanding balance.
	amount := min3(line.AmountResidual, candidate.AmountResidual, mathAbs(line.Amount))
	if amount <= residualTolerance {
		return nil, platformerrors.Conflict("nothing left to reconcile")
	}
	amount = roundTo4(amount)

	lineNewResidual := roundTo4(line.AmountResidual - amount)
	lineReconciled := lineNewResidual <= residualTolerance
	candNewResidual := roundTo4(candidate.AmountResidual - amount)
	candReconciled := candNewResidual <= residualTolerance

	var matchingNumber *string
	if candReconciled {
		num := newMatchingNumber(line.ID, candidate.ID)
		matchingNumber = &num
		fr := &bankstatement.FullReconcile{MatchingNumber: num}
		if err := uc.repo.CreateFullReconcile(ctx, fr); err != nil {
			return nil, err
		}
	}

	pr := &bankstatement.PartialReconcile{
		Amount:         amount,
		AmountCurrency: amount,
		Currency:       line.Currency,
	}
	if candidate.Balance > 0 {
		pr.DebitMoveID = candidate.MoveID
		pr.DebitLineID = candidate.ID
		pr.CreditMoveID = move.ID
		pr.CreditLineID = counterpartID
	} else {
		pr.DebitMoveID = move.ID
		pr.DebitLineID = counterpartID
		pr.CreditMoveID = candidate.MoveID
		pr.CreditLineID = candidate.ID
	}
	if err := uc.repo.CreatePartialReconcile(ctx, pr); err != nil {
		return nil, err
	}

	// Persist reconcile state on both accounting lines and the statement line.
	if err := uc.accountingSvc.UpdateMoveLineReconcile(ctx, candidate.ID, candReconciled, candNewResidual, matchingNumber); err != nil {
		return nil, err
	}
	if err := uc.accountingSvc.UpdateMoveLineReconcile(ctx, counterpartID, lineReconciled, lineNewResidual, matchingNumber); err != nil {
		return nil, err
	}
	if err := uc.repo.UpdateStatementLineReconcileState(ctx, line.ID, lineReconciled, lineNewResidual, matchingNumber); err != nil {
		return nil, err
	}

	// Recompute the invoice's payment state once the candidate line is touched.
	paymentState, err := uc.recomputeMovePaymentState(ctx, candidate.MoveID)
	if err != nil {
		return nil, err
	}

	uc.logger.InfoContext(ctx, "statement line reconciled",
		"statement_line_id", line.ID, "candidate_line_id", candidate.ID,
		"amount", amount, "fully_reconciled", lineReconciled)

	return &ReconcileResult{
		StatementLineID:     line.ID,
		CandidateLineID:     candidate.ID,
		Amount:              amount,
		StatementReconciled: lineReconciled,
		CandidateReconciled: candReconciled,
		MatchingNumber:      matchingNumber,
		PaymentState:        paymentState,
	}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Undo
// ─────────────────────────────────────────────────────────────────────────────

// UndoReconcile reverses every partial reconcile involving the statement line,
// restoring the outstanding balances of the involved accounting lines.
func (uc *UseCase) UndoReconcile(ctx context.Context, lineID int64) error {
	line, err := uc.repo.GetStatementLineByID(ctx, lineID)
	if err != nil {
		return err
	}
	if !line.Reconciled {
		return platformerrors.Conflict("statement line is not reconciled")
	}
	if line.MoveID == nil || *line.MoveID <= 0 {
		return nil
	}

	move, err := uc.accountingSvc.GetMove(ctx, *line.MoveID)
	if err != nil {
		return err
	}
	counterpartID, err := findCounterpartLine(move, line.ID)
	if err != nil {
		return err
	}

	partials, err := uc.repo.ListPartialReconcilesByLine(ctx, counterpartID)
	if err != nil {
		return err
	}

	var touchedMoveIDs = make(map[int64]struct{})
	for _, pr := range partials {
		var counterpart int64
		switch pr.DebitLineID {
		case counterpartID:
			counterpart = pr.CreditLineID
		default:
			counterpart = pr.DebitLineID
		}

		other, err := uc.accountingSvc.GetMoveLine(ctx, counterpart)
		if err != nil {
			return err
		}
		if err := uc.accountingSvc.UpdateMoveLineReconcile(ctx, other.ID, false, roundTo4(other.AmountResidual+pr.Amount), nil); err != nil {
			return err
		}
		touchedMoveIDs[other.MoveID] = struct{}{}

		if err := uc.repo.DeletePartialReconcile(ctx, pr.ID); err != nil {
			return err
		}
	}

	// Restore the statement side (counterpart line + statement line bookkeeping).
	if err := uc.accountingSvc.UpdateMoveLineReconcile(ctx, counterpartID, false, mathAbs(line.Amount), nil); err != nil {
		return err
	}
	if err := uc.repo.UpdateStatementLineReconcileState(ctx, line.ID, false, mathAbs(line.Amount), nil); err != nil {
		return err
	}

	for moveID := range touchedMoveIDs {
		if _, err := uc.recomputeMovePaymentState(ctx, moveID); err != nil {
			return err
		}
	}

	uc.logger.InfoContext(ctx, "statement line reconciliation undone", "statement_line_id", lineID)
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// findCounterpartLine locates the non-liquidity line of a statement line's move.
// The liquidity line carries the statement_line_id; the counterpart is the other line.
func findCounterpartLine(move *accounting.AccountMove, statementLineID int64) (int64, error) {
	var counterpart int64
	found := false
	for _, l := range move.Lines {
		if l.StatementLineID != nil && *l.StatementLineID == statementLineID {
			continue // liquidity line
		}
		if found {
			return 0, platformerrors.Internal("statement move has more than one counterpart line", nil)
		}
		counterpart = l.ID
		found = true
	}
	if !found {
		return 0, platformerrors.Internal("statement move counterpart line not found", nil)
	}
	return counterpart, nil
}

// recomputeMovePaymentState derives the residual of a move from its reconcilable lines
// (excluding bank statement lines) and pushes the payment state via the accounting service.
func (uc *UseCase) recomputeMovePaymentState(ctx context.Context, moveID int64) (accounting.PaymentState, error) {
	move, err := uc.accountingSvc.GetMove(ctx, moveID)
	if err != nil {
		return "", err
	}

	var residual float64
	hasReconcilable := false
	for _, l := range move.Lines {
		if l.StatementLineID != nil {
			continue
		}
		if l.Reconcile {
			hasReconcilable = true
			if !l.Reconciled {
				residual += l.AmountResidual
			}
		}
	}
	residual = roundTo4(residual)

	var state accounting.PaymentState
	switch {
	case residual <= residualTolerance:
		state = accounting.PaymentStatePaid
	case residual < move.AmountTotal-residualTolerance:
		state = accounting.PaymentStatePartial
	default:
		state = accounting.PaymentStateNotPaid
	}
	if !hasReconcilable {
		state = accounting.PaymentStateNotPaid
	}

	if err := uc.accountingSvc.UpdatePaymentStatus(ctx, moveID, state, residual); err != nil {
		return "", err
	}
	uc.logger.DebugContext(ctx, "move payment status updated", "move_id", moveID, "state", state, "residual", residual)
	return state, nil
}

func min3(a, b, c float64) float64 {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func newMatchingNumber(statementLineID, candidateLineID int64) string {
	return fmt.Sprintf("FR-%d-%d-%d", time.Now().UTC().UnixNano(), statementLineID, candidateLineID)
}
