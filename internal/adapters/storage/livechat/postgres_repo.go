package livechatstorage

import (
	"context"
	"cashflow_backend/internal/domain/livechat"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) GetChannel(ctx context.Context, id int64) (*livechat.Channel, error) { return nil, nil }
func (r *PostgresRepo) ListChannels(ctx context.Context, companyID int64) ([]livechat.Channel, error) { return nil, nil }
func (r *PostgresRepo) CreateChannel(ctx context.Context, channel *livechat.Channel) error { return nil }
func (r *PostgresRepo) UpdateChannel(ctx context.Context, channel *livechat.Channel) error { return nil }
func (r *PostgresRepo) GetSession(ctx context.Context, id int64) (*livechat.Session, error) { return nil, nil }
func (r *PostgresRepo) GetSessionByUUID(ctx context.Context, uuid string) (*livechat.Session, error) { return nil, nil }
func (r *PostgresRepo) CreateSession(ctx context.Context, session *livechat.Session) error { return nil }
func (r *PostgresRepo) UpdateSession(ctx context.Context, session *livechat.Session) error { return nil }
func (r *PostgresRepo) ListSessions(ctx context.Context, channelID int64) ([]livechat.Session, error) { return nil, nil }
func (r *PostgresRepo) CreateMessage(ctx context.Context, msg *livechat.Message) error { return nil }
func (r *PostgresRepo) ListMessages(ctx context.Context, sessionID int64) ([]livechat.Message, error) { return nil, nil }
func (r *PostgresRepo) ListCannedResponses(ctx context.Context, companyID int64) ([]livechat.CannedResponse, error) { return nil, nil }
