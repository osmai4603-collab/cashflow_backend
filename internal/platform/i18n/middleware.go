package i18n

import (
	"context"
	"net/http"
	"strings"
)

// Middleware extracts the language from the request and adds it to the context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			lang = r.Header.Get("Accept-Language")
			if lang != "" {
				// Take the first preferred language (e.g., "en-US,en;q=0.9" -> "en-US")
				lang = strings.Split(lang, ",")[0]
				// Normalize to Odoo format (e.g., en-US -> en_US)
				lang = strings.Replace(lang, "-", "_", -1)
			}
		}

		if lang == "" {
			lang = DefaultLang
		}

		ctx := context.WithValue(r.Context(), LangKey, lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetLang returns the current language from the context.
func GetLang(ctx context.Context) string {
	if lang, ok := ctx.Value(LangKey).(string); ok {
		return lang
	}
	return DefaultLang
}
