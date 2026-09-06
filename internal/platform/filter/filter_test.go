package filter_test

import (
	"testing"

	"cashflow_backend/internal/platform/filter"
)

func TestBuildWhereClause_AllowedFields(t *testing.T) {
	allowed := map[string]string{
		"name":        "partners.name",
		"is_customer": "partners.is_customer",
		"country":     "partners.country_code",
		"balance":     "partners.total_invoiced",
		"deleted_at":  "partners.deleted_at",
	}

	f := filter.NewFilter().
		Add("is_customer", filter.OpEqual, true).
		Add("name", filter.OpILike, "Acme").
		Add("balance", filter.OpGreaterThan, 1000).
		Add("deleted_at", filter.OpIsNull, nil)

	where, args, nextIdx, err := f.BuildWhereClause(allowed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedWhere := "WHERE partners.is_customer = $1 AND partners.name ILIKE $2 AND partners.total_invoiced > $3 AND partners.deleted_at IS NULL"
	if where != expectedWhere {
		t.Errorf("expected where clause:\n%s\ngot:\n%s", expectedWhere, where)
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}

	if args[0] != true {
		t.Errorf("expected arg[0] = true, got %v", args[0])
	}
	if args[1] != "%Acme%" {
		t.Errorf("expected arg[1] = %%Acme%%, got %v", args[1])
	}
	if args[2] != 1000 {
		t.Errorf("expected arg[2] = 1000, got %v", args[2])
	}
	if nextIdx != 4 {
		t.Errorf("expected next index 4, got %d", nextIdx)
	}
}

func TestBuildWhereClause_DisallowedField(t *testing.T) {
	allowed := map[string]string{
		"name": "partners.name",
	}

	f := filter.NewFilter().Add("password_hash", filter.OpEqual, "secret")
	_, _, _, err := f.BuildWhereClause(allowed, 1)
	if err == nil {
		t.Errorf("expected error when filtering on disallowed field")
	}
}

func TestBuildWhereClause_InOperator(t *testing.T) {
	allowed := map[string]string{
		"status": "orders.status",
	}

	f := filter.NewFilter().Add("status", filter.OpIn, []string{"draft", "sent", "sale"})
	where, args, nextIdx, err := f.BuildWhereClause(allowed, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedWhere := "WHERE orders.status IN ($1, $2, $3)"
	if where != expectedWhere {
		t.Errorf("expected %s, got %s", expectedWhere, where)
	}
	if len(args) != 3 || nextIdx != 4 {
		t.Errorf("expected 3 args and nextIdx=4, got %d args, nextIdx=%d", len(args), nextIdx)
	}
}

func TestBuildWhereClause_EmptyFilter(t *testing.T) {
	var f *filter.Filter
	where, args, nextIdx, err := f.BuildWhereClause(nil, 1)
	if err != nil || where != "" || len(args) != 0 || nextIdx != 1 {
		t.Errorf("expected empty where for nil filter")
	}
}
