package stock

import (
	platformerrors "cashflow_backend/internal/platform/errors"
	"context"
)

type ProductPackaging struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	ProductID     int64   `json:"product_id"`
	Barcode       string  `json:"barcode,omitempty"`
	Qty           float64 `json:"qty"`
	PackageTypeID *int64  `json:"package_type_id,omitempty"`
	CompanyID     int64   `json:"company_id"`
	Active        bool    `json:"active"`
}

func (p *ProductPackaging) Validate() error {
	if p.Name == "" || p.ProductID <= 0 || p.Qty <= 0 {
		return platformerrors.Validation("packaging name, product and positive quantity are required", nil)
	}
	return nil
}

type StockPackageType struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Height    float64 `json:"height"`
	Width     float64 `json:"width"`
	Length    float64 `json:"length"`
	MaxWeight float64 `json:"max_weight"`
	Barcode   string  `json:"barcode,omitempty"`
	Sequence  int     `json:"sequence"`
	CompanyID *int64  `json:"company_id,omitempty"`
}

type StockPackage struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	PackageTypeID *int64  `json:"package_type_id,omitempty"`
	LocationID    int64   `json:"location_id"`
	CompanyID     int64   `json:"company_id"`
	Weight        float64 `json:"weight"`
}

func (p *StockPackage) Validate() error {
	if p.Name == "" || p.LocationID <= 0 || p.Weight < 0 {
		return platformerrors.Validation("package name and location are required and weight cannot be negative", nil)
	}
	return nil
}

type PackagingRepository interface {
	CreatePackaging(context.Context, *ProductPackaging) error
	GetPackagingByID(context.Context, int64) (*ProductPackaging, error)
	CreatePackageType(context.Context, *StockPackageType) error
	GetPackageTypeByID(context.Context, int64) (*StockPackageType, error)
	CreatePackage(context.Context, *StockPackage) error
	GetPackageByID(context.Context, int64) (*StockPackage, error)
}
