package xsd

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"cashflow_backend/internal/domain/accounting"
)

type ValidationError struct {
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Message string `json:"message"`
	Code    string `json:"code"`
}
type XSDValidator struct{}

func NewXSDValidator() (*XSDValidator, error) { return &XSDValidator{}, nil }
func (v *XSDValidator) Validate(content []byte, _ accounting.EDIFormat) error {
	decoder := xml.NewDecoder(bytes.NewReader(content))
	depth := 0
	rootSeen := false
	required := map[string]bool{"ID": false, "IssueDate": false, "DocumentCurrencyCode": false, "LegalMonetaryTotal": false}
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				if !rootSeen {
					return fmt.Errorf("UBL document has no root element")
				}
				for field, seen := range required {
					if !seen {
						return fmt.Errorf("UBL document is missing required element %s", field)
					}
				}
				return nil
			}
			return err
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if !rootSeen {
				rootSeen = true
				if value.Name.Local != "Invoice" {
					return fmt.Errorf("UBL root must be Invoice, got %s", value.Name.Local)
				}
			}
			if _, ok := required[value.Name.Local]; ok {
				required[value.Name.Local] = true
			}
		case xml.EndElement:
			depth--
			if depth < 0 {
				return fmt.Errorf("unexpected closing element %s", value.Name.Local)
			}
		}
	}
}
func (v *XSDValidator) ValidateZATCA(content []byte, format accounting.EDITransactionType) []ValidationError {
	if err := v.Validate(content, accounting.EDIFormatZatcaPhase2); err != nil {
		return []ValidationError{{Message: err.Error(), Code: "invalid_xml"}}
	}
	return nil
}
