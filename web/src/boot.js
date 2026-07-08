// SPDX-License-Identifier: MIT

import { normalizeWorkplaceProvider, returnToParam } from './lib/feishu-workplace.js';

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
