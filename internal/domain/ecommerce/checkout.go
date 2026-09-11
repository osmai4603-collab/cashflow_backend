package ecommerce

// CheckoutRequest holds data for the final step of the eCommerce flow.
type CheckoutRequest struct {
	CartID            int64  `json:"cart_id"`
	ShippingAddressID int64  `json:"shipping_address_id"`
	InvoiceAddressID  int64  `json:"invoice_address_id"`
	DeliveryMethodID  int64  `json:"delivery_method_id"`
	PaymentMethodID   int64  `json:"payment_method_id"`
	PaymentToken      string `json:"payment_token,omitempty"` // For external providers
}

// CheckoutResult contains the outcome of a successful checkout.
type CheckoutResult struct {
	OrderID       int64  `json:"order_id"`
	OrderNumber   string `json:"order_number"`
	TransactionID string `json:"transaction_id,omitempty"`
	RedirectURL   string `json:"redirect_url,omitempty"` // For 3D Secure or Hosted payment
}
