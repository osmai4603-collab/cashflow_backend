package bankstatement_test

import (
	"testing"

	"cashflow_backend/internal/domain/bankstatement"
)

func mustRounding(method bankstatement.RoundingMethod, granularity float64) *bankstatement.CashRounding {
	c := &bankstatement.CashRounding{
		Name:           "Test rounding",
		RoundingMethod: method,
		Rounding:       granularity,
	}
	if err := c.Validate(); err != nil {
		panic(err)
	}
	return c
}

func TestCashRoundingHalfUp(t *testing.T) {
	c := mustRounding(bankstatement.RoundingMethodHalfUp, 0.05)
	cases := map[float64]float64{
		10.01: 10.00,
		10.02: 10.00,
		10.03: 10.05,
		10.04: 10.05,
		10.05: 10.05,
		10.06: 10.05,
		10.07: 10.05,
		10.08: 10.10,
	}
	for in, want := range cases {
		if got := c.Round(in); got != want {
			t.Errorf("Round(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestCashRoundingUpDown(t *testing.T) {
	if got := mustRounding(bankstatement.RoundingMethodUP, 1).Round(10.2); got != 11 {
		t.Fatalf("UP Round(10.2) = %v, want 11", got)
	}
	if got := mustRounding(bankstatement.RoundingMethodDOWN, 1).Round(10.2); got != 10 {
		t.Fatalf("DOWN Round(10.2) = %v, want 10", got)
	}
	if got := mustRounding(bankstatement.RoundingMethodDOWN, 1).Round(-10.2); got != -11 {
		t.Fatalf("DOWN Round(-10.2) = %v, want -11", got)
	}
}

func TestCashRoundingRoundDiff(t *testing.T) {
	if got := mustRounding(bankstatement.RoundingMethodHalfUp, 1).RoundDiff(10.4); got != -0.4 {
		t.Fatalf("RoundDiff(10.4) half-up = %v, want -0.4", got)
	}
	if got := mustRounding(bankstatement.RoundingMethodUP, 1).RoundDiff(10.4); got != 0.6 {
		t.Fatalf("RoundDiff(10.4) UP = %v, want 0.6", got)
	}
	if got := mustRounding(bankstatement.RoundingMethodDOWN, 1).RoundDiff(10.4); got != -0.4 {
		t.Fatalf("RoundDiff(10.4) DOWN = %v, want -0.4", got)
	}
}
