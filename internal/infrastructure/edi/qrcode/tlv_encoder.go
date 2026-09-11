package qrcode

import (
	"encoding/base64"
	"fmt"
	"sort"
)

type TLVTag int

const (
	TLVSellerName     TLVTag = 1
	TLVVATNumber      TLVTag = 2
	TLVTimestamp      TLVTag = 3
	TLVTotalWithVAT   TLVTag = 4
	TLVVATAmount      TLVTag = 5
	TLVInvoiceHash    TLVTag = 6
	TLVECDSASignature TLVTag = 7
	TLVECDSAPublicKey TLVTag = 8
	TLVCertSignature  TLVTag = 9
)

func EncodeTLV(fields map[TLVTag]string) ([]byte, error) {
	keys := make([]int, 0, len(fields))
	for tag, value := range fields {
		if tag < 1 || tag > 255 || len([]byte(value)) > 255 {
			return nil, fmt.Errorf("invalid TLV field %d", tag)
		}
		keys = append(keys, int(tag))
	}
	sort.Ints(keys)
	result := make([]byte, 0)
	for _, key := range keys {
		value := []byte(fields[TLVTag(key)])
		result = append(result, byte(key), byte(len(value)))
		result = append(result, value...)
	}
	return result, nil
}

func EncodeTLVBase64(fields map[TLVTag]string) (string, error) {
	data, err := EncodeTLV(fields)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}
