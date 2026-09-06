package partner_test

import (
	"strings"
	"testing"

	"cashflow_backend/internal/domain/partner"
)

func TestPartner_Validate(t *testing.T) {
	tests := []struct {
		name    string
		partner partner.Partner
		wantErr bool
		errKey  string
	}{
		{
			name: "valid individual partner",
			partner: partner.Partner{
				Name:       "John Doe",
				Email:      "john@example.com",
				Type:       partner.PartnerTypeIndividual,
				IsCustomer: true,
			},
			wantErr: false,
		},
		{
			name: "valid company partner",
			partner: partner.Partner{
				Name:       "Acme Corp",
				Email:      "CONTACT@ACME.COM",
				Type:       partner.PartnerTypeCompany,
				IsSupplier: true,
			},
			wantErr: false,
		},
		{
			name: "empty name fails",
			partner: partner.Partner{
				Name: "   ",
				Type: partner.PartnerTypeIndividual,
			},
			wantErr: true,
			errKey:  "name",
		},
		{
			name: "name exceeding 255 characters fails",
			partner: partner.Partner{
				Name: strings.Repeat("A", 256),
				Type: partner.PartnerTypeIndividual,
			},
			wantErr: true,
			errKey:  "name",
		},
		{
			name: "invalid partner type fails",
			partner: partner.Partner{
				Name: "Test Partner",
				Type: "alien",
			},
			wantErr: true,
			errKey:  "type",
		},
		{
			name: "invalid email format fails",
			partner: partner.Partner{
				Name:  "Test Partner",
				Email: "not-an-email",
				Type:  partner.PartnerTypeIndividual,
			},
			wantErr: true,
			errKey:  "email",
		},
		{
			name: "self-referential parent_id fails",
			partner: partner.Partner{
				ID:       10,
				Name:     "Test Partner",
				Type:     partner.PartnerTypeCompany,
				ParentID: ptr(int64(10)),
			},
			wantErr: true,
			errKey:  "parent_id",
		},
		{
			name: "different parent_id succeeds",
			partner: partner.Partner{
				ID:       10,
				Name:     "Branch Office",
				Type:     partner.PartnerTypeCompany,
				ParentID: ptr(int64(5)),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.partner.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPartner_EmailNormalization(t *testing.T) {
	p := partner.Partner{
		Name:  " Jane Doe ",
		Email: " JANE.DOE@Example.COM ",
		Type:  partner.PartnerTypeIndividual,
	}

	if err := p.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	if p.Name != "Jane Doe" {
		t.Errorf("expected trimmed name 'Jane Doe', got %q", p.Name)
	}

	if p.Email != "jane.doe@example.com" {
		t.Errorf("expected lowercase trimmed email 'jane.doe@example.com', got %q", p.Email)
	}
}

func ptr[T any](v T) *T {
	return &v
}
