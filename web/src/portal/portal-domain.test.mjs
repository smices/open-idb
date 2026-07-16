import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

function source(name) {
  return readFileSync(new URL(name, import.meta.url), 'utf8');
}

test('portal application directory owns its API request and stale-request protection', () => {
  const api = readFileSync(new URL('../lib/api.ts', import.meta.url), 'utf8');
  const hook = source('./usePortalApplications.js');

  assert.match(api, /portalApplications:\s*\(options\?[^)]*\)\s*=>\s*apiRequest<PortalApplicationsResponse>\('\/api\/portal\/applications'/);
  assert.match(hook, /AbortController/);
  assert.match(hook, /requestIdRef/);
  assert.doesNotMatch(hook, /useLoader|admin-pages/);
});

test('portal home renders application, empty, and retryable error states without access decisions', () => {
  const home = source('./PortalHomePage.jsx');

  assert.match(home, /applications\.map/);
  assert.match(home, /portal\.empty/);
  assert.match(home, /portal\.fetchFailed/);
  assert.match(home, /reload/);
  assert.match(home, /entry_url/);
  assert.doesNotMatch(home, /has_access|roles|permissions|useLoader|admin-pages/);
});

test('portal shell owns portal navigation and profile routing', () => {
  const shell = source('./PortalShell.jsx');

  assert.match(shell, /\/portal\/profile/);
  assert.match(shell, /PortalHomePage/);
  assert.match(shell, /ProfilePage/);
  assert.doesNotMatch(shell, /useLoader|admin-pages/);
});

test('portal profile is a complete read-only user detail view', () => {
  const profile = source('./ProfilePage.jsx');

  assert.match(profile, /Descriptions/);
  assert.match(profile, /english_name|EnglishName/);
  assert.match(profile, /employee_no|EmployeeNo/);
  assert.match(profile, /job_title|JobTitle/);
  assert.match(profile, /lifecycle_status|LifecycleStatus/);
  assert.doesNotMatch(profile, /updateMe|updatePassword|Form.useForm|<Form|<Input|<Button/);
});

test('portal application cards are direct links when an entry URL is configured', () => {
  const home = source('./PortalHomePage.jsx');

  assert.match(home, /<a[^>]+href={application.entry_url}/);
  assert.match(home, /application\.entry_url\.startsWith\('\/'\)/);
  assert.match(home, /target=\{isSameOriginEntry \? undefined : '_blank'\}/);
  assert.doesNotMatch(home, /<Button[^>]+href={application.entry_url}/);
});
