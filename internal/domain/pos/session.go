package pos

import (
	"fmt"
	"math"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type SessionState string

const (
	SessionStateOpeningControl SessionState = "opening_control"
	SessionStateOpened         SessionState = "opened"
	SessionStateClosingControl SessionState = "closing_control"
	SessionStateClosed         SessionState = "closed"
)

type PosSession struct {
	ID                       int64        `json:"id"`
	ConfigID                 int64        `json:"config_id"`
	UserID                   int64        `json:"user_id"`
	Name                     string       `json:"name"`
	State                    SessionState `json:"state"`
	StartAt                  time.Time    `json:"start_at"`
	StopAt                   *time.Time   `json:"stop_at,omitempty"`
	CashRegisterBalanceStart float64      `json:"cash_register_balance_start"`
	CashRegisterBalanceEnd   float64      `json:"cash_register_balance_end"`
	CashRegisterBalanceReal  float64      `json:"cash_register_balance_real"`
	CashRegisterDifference   float64      `json:"cash_register_difference"`
	TotalOrdersCount         int          `json:"total_orders_count"`
	TotalPaymentsAmount      float64      `json:"total_payments_amount"`
	StockPickingID           *int64       `json:"stock_picking_id,omitempty"`
	AccountMoveID            *int64       `json:"account_move_id,omitempty"`
	CompanyID                int64        `json:"company_id"`
}

func (session *PosSession) Open(openingBalance float64) error {
	if session.State != "" && session.State != SessionStateOpeningControl {
		return invalidState("session", session.State)
	}
	if openingBalance < 0 || math.IsNaN(openingBalance) || math.IsInf(openingBalance, 0) {
		return platformerrors.Validation("opening cash balance must be non-negative", map[string]string{"opening_balance": "must be finite and not negative"})
	}
	session.CashRegisterBalanceStart = roundMoney(openingBalance)
	session.State = SessionStateOpened
	if session.StartAt.IsZero() {
		session.StartAt = time.Now().UTC()
	}
	return nil
}

func (session *PosSession) BeginClosing(expectedCash float64) error {
	if session.State != SessionStateOpened {
		return invalidState("session", session.State)
	}
	if expectedCash < 0 {
		return platformerrors.Validation("expected cash balance must be non-negative", nil)
	}
	session.CashRegisterBalanceReal = roundMoney(expectedCash)
	session.State = SessionStateClosingControl
	return nil
}

func (session *PosSession) Close(countedCash float64) error {
	if session.State != SessionStateClosingControl {
		return invalidState("session", session.State)
	}
	if countedCash < 0 {
		return platformerrors.Validation("counted cash balance must be non-negative", nil)
	}
	session.CashRegisterBalanceEnd = roundMoney(countedCash)
	session.CashRegisterDifference = roundMoney(countedCash - session.CashRegisterBalanceReal)
	session.State = SessionStateClosed
	closedAt := time.Now().UTC()
	session.StopAt = &closedAt
	return nil
}

func (session *PosSession) AddPayment(amount float64) error {
	if session.State != SessionStateOpened {
		return invalidState("session", session.State)
	}
	if amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return platformerrors.Validation("payment amount must be positive and finite", nil)
	}
	session.TotalPaymentsAmount = roundMoney(session.TotalPaymentsAmount + amount)
	return nil
}

func (session *PosSession) Validate() error {
	if session.ConfigID <= 0 || session.UserID <= 0 || session.CompanyID <= 0 {
		return platformerrors.Validation("session references are required", map[string]string{"references": "config, user, and company must be valid"})
	}
	if session.State == "" {
		session.State = SessionStateOpeningControl
	}
	switch session.State {
	case SessionStateOpeningControl, SessionStateOpened, SessionStateClosingControl, SessionStateClosed:
		return nil
	default:
		return platformerrors.Validation(fmt.Sprintf("invalid session state '%s'", session.State), nil)
	}
}

func roundMoney(value float64) float64 {
	return math.Round(value*10000) / 10000
}
