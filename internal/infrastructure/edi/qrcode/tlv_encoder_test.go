package qrcode

import "testing"

func TestEncodeTLV(t *testing.T) {
	data, err := EncodeTLV(map[TLVTag]string{TLVVATNumber: "123", TLVSellerName: "Seller"})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 13 || data[0] != byte(TLVSellerName) {
		t.Fatalf("unexpected TLV %v", data)
	}
}
