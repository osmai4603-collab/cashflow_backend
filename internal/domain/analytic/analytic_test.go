package analytic

import (
	"errors"
	"testing"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

func TestDistributionKey(t *testing.T) {
	tests := []struct {
		name string
		ids  []int64
		want string
	}{
		{"single account", []int64{1}, "1"},
		{"combines and sorts", []int64{5, 1, 3}, "1,3,5"},
		{"preserves order deterministically", []int64{9, 2, 7}, "2,7,9"},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DistributionKey(tt.ids); got != tt.want {
				t.Fatalf("DistributionKey(%v) = %q, want %q", tt.ids, got, tt.want)
			}
		})
	}
}

func TestMapFromAccountIDs(t *testing.T) {
	d := MapFromAccountIDs([]int64{4})
	if len(d) != 1 || d["4"] != 100 {
		t.Fatalf("expected {\"4\":100}, got %v", d)
	}

	combo := MapFromAccountIDs([]int64{1, 5})
	if len(combo) != 1 || combo["1,5"] != 100 {
		t.Fatalf("expected {\"1,5\":100}, got %v", combo)
	}

	if got := MapFromAccountIDs(nil); got != nil {
		t.Fatalf("expected nil for empty input, got %v", got)
	}
}

func TestAnalyticDistribution_AccountIDs(t *testing.T) {
	d := AnalyticDistribution{
		"1":   50,
		"2,5": 50,
	}
	ids := d.AccountIDs()
	if len(ids) != 3 {
		t.Fatalf("expected 3 account ids, got %v", ids)
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	for _, want := range []int64{1, 2, 5} {
		if !seen[want] {
			t.Fatalf("expected account %d in %v", want, ids)
		}
	}
}

func TestAnalyticDistribution_Validate100(t *testing.T) {
	t.Run("sums to 100", func(t *testing.T) {
		d := AnalyticDistribution{"1": 50, "2": 50}
		if err := d.Validate100(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("fails when sum differs", func(t *testing.T) {
		d := AnalyticDistribution{"1": 60, "2": 50}
		err := d.Validate100()
		if err == nil {
			t.Fatal("expected validation error for a 110% distribution")
		}
		var appErr *platformerrors.AppError
		if !errors.As(err, &appErr) || appErr.Code != platformerrors.CodeValidation {
			t.Fatalf("expected VALIDATION error, got %v", err)
		}
	})

	t.Run("empty is valid", func(t *testing.T) {
		if err := (AnalyticDistribution{}).Validate100(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestMerge(t *testing.T) {
	t.Run("preserves unchanged old keys", func(t *testing.T) {
		old := AnalyticDistribution{"1": 50, "2": 50}
		newDist := AnalyticDistribution{"2": 100}

		res := Merge(old, newDist)
		if res.Merged["1"] != 50 {
			t.Fatalf("key '1' should be preserved, got %v", res.Merged)
		}
		if res.Merged["2"] != 100 {
			t.Fatalf("key '2' should take the new value, got %v", res.Merged)
		}
	})

	t.Run("reports changed when value overridden", func(t *testing.T) {
		old := AnalyticDistribution{"1": 50, "2": 50}
		newDist := AnalyticDistribution{"1": 60}

		res := Merge(old, newDist)
		if !res.Changed {
			t.Fatal("expected Changed=true when overriding an existing key")
		}
	})

	t.Run("reports no change on identical input", func(t *testing.T) {
		old := AnalyticDistribution{"1": 100}
		res := Merge(old, AnalyticDistribution{"1": 100})
		if res.Changed {
			t.Fatal("expected Changed=false for identical distributions")
		}
	})

	t.Run("empty new returns old unchanged", func(t *testing.T) {
		old := AnalyticDistribution{"1": 100}
		res := Merge(old, nil)
		if res.Changed || res.Merged["1"] != 100 {
			t.Fatalf("expected old distribution preserved, got %v changed=%v", res.Merged, res.Changed)
		}
	})

	t.Run("empty old adopts new", func(t *testing.T) {
		newDist := AnalyticDistribution{"2": 100}
		res := Merge(nil, newDist)
		if !res.Changed || res.Merged["2"] != 100 {
			t.Fatalf("expected new distribution adopted, got %v changed=%v", res.Merged, res.Changed)
		}
	})
}

func TestAnalyticPlan_Validate(t *testing.T) {
	t.Run("valid plan defaults to optional", func(t *testing.T) {
		p := &AnalyticPlan{Name: "Project"}
		if err := p.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.DefaultApplicability != AppOptional {
			t.Fatalf("expected default applicability optional, got %s", p.DefaultApplicability)
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		p := &AnalyticPlan{}
		if err := p.Validate(); err == nil {
			t.Fatal("expected validation error for empty name")
		}
	})

	t.Run("rejects self parent", func(t *testing.T) {
		self := int64(3)
		p := &AnalyticPlan{ID: 3, Name: "Plan", ParentID: &self}
		if err := p.Validate(); err == nil {
			t.Fatal("expected validation error for circular parent")
		}
	})
}

func TestAnalyticLine_Validate(t *testing.T) {
	line := &AnalyticLine{
		Name:      "Consulting services",
		Date:      time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		AccountID: 1,
		UserID:    1,
		CompanyID: 1,
	}
	if err := line.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("rejects missing account (G4)", func(t *testing.T) {
		l := &AnalyticLine{
			Name:      "x",
			Date:      time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			UserID:    1,
			CompanyID: 1,
		}
		if err := l.Validate(); err == nil {
			t.Fatal("expected validation error when no analytic account is set")
		}
	})
}

func TestDistributionModel_Validate(t *testing.T) {
	t.Run("valid model", func(t *testing.T) {
		m := &DistributionModel{Distribution: AnalyticDistribution{"1": 100}}
		if err := m.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects invalid distribution", func(t *testing.T) {
		m := &DistributionModel{Distribution: AnalyticDistribution{"1": 70, "2": 40}}
		if err := m.Validate(); err == nil {
			t.Fatal("expected validation error for a 110% distribution")
		}
	})

	t.Run("rejects both partner and category", func(t *testing.T) {
		pid := int64(1)
		cid := int64(2)
		m := &DistributionModel{
			PartnerID:         &pid,
			PartnerCategoryID: &cid,
			Distribution:      AnalyticDistribution{"1": 100},
		}
		if err := m.Validate(); err == nil {
			t.Fatal("expected validation error when both partner and category are set")
		}
	})
}

func TestDebitCreditBalance(t *testing.T) {
	b := DebitCreditBalance{Debit: 100, Credit: 450}
	b.ComputeBalance()
	if b.Balance != 350 {
		t.Fatalf("expected balance 350, got %f", b.Balance)
	}
}