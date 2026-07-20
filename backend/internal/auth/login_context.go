// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type LoginMode string

const (
	LoginModeApp         LoginMode = "app"
	LoginModeUser        LoginMode = "user"
	LoginModeAdmin       LoginMode = "admin"
	LoginModeEntityAdmin LoginMode = "entity_admin"
)

var loginProviderAliases = map[string]string{
	"lark": "feishu",
}

type LoginContextEntity struct {
	ID           string `json:"id,omitempty"`
	Slug         string `json:"slug,omitempty"`
	Name         string `json:"name,omitempty"`
	BrandName    string `json:"brand_name,omitempty"`
	LogoURL      string `json:"logo_url,omitempty"`
	LoginMessage string `json:"login_message,omitempty"`
}

type LoginContextApplication struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

type LoginContextEntityResolver interface {
	GetLoginContextEntityBySlug(ctx context.Context, slug string) (LoginContextEntity, error)
}

type LoginContextApplicationResolver interface {
	GetLoginContextApplicationByClientID(ctx context.Context, clientID string) (LoginContext, error)
}

type LoginContextDefaultEntityResolver interface {
	GetDefaultLoginContext(ctx context.Context) (LoginContext, error)
}

type LoginContext struct {
	Mode                 LoginMode                `json:"mode"`
	Entity               *LoginContextEntity      `json:"entity"`
	Application          *LoginContextApplication `json:"application"`
	Methods              []string                 `json:"methods"`
	AllowEntitySelection bool                     `json:"allow_entity_selection"`
	Reason               string                   `json:"reason,omitempty"`
	ReturnTo             string                   `json:"return_to"`
	PreferredProvider    string                   `json:"preferred_provider,omitempty"`
	AutoRedirectURL      string                   `json:"auto_redirect_url,omitempty"`
}

func resolveLoginContext(path string, returnTo string) LoginContext {
	normalizedPath := cleanLoginContextPath(path)
	safeReturn := safeReturnTo(returnTo)
	parts := strings.Split(strings.Trim(normalizedPath, "/"), "/")

	if normalizedPath == "/admin/login" {
		return LoginContext{
			Mode:                 LoginModeAdmin,
			Methods:              []string{"password"},
			AllowEntitySelection: false,
			Reason:               "admin",
			ReturnTo:             safeReturnToWithDefault(returnTo, "/admin"),
		}
	}

	if len(parts) >= 4 && parts[0] == "t" && parts[2] == "admin" && parts[3] == "login" {
		return LoginContext{
			Mode:                 LoginModeEntityAdmin,
			Entity:               &LoginContextEntity{Slug: strings.TrimSpace(parts[1])},
			Methods:              []string{"password"},
			AllowEntitySelection: false,
			Reason:               "path",
			ReturnTo:             safeReturn,
		}
	}

	if normalizedPath == "/auth/continue" {
		return LoginContext{
			Mode:                 LoginModeApp,
			Methods:              []string{"password", "feishu"},
			AllowEntitySelection: false,
			Reason:               "application",
			ReturnTo:             safeReturn,
		}
	}

	return LoginContext{
		Mode:                 LoginModeUser,
		Methods:              []string{"password"},
		AllowEntitySelection: false,
		Reason:               "default",
		ReturnTo:             safeReturn,
	}
}

func cleanLoginContextPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

func (h Handler) loginContext(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = r.URL.Path
	}
	returnTo := r.URL.Query().Get("return_to")
	preferredProvider := preferredProviderFromRequest(r, returnTo)
	ctx := resolveLoginContext(path, returnTo)
	if clientID := oidcClientIDFromReturnTo(returnTo); clientID != "" {
		if resolver, ok := h.service.(LoginContextApplicationResolver); ok {
			if appCtx, err := resolver.GetLoginContextApplicationByClientID(r.Context(), clientID); err == nil {
				ctx = appCtx
				ctx.ReturnTo = safeReturnTo(returnTo)
			}
		}
	} else if ctx.Mode == LoginModeUser || (ctx.Mode == LoginModeApp && preferredProvider == "feishu") {
		if resolver, ok := h.service.(LoginContextDefaultEntityResolver); ok {
			if defaultCtx, err := resolver.GetDefaultLoginContext(r.Context()); err == nil {
				if ctx.Mode == LoginModeApp {
					ctx.Entity = defaultCtx.Entity
					ctx.Methods = defaultCtx.Methods
					ctx.AllowEntitySelection = false
					ctx.Reason = "workplace_default_entity"
				} else {
					ctx = defaultCtx
				}
				ctx.ReturnTo = safeReturnTo(returnTo)
			}
		}
	}
	ctx.applyPreferredProvider(preferredProvider)
	if ctx.Entity != nil && ctx.Entity.Slug != "" {
		if resolver, ok := h.service.(LoginContextEntityResolver); ok {
			if entity, err := resolver.GetLoginContextEntityBySlug(r.Context(), ctx.Entity.Slug); err == nil {
				ctx.Entity = &entity
			}
		}
	}
	ctx.applyAutoRedirect()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ctx)
}

func oidcClientIDFromReturnTo(returnTo string) string {
	safeReturn := safeReturnTo(returnTo)
	u, err := url.Parse(safeReturn)
	if err != nil {
		return ""
	}
	if u.Path != "/oauth2/authorize" && u.Path != "/api/oauth2/authorize" {
		return ""
	}
	return strings.TrimSpace(u.Query().Get("client_id"))
}

func preferredProviderFromRequest(r *http.Request, returnTo string) string {
	if provider := normalizePreferredProvider(r.URL.Query().Get("idp")); provider != "" {
		return provider
	}
	if provider := normalizePreferredProvider(r.URL.Query().Get("login_provider")); provider != "" {
		return provider
	}
	if provider := normalizePreferredProvider(r.URL.Query().Get("workplace")); provider != "" {
		return provider
	}
	return preferredProviderFromReturnTo(returnTo)
}

func preferredProviderFromReturnTo(returnTo string) string {
	safeReturn := safeReturnTo(returnTo)
	u, err := url.Parse(safeReturn)
	if err != nil {
		return ""
	}
	if u.Path != "/oauth2/authorize" && u.Path != "/api/oauth2/authorize" {
		return ""
	}
	if provider := normalizePreferredProvider(u.Query().Get("idp")); provider != "" {
		return provider
	}
	return normalizePreferredProvider(u.Query().Get("login_provider"))
}

func normalizePreferredProvider(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if alias := loginProviderAliases[provider]; alias != "" {
		return alias
	}
	if !isSafeProviderName(provider) {
		return ""
	}
	return provider
}

func (ctx *LoginContext) applyPreferredProvider(provider string) {
	if provider == "" || !loginMethodAllowed(ctx.Methods, provider) {
		return
	}
	ctx.PreferredProvider = provider
}

func (ctx *LoginContext) applyAutoRedirect() {
	if ctx.PreferredProvider == "" || ctx.Entity == nil || strings.TrimSpace(ctx.Entity.ID) == "" {
		return
	}
	values := url.Values{}
	values.Set("entity_id", strings.TrimSpace(ctx.Entity.ID))
	if ctx.ReturnTo != "" {
		values.Set("return_to", ctx.ReturnTo)
	}
	ctx.AutoRedirectURL = "/api/auth/" + ctx.PreferredProvider + "/login?" + values.Encode()
}

func loginMethodAllowed(methods []string, provider string) bool {
	for _, method := range methods {
		if strings.EqualFold(strings.TrimSpace(method), provider) {
			return true
		}
	}
	return false
}

func isSafeProviderName(provider string) bool {
	if provider == "" {
		return false
	}
	for _, r := range provider {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}
