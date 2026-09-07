package notificationbus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresChannel = "cashflow_notification"

type postgresEvent struct {
	Origin  string          `json:"origin"`
	UserID  int64           `json:"user_id"`
	Channel string          `json:"channel"`
	Payload json.RawMessage `json:"payload"`
}

// PostgresRelay adds cross-process delivery to a local Bus.
type PostgresRelay struct {
	pool   *pgxpool.Pool
	local  *Bus
	origin string
}

func NewPostgresRelay(pool *pgxpool.Pool, local *Bus) *PostgresRelay {
	return &PostgresRelay{pool: pool, local: local, origin: processID()}
}

func (r *PostgresRelay) NotifyUser(ctx context.Context, userID int64, channel string, payload any) error {
	if err := r.local.NotifyUser(ctx, userID, channel, payload); err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	event, err := json.Marshal(postgresEvent{Origin: r.origin, UserID: userID, Channel: channel, Payload: encoded})
	if err != nil {
		return fmt.Errorf("marshal notification event: %w", err)
	}
	_, err = r.pool.Exec(ctx, "SELECT pg_notify($1, $2)", postgresChannel, string(event))
	return err
}

// Start listens for events emitted by other service instances until ctx is cancelled.
func (r *PostgresRelay) Start(ctx context.Context) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire notification listener: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "LISTEN "+postgresChannel); err != nil {
		return fmt.Errorf("listen for notifications: %w", err)
	}
	for {
		notification, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("wait for notification: %w", err)
		}
		var event postgresEvent
		if err := json.Unmarshal([]byte(notification.Payload), &event); err != nil || event.Origin == r.origin {
			continue
		}
		var payload any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			continue
		}
		if err := r.local.NotifyUser(ctx, event.UserID, event.Channel, payload); err != nil && ctx.Err() != nil {
			return nil
		}
	}
}

func processID() string {
	value := make([]byte, 12)
	if _, err := rand.Read(value); err != nil {
		return fmt.Sprintf("fallback-%p", value)
	}
	return hex.EncodeToString(value)
}
