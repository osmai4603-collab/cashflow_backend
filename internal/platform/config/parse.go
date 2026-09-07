package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// setField assigns a string value onto a struct field based on its Go type,
// mirroring Odoo's config parsing (bool/int/float/duration/text/comma lists).
func setField(field reflect.Value, raw string) error {
	if !field.CanSet() {
		return fmt.Errorf("field not settable")
	}
	// time.Duration is an int64-backed named type; handle before generic ints so
	// "5s"/"2m" values are accepted alongside bare numeric seconds.
	if field.Type() == reflect.TypeOf(time.Duration(0)) {
		return setDuration(field, raw)
	}
	raw = strings.TrimSpace(raw)
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
		return nil
	case reflect.Bool:
		b, err := parseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid int %q", raw)
		}
		field.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid uint %q", raw)
		}
		field.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float %q", raw)
		}
		field.SetFloat(f)
		return nil
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			field.Set(reflect.ValueOf(parseSlice(raw)))
			return nil
		}
		return fmt.Errorf("unsupported slice element type %s", field.Type())
	}
	return fmt.Errorf("unsupported field type %s", field.Type())
}

// SetEnvValue assigns a raw environment string onto a reflected field,
// reporting an error wrapped with the env name when coercion fails.
func SetEnvValue(field reflect.Value, envName, raw string) error {
	if err := setField(field, raw); err != nil {
		return fmt.Errorf("env %s: %w", envName, err)
	}
	return nil
}

func setDuration(field reflect.Value, raw string) error {
	d, err := time.ParseDuration(raw)
	if err == nil {
		field.SetInt(int64(d))
		return nil
	}
	// Fallback for bare numeric seconds, matching the legacy *_SECONDS env vars.
	secs, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid duration %q", raw)
	}
	field.SetInt(secs * int64(time.Second))
	return nil
}

func parseBool(raw string) (bool, error) {
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off", "":
		return false, nil
	}
	return false, fmt.Errorf("invalid bool %q", raw)
}

// parseSlice splits a comma-separated list, trimming entries and dropping empties.
// It returns nil when nothing remains.
func parseSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}