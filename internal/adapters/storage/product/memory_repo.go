package productstorage

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"cashflow_backend/internal/domain/product"
	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/filter"
	"cashflow_backend/internal/platform/pagination"
)

// MemoryRepo is a thread-safe in-memory implementation of product.Repository.
type MemoryRepo struct {
	mu             sync.RWMutex
	templates      map[int64]*product.ProductTemplate
	variants       map[int64]*product.ProductVariant
	categories     map[int64]*product.ProductCategory
	uoms           map[int64]*product.UnitOfMeasure
	pricelists     map[int64]*product.Pricelist
	pricelistItems map[int64]*product.PricelistItem

	lastTemplateID      int64
	lastVariantID       int64
	lastCategoryID      int64
	lastUoMID           int64
	lastPricelistID     int64
	lastPricelistItemID int64
}

// NewMemoryRepo creates an initialized MemoryRepo with standard seed data.
func NewMemoryRepo() *MemoryRepo {
	r := &MemoryRepo{
		templates:      make(map[int64]*product.ProductTemplate),
		variants:       make(map[int64]*product.ProductVariant),
		categories:     make(map[int64]*product.ProductCategory),
		uoms:           make(map[int64]*product.UnitOfMeasure),
		pricelists:     make(map[int64]*product.Pricelist),
		pricelistItems: make(map[int64]*product.PricelistItem),
	}

	// Seed Standard Units of Measure
	now := time.Now().UTC()
	standardUoMs := []product.UnitOfMeasure{
		{ID: 1, Name: "Units", Category: "unit", Ratio: 1.0, Rounding: 0.001, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Name: "Dozens", Category: "unit", Ratio: 12.0, Rounding: 0.001, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Name: "kg", Category: "weight", Ratio: 1.0, Rounding: 0.001, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Name: "g", Category: "weight", Ratio: 0.001, Rounding: 0.001, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 5, Name: "Liters", Category: "volume", Ratio: 1.0, Rounding: 0.001, Active: true, CreatedAt: now, UpdatedAt: now},
		{ID: 6, Name: "Hours", Category: "time", Ratio: 1.0, Rounding: 0.01, Active: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, u := range standardUoMs {
		clone := u
		r.uoms[u.ID] = &clone
	}
	r.lastUoMID = 6

	// Seed Standard Category: All
	rootCat := product.ProductCategory{
		ID:           1,
		Name:         "All",
		CompleteName: "All",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	r.categories[1] = &rootCat
	r.lastCategoryID = 1

	// Seed Standard Pricelist: Public Pricelist
	pubPricelist := product.Pricelist{
		ID:       1,
		Name:     "Public Pricelist",
		Currency: "USD",
		Active:   true,
	}
	pubPricelist.Audit.CreatedAt = now
	pubPricelist.Audit.UpdatedAt = now
	r.pricelists[1] = &pubPricelist
	r.lastPricelistID = 1

	return r
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Templates
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateTemplate(ctx context.Context, pt *product.ProductTemplate) error {
	if err := pt.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastTemplateID++
	pt.ID = r.lastTemplateID
	pt.Active = true

	now := time.Now().UTC()
	if pt.Audit.CreatedAt.IsZero() {
		pt.Audit.CreatedAt = now
	}
	if pt.Audit.UpdatedAt.IsZero() {
		pt.Audit.UpdatedAt = now
	}

	clone := *pt
	r.templates[pt.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetTemplateByID(ctx context.Context, id int64) (*product.ProductTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pt, exists := r.templates[id]
	if !exists || !pt.Active {
		return nil, platformerrors.NotFound("product template not found")
	}

	clone := *pt
	r.enrichTemplateRelations(&clone)
	return &clone, nil
}

func (r *MemoryRepo) UpdateTemplate(ctx context.Context, pt *product.ProductTemplate) error {
	if err := pt.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.templates[pt.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product template not found")
	}

	pt.Audit.UpdatedAt = time.Now().UTC()
	clone := *pt
	r.templates[pt.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteTemplate(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.templates[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product template not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()

	// Also deactivate associated variants
	for _, v := range r.variants {
		if v.TemplateID == id && v.Active {
			v.Active = false
			v.Audit.UpdatedAt = time.Now().UTC()
		}
	}

	return nil
}

func (r *MemoryRepo) ListTemplates(ctx context.Context, f *filter.Filter, page pagination.PageRequest) (pagination.PageResult[product.ProductTemplate], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*product.ProductTemplate
	for _, pt := range r.templates {
		if !r.matchesTemplateFilter(pt, f) {
			continue
		}
		clone := *pt
		r.enrichTemplateRelations(&clone)
		filtered = append(filtered, &clone)
	}

	// Sorting
	sort.Slice(filtered, func(i, j int) bool {
		switch strings.ToLower(page.SortBy) {
		case "name":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Name > filtered[j].Name
			}
			return filtered[i].Name < filtered[j].Name
		case "sale_price":
			if page.OrderDirection() == "DESC" {
				return filtered[i].SalePrice > filtered[j].SalePrice
			}
			return filtered[i].SalePrice < filtered[j].SalePrice
		case "created_at":
			if page.OrderDirection() == "DESC" {
				return filtered[i].Audit.CreatedAt.After(filtered[j].Audit.CreatedAt)
			}
			return filtered[i].Audit.CreatedAt.Before(filtered[j].Audit.CreatedAt)
		default:
			if page.OrderDirection() == "DESC" {
				return filtered[i].ID > filtered[j].ID
			}
			return filtered[i].ID < filtered[j].ID
		}
	})

	totalItems := int64(len(filtered))
	offset := page.Offset()
	limit := page.LimitClamped()

	var items []product.ProductTemplate
	if offset < len(filtered) {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		for _, pt := range filtered[offset:end] {
			items = append(items, *pt)
		}
	}

	return pagination.NewPageResult(items, totalItems, page), nil
}

func (r *MemoryRepo) enrichTemplateRelations(pt *product.ProductTemplate) {
	if pt.CategoryID != nil {
		if cat, ok := r.categories[*pt.CategoryID]; ok && cat.Active {
			catClone := *cat
			pt.Category = &catClone
		}
	}
	if pt.UoMID != nil {
		if uom, ok := r.uoms[*pt.UoMID]; ok && uom.Active {
			uomClone := *uom
			pt.UoM = &uomClone
		}
	}
}

func (r *MemoryRepo) matchesTemplateFilter(pt *product.ProductTemplate, f *filter.Filter) bool {
	if f == nil || len(f.Criteria) == 0 {
		return pt.Active
	}

	for _, c := range f.Criteria {
		switch strings.ToLower(c.Field) {
		case "active":
			if val, ok := c.Value.(bool); ok && pt.Active != val {
				return false
			}
		case "type":
			if val, ok := c.Value.(string); ok && string(pt.Type) != val {
				return false
			}
		case "sale_ok":
			if val, ok := c.Value.(bool); ok && pt.SaleOK != val {
				return false
			}
		case "purchase_ok":
			if val, ok := c.Value.(bool); ok && pt.PurchaseOK != val {
				return false
			}
		case "category_id":
			if pt.CategoryID == nil {
				return false
			}
			switch v := c.Value.(type) {
			case int64:
				if *pt.CategoryID != v {
					return false
				}
			case int:
				if *pt.CategoryID != int64(v) {
					return false
				}
			case float64:
				if *pt.CategoryID != int64(v) {
					return false
				}
			}
		case "internal_ref":
			if val, ok := c.Value.(string); ok && !strings.EqualFold(pt.InternalRef, val) {
				return false
			}
		case "barcode":
			if val, ok := c.Value.(string); ok && !strings.EqualFold(pt.Barcode, val) {
				return false
			}
		case "name":
			if val, ok := c.Value.(string); ok {
				if c.Operator == filter.OpILike || c.Operator == filter.OpLike {
					if !strings.Contains(strings.ToLower(pt.Name), strings.ToLower(val)) {
						return false
					}
				} else if pt.Name != val {
					return false
				}
			}
		}
	}

	return true
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Variants
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateVariant(ctx context.Context, pv *product.ProductVariant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastVariantID++
	pv.ID = r.lastVariantID
	pv.Active = true

	now := time.Now().UTC()
	if pv.Audit.CreatedAt.IsZero() {
		pv.Audit.CreatedAt = now
	}
	if pv.Audit.UpdatedAt.IsZero() {
		pv.Audit.UpdatedAt = now
	}

	clone := *pv
	r.variants[pv.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetVariantByID(ctx context.Context, id int64) (*product.ProductVariant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pv, exists := r.variants[id]
	if !exists || !pv.Active {
		return nil, platformerrors.NotFound("product variant not found")
	}

	clone := *pv
	return &clone, nil
}

func (r *MemoryRepo) GetVariantsByTemplateID(ctx context.Context, templateID int64) ([]product.ProductVariant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []product.ProductVariant
	for _, pv := range r.variants {
		if pv.TemplateID == templateID && pv.Active {
			result = append(result, *pv)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

func (r *MemoryRepo) UpdateVariant(ctx context.Context, pv *product.ProductVariant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.variants[pv.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product variant not found")
	}

	pv.Audit.UpdatedAt = time.Now().UTC()
	clone := *pv
	r.variants[pv.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteVariant(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.variants[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product variant not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Product Categories
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateCategory(ctx context.Context, c *product.ProductCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastCategoryID++
	c.ID = r.lastCategoryID
	c.Active = true

	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	clone := *c
	r.categories[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetCategoryByID(ctx context.Context, id int64) (*product.ProductCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cat, exists := r.categories[id]
	if !exists || !cat.Active {
		return nil, platformerrors.NotFound("product category not found")
	}

	clone := *cat
	return &clone, nil
}

func (r *MemoryRepo) UpdateCategory(ctx context.Context, c *product.ProductCategory) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.categories[c.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product category not found")
	}

	c.UpdatedAt = time.Now().UTC()
	clone := *c
	r.categories[c.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteCategory(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.categories[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("product category not found")
	}

	// Check if active products rely on this category
	for _, pt := range r.templates {
		if pt.Active && pt.CategoryID != nil && *pt.CategoryID == id {
			return platformerrors.Conflict("cannot delete category referenced by active products")
		}
	}

	existing.Active = false
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListCategories(ctx context.Context) ([]product.ProductCategory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []product.ProductCategory
	for _, c := range r.categories {
		if c.Active {
			result = append(result, *c)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CompleteName < result[j].CompleteName
	})

	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Units of Measure (UoM)
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreateUoM(ctx context.Context, u *product.UnitOfMeasure) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastUoMID++
	u.ID = r.lastUoMID
	u.Active = true

	now := time.Now().UTC()
	u.CreatedAt = now
	u.UpdatedAt = now

	clone := *u
	r.uoms[u.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetUoMByID(ctx context.Context, id int64) (*product.UnitOfMeasure, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, exists := r.uoms[id]
	if !exists || !u.Active {
		return nil, platformerrors.NotFound("unit of measure not found")
	}

	clone := *u
	return &clone, nil
}

func (r *MemoryRepo) UpdateUoM(ctx context.Context, u *product.UnitOfMeasure) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.uoms[u.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("unit of measure not found")
	}

	u.UpdatedAt = time.Now().UTC()
	clone := *u
	r.uoms[u.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeleteUoM(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.uoms[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("unit of measure not found")
	}

	// Check if active products rely on this UoM
	for _, pt := range r.templates {
		if pt.Active && pt.UoMID != nil && *pt.UoMID == id {
			return platformerrors.Conflict("cannot delete unit of measure referenced by active products")
		}
	}

	existing.Active = false
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListUoMs(ctx context.Context) ([]product.UnitOfMeasure, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []product.UnitOfMeasure
	for _, u := range r.uoms {
		if u.Active {
			result = append(result, *u)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Category == result[j].Category {
			return result[i].Ratio < result[j].Ratio
		}
		return result[i].Category < result[j].Category
	})

	return result, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Pricelists
// ─────────────────────────────────────────────────────────────────────────────

func (r *MemoryRepo) CreatePricelist(ctx context.Context, pl *product.Pricelist) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastPricelistID++
	pl.ID = r.lastPricelistID
	pl.Active = true

	now := time.Now().UTC()
	if pl.Audit.CreatedAt.IsZero() {
		pl.Audit.CreatedAt = now
	}
	if pl.Audit.UpdatedAt.IsZero() {
		pl.Audit.UpdatedAt = now
	}

	clone := *pl
	r.pricelists[pl.ID] = &clone
	return nil
}

func (r *MemoryRepo) GetPricelistByID(ctx context.Context, id int64) (*product.Pricelist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pl, exists := r.pricelists[id]
	if !exists || !pl.Active {
		return nil, platformerrors.NotFound("pricelist not found")
	}

	clone := *pl
	// Attach items
	var items []product.PricelistItem
	for _, item := range r.pricelistItems {
		if item.PricelistID == id {
			items = append(items, *item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	clone.Items = items

	return &clone, nil
}

func (r *MemoryRepo) UpdatePricelist(ctx context.Context, pl *product.Pricelist) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.pricelists[pl.ID]
	if !exists || !existing.Active {
		return platformerrors.NotFound("pricelist not found")
	}

	pl.Audit.UpdatedAt = time.Now().UTC()
	clone := *pl
	r.pricelists[pl.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeletePricelist(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.pricelists[id]
	if !exists || !existing.Active {
		return platformerrors.NotFound("pricelist not found")
	}

	existing.Active = false
	existing.Audit.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *MemoryRepo) ListPricelists(ctx context.Context) ([]product.Pricelist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []product.Pricelist
	for _, pl := range r.pricelists {
		if pl.Active {
			clone := *pl
			result = append(result, clone)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result, nil
}

func (r *MemoryRepo) AddPricelistItem(ctx context.Context, item *product.PricelistItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pricelists[item.PricelistID]; !exists {
		return platformerrors.NotFound("pricelist not found")
	}

	r.lastPricelistItemID++
	item.ID = r.lastPricelistItemID

	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now

	clone := *item
	r.pricelistItems[item.ID] = &clone
	return nil
}

func (r *MemoryRepo) DeletePricelistItem(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.pricelistItems[id]; !exists {
		return platformerrors.NotFound("pricelist item not found")
	}

	delete(r.pricelistItems, id)
	return nil
}

func (r *MemoryRepo) GetPricelistItems(ctx context.Context, pricelistID int64) ([]product.PricelistItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var items []product.PricelistItem
	for _, item := range r.pricelistItems {
		if item.PricelistID == pricelistID {
			items = append(items, *item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	return items, nil
}
