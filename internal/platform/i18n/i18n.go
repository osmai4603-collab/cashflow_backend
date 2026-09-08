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

// TranslationString represents a multi-language string stored as JSONB.
// Key is the language code (e.g., "en_US", "ar_001"), value is the translation.
type TranslationString map[string]string

// Get returns the translation for the given language, falling back to English or any available value.
func (t TranslationString) Get(lang string) string {
	if val, ok := t[lang]; ok && val != "" {
		return val
	}
	if val, ok := t[DefaultLang]; ok && val != "" {
		return val
	}
	// Fallback to the first available translation
	for _, val := range t {
		if val != "" {
			return val
		}
	}
	return ""
}

// GetLocalized returns the translation based on the language found in the context.
func (t TranslationString) GetLocalized(ctx context.Context) string {
	lang, ok := ctx.Value(LangKey).(string)
	if !ok || lang == "" {
		lang = DefaultLang
	}
	return t.Get(lang)
}

// Scan implements the sql.Scanner interface for database retrieval.
func (t *TranslationString) Scan(value interface{}) error {
	if value == nil {
		*t = make(TranslationString)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, t)
}

// Value implements the driver.Valuer interface for database storage.
func (t TranslationString) Value() (driver.Value, error) {
	if len(t) == 0 {
		return nil, nil
	}
	return json.Marshal(t)
}

// Set sets the translation for a specific language.
func (t TranslationString) Set(lang, value string) {
	t[lang] = value
}

// NewTranslation returns a new TranslationString with a default value for English.
func NewTranslation(val string) TranslationString {
	return TranslationString{
		DefaultLang: val,
	}
}
