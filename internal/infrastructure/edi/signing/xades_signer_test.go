package signing

import (
	"cashflow_backend/internal/domain/accounting"
	"testing"
)

func TestXAdESSignAndVerify(t *testing.T) {
	certificate, privateKey, err := GenerateTestCertificate()
	if err != nil {
		t.Fatal(err)
	}
	signed, err := (&XAdESSigner{}).Sign([]byte(`<Invoice><cbc>ID</cbc></Invoice>`), &accounting.EDICertificate{CertContent: certificate, PrivateKey: privateKey})
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(signed, certificate); err != nil {
		t.Fatal(err)
	}
}
