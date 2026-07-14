package adminapi

import "testing"

func TestOIDCClientForAuditRemovesAllSecrets(t *testing.T) {
	original := OIDCClientResponse{
		ID:                 "client-id",
		ClientID:           "public-client-id",
		ClientSecret:       "readable-client-secret",
		WorkplaceAppSecret: "workplace-secret",
	}

	got := oidcClientForAudit(original)
	if got.ClientSecret != "" {
		t.Fatalf("ClientSecret = %q, want empty", got.ClientSecret)
	}
	if got.WorkplaceAppSecret != "" {
		t.Fatalf("WorkplaceAppSecret = %q, want empty", got.WorkplaceAppSecret)
	}
	if got.ClientID != original.ClientID || got.ID != original.ID {
		t.Fatalf("non-secret fields changed: %#v", got)
	}
	if original.ClientSecret == "" || original.WorkplaceAppSecret == "" {
		t.Fatal("oidcClientForAudit mutated the API response")
	}
}
