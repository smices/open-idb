// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestListPortalApplicationsUsesInternalApplicationAppURL(t *testing.T) {
	const configuredInternalApplicationURL = "COALESCE(NULLIF(application.config ->> 'entry_url', ''), application.config ->> 'app_url') AS entry_url"
	if !strings.Contains(listPortalApplications, configuredInternalApplicationURL) {
		t.Fatalf("portal application query does not expose configured internal_app app_url as entry_url:\n%s", listPortalApplications)
	}
	if !strings.Contains(listPortalApplications, "oidc.redirect_uris[1] AS oidc_redirect_uri") {
		t.Fatalf("portal application query does not expose OIDC redirect URI for an application entry fallback:\n%s", listPortalApplications)
	}
}
