// SPDX-License-Identifier: MIT

package i18n

import (
	"context"
	"net/http"
	"strings"
)

type localeContextKey struct{}

// Middleware extracts locale from Accept-Language header and injects
// it into the request context. Falls back to en-US if not found.
func Middleware(catalog Catalog) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := extractLocale(r)
			ctx := WithLocale(r.Context(), locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// WithLocale returns a context with the locale set.
func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeContextKey{}, locale)
}

// LocaleFromContext extracts the locale from the context.
func LocaleFromContext(ctx context.Context) string {
	if locale, ok := ctx.Value(localeContextKey{}).(string); ok {
		return locale
	}
	return LocaleEnglishUS
}

// Localize returns the localized message for the given code using
// the locale from the context. Falls back to en-US, then to the code.
func Localize(ctx context.Context, catalog Catalog, code string) string {
	locale := LocaleFromContext(ctx)
	return catalog.Message(locale, code)
}

// extractLocale parses the Accept-Language header and returns the best
// matching locale. Currently supports en-US and zh-CN.
func extractLocale(r *http.Request) string {
	header := r.Header.Get("Accept-Language")
	if header == "" {
		return LocaleEnglishUS
	}

	// Parse Accept-Language: zh-CN,zh;q=0.9,en-US;q=0.8,en;q=0.7
	parts := strings.Split(header, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		switch strings.ToLower(lang) {
		case "zh-cn", "zh":
			return LocaleChineseCN
		case "en-us", "en":
			return LocaleEnglishUS
		}
	}

	return LocaleEnglishUS
}
