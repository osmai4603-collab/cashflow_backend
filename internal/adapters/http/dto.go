package httpadapter

import (
	"time"

	"cashflow_backend/internal/domain"
	"cashflow_backend/internal/usecase"
)

type CreateTransactionRequest struct {
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
}

type TransactionResponse struct {
	ID          string    `json:"id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type SummaryResponse struct {
	TotalIncome  float64 `json:"total_income"`
	TotalExpense float64 `json:"total_expense"`
	NetCashflow  float64 `json:"net_cashflow"`
	Count        int     `json:"count"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func toTransactionResponse(tx *domain.Transaction) TransactionResponse {
	return TransactionResponse{
		ID:          tx.ID,
		Amount:      tx.Amount,
		Type:        string(tx.Type),
		Description: tx.Description,
		CreatedAt:   tx.CreatedAt,
	}
}

func toSummaryResponse(s *usecase.Summary) SummaryResponse {
	return SummaryResponse{
		TotalIncome:  s.TotalIncome,
		TotalExpense: s.TotalExpense,
		NetCashflow:  s.NetCashflow,
		Count:        s.Count,
	}
}
