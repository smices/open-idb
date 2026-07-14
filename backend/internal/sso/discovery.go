// SPDX-License-Identifier: MIT

package sso

import "strings"

type DiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JWKSURI                           string   `json:"jwks_uri"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	SubjectTypesSupported             []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

func BuildDiscoveryDocument(issuer string) DiscoveryDocument {
	return BuildDiscoveryDocumentWithEndpointPrefix(issuer, "")
}

func BuildDiscoveryDocumentWithEndpointPrefix(issuer string, endpointPrefix string) DiscoveryDocument {
	normalizedIssuer := strings.TrimRight(issuer, "/")
	normalizedPrefix := strings.TrimRight(endpointPrefix, "/")
	if normalizedPrefix != "" && !strings.HasPrefix(normalizedPrefix, "/") {
		normalizedPrefix = "/" + normalizedPrefix
	}

	return DiscoveryDocument{
		Issuer:                            normalizedIssuer,
		AuthorizationEndpoint:             normalizedIssuer + normalizedPrefix + "/oauth2/authorize",
		TokenEndpoint:                     normalizedIssuer + normalizedPrefix + "/oauth2/token",
		UserinfoEndpoint:                  normalizedIssuer + normalizedPrefix + "/oauth2/userinfo",
		JWKSURI:                           normalizedIssuer + normalizedPrefix + "/.well-known/jwks.json",
		ResponseTypesSupported:            []string{"code"},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{SigningAlgorithmRS256},
		ScopesSupported:                   []string{"openid", "profile", "email", "directory:read"},
		TokenEndpointAuthMethodsSupported: []string{"client_secret_basic", "client_secret_post", "none"},
		CodeChallengeMethodsSupported:     []string{CodeChallengeS256},
	}
}
