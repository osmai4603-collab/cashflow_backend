package config

import (
	"fmt"
	"os"
	"reflect"
)

func originalLookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// LoadEnvironment applies environment variables onto cfg using the env struct
// tags, mirroring Mattermost's reflection-based MM_ override. Canonical names
// come first, then legacy aliases (e.g. PGHOST for DB_HOST). lookup defaults to
// os.LookupEnv.
func LoadEnvironment(cfg *Configuration, lookup func(string) (string, bool)) error {
	if lookup == nil {
		lookup = originalLookupEnv
	}
	var errs []error
	IterateFields(cfg, func(_ string, field reflect.StructField, value reflect.Value) {
		name := EnvTag(field)
		if name == "" {
			return
		}
		raw, ok := EnvValueFor(lookup, name)
		if !ok {
			return
		}
		if err := SetEnvValue(value, name, raw); err != nil {
			errs = append(errs, err)
		}
	})
	return joinErrs(errs)
}

func joinErrs(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("environment overrides failed: %w", errs[0])
}