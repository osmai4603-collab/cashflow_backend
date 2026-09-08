package stockusecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// BarcodeResult holds the parsed components of a GS1 or standard barcode.
type BarcodeResult struct {
	GTIN     string  `json:"gtin"`
	Lot      string  `json:"lot"`
	Serial   string  `json:"serial"`
	Expiry   string  `json:"expiry"`
	Quantity float64 `json:"quantity"`
}

// ParseBarcode decodes a barcode string using GS1-128 rules or standard fallback.
// Odoo 19 uses barcode nomenclatures for this purpose.
func (uc *UseCase) ParseBarcode(ctx context.Context, barcode string) (*BarcodeResult, error) {
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		return nil, fmt.Errorf("empty barcode")
	}

	// Basic GS1-128 AI (Application Identifier) parsing logic
	// (01) GTIN - 14 digits
	// (10) Lot - variable length (up to 20)
	// (17) Expiry - 6 digits (YYMMDD)
	// (21) Serial - variable length
	// (30) Qty - variable length

	res := &BarcodeResult{Quantity: 1.0}

	// Simplified parser for common AIs
	curr := barcode
	for len(curr) >= 2 {
		ai := curr[:2]
		switch ai {
		case "01": // GTIN
			if len(curr) < 16 {
				return nil, fmt.Errorf("invalid GTIN AI 01 length")
			}
			res.GTIN = curr[2:16]
			curr = curr[16:]
		case "10": // Lot
			// In real GS1, variable fields end with FNC1 (mapped to ASCII 29 in some scanners)
			// Here we assume simple concatenation or end of string.
			res.Lot = curr[2:]
			curr = ""
		case "17": // Expiry
			if len(curr) < 8 {
				return nil, fmt.Errorf("invalid Expiry AI 17 length")
			}
			res.Expiry = curr[2:8]
			curr = curr[8:]
		case "21": // Serial
			res.Serial = curr[2:]
			curr = ""
		case "30": // Quantity
			qtyStr := curr[2:]
			if q, err := strconv.ParseFloat(qtyStr, 64); err == nil {
				res.Quantity = q
			}
			curr = ""
		default:
			// Fallback: entire barcode is a single identifier (e.g. Serial or GTIN)
			if res.GTIN == "" && res.Serial == "" && res.Lot == "" {
				res.GTIN = barcode
			}
			curr = ""
		}
	}

	return res, nil
}
