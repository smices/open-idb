# Feishu Workplace Login Diagnostics Design

## Goal

When Feishu Workplace SSO or the Portal application directory fails, keep the
user on the failing page and expose a safe, copyable diagnostic bundle for
support and engineering triage.

## Scope

- Replace the automatic redirect after a Workplace SSO failure with an inline
  error state that offers retry, return-to-login, and copy-diagnostics actions.
- Include a redacted diagnostic record: event code, failed stage, HTTP status
  when available, backend trace ID when available, timestamp, route, and
  browser user agent.
- Do not include authorization codes, cookies, OAuth client credentials,
  request bodies, or raw backend error payloads.
- Add the same copyable diagnostic mechanism to Portal application-directory
  failures.
- Return distinct server-side Feishu login codes for an identity that was not
  synchronized and an identity that is inactive.

## Architecture

`web/src/lib/diagnostics.js` owns pure, testable redaction, formatting, and
clipboard support. `workplace-continue.js` uses it without importing React,
preserving its intentionally small standalone bundle. The React Portal uses a
small `DiagnosticDetails` component built from the same formatter.

The API client will retain non-sensitive response metadata on thrown errors:
HTTP status, request path, and an optional `X-Request-ID`/`X-Trace-ID` header.
The backend continues to log full operational detail server-side and maps
identity lifecycle failures to stable public error codes.

## User Flow

1. A Workplace SSO stage fails.
2. The page displays a plain-language failure message and remains visible.
3. The user may retry, return to login, or copy the structured diagnostic text.
4. The diagnostic text can be pasted into a support conversation without
   exposing credentials.
5. Portal catalogue failures present the same copy action next to retry.

## Verification

- Unit tests prove redaction removes sensitive keys and URL parameters.
- Unit tests prove Workplace failures render a stable diagnostic bundle and do
  not schedule an automatic redirect.
- Unit tests prove Portal errors expose copyable diagnostics.
- Go tests cover the two new stable identity error classifications.
- Run the focused frontend and backend test suites, then the full frontend
  check after dependencies are available.
