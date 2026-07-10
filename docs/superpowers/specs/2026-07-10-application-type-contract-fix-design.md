# Application Type Contract Fix Design

## Problem

The applications admin page submits `oidc`, `saml`, or `custom` when creating an application. The persisted application contract only permits `oidc_client`, `api_client`, or `internal_app`. The create handler currently checks only that the type is non-empty, so invalid input reaches PostgreSQL, violates `applications_type_check`, and is returned as an HTTP 500 error.

The Chrome ad-blocking and permissions-policy console messages are unrelated to the failed application insert.

## Contract

The existing database and product specification remain authoritative. Application types are exactly:

- `oidc_client`
- `api_client`
- `internal_app`

No database migration or legacy alias is introduced.

## Frontend

The create form will offer only the three contract values. Its default value will be `oidc_client`, and the request payload will preserve the selected contract value without translation or remapping.

Chinese and English translations will use human-readable labels for all three values. Existing application rows will therefore render labels through the same translation-key pattern as the form.

## Backend

The create handler will validate `type` before calling the service or database. A value outside the three-value contract will return HTTP 400 with the stable error code `invalid_application_type`. Missing `name` or `type` will retain the existing `invalid_request` behavior.

The database check constraint remains the final integrity boundary; handler validation provides a clear API error at the request boundary.

## Testing

Backend handler tests will prove that:

- each permitted type reaches the application service;
- an unsupported type returns HTTP 400 and never calls the service;
- the error response uses `invalid_application_type`.

Frontend tests or the project's existing static UI contract checks will prove that the create form and default payload use only `oidc_client`, `api_client`, and `internal_app`. Verification will include the focused tests, the relevant backend test suite, frontend checks, and a production build.

## Scope

This fix does not add SAML support, change the database schema, redesign application onboarding, or address the unrelated Chrome console warnings.
