package examples

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DomainEntity represents a pure enterprise domain entity.
// Notice it contains no database or SQL tags.
type DomainEntity struct {
	ID        string
	Name      string
	Version   int64
	UpdatedAt time.Time
}

// EntityRepositoryPort is the Outbound Port defined by the application/domain layer.
// Following Go's "Accept interfaces, return structs" guideline, this interface belongs
// to the consumer layer, while the infrastructure layer provides the implementation.
type EntityRepositoryPort interface {
	GetByID(ctx context.Context, id string) (*DomainEntity, error)
	Save(ctx context.Context, entity *DomainEntity) error
}

// CachePort is the Outbound Port for caching operations.
type CachePort interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// PostgresRepositoryAdapter implements EntityRepositoryPort using a tuned *sql.DB pool.
type PostgresRepositoryAdapter struct {
	db *sql.DB
}

// NewPostgresRepositoryAdapter returns a concrete struct pointer (idiomatic Go).
func NewPostgresRepositoryAdapter(db *sql.DB) *PostgresRepositoryAdapter {
	return &PostgresRepositoryAdapter{db: db}
}

// ConfigureDBPool tunes the database connection pool using official Go database/sql guidelines.
func ConfigureDBPool(db *sql.DB, maxOpen, maxIdle int, maxLifetime, maxIdleTime time.Duration) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	db.SetConnMaxIdleTime(maxIdleTime)
}

func (a *PostgresRepositoryAdapter) GetByID(ctx context.Context, id string) (*DomainEntity, error) {
	query := `SELECT id, name, version, updated_at FROM entities WHERE id = $1 LIMIT 1`

	var entity DomainEntity
	err := a.db.QueryRowContext(ctx, query, id).Scan(
		&entity.ID,
		&entity.Name,
		&entity.Version,
		&entity.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("entity not found")
		}
		return nil, fmt.Errorf("postgres query failed: %w", err)
	}

	return &entity, nil
}

func (a *PostgresRepositoryAdapter) Save(ctx context.Context, entity *DomainEntity) error {
	query := `
		INSERT INTO entities (id, name, version, updated_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET name = EXCLUDED.name,
		    version = EXCLUDED.version + 1,
		    updated_at = EXCLUDED.updated_at
	`
	_, err := a.db.ExecContext(ctx, query, entity.ID, entity.Name, entity.Version, entity.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres save failed: %w", err)
	}
	return nil
}

// InProcessCacheAdapter simulates an infrastructure cache adapter (e.g. Redis client wrapper).
type InProcessCacheAdapter struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

func NewInProcessCacheAdapter() *InProcessCacheAdapter {
	return &InProcessCacheAdapter{
		items: make(map[string]cacheItem),
	}
}

func (c *InProcessCacheAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists || time.Now().UTC().After(item.expiresAt) {
		return nil, errors.New("cache miss")
	}

	return item.data, nil
}

func (c *InProcessCacheAdapter) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheItem{
		data:      value,
		expiresAt: time.Now().UTC().Add(ttl),
	}
	return nil
}

func (c *InProcessCacheAdapter) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
	return nil
}
