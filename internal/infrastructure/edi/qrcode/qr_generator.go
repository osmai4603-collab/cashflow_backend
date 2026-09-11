package qrcode

import (
	"github.com/skip2/go-qrcode"
)

func GenerateQRCode(tlvData []byte, size int) ([]byte, error) {
	if size <= 0 {
		size = 256
	}
	return qrcode.Encode(string(tlvData), qrcode.Medium, size)
}
