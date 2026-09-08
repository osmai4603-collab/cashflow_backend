package stock

import (
	"fmt"
	"strings"
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
	"cashflow_backend/internal/platform/i18n"
)

// LocationUsage defines the categorization of a stock location (stock.location usage in Odoo).
type LocationUsage string

const (
	LocationUsageInternal   LocationUsage = "internal"   // Physical storage inside company
	LocationUsageSupplier   LocationUsage = "supplier"   // Vendor location (source for incoming shipments)
	LocationUsageCustomer   LocationUsage = "customer"   // Customer location (dest for outgoing shipments)
	LocationUsageInventory  LocationUsage = "inventory"  // Virtual location for inventory loss/adjustment/scrap
	LocationUsageTransit    LocationUsage = "transit"    // Virtual location for inter-warehouse transit
	LocationUsageProduction LocationUsage = "production" // Virtual location for manufacturing
	LocationUsageView       LocationUsage = "view"       // Non-locatable folder/grouping node
)

// StockLocation represents a storage location or virtual stock node (stock.location in Odoo).
type StockLocation struct {
	ID             int64         `json:"id"`
	Name           i18n.TranslationString        `json:"name"`
	CompleteName   i18n.TranslationString        `json:"complete_name"`
	Usage          LocationUsage `json:"usage"`
	ParentID       *int64        `json:"parent_id,omitempty"`
	ScrapLocation  bool          `json:"scrap_location"`
	ReturnLocation bool          `json:"return_location"`
	CompanyID      *int64        `json:"company_id,omitempty"`
	Active         bool          `json:"active"`
	// ValuationAccountID defines the valuation boundary for this location (stock.location.valuation_account_id in Odoo).
	ValuationAccountID *int64 `json:"valuation_account_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          *int64    `json:"created_by,omitempty"`
	UpdatedBy          *int64    `json:"updated_by,omitempty"`
}

// Validate checks business invariants for StockLocation.
func (l *StockLocation) Validate() error {
	name := strings.TrimSpace(string(l.Name))
	if name == "" {
		return platformerrors.Validation("location name is required", map[string]string{
			"name": "cannot be empty",
		})
	}
	l.Name = i18n.NewTranslation(name)

	if l.Usage == "" {
		l.Usage = LocationUsageInternal
	}

	switch l.Usage {
	case LocationUsageInternal, LocationUsageSupplier, LocationUsageCustomer,
		LocationUsageInventory, LocationUsageTransit, LocationUsageProduction, LocationUsageView:
		// Valid usage
	default:
		return platformerrors.Validation("invalid location usage", map[string]string{
			"usage": fmt.Sprintf("must be one of [%s, %s, %s, %s, %s, %s, %s]",
				LocationUsageInternal, LocationUsageSupplier, LocationUsageCustomer,
				LocationUsageInventory, LocationUsageTransit, LocationUsageProduction, LocationUsageView),
		})
	}

	if l.ParentID != nil && l.ID > 0 && *l.ParentID == l.ID {
		return platformerrors.Validation("circular reference in location hierarchy", map[string]string{
			"parent_id": "location cannot be its own parent",
		})
	}

	if len(l.CompleteName) == 0 {
		l.CompleteName = l.Name
	}

	return nil
}

// ComputeCompleteName updates CompleteName based on the parent location's CompleteName.
func (l *StockLocation) ComputeCompleteName(parentCompleteName string) {
	parentCompleteName = strings.TrimSpace(parentCompleteName)
	if parentCompleteName != "" {
		l.CompleteName = i18n.NewTranslation(parentCompleteName + "/" + l.Name.Get(i18n.DefaultLang))
	} else {
		l.CompleteName = l.Name
	}
}
