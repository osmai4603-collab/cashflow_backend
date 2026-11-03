package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AuditHasher computes the hash that binds one audit log to the prior log.
type AuditHasher struct{}

func (AuditHasher) ComputeHash(prevHash string, entityID uuid.UUID, action string, payload []byte, timestamp time.Time) string {
	raw := fmt.Sprintf("%s|%s|%s|%s|%d", prevHash, entityID.String(), action, payload, timestamp.UnixNano())
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// VerifyAuditIntegrity checks the ordered audit log chain and its payload hashes.
func (r *PostgresRepository) VerifyAuditIntegrity(ctx context.Context) (bool, error) {
	const query = `
		SELECT id, entity_id, action, COALESCE(payload_after::text, ''),
			COALESCE(previous_log_hash, ''), current_log_hash, timestamp
		FROM financial_audit_logs
		ORDER BY id ASC`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return false, fmt.Errorf("query financial audit logs: %w", err)
	}
	defer rows.Close()

	var expectedPrevious string
	hasher := AuditHasher{}
	for rows.Next() {
		var id int64
		var entityID uuid.UUID
		var action, payload, previousHash, currentHash string
		var timestamp time.Time
		if err := rows.Scan(&id, &entityID, &action, &payload, &previousHash, &currentHash, &timestamp); err != nil {
			return false, fmt.Errorf("scan financial audit log %d: %w", id, err)
		}
		if previousHash != expectedPrevious {
			return false, fmt.Errorf("tamper detected at audit log id %d: broken previous hash link", id)
		}
		calculated := hasher.ComputeHash(previousHash, entityID, action, []byte(payload), timestamp)
		if calculated != currentHash {
			return false, fmt.Errorf("tamper detected at audit log id %d: payload hash mismatch", id)
		}
		expectedPrevious = currentHash
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate financial audit logs: %w", err)
	}
	return true, nil
}
