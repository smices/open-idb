// SPDX-License-Identifier: MIT

package generated

import (
	"strings"
	"testing"
)

func TestListOIDCClientsDoesNotSelectReadableSecrets(t *testing.T) {
	projection := strings.SplitN(listOIDCClients, "FROM oidc_clients", 2)[0]
	if strings.Contains(projection, "client_secret_hash") {
		t.Fatal("OIDC client list query selects client_secret_hash")
	}
	if strings.Contains(projection, "workplace_app_secret") {
		t.Fatal("OIDC client list query selects workplace_app_secret")
	}
}
