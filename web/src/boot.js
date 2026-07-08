// SPDX-License-Identifier: MIT

function normalizeWorkplaceProvider(provider) {
  const value = (provider || '').trim().toLowerCase();
  if (value === 'lark') return 'feishu';
  return value === 'feishu' ? value : '';
}

function returnToParam(returnToValue, name) {
  try {
    return new URL(returnToValue, window.location.origin).searchParams.get(name) || '';
  } catch {
    return '';
  }
}

function workplaceProvider(params) {
  const returnTo = params.get('return_to') || '';
  return normalizeWorkplaceProvider(
    params.get('workplace') ||
      params.get('workplace_provider') ||
      params.get('sso_provider') ||
      returnToParam(returnTo, 'workplace') ||
      returnToParam(returnTo, 'workplace_provider') ||
      returnToParam(returnTo, 'sso_provider'),
  );
}

const params = new URLSearchParams(window.location.search);
const isFeishuWorkplaceContinue = window.location.pathname === '/auth/continue' && workplaceProvider(params) === 'feishu';

if (isFeishuWorkplaceContinue) {
  import('./workplace-continue.js');
} else {
  import('./main.jsx');
}
