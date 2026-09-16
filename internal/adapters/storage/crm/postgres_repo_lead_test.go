package crmstorage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cashflow_backend/internal/domain/crm"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"

	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool returns a pool connected to the local PostgreSQL instance, or nil to
// skip when no database is configured (e.g. unit-only CI environments).
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("CASHFLOW_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/cashflow?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping: cannot parse database url: %v", err)
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()
	if err := pool.Ping(ctx2); err != nil {
		pool.Close()
		t.Skipf("skipping: database is unreachable: %v", err)
	}
	return pool
}

// insertNullColumnLead inserts a lead whose every nullable text column is NULL
// (the exact shape that previously crashed `failed to scan lead`).
func insertNullColumnLead(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, leadType string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(ctx, `
		INSERT INTO crm_leads (name, type, stage_id, expected_revenue, prorated_revenue, probability, priority, active, company_id)
		VALUES ($1, $2, 1, 0.0000, 0.0000, 0.00, '1', true, 1)
		RETURNING id
	`, name, leadType).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert null-column lead: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM crm_leads WHERE id = $1`, id)
	})
	return id
}

func TestPostgresRepo_GetLeadByID_ScansNullTextColumns(t *testing.T) {
	pool := testPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	repo := NewPostgresRepo(pool)
	name := fmt.Sprintf("null-lead-%d", time.Now().UnixNano())
	id := insertNullColumnLead(t, ctx, pool, name, string(crm.LeadTypeLead))

	lead, err := repo.GetLeadByID(ctx, id)
	if err != nil {
		t.Fatalf("GetLeadByID must not fail on NULL text columns: %v", err)
	}
	if lead.PartnerName != "" || lead.ContactName != "" || lead.EmailFrom != "" ||
		lead.Phone != "" || lead.Source != "" || lead.LostFeedback != "" || lead.Notes != "" {
		t.Fatalf("expected NULL text columns to be read as empty strings, got %+v", lead)
	}
}

func TestPostgresRepo_ListLeads_ScansNullTextColumns(t *testing.T) {
	pool := testPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	repo := NewPostgresRepo(pool)
	name := fmt.Sprintf("null-list-%d", time.Now().UnixNano())
	insertNullColumnLead(t, ctx, pool, name, string(crm.LeadTypeLead))

	f := filter.NewFilter().Add("name", filter.OpEqual, name)
	result, err := repo.ListLeads(ctx, f, pagination.PageRequest{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListLeads must not fail on NULL text columns: %v", err)
	}
	if result.TotalItems != 1 || len(result.Items) != 1 {
		t.Fatalf("expected exactly 1 lead, got total=%d items=%d", result.TotalItems, len(result.Items))
	}
	lead := result.Items[0]
	if lead.PartnerName != "" || lead.ContactName != "" || lead.EmailFrom != "" ||
		lead.Phone != "" || lead.Source != "" || lead.LostFeedback != "" || lead.Notes != "" {
		t.Fatalf("expected NULL text columns to be read as empty strings, got %+v", lead)
	}
}

func TestPostgresRepo_GetPipeline_ScansNullTextColumns(t *testing.T) {
	pool := testPool(t)
	if pool == nil {
		return
	}
	defer pool.Close()

	ctx := context.Background()
	repo := NewPostgresRepo(pool)
	name := fmt.Sprintf("null-opp-%d", time.Now().UnixNano())
	insertNullColumnLead(t, ctx, pool, name, string(crm.LeadTypeOpportunity))

	pipeline, err := repo.GetPipeline(ctx, nil)
	if err != nil {
		t.Fatalf("GetPipeline must not fail on NULL text columns: %v", err)
	}
	for _, stage := range pipeline {
		for _, opp := range stage.Opportunities {
			if opp.Name == name {
				if opp.PartnerName != "" || opp.ContactName != "" || opp.EmailFrom != "" ||
					opp.Phone != "" || opp.Source != "" || opp.LostFeedback != "" || opp.Notes != "" {
					t.Fatalf("expected NULL text columns to be read as empty strings, got %+v", opp)
				}
				return
			}
		}
	}
	t.Fatalf("opportunity %q not found in pipeline output", name)
}