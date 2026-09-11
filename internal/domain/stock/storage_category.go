package stock

import platformerrors "cashflow_backend/internal/platform/errors"

type StorageCategory struct {
	ID              int64                     `json:"id"`
	Name            string                    `json:"name"`
	MaxWeight       float64                   `json:"max_weight"`
	AllowNewProduct string                    `json:"allow_new_product"`
	CompanyID       int64                     `json:"company_id"`
	CapacityIDs     []StorageCategoryCapacity `json:"capacities,omitempty"`
}

type StorageCategoryCapacity struct {
	ID                int64 `json:"id"`
	StorageCategoryID int64 `json:"storage_category_id"`
	PackageTypeID     int64 `json:"package_type_id"`
	Quantity          int   `json:"quantity"`
}

type PutawayRule struct {
	ID                int64  `json:"id"`
	ProductID         *int64 `json:"product_id,omitempty"`
	CategoryID        *int64 `json:"category_id,omitempty"`
	LocationInID      int64  `json:"location_in_id"`
	LocationOutID     int64  `json:"location_out_id"`
	StorageCategoryID *int64 `json:"storage_category_id,omitempty"`
	Sequence          int    `json:"sequence"`
	CompanyID         int64  `json:"company_id"`
	Active            bool   `json:"active"`
}

func (c *StorageCategory) Validate() error {
	if c.Name == "" || c.MaxWeight < 0 {
		return platformerrors.Validation("storage category name and non-negative max weight are required", nil)
	}
	switch c.AllowNewProduct {
	case "", "same", "mixed":
	default:
		return platformerrors.Validation("invalid storage category product policy", nil)
	}
	return nil
}

func (r *PutawayRule) Validate() error {
	if r.LocationInID <= 0 || r.LocationOutID <= 0 || r.LocationInID == r.LocationOutID {
		return platformerrors.Validation("putaway locations must be valid and different", nil)
	}
	return nil
}
