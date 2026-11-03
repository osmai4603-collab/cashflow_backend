package internal

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cashflow_backend/internal/features/accounting"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestVerifyAuditIntegrityDetectsPayloadChanges(t *testing.T) {
	entityID := uuid.New()
	timestamp := time.Date(2026, 11, 3, 4, 5, 6, 123456000, time.UTC)
	payload := `{"event": "posted"}`
	hash := (AuditHasher{}).ComputeHash("", entityID, "POST", []byte(payload), timestamp)
	rows := &fakeRows{values: [][]any{{
		int64(1), entityID, "POST", payload, "", hash, timestamp,
	}}}
	repository := &PostgresRepository{pool: &fakeQueryPool{rows: rows}}

	valid, err := repository.VerifyAuditIntegrity(context.Background())
	if err != nil || !valid {
		t.Fatalf("VerifyAuditIntegrity() = %v, %v; want valid chain", valid, err)
	}

	rows = &fakeRows{values: [][]any{{
		int64(1), entityID, "POST", `{"event": "modified"}`, "", hash, timestamp,
	}}}
	repository.pool = &fakeQueryPool{rows: rows}
	valid, err = repository.VerifyAuditIntegrity(context.Background())
	if err == nil || valid {
		t.Fatalf("VerifyAuditIntegrity() after payload modification = %v, %v; want integrity error", valid, err)
	}
}

func TestAuditHasherBindsPreviousHash(t *testing.T) {
	hasher := AuditHasher{}
	entityID := uuid.New()
	timestamp := time.Now().UTC()
	payload := []byte(`{"amount": 10}`)

	first := hasher.ComputeHash("", entityID, "POST", payload, timestamp)
	second := hasher.ComputeHash("prior", entityID, "POST", payload, timestamp)
	if first == second {
		t.Fatal("ComputeHash() did not bind the previous hash")
	}
}

func TestInsertAuditLogBuildsChainFromCanonicalPayload(t *testing.T) {
	const previousHash = "previous-hash"
	const canonicalPayload = `{"event": "posted"}`
	tx := newFakeTx()
	tx.row = func(query string) pgx.Row {
		switch {
		case strings.Contains(query, "COALESCE($1::jsonb::text"):
			return fakeRow{values: []any{canonicalPayload}}
		case strings.Contains(query, "SELECT current_log_hash"):
			return fakeRow{values: []any{previousHash}}
		default:
			return fakeRow{err: errors.New("unexpected query: " + query)}
		}
	}

	timestamp := time.Date(2026, 11, 3, 4, 5, 6, 123456789, time.UTC)
	audit := &accounting.FinancialAuditLog{
		EntityType: "account_move", EntityID: uuid.New(), Action: "POST",
		PayloadAfter: []byte(`{"event":"posted"}`), Timestamp: timestamp,
	}
	if err := insertAuditLog(context.Background(), tx, audit); err != nil {
		t.Fatalf("insertAuditLog() error = %v", err)
	}

	if audit.PreviousLogHash != previousHash {
		t.Fatalf("previous hash = %q, want %q", audit.PreviousLogHash, previousHash)
	}
	if audit.Timestamp.Nanosecond() != 123456000 {
		t.Fatalf("audit timestamp precision = %d ns, want microsecond precision", audit.Timestamp.Nanosecond())
	}
	wantHash := (AuditHasher{}).ComputeHash(
		previousHash, audit.EntityID, audit.Action, []byte(canonicalPayload), audit.Timestamp,
	)
	if audit.CurrentLogHash != wantHash {
		t.Fatalf("current hash = %q, want %q", audit.CurrentLogHash, wantHash)
	}
	if len(tx.execQueries) != 2 || !strings.Contains(tx.execQueries[0], "pg_advisory_xact_lock") ||
		!strings.Contains(tx.execQueries[1], "INSERT INTO financial_audit_logs") {
		t.Fatalf("audit write queries = %v", tx.execQueries)
	}
}
