import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

import { formatDiagnostic } from './diagnostics.js';

test('formats a redacted diagnostic bundle without credentials', () => {
  const text = formatDiagnostic({
    code: 'auth_code_failed',
    stage: 'exchange',
    route: '/auth/continue?code=secret-code&state=secret-state&return_to=%2Fportal',
    error: { message: 'upstream unavailable', status: 502, traceId: 'trace-123' },
    timestamp: '2026-07-17T00:00:00.000Z',
    userAgent: 'test-agent',
  });

  assert.match(text, /code=auth_code_failed/);
  assert.match(text, /stage=exchange/);
  assert.match(text, /http_status=502/);
  assert.match(text, /trace_id=trace-123/);
  assert.doesNotMatch(text, /secret-code|secret-state/);
});

test('workplace failure stays visible and exposes a copy action', () => {
  const source = readFileSync(new URL('../workplace-continue.js', import.meta.url), 'utf8');

  assert.match(source, /copyDiagnostic/);
  assert.doesNotMatch(source, /window\.setTimeout\(\(\) =>\s*\{[\s\S]*redirectToLogin/);
});
