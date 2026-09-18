package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// Private unexported context key to prevent collisions.
type localeKey struct{}
type printerKey struct{}

// I18nManager coordinates language matching and catalog printing.
type I18nManager struct {
	matcher  language.Matcher
	fallback language.Tag
}

// NewI18nManager creates a pre-warmed, thread-safe I18nManager.
func NewI18nManager(supported []language.Tag, fallback language.Tag) *I18nManager {
	return &I18nManager{
		matcher:  language.NewMatcher(supported),
		fallback: fallback,
	}
}

// Middleware intercepts HTTP requests and injects the resolved language into context.
func (m *I18nManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var candidateTags []language.Tag

		// 1. Explicit query parameter (?lang=...)
		if q := r.URL.Query().Get("lang"); q != "" {
			if tag, err := language.Parse(q); err == nil {
				candidateTags = append(candidateTags, tag)
			}
		}

		// 2. Custom header (X-App-Language: ar-SA)
		if custom := r.Header.Get("X-App-Language"); custom != "" && len(candidateTags) == 0 {
			if tag, err := language.Parse(custom); err == nil {
				candidateTags = append(candidateTags, tag)
			}
		}

		// 3. Accept-Language header
		if len(candidateTags) == 0 {
			if accept := r.Header.Get("Accept-Language"); accept != "" {
				tags, _, _ := language.ParseAcceptLanguage(accept)
				candidateTags = append(candidateTags, tags...)
			}
		}

		// Match candidate against supported locales
		var resolved language.Tag
		if len(candidateTags) > 0 {
			tag, _, _ := m.matcher.Match(candidateTags...)
			resolved = tag
		} else {
			resolved = m.fallback
		}

		// Instantiate printer for the resolved tag
		printer := message.NewPrinter(resolved)

		// Inject into context
		ctx := context.WithValue(r.Context(), localeKey{}, resolved)
		ctx = context.WithValue(ctx, printerKey{}, printer)

		// Set response header to notify client
		w.Header().Set("Content-Language", resolved.String())

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetLocale safely retrieves the resolved language tag from context.
func GetLocale(ctx context.Context, defaultTag language.Tag) language.Tag {
	if tag, ok := ctx.Value(localeKey{}).(language.Tag); ok {
		return tag
	}
	return defaultTag
}

// GetPrinter safely retrieves the message.Printer from context.
func GetPrinter(ctx context.Context, defaultTag language.Tag) *message.Printer {
	if p, ok := ctx.Value(printerKey{}).(*message.Printer); ok {
		return p
	}
	return message.NewPrinter(defaultTag)
}

// ProblemDetails represents an RFC 7807 compliant error payload.
type ProblemDetails struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Status   int            `json:"status"`
	Code     string         `json:"code"`
	Detail   string         `json:"detail"`
	Language string         `json:"language"`
	Invalid  map[string]any `json:"invalid_params,omitempty"`
}

// WriteProblemDetails writes a localized RFC 7807 error response.
func WriteProblemDetails(w http.ResponseWriter, r *http.Request, status int, code, titleKey, detailKey string, params map[string]any) {
	ctx := r.Context()
	defaultTag := language.MustParse("ar-SA")
	printer := GetPrinter(ctx, defaultTag)
	locale := GetLocale(ctx, defaultTag)

	prob := ProblemDetails{
		Type:     fmt.Sprintf("https://api.cashflow.com/errors/%s", code),
		Title:    printer.Sprintf(titleKey),
		Status:   status,
		Code:     code,
		Detail:   printer.Sprintf(detailKey),
		Language: locale.String(),
		Invalid:  params,
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
