// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"strings"

	"github.com/smices/open-idb/internal/db/generated"
)

type entitySlugResolver interface {
	GetEntityBySlug(ctx context.Context, slug string) (generated.BusinessEntity, error)
}

func resolveEntityRef(ctx context.Context, resolver entitySlugResolver, entityRef string) (string, error) {
	entityRef = strings.TrimSpace(entityRef)
	if entityULID, err := parseULID(entityRef); err == nil {
		return entityULID, nil
	}
	entity, err := resolver.GetEntityBySlug(ctx, entityRef)
	if err != nil {
		return "", err
	}
	return ulidString(entity.ID), nil
}
