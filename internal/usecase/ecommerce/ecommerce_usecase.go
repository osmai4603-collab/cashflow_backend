package ecommerceusecase

import (
	"context"
	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/domain/sale"
)

type SaleService interface {
	CreateOrder(ctx context.Context, order *sale.SaleOrder) error
}

type UseCase struct {
	repo ecommerce.Repository
	sale SaleService
}

func New(repo ecommerce.Repository, sale SaleService) *UseCase {
	return &UseCase{repo: repo, sale: sale}
}

func (u *UseCase) AddToCart(ctx context.Context, websiteID int64, sessionUUID string, productID int64, qty float64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) UpdateCartLine(ctx context.Context, cartID int64, lineID int64, qty float64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) RemoveFromCart(ctx context.Context, cartID int64, lineID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) GetCart(ctx context.Context, sessionUUID string, partnerID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) GetCategories(ctx context.Context) ([]ecommerce.Category, error) {
	return nil, nil
}

func (u *UseCase) GetProducts(ctx context.Context, f interface{}) ([]ecommerce.Product, error) {
	return nil, nil
}

func (u *UseCase) GetProductByID(ctx context.Context, id int64) (*ecommerce.Product, error) {
	return nil, nil
}

func (u *UseCase) ApplyCoupon(ctx context.Context, cartID int64, code string) error {
	return nil
}

func (u *UseCase) SetCheckoutAddresses(ctx context.Context, cartID, shippingAddrID, invoiceAddrID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) SetShippingMethod(ctx context.Context, cartID, methodID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) ProcessCheckout(ctx context.Context, req ecommerce.CheckoutRequest) (*ecommerce.CheckoutResult, error) {
	return nil, nil
}
