package ecommercestorage

import (
	"context"
	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/platform/pagination"
	"sync"
)

type MemoryRepo struct {
	mu    sync.RWMutex
	carts map[int64]*ecommerce.Cart
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		carts: make(map[int64]*ecommerce.Cart),
	}
}

func (r *MemoryRepo) ListCategories(ctx context.Context, websiteID int64) ([]ecommerce.CategoryView, error) {
	return nil, nil
}

func (r *MemoryRepo) ListProducts(ctx context.Context, websiteID int64, filter ecommerce.CatalogFilter, page pagination.PageRequest) (pagination.PageResult[ecommerce.ProductWebView], error) {
	return pagination.PageResult[ecommerce.ProductWebView]{}, nil
}

func (r *MemoryRepo) GetProductBySlug(ctx context.Context, websiteID int64, slug string) (*ecommerce.ProductWebView, error) {
	return nil, nil
}

func (r *MemoryRepo) GetCartBySession(ctx context.Context, websiteID int64, sessionUUID string) (*ecommerce.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.carts {
		if c.WebsiteID == websiteID && c.SessionUUID == sessionUUID {
			return c, nil
		}
	}
	return nil, nil
}

func (r *MemoryRepo) GetCartByID(ctx context.Context, id int64) (*ecommerce.Cart, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.carts[id], nil
}

func (r *MemoryRepo) SaveCart(ctx context.Context, cart *ecommerce.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.carts[cart.ID] = cart
	return nil
}

func (r *MemoryRepo) DeleteCart(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.carts, id)
	return nil
}

func (r *MemoryRepo) CreateSaleOrderFromCart(ctx context.Context, cartID int64) (int64, string, error) {
	return 0, "", nil
}
