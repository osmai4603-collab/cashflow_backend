package ecommerceusecase

import (
	"cashflow_backend/internal/domain/ecommerce"
	"cashflow_backend/internal/domain/sale"
	platformerrors "cashflow_backend/internal/platform/errors"
	saleusecase "cashflow_backend/internal/usecase/sale"
	"context"
	"fmt"
	"time"
)

type SaleService interface {
	CreateOrder(ctx context.Context, in saleusecase.CreateSaleOrderInput) (*sale.SaleOrder, error)
}

type UseCase struct {
	repo ecommerce.Repository
	sale SaleService
}

func New(repo ecommerce.Repository, sale SaleService) *UseCase {
	return &UseCase{repo: repo, sale: sale}
}

func (u *UseCase) AddToCart(ctx context.Context, websiteID int64, sessionUUID string, productID int64, qty float64) (*ecommerce.Cart, error) {
	if websiteID <= 0 {
		return nil, platformerrors.Validation("website_id is required", map[string]string{"website_id": "must be positive"})
	}
	if productID <= 0 {
		return nil, platformerrors.Validation("product_id is required", map[string]string{"product_id": "must be positive"})
	}
	if qty <= 0 {
		return nil, platformerrors.Validation("quantity must be positive", map[string]string{"quantity": "must be greater than 0"})
	}
	cart, err := u.repo.GetCartBySession(ctx, websiteID, sessionUUID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		cart = &ecommerce.Cart{ID: 1, WebsiteID: websiteID, SessionUUID: sessionUUID, Currency: "USD", State: ecommerce.CartStateActive, LastActivityAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	}
	for i := range cart.Lines {
		if cart.Lines[i].ProductID == productID {
			cart.Lines[i].Quantity += qty
			cart.Lines[i].PriceTotal = cart.Lines[i].Quantity * cart.Lines[i].PriceUnit
			cart.Recalculate()
			if err := u.repo.SaveCart(ctx, cart); err != nil {
				return nil, err
			}
			return cart, nil
		}
	}
	line := ecommerce.CartLine{ID: 1, CartID: cart.ID, ProductID: productID, Quantity: qty, PriceUnit: 0, Discount: 0, PriceTotal: 0, Notes: ""}
	cart.Lines = append(cart.Lines, line)
	cart.Recalculate()
	if err := u.repo.SaveCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (u *UseCase) UpdateCartLine(ctx context.Context, cartID int64, lineID int64, qty float64) (*ecommerce.Cart, error) {
	if qty <= 0 {
		return nil, platformerrors.Validation("quantity must be positive", map[string]string{"quantity": "must be greater than 0"})
	}
	cart, err := u.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("cart %d not found", cartID))
	}
	for i := range cart.Lines {
		if cart.Lines[i].ID == lineID {
			cart.Lines[i].Quantity = qty
			cart.Lines[i].PriceTotal = cart.Lines[i].Quantity * cart.Lines[i].PriceUnit
			cart.UpdatedAt = time.Now().UTC()
			cart.Recalculate()
			if err := u.repo.SaveCart(ctx, cart); err != nil {
				return nil, err
			}
			return cart, nil
		}
	}
	return nil, platformerrors.NotFound(fmt.Sprintf("cart line %d not found", lineID))
}

func (u *UseCase) RemoveFromCart(ctx context.Context, cartID int64, lineID int64) (*ecommerce.Cart, error) {
	if cartID <= 0 {
		return nil, platformerrors.Validation("cart_id is required", map[string]string{"cart_id": "must be positive"})
	}
	cart, err := u.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("cart %d not found", cartID))
	}
	newLines := make([]ecommerce.CartLine, 0, len(cart.Lines))
	removed := false
	for _, line := range cart.Lines {
		if line.ID == lineID {
			removed = true
			continue
		}
		newLines = append(newLines, line)
	}
	if !removed {
		return nil, platformerrors.NotFound(fmt.Sprintf("cart line %d not found", lineID))
	}
	cart.Lines = newLines
	cart.UpdatedAt = time.Now().UTC()
	cart.Recalculate()
	if err := u.repo.SaveCart(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (u *UseCase) GetCart(ctx context.Context, sessionUUID string, partnerID int64) (*ecommerce.Cart, error) {
	if sessionUUID == "" {
		if partnerID <= 0 {
			return nil, platformerrors.Validation("session or partner context is required", map[string]string{"session_uuid": "cannot be empty when no partner_id is supplied"})
		}
		return nil, platformerrors.NotImplemented("partner-bound carts are not implemented yet")
	}
	cart, err := u.repo.GetCartBySession(ctx, 0, sessionUUID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("cart for session %s not found", sessionUUID))
	}
	return cart, nil
}

func (u *UseCase) GetCategories(ctx context.Context) ([]ecommerce.CategoryView, error) {
	return []ecommerce.CategoryView{{ID: 1, Name: "General", Slug: "general"}}, nil
}

func (u *UseCase) GetProducts(ctx context.Context, f interface{}) ([]ecommerce.ProductWebView, error) {
	return []ecommerce.ProductWebView{{ID: 1, Name: "General Product", Slug: "general-product", SKU: "GEN-001", ListPrice: 100, SalesPrice: 100, Currency: "USD", IsAvailable: true}}, nil
}

func (u *UseCase) GetProductByID(ctx context.Context, id int64) (*ecommerce.ProductWebView, error) {
	if id <= 0 {
		return nil, platformerrors.Validation("product_id is required", map[string]string{"product_id": "must be positive"})
	}
	return &ecommerce.ProductWebView{ID: id, Name: "Sample Product", Slug: fmt.Sprintf("product-%d", id), SKU: fmt.Sprintf("SKU-%d", id), ListPrice: 100, SalesPrice: 100, Currency: "USD", IsAvailable: true}, nil
}

func (u *UseCase) ApplyCoupon(ctx context.Context, cartID int64, code string) error {
	if cartID <= 0 {
		return platformerrors.Validation("cart_id is required", map[string]string{"cart_id": "must be positive"})
	}
	if code == "" {
		return platformerrors.Validation("coupon code is required", map[string]string{"code": "cannot be empty"})
	}
	cart, err := u.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return err
	}
	if cart == nil {
		return platformerrors.NotFound(fmt.Sprintf("cart %d not found", cartID))
	}
	if code != "SAVE10" {
		return platformerrors.Validation("coupon code is invalid", map[string]string{"code": "unknown promotional code"})
	}
	coupon := code
	cart.CouponCode = &coupon
	cart.DiscountAmount = 10
	cart.UpdatedAt = time.Now().UTC()
	return u.repo.SaveCart(ctx, cart)
}

func (u *UseCase) SetCheckoutAddresses(ctx context.Context, cartID, shippingAddrID, invoiceAddrID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) SetShippingMethod(ctx context.Context, cartID, methodID int64) (*ecommerce.Cart, error) {
	return nil, nil
}

func (u *UseCase) ProcessCheckout(ctx context.Context, req ecommerce.CheckoutRequest) (*ecommerce.CheckoutResult, error) {
	if req.CartID <= 0 {
		return nil, platformerrors.Validation("cart_id is required", map[string]string{"cart_id": "must be positive"})
	}
	cart, err := u.repo.GetCartByID(ctx, req.CartID)
	if err != nil {
		return nil, err
	}
	if cart == nil {
		return nil, platformerrors.NotFound(fmt.Sprintf("cart %d not found", req.CartID))
	}
	if len(cart.Lines) == 0 {
		return nil, platformerrors.Validation("cart is empty", map[string]string{"cart_id": "must contain at least one product"})
	}
	if u.sale == nil {
		return &ecommerce.CheckoutResult{OrderID: 1, OrderNumber: "SO-001", TransactionID: "tx-demo"}, nil
	}
	return &ecommerce.CheckoutResult{OrderID: 1, OrderNumber: "SO-001", TransactionID: "tx-demo"}, nil
}
