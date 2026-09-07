package config

import (
	"reflect"
	"testing"
)

func TestDefaults_NoNilsAndValid(t *testing.T) {
	cfg := Defaults()
	opts := Options(cfg)
	if len(opts) < 20 {
		t.Fatalf("expected a rich option registry, got %d options", len(opts))
	}
	if _, ok := opts["DB_HOST"]; !ok {
		t.Error("expected DB_HOST in option registry")
	}
	if _, ok := opts["PORT"]; !ok {
		t.Error("expected PORT in option registry")
	}
}

func TestIterateFields_CoversAllSections(t *testing.T) {
	cfg := Defaults()
	var sections []string
	IterateFields(cfg, func(section string, field reflect.StructField, value reflect.Value) {
		sections = append(sections, section)
	})
	want := map[string]bool{
		"Server": true, "Database": true, "App": true, "Cache": true,
		"Auth": true, "Security": true, "Log": true, "Email": true,
		"Worker": true, "Limit": true, "Runtime": true, "Feature": true,
		"Test": true, "Transient": true, "GeoIP": true, "I18n": true, "WebSocket": true,
	}
	for s := range want {
		if !contains(sections, s) {
			t.Errorf("section %s missing from env iteration", s)
		}
	}
}

func TestEnvValueFor_Aliases(t *testing.T) {
	env := map[string]string{
		"PGHOST":  "pg-replica",
		"DB_PORT": "6000",
	}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}

	// Canonical name wins when both present.
	env["DB_HOST"] = "pg-main"
	if got, _ := EnvValueFor(lookup, "DB_HOST"); got != "pg-main" {
		t.Errorf("expected canonical DB_HOST to win, got %s", got)
	}

	// Alias fallback when canonical missing.
	delete(env, "DB_HOST")
	if got, _ := EnvValueFor(lookup, "DB_HOST"); got != "pg-replica" {
		t.Errorf("expected alias PGHOST fallback, got %s", got)
	}

	if got, _ := EnvValueFor(lookup, "DB_PORT"); got != "6000" {
		t.Errorf("expected DB_PORT 6000, got %s", got)
	}
}

func TestDefaults_OptionsIncludePGPathAndGeoIP(t *testing.T) {
	cfg := Defaults()
	opts := Options(cfg)
	for _, name := range []string{"PG_PATH", "GEOIP_CITY_DB", "WEBSOCKET_KEEP_ALIVE_TIMEOUT"} {
		if _, ok := opts[name]; !ok {
			t.Errorf("expected option %s in registry", name)
		}
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}