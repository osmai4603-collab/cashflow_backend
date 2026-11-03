package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
)

// =============================================================================
// Resilient Duration Type
// =============================================================================

// Duration wraps time.Duration to provide flexible JSON unmarshaling supporting
// both human-readable strings (e.g., "5s", "10m", "1h") and raw numeric seconds.
type Duration time.Duration

// Duration returns the standard time.Duration representation.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// String returns the duration formatted as a string.
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON serializes the duration as a standard Go duration string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON parses both human duration strings and numeric seconds.
func (d *Duration) UnmarshalJSON(b []byte) error {
	raw, err := decodeJSONValue(b)
	if err != nil {
		return err
	}

	switch val := raw.(type) {
	case nil:
		*d = Duration(0)
		return nil
	case string:
		if val == "" {
			*d = Duration(0)
			return nil
		}
		parsed, err := time.ParseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid duration string %q: %w", val, err)
		}
		*d = Duration(parsed)
		return nil
	case json.Number:
		seconds, err := decimal.NewFromString(val.String())
		if err != nil {
			return fmt.Errorf("invalid duration seconds %q: %w", val, err)
		}
		nanoseconds := seconds.Mul(decimal.NewFromInt(int64(time.Second))).BigInt()
		if !nanoseconds.IsInt64() {
			return fmt.Errorf("duration seconds %q exceed supported range", val)
		}
		*d = Duration(time.Duration(nanoseconds.Int64()))
		return nil
	default:
		return errors.New("duration must be either a string (e.g. '5s') or a numeric count of seconds")
	}
}

// =============================================================================
// Self-Masking Secret String Type
// =============================================================================

// SecretString encapsulates sensitive strings (passwords, tokens, keys) and prevents
// unintentional plaintext exposure during fmt printing or JSON serialization.
type SecretString string

// String returns masked asterisks instead of the raw sensitive content.
func (s SecretString) String() string {
	if s == "" {
		return ""
	}
	return "***"
}

// MarshalJSON serializes the string as masked asterisks.
func (s SecretString) MarshalJSON() ([]byte, error) {
	if s == "" {
		return []byte(`""`), nil
	}
	return []byte(`"***"`), nil
}

// Expose explicitly returns the raw underlying secret string when needed for operations.
func (s SecretString) Expose() string {
	return string(s)
}

// =============================================================================
// Resilient ByteSize Type
// =============================================================================

// ByteSize represents a quantity of data in bytes with flexible JSON unmarshaling.
type ByteSize int64

const (
	Byte     ByteSize = 1
	Kilobyte          = 1024 * Byte
	Megabyte          = 1024 * Kilobyte
	Gigabyte          = 1024 * Megabyte
	Terabyte          = 1024 * Gigabyte
)

// Bytes returns the size in raw integer bytes.
func (b ByteSize) Bytes() int64 {
	return int64(b)
}

func (b ByteSize) String() string {
	switch {
	case b >= Terabyte && b%Terabyte == 0:
		return fmt.Sprintf("%dTB", b/Terabyte)
	case b >= Gigabyte && b%Gigabyte == 0:
		return fmt.Sprintf("%dGB", b/Gigabyte)
	case b >= Megabyte && b%Megabyte == 0:
		return fmt.Sprintf("%dMB", b/Megabyte)
	case b >= Kilobyte && b%Kilobyte == 0:
		return fmt.Sprintf("%dKB", b/Kilobyte)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// MarshalJSON formats size as a human-readable string (e.g. "512MB").
func (b ByteSize) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

// UnmarshalJSON parses both human size strings ("512MB", "1GB") and raw numeric byte counts.
func (b *ByteSize) UnmarshalJSON(data []byte) error {
	v, err := decodeJSONValue(data)
	if err != nil {
		return err
	}

	switch val := v.(type) {
	case nil:
		*b = 0
		return nil
	case json.Number:
		num, err := decimal.NewFromString(val.String())
		if err != nil {
			return fmt.Errorf("invalid numeric byte size %q: %w", val, err)
		}
		size := num.BigInt()
		if !size.IsInt64() {
			return fmt.Errorf("byte size %q exceeds supported range", val)
		}
		*b = ByteSize(size.Int64())
		return nil
	case string:
		parsed, err := ParseByteSize(val)
		if err != nil {
			return err
		}
		*b = parsed
		return nil
	default:
		return errors.New("size must be either a string (e.g. '512MB') or raw numeric bytes")
	}
}

func decodeJSONValue(data []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("unexpected trailing JSON value")
		}
		return nil, err
	}

	return value, nil
}

// ParseByteSize parses a human-readable size string into ByteSize.
func ParseByteSize(s string) (ByteSize, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	i := 0
	for i < len(s) && (unicode.IsDigit(rune(s[i])) || s[i] == '.') {
		i++
	}

	numStr := strings.TrimSpace(s[:i])
	unitStr := strings.ToUpper(strings.TrimSpace(s[i:]))

	num, err := decimal.NewFromString(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric portion %q: %w", numStr, err)
	}

	var multiplier ByteSize
	switch unitStr {
	case "", "B", "BYTES":
		multiplier = Byte
	case "K", "KB", "KIB":
		multiplier = Kilobyte
	case "M", "MB", "MIB":
		multiplier = Megabyte
	case "G", "GB", "GIB":
		multiplier = Gigabyte
	case "T", "TB", "TIB":
		multiplier = Terabyte
	default:
		return 0, fmt.Errorf("unknown size unit %q (supported: B, KB, MB, GB, TB)", unitStr)
	}

	size := num.Mul(decimal.NewFromInt(int64(multiplier))).BigInt()
	if !size.IsInt64() {
		return 0, fmt.Errorf("size %q exceeds supported range", s)
	}
	return ByteSize(size.Int64()), nil
}
