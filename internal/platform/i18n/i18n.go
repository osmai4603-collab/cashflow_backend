package i18n

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LanguageKey is the type for context keys
type contextKey string

var (
	// codeTranslations stores [lang][source_text]translation
	codeTranslations = make(map[string]map[string]string)
	mu               sync.RWMutex
)

// LoadTranslations loads a map of translations for a specific language
func LoadTranslations(lang string, translations map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := codeTranslations[lang]; !ok {
		codeTranslations[lang] = make(map[string]string)
	}
	for k, v := range translations {
		codeTranslations[lang][k] = v
	}
}

// T (GetText) returns the translated version of a string based on context.
// Named T instead of _ to avoid conflicts with blank identifier in Go.
func T(ctx context.Context, source string) string {
	lang, ok := ctx.Value(LangKey).(string)
	if !ok || lang == "" {
		lang = DefaultLang
	}

	mu.RLock()
	defer mu.RUnlock()
	if translations, ok := codeTranslations[lang]; ok {
		if val, ok := translations[source]; ok {
			return val
		}
	}
	return source
}

// LoadFromDirectory scans a directory for JSON translation files (e.g., ar_SA.json).
func LoadFromDirectory(dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			lang := strings.TrimSuffix(d.Name(), ".json")
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var translations map[string]string
			if err := json.Unmarshal(data, &translations); err != nil {
				return err
			}

			LoadTranslations(lang, translations)
		}
		return nil
	})
}


const (
	// LangKey is the key used to store the language in the context
	LangKey contextKey = "lang"
	// DefaultLang is the fallback language
	DefaultLang = "en_US"
)

// TranslationString stores a human-readable label that can be displayed in the active locale.
// The project historically uses a simple string value for names and complete names, while the
// underlying translation registry remains available through i18n.T / LoadTranslations.
type TranslationString string

// Get returns the underlying label, preserving compatibility with callers that lookup by language.
func (t TranslationString) Get(lang string) string {
	if t == "" {
		return ""
	}
	if lang == "" {
		lang = DefaultLang
	}
	mu.RLock()
	defer mu.RUnlock()
	if translations, ok := codeTranslations[lang]; ok {
		if val, ok := translations[string(t)]; ok {
			return val
		}
	}
	return string(t)
}

// GetLocalized returns the translation based on the language found in the context.
func (t TranslationString) GetLocalized(ctx context.Context) string {
	if t == "" {
		return ""
	}
	if lang, ok := ctx.Value(LangKey).(string); ok && lang != "" {
		if translated := t.Get(lang); translated != string(t) {
			return translated
		}
	}
	return string(t)
}

// Scan implements the sql.Scanner interface for database retrieval.
func (t *TranslationString) Scan(value interface{}) error {
	if value == nil {
		*t = ""
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for TranslationString scan")
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*t = ""
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(bytes, &m); err == nil && len(m) > 0 {
		if val, ok := m[DefaultLang]; ok && val != "" {
			*t = TranslationString(val)
			return nil
		}
		for _, val := range m {
			if val != "" {
				*t = TranslationString(val)
				return nil
			}
		}
		*t = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(bytes, &s); err == nil {
		*t = TranslationString(s)
		return nil
	}
	*t = TranslationString(string(bytes))
	return nil
}

// Value implements the driver.Valuer interface for database storage.
func (t TranslationString) Value() (driver.Value, error) {
	if t == "" {
		return nil, nil
	}
	return []byte(t), nil
}

// Set updates the label value.
func (t *TranslationString) Set(lang, value string) {
	*t = TranslationString(value)
}

// NewTranslation returns a new TranslationString with the supplied value.
func NewTranslation(val string) TranslationString {
	return TranslationString(val)
}
