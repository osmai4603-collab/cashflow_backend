package expense_test

import (
	"testing"

	"cashflow_backend/internal/domain/expense"
)

func TestExpenseValidate_DefaultsDraftAndPaymentMode(t *testing.T) {
	e := expense.Expense{Name: "Taxi", EmployeeID: 1, TotalAmount: 25}

	if err := e.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
	if e.State != expense.StateDraft {
		t.Fatalf("state = %q, want draft", e.State)
	}
	if e.PaymentMode != expense.PaymentOwnAccount {
		t.Fatalf("payment mode = %q, want own_account", e.PaymentMode)
	}
	if e.Quantity != 1 {
		t.Fatalf("quantity = %v, want 1", e.Quantity)
	}
}

func TestExpenseValidate_RejectsInvalidWorkflowValues(t *testing.T) {
	tests := []struct {
		name string
		expense.Expense
	}{
		{name: "invalid payment mode", Expense: expense.Expense{Name: "Taxi", EmployeeID: 1, TotalAmount: 25, PaymentMode: "cash"}},
		{name: "invalid state", Expense: expense.Expense{Name: "Taxi", EmployeeID: 1, TotalAmount: 25, State: "cancelled"}},
		{name: "zero submitted expense", Expense: expense.Expense{Name: "Taxi", EmployeeID: 1, State: expense.StateSubmitted}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.Expense.Validate(); err == nil {
				t.Fatal("Validate() returned nil, want validation error")
			}
		})
	}
}

func TestExpenseValidate_AllOdooStatesAreAccepted(t *testing.T) {
	states := []expense.ExpenseState{
		expense.StateDraft,
		expense.StateSubmitted,
		expense.StateApproved,
		expense.StatePosted,
		expense.StateInPayment,
		expense.StatePaid,
		expense.StateRefused,
	}

	for _, state := range states {
		e := expense.Expense{Name: "Hotel", EmployeeID: 1, TotalAmount: 100, State: state}
		if err := e.Validate(); err != nil {
			t.Errorf("state %q rejected: %v", state, err)
		}
	}
}