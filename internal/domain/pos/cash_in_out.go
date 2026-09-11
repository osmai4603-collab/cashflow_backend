package pos

import (
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type CashMovementType string

const (
	CashMovementIn  CashMovementType = "in"
	CashMovementOut CashMovementType = "out"
)

type CashInOutMovement struct {
	ID        int64            `json:"id"`
	SessionID int64            `json:"session_id"`
	Type      CashMovementType `json:"type"`
	Amount    float64          `json:"amount"`
	Reason    string           `json:"reason"`
	UserID    int64            `json:"user_id"`
}

func (movement *CashInOutMovement) Validate() error {
	movement.Reason = strings.TrimSpace(movement.Reason)
	if movement.SessionID <= 0 || movement.UserID <= 0 || movement.Amount <= 0 || movement.Reason == "" {
		return platformerrors.Validation("cash movement requires session, user, positive amount, and reason", nil)
	}
	if movement.Type != CashMovementIn && movement.Type != CashMovementOut {
		return platformerrors.Validation("invalid cash movement type", nil)
	}
	return nil
}
