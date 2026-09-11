package repair

type RepairOrderLine struct {
	ID         int64   `json:"id"`
	RepairID   int64   `json:"repair_id"`
	ProductID  int64   `json:"product_id"`
	Quantity   float64 `json:"quantity"`
	PriceUnit  float64 `json:"price_unit"`
	PriceTotal float64 `json:"price_total"`
}

func (l *RepairOrderLine) CalculateLineTotal() {
	l.PriceTotal = l.Quantity * l.PriceUnit
}

func (l *RepairOrderLine) Validate() error {
	if l.ProductID <= 0 {
		return errInvalid("line product ID is required")
	}
	if l.Quantity <= 0 {
		return errInvalid("line quantity must be greater than zero")
	}
	if l.PriceUnit < 0 {
		return errInvalid("line price unit cannot be negative")
	}
	return nil
}
