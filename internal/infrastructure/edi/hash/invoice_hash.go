package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
)

func ComputeInvoiceHash(xmlContent []byte) (string, error) {
	var document any
	if err := xml.Unmarshal(xmlContent, &document); err != nil {
		return "", err
	}
	digest := sha256.Sum256(xmlContent)
	return hex.EncodeToString(digest[:]), nil
}

func ComputeInvoiceHashBase64(xmlContent []byte) (string, error) {
	digest, err := ComputeInvoiceHash(xmlContent)
	if err != nil {
		return "", err
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(decoded), nil
}
