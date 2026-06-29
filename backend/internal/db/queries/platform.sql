-- SPDX-License-Identifier: MIT

-- name: GetPlatformSettings :one
SELECT id, platform_name, logo_url, favicon_url, title_suffix, updated_at
FROM platform_settings
WHERE id = 1;

-- name: UpsertPlatformSettings :one
INSERT INTO platform_settings (id, platform_name, logo_url, favicon_url, title_suffix, updated_at)
VALUES (1, $1, $2, $3, $4, now())
ON CONFLICT (id) DO UPDATE SET
    platform_name = EXCLUDED.platform_name,
    logo_url = EXCLUDED.logo_url,
    favicon_url = EXCLUDED.favicon_url,
    title_suffix = EXCLUDED.title_suffix,
    updated_at = now()
RETURNING id, platform_name, logo_url, favicon_url, title_suffix, updated_at;
