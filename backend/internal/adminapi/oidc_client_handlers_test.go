// SPDX-License-Identifier: MIT

package adminapi

import (
	"testing"

	"github.com/smices/open-idb/internal/db/generated"
)

func TestOIDCClientDetailExposesSecretRequirement(t *testing.T) {
	response := oidcClientFromRow(generated.OidcClient{SecretRequired: true})

	if !response.SecretRequired {
		t.Fatal("secret_required = false, want true")
	}
}
