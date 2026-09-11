package pos

import "time"

type SyncBatch struct {
	ID             int64      `json:"id"`
	SessionID      int64      `json:"session_id"`
	IdempotencyKey string     `json:"idempotency_key"`
	Orders         []PosOrder `json:"orders"`
	ReceivedAt     time.Time  `json:"received_at"`
}

type SyncResult struct {
	SessionID      int64   `json:"session_id"`
	IdempotencyKey string  `json:"idempotency_key"`
	Accepted       int     `json:"accepted"`
	Duplicate      bool    `json:"duplicate"`
	OrderIDs       []int64 `json:"order_ids"`
}
