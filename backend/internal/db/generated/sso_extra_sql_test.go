// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestGlobalSSOTokenLookupRejectsCrossEntityHashAmbiguity(t *testing.T) {
	if !strings.Contains(getSSOTokenByHashGlobally, "duplicate.entity_id <> token.entity_id") {
		t.Fatal("global token lookup does not reject a hash shared by multiple entities")
	}
}

func TestGlobalSSOTokenRevocationRejectsCrossEntityHashAmbiguity(t *testing.T) {
	if !strings.Contains(revokeSSOTokenByHashGlobally, "duplicate.entity_id <> token.entity_id") {
		t.Fatal("global token revocation does not reject a hash shared by multiple entities")
	}
}

func TestCreateOIDCClientRequiresSecretForNewManagedClients(t *testing.T) {
	if !strings.Contains(createOIDCClient, "secret_required") || !strings.Contains(createOIDCClient, "true") {
		t.Fatal("CreateOIDCClient does not opt new managed clients into secret verification")
	}
}
