import assert from 'node:assert/strict';
import test from 'node:test';

import { safeReturnTo } from './navigation.js';

const origin = 'https://auth.example.test';

test('safeReturnTo preserves same-origin application paths and queries', () => {
  const value = '/oauth2/authorize?client_id=demo&redirect_uri=https%3A%2F%2Fapp.example.test%2Fcallback';

  assert.equal(safeReturnTo(value, '/portal', origin), value);
});

test('safeReturnTo rejects absolute and network-path redirects', () => {
  for (const value of [
    'https://evil.example/steal',
    '//evil.example/steal',
    String.raw`/\evil.example/steal`,
    '/.//evil.example/steal',
    '/foo/..//evil.example/steal',
    'javascript:alert(1)',
  ]) {
    assert.equal(safeReturnTo(value, '/portal', origin), '/portal', value);
  }
});

test('safeReturnTo removes fragments before navigation', () => {
  assert.equal(safeReturnTo('/portal?tab=apps#secret', '/portal', origin), '/portal?tab=apps');
});
