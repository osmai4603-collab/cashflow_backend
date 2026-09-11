package hash

import "testing"

func TestComputeInvoiceHash(t *testing.T) {
	first, err := ComputeInvoiceHash([]byte(`<Invoice/>`))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := ComputeInvoiceHash([]byte(`<Invoice/>`))
	if first != second || first == "" {
		t.Fatal("expected deterministic invoice hash")
	}
}
