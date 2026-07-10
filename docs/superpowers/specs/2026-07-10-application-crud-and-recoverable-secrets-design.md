# Application CRUD and Recoverable OIDC Secrets Design

## Goal

Replace the current split application/access/OIDC drawer experience with a focused application CRUD workflow. New, edit, view, and delete are separate actions. Forms show fields relevant to the selected application type; view exposes the complete configuration as copyable JSON for AI-assisted integration work.

## Scope

The application list exposes only View, Edit, and Delete. The creation action opens the typed application form. Application role assignments and the standalone OIDC configuration drawer are removed from this page.

The supported application types remain `oidc_client`, `api_client`, and `internal_app`.

## Data Model

`applications` gains:

- `description TEXT NOT NULL DEFAULT ''`
- `config JSONB NOT NULL DEFAULT '{}'::jsonb`

`config` stores non-secret type-specific metadata:

- `api_client`: `client_id`, `audience`, `allowed_scopes`
- `internal_app`: `app_id`, `entry_url`

`oidc_clients` gains `client_secret_encrypted BYTEA`. It stores an encrypted recoverable copy of the OIDC client secret while the existing `client_secret_hash` remains the only value used to authenticate token requests.

Existing OIDC clients have no recoverable secret because their current hash cannot be reversed. Their detail response must set `client_secret_available` to false and explain that rotating the secret creates a recoverable value.

## Secret Protection

The server reads `IDB_OIDC_SECRET_ENCRYPTION_KEY`, a base64-encoded 32-byte deployment secret. The process must fail configuration validation when the key is absent or malformed.

Secrets use AES-256-GCM with a fresh random nonce for every encryption. The persisted byte value is `nonce || ciphertext`; the plaintext is never logged. Creating or rotating an OIDC client updates both the hash and encrypted copy in the same database write.

Only authenticated administrator-facing application detail responses decrypt `client_secret_encrypted`. The View modal warns that copied JSON contains a credential. API list responses and application edit payloads never include the secret.

## CRUD API

Add an application detail response that includes common fields, `config`, the optional OIDC client configuration, and `client_secret` only when it is recoverable.

Add a create/update request contract with common fields plus a type-specific configuration object. For `oidc_client`, the service creates or updates the application and OIDC client in one database transaction. Creation returns the complete detail object including the newly generated encrypted/recoverable secret. Updates retain the current secret unless an explicit rotation endpoint is invoked.

For `api_client` and `internal_app`, the service persists the typed JSONB configuration. No unsupported token or runtime behavior is implied by these metadata fields.

## User Interface

### New and Edit

One shared form component powers both modes:

- Common fields: name, type, status, description.
- OIDC: client ID, redirect URIs, scopes, grant types, response types, PKCE, and workplace settings.
- API Client: client ID, audience, allowed scopes.
- Internal App: app ID and entry URL.

Type is selectable for new applications and read-only in edit mode. Edit uses the same fields and persists all fields supported by its type.

### View

View is a dedicated read-only modal/drawer, not an edit form. It presents the complete configuration and a `Copy JSON` action that copies the exact detail response, including `client_secret` when available. The warning appears adjacent to the copy action.

### List

The table retains name, type, status, and update time. Its only actions are View, Edit, and Delete. Delete keeps its confirmation dialog.

## Error Handling

- Invalid type-specific payloads return HTTP 400 with a stable application validation error.
- A missing recoverable-secret key prevents server startup rather than storing a secret unsafely.
- Legacy clients without encrypted secrets return successfully with `client_secret_available: false`; viewing them does not fail.
- Clipboard write failures surface an inline UI error near the copy action.

## Testing

Tests cover AES-GCM encryption/decryption and malformed keys, atomic OIDC creation, secret rotation, legacy-client detail behavior, typed API validation, and tenant isolation.

Frontend checks cover dynamic fields per application type, separate View/Edit actions, JSON copy payload construction, clipboard success/failure feedback, and absence of the access/OIDC drawer on the application page.

## Out of Scope

- Adding real API-client token issuance or authorization behavior.
- Adding an internal-application runtime integration protocol.
- Returning secrets from list or edit endpoints.
- Reworking unrelated identity-source secret storage.
