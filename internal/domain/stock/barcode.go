package stock

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type BarcodeEncoding string

const (
	BarcodeEAN13   BarcodeEncoding = "ean13"
	BarcodeEAN8    BarcodeEncoding = "ean8"
	BarcodeUPCA    BarcodeEncoding = "upca"
	BarcodeCode128 BarcodeEncoding = "code128"
	BarcodeGS1     BarcodeEncoding = "gs1_128"
	BarcodeQR      BarcodeEncoding = "qr"
)

type BarcodeRuleType string

const (
	BarcodeRuleProduct  BarcodeRuleType = "product"
	BarcodeRuleWeight   BarcodeRuleType = "weight"
	BarcodeRulePrice    BarcodeRuleType = "price"
	BarcodeRuleCustomer BarcodeRuleType = "customer"
	BarcodeRuleLot      BarcodeRuleType = "lot"
	BarcodeRulePackage  BarcodeRuleType = "package"
	BarcodeRuleLocation BarcodeRuleType = "location"
)

type BarcodeNomenclature struct {
	ID        int64         `json:"id"`
	Name      string        `json:"name"`
	CompanyID int64         `json:"company_id"`
	Rules     []BarcodeRule `json:"rules,omitempty"`
}

type BarcodeRule struct {
	ID             int64           `json:"id"`
	NomenclatureID int64           `json:"nomenclature_id"`
	Name           string          `json:"name"`
	Sequence       int             `json:"sequence"`
	Encoding       BarcodeEncoding `json:"encoding"`
	Type           BarcodeRuleType `json:"type"`
	Pattern        string          `json:"pattern"`
	GS1ContentType string          `json:"gs1_content_type,omitempty"`
	Associated     bool            `json:"associated"`
}

type ParseResult struct {
	Type     BarcodeRuleType `json:"type"`
	Value    string          `json:"value"`
	BaseCode string          `json:"base_code"`
	Quantity *float64        `json:"quantity,omitempty"`
	Price    *float64        `json:"price,omitempty"`
	LotName  string          `json:"lot_name,omitempty"`
}

type BarcodeParser struct{ nomenclature *BarcodeNomenclature }

func NewBarcodeParser(nomenclature *BarcodeNomenclature) *BarcodeParser {
	return &BarcodeParser{nomenclature: nomenclature}
}

func (p *BarcodeParser) Parse(barcode string) (*ParseResult, error) {
	if p == nil || p.nomenclature == nil || barcode == "" {
		return nil, platformerrors.Validation("barcode and nomenclature are required", nil)
	}
	for _, rule := range p.nomenclature.Rules {
		matched, err := regexp.MatchString(rule.Pattern, barcode)
		if err != nil {
			return nil, platformerrors.Validation("invalid barcode rule pattern", nil)
		}
		if matched {
			return &ParseResult{Type: rule.Type, Value: barcode, BaseCode: barcode}, nil
		}
	}
	return nil, platformerrors.NotFound("no barcode rule matched", nil)
}

type GS1Parser struct{}
type GS1Element struct {
	AI    string `json:"ai"`
	Value string `json:"value"`
	Label string `json:"label"`
}

func (g *GS1Parser) ParseGS1(barcode string) ([]GS1Element, error) {
	barcode = strings.ReplaceAll(barcode, "(", "")
	barcode = strings.ReplaceAll(barcode, ")", "")
	if barcode == "" {
		return nil, platformerrors.Validation("GS1 barcode is required", nil)
	}
	var result []GS1Element
	for len(barcode) > 0 {
		if len(barcode) < 2 {
			return nil, platformerrors.Validation("invalid GS1 application identifier", nil)
		}
		ai := barcode[:2]
		barcode = barcode[2:]
		length := 0
		label := ""
		switch ai {
		case "01":
			length, label = 14, "GTIN"
		case "17":
			length, label = 6, "Expiration date"
		case "10":
			label = "Batch/Lot"
		case "21":
			label = "Serial"
		default:
			return nil, fmt.Errorf("unsupported GS1 AI %s", ai)
		}
		if length == 0 {
			separator := strings.IndexByte(barcode, 29)
			if separator < 0 {
				length = len(barcode)
			} else {
				length = separator
			}
		}
		if len(barcode) < length {
			return nil, platformerrors.Validation("truncated GS1 element", nil)
		}
		value := barcode[:length]
		barcode = strings.TrimPrefix(barcode[length:], string(rune(29)))
		result = append(result, GS1Element{AI: ai, Value: value, Label: label})
	}
	return result, nil
}

func ParseWeightedBarcode(value string) (*ParseResult, error) {
	if len(value) != 13 {
		return nil, platformerrors.Validation("weighted barcode must be 13 digits", nil)
	}
	quantity, err := strconv.ParseFloat(value[7:12], 64)
	if err != nil {
		return nil, platformerrors.Validation("invalid weighted barcode quantity", nil)
	}
	return &ParseResult{Type: BarcodeRuleWeight, BaseCode: value[:7], Value: value, Quantity: &quantity}, nil
}
