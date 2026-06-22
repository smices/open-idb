// SPDX-License-Identifier: MIT

//go:build tools
// +build tools

package tools

import (
	_ "github.com/go-chi/chi/v5"
	_ "github.com/google/go-cmp/cmp"
	_ "github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/pressly/goose/v3/cmd/goose"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"
	_ "github.com/testcontainers/testcontainers-go"
	_ "github.com/testcontainers/testcontainers-go/modules/postgres"
	_ "go.uber.org/zap"
)
