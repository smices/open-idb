// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestListPortalApplicationsUsesInternalApplicationAppURL(t *testing.T) {
	const configuredInternalApplicationURL = "COALESCE(NULLIF(config ->> 'entry_url', ''), config ->> 'app_url') AS entry_url"
	if !strings.Contains(listPortalApplications, configuredInternalApplicationURL) {
		t.Fatalf("portal application query does not expose configured internal_app app_url as entry_url:\n%s", listPortalApplications)
	}
}
