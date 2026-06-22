// SPDX-License-Identifier: MIT

package sso

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenSubject struct {
	EntityID              string
	UserID                string
	ClientID              string
	SessionID             string
	Name                  string
	Email                 string
	PhoneNumber           string
	Picture               string
	PreferredUsername     string
	Locale                string
	IdentitySources       []string
	Roles                 []string
	PermissionsVersion    int64
	ResourceScopesVersion int64
}

func BuildIDTokenClaims(issuer string, subject TokenSubject, now time.Time, ttl time.Duration) jwt.MapClaims {
	claims := jwt.MapClaims{
		"iss":              issuer,
		"sub":              subject.UserID,
		"aud":              subject.ClientID,
		"iat":              jwt.NewNumericDate(now),
		"exp":              jwt.NewNumericDate(now.Add(ttl)),
		"entity_id":        subject.EntityID,
		"sid":              subject.SessionID,
		"locale":           subject.Locale,
		"identity_sources": subject.IdentitySources,
	}
	addIfNotEmpty(claims, "name", subject.Name)
	addIfNotEmpty(claims, "email", subject.Email)
	addIfNotEmpty(claims, "phone_number", subject.PhoneNumber)
	addIfNotEmpty(claims, "picture", subject.Picture)
	addIfNotEmpty(claims, "preferred_username", subject.PreferredUsername)
	return claims
}

func BuildAccessTokenClaims(issuer string, subject TokenSubject, scopes []string, now time.Time, ttl time.Duration) jwt.MapClaims {
	permVersion := subject.PermissionsVersion
	if permVersion == 0 {
		permVersion = 1
	}
	rsVersion := subject.ResourceScopesVersion
	if rsVersion == 0 {
		rsVersion = 1
	}
	return jwt.MapClaims{
		"iss":                     issuer,
		"sub":                     subject.UserID,
		"aud":                     subject.ClientID,
		"iat":                     jwt.NewNumericDate(now),
		"exp":                     jwt.NewNumericDate(now.Add(ttl)),
		"entity_id":               subject.EntityID,
		"sid":                     subject.SessionID,
		"scope":                   strings.Join(scopes, " "),
		"roles":                   subject.Roles,
		"permissions_version":     permVersion,
		"resource_scopes_version": rsVersion,
	}
}

func SignRS256(claims jwt.MapClaims, keyID string, key *rsa.PrivateKey) (string, error) {
	if keyID == "" {
		return "", fmt.Errorf("key id is required")
	}
	if key == nil {
		return "", fmt.Errorf("rsa private key is required")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	return token.SignedString(key)
}

func addIfNotEmpty(claims jwt.MapClaims, key string, value string) {
	if value != "" {
		claims[key] = value
	}
}
