-- SPDX-License-Identifier: MIT

-- Hash-only OIDC lookups are global because the token endpoint, UserInfo, and
-- revocation endpoint cannot always know the entity before resolving a token.
-- Build these indexes without blocking production token writes. Dropping an
-- incomplete same-name index first also makes a failed concurrent migration
-- safe to retry.
-- +goose NO TRANSACTION

-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS idx_oauth_authorization_codes_code_hash;
CREATE INDEX CONCURRENTLY idx_oauth_authorization_codes_code_hash
    ON oauth_authorization_codes(code_hash);

DROP INDEX CONCURRENTLY IF EXISTS idx_oauth_tokens_token_hash;
CREATE INDEX CONCURRENTLY idx_oauth_tokens_token_hash
    ON oauth_tokens(token_hash);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_oauth_tokens_token_hash;
DROP INDEX CONCURRENTLY IF EXISTS idx_oauth_authorization_codes_code_hash;
