// SPDX-License-Identifier: MIT

import { feishuAuthCodeFromBridge, normalizeWorkplaceProvider, queryAuthCode, returnToParam } from './lib/feishu-workplace.js';

const API_TARGET = import.meta.env.VITE_API_TARGET || import.meta.env.PUBLIC_API_TARGET || '';

const THEME_KEY = 'idb-theme-mode';
const MESSAGES = {
  'zh-CN': {
    eyebrow: '飞书工作台 SSO',
    title: '正在进入应用',
    titleDocument: '正在进入应用 - IdBridge',
    error: '进入应用失败，正在返回登录页。',
    steps: [
      { key: 'context', label: '解析访问上下文', detail: '正在确认应用与企业身份' },
      { key: 'feishu', label: '获取飞书授权', detail: '正在连接飞书工作台' },
      { key: 'enter', label: '进入应用', detail: '正在完成登录并跳转' },
    ],
  },
  'en-US': {
    eyebrow: 'Feishu Workplace SSO',
    title: 'Opening your app',
    titleDocument: 'Opening your app - IdBridge',
    error: 'Unable to open the app. Returning to sign-in.',
    steps: [
      { key: 'context', label: 'Resolving access context', detail: 'Checking the application and organization identity' },
      { key: 'feishu', label: 'Requesting Feishu authorization', detail: 'Connecting to Feishu Workplace' },
      { key: 'enter', label: 'Entering the app', detail: 'Completing sign-in and redirecting' },
    ],
  },
};

const locale = resolveLocale();
const copy = MESSAGES[locale];
const steps = copy.steps;

function resolveLocale() {
  return navigator.language?.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US';
}

function resolveThemeMode() {
  try {
    const stored = localStorage.getItem(THEME_KEY);
    return stored === 'dark' || stored === 'light' ? stored : 'system';
  } catch {
    return 'system';
  }
}

function queryString(params) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params || {})) {
    if (value === undefined || value === null || value === '') continue;
    search.set(key, String(value));
  }
  return search.toString() ? `?${search.toString()}` : '';
}

async function apiRequest(path, options = {}) {
  const headers = { ...(options.headers || {}) };
  const hasFormBody = options.body instanceof FormData;
  const body = hasFormBody || options.skipJson ? options.body : options.body ? JSON.stringify(options.body) : undefined;

  if (!hasFormBody && options.body && !options.headers?.['Content-Type']) {
    headers['Content-Type'] = 'application/json';
  }

  const response = await fetch(`${API_TARGET}${path}`, {
    method: options.method || 'GET',
    headers,
    credentials: 'include',
    body,
    signal: options.signal,
    redirect: 'follow',
  });

  if (response.status === 204) return undefined;
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    const payload = await response.json();
    if (!response.ok) {
      throw new Error(payload?.error_description || payload?.message || 'Request failed');
    }
    return payload;
  }

  const text = await response.text();
  if (!response.ok) throw new Error(text || 'Request failed');
  return text;
}

function injectStyles() {
  document.documentElement.lang = locale;
  document.documentElement.dataset.workplaceTheme = resolveThemeMode();

  const style = document.createElement('style');
  style.textContent = `
    :root {
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      color-scheme: light;
      --workplace-bg: #f5f7f6;
      --workplace-panel: rgba(255, 255, 255, 0.92);
      --workplace-panel-solid: #ffffff;
      --workplace-text: #111827;
      --workplace-muted: #4b5563;
      --workplace-soft: #6b7280;
      --workplace-line: rgba(17, 24, 39, 0.12);
      --workplace-frame: rgba(17, 24, 39, 0.08);
      --workplace-accent: #0f766e;
      --workplace-accent-strong: #0d9488;
      --workplace-accent-soft: #ecfdf5;
      --workplace-accent-frame: rgba(15, 118, 110, 0.16);
      --workplace-active-border: rgba(15, 118, 110, 0.32);
      --workplace-check: #ffffff;
      --workplace-track: #e5e7eb;
      --workplace-shadow: 0 24px 80px rgba(17, 24, 39, 0.13);
      --workplace-error-bg: #fffbeb;
      --workplace-error-border: rgba(217, 119, 6, 0.28);
      --workplace-error-text: #92400e;
      background: var(--workplace-bg);
      color: var(--workplace-text);
    }

    :root[data-workplace-theme="dark"] {
      color-scheme: dark;
      --workplace-bg: #0d1110;
      --workplace-panel: rgba(18, 25, 24, 0.92);
      --workplace-panel-solid: #121918;
      --workplace-text: #f8fafc;
      --workplace-muted: #cbd5e1;
      --workplace-soft: #94a3b8;
      --workplace-line: rgba(226, 232, 240, 0.14);
      --workplace-frame: rgba(226, 232, 240, 0.1);
      --workplace-accent: #5eead4;
      --workplace-accent-strong: #2dd4bf;
      --workplace-accent-soft: rgba(45, 212, 191, 0.16);
      --workplace-accent-frame: rgba(94, 234, 212, 0.16);
      --workplace-active-border: rgba(94, 234, 212, 0.34);
      --workplace-check: #ffffff;
      --workplace-track: rgba(148, 163, 184, 0.22);
      --workplace-shadow: 0 24px 90px rgba(0, 0, 0, 0.38);
      --workplace-error-bg: rgba(120, 53, 15, 0.2);
      --workplace-error-border: rgba(251, 191, 36, 0.3);
      --workplace-error-text: #fde68a;
    }

    @media (prefers-color-scheme: dark) {
      :root[data-workplace-theme="system"] {
        color-scheme: dark;
        --workplace-bg: #0d1110;
        --workplace-panel: rgba(18, 25, 24, 0.92);
        --workplace-panel-solid: #121918;
        --workplace-text: #f8fafc;
        --workplace-muted: #cbd5e1;
        --workplace-soft: #94a3b8;
        --workplace-line: rgba(226, 232, 240, 0.14);
        --workplace-frame: rgba(226, 232, 240, 0.1);
        --workplace-accent: #5eead4;
        --workplace-accent-strong: #2dd4bf;
        --workplace-accent-soft: rgba(45, 212, 191, 0.16);
        --workplace-accent-frame: rgba(94, 234, 212, 0.16);
        --workplace-active-border: rgba(94, 234, 212, 0.34);
        --workplace-check: #ffffff;
        --workplace-track: rgba(148, 163, 184, 0.22);
        --workplace-shadow: 0 24px 90px rgba(0, 0, 0, 0.38);
        --workplace-error-bg: rgba(120, 53, 15, 0.2);
        --workplace-error-border: rgba(251, 191, 36, 0.3);
        --workplace-error-text: #fde68a;
      }
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      min-width: 320px;
      min-height: 100dvh;
      background: var(--workplace-bg);
    }

    .workplace-loading {
      position: relative;
      display: grid;
      min-height: 100dvh;
      overflow: hidden;
      padding: 28px;
      place-items: center;
      background: var(--workplace-bg);
    }

    .workplace-loading::before,
    .workplace-loading::after {
      position: absolute;
      content: "";
      border: 1px solid var(--workplace-frame);
      pointer-events: none;
    }

    .workplace-loading::before {
      inset: 24px;
    }

    .workplace-loading::after {
      inset: 36px;
      border-color: var(--workplace-accent-frame);
    }

    .workplace-panel {
      position: relative;
      z-index: 1;
      width: min(520px, 100%);
      padding: 36px;
      overflow: hidden;
      border: 1px solid var(--workplace-line);
      border-radius: 8px;
      background: var(--workplace-panel);
      box-shadow: var(--workplace-shadow);
    }

    .workplace-panel::before {
      position: absolute;
      inset: 0 0 auto;
      height: 3px;
      background: var(--workplace-accent);
      content: "";
    }

    .workplace-panel::after {
      position: absolute;
      inset: 16px;
      border: 1px solid var(--workplace-frame);
      border-radius: 6px;
      content: "";
      pointer-events: none;
    }

    .workplace-panel > * {
      position: relative;
      z-index: 1;
    }

    .workplace-topline {
      display: flex;
      align-items: center;
      justify-content: space-between;
      gap: 18px;
    }

    .workplace-brand {
      display: flex;
      align-items: center;
      gap: 12px;
      min-width: 0;
      color: var(--workplace-text);
      font-size: 15px;
      font-weight: 650;
    }

    .workplace-brand img {
      width: 34px;
      height: 34px;
      flex: 0 0 auto;
      border-radius: 7px;
      object-fit: contain;
    }

    .workplace-signal {
      display: inline-flex;
      align-items: end;
      gap: 3px;
      height: 26px;
      padding: 6px 8px;
      border: 1px solid var(--workplace-line);
      border-radius: 999px;
      background: var(--workplace-panel-solid);
    }

    .workplace-signal span {
      width: 3px;
      border-radius: 999px;
      background: var(--workplace-accent);
      opacity: 0.72;
      transform-origin: bottom;
      animation: idb-workplace-signal 1.15s ease-in-out infinite;
    }

    .workplace-signal span:nth-child(1) {
      height: 7px;
    }

    .workplace-signal span:nth-child(2) {
      height: 11px;
      animation-delay: 120ms;
    }

    .workplace-signal span:nth-child(3) {
      height: 15px;
      animation-delay: 240ms;
    }

    .workplace-brand span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .workplace-copy {
      margin-top: 42px;
    }

    .workplace-eyebrow {
      margin: 0 0 10px;
      color: var(--workplace-accent);
      font-size: 13px;
      font-weight: 700;
    }

    .workplace-title {
      margin: 0;
      color: var(--workplace-text);
      font-size: 30px;
      font-weight: 760;
      letter-spacing: 0;
      line-height: 1.15;
      text-wrap: balance;
    }

    .workplace-detail {
      min-height: 52px;
      margin: 14px 0 0;
      color: var(--workplace-muted);
      font-size: 16px;
      line-height: 1.65;
      text-wrap: pretty;
    }

    .workplace-progress {
      position: relative;
      height: 3px;
      margin: 34px 0 26px;
      overflow: hidden;
      border-radius: 999px;
      background: var(--workplace-track);
    }

    .workplace-progress span {
      position: absolute;
      inset: 0 auto 0 0;
      width: 38%;
      border-radius: inherit;
      background: var(--workplace-accent);
      transform: translateX(-100%);
      animation: idb-workplace-progress 1.35s ease-in-out infinite;
    }

    .workplace-steps {
      display: grid;
      gap: 10px;
      margin: 0;
      padding: 0;
      list-style: none;
    }

    .workplace-step {
      display: grid;
      grid-template-columns: 22px 1fr;
      gap: 10px;
      align-items: center;
      min-height: 30px;
      color: var(--workplace-soft);
      font-size: 14px;
    }

    .workplace-step-dot {
      display: grid;
      width: 22px;
      height: 22px;
      place-items: center;
      border: 1px solid var(--workplace-line);
      border-radius: 999px;
      background: var(--workplace-panel-solid);
    }

    .workplace-step-dot::after {
      width: 6px;
      height: 6px;
      border-radius: 999px;
      background: var(--workplace-line);
      content: "";
    }

    .workplace-step.is-active {
      color: var(--workplace-text);
      font-weight: 650;
    }

    .workplace-step.is-active .workplace-step-dot {
      border-color: var(--workplace-active-border);
      background: var(--workplace-accent-soft);
    }

    .workplace-step.is-active .workplace-step-dot::after {
      background: var(--workplace-accent);
      animation: idb-workplace-pulse 1.2s ease-in-out infinite;
    }

    .workplace-step.is-done {
      color: var(--workplace-muted);
    }

    .workplace-step.is-done .workplace-step-dot {
      border-color: var(--workplace-accent-strong);
      background: var(--workplace-accent-strong);
    }

    .workplace-step.is-done .workplace-step-dot::after {
      width: 8px;
      height: 8px;
      border-radius: 0;
      background: transparent;
      border-right: 2px solid var(--workplace-check);
      border-bottom: 2px solid var(--workplace-check);
      transform: translateY(-1px) rotate(45deg);
    }

    .workplace-error {
      display: none;
      margin-top: 22px;
      padding: 12px 14px;
      border: 1px solid var(--workplace-error-border);
      border-radius: 8px;
      background: var(--workplace-error-bg);
      color: var(--workplace-error-text);
      font-size: 14px;
      line-height: 1.55;
    }

    .workplace-error.is-visible {
      display: block;
    }

    @keyframes idb-workplace-progress {
      0% { transform: translateX(-100%); }
      52% { transform: translateX(92%); }
      100% { transform: translateX(264%); }
    }

    @keyframes idb-workplace-signal {
      0%, 100% { transform: scaleY(0.68); opacity: 0.58; }
      50% { transform: scaleY(1); opacity: 1; }
    }

    @keyframes idb-workplace-pulse {
      0%, 100% { transform: scale(0.9); opacity: 0.74; }
      50% { transform: scale(1.35); opacity: 1; }
    }

    @media (max-width: 620px) {
      .workplace-loading {
        padding: 18px;
      }

      .workplace-loading::before {
        inset: 12px;
      }

      .workplace-loading::after {
        inset: 22px;
      }

      .workplace-panel {
        padding: 26px;
      }

      .workplace-copy {
        margin-top: 34px;
      }

      .workplace-title {
        font-size: 26px;
      }
    }

    @media (prefers-reduced-motion: reduce) {
      .workplace-progress span,
      .workplace-signal span,
      .workplace-step.is-active .workplace-step-dot::after {
        animation: none;
      }

      .workplace-progress span {
        transform: translateX(60%);
      }
    }
  `;
  document.head.appendChild(style);
}

function renderShell() {
  injectStyles();
  document.title = copy.titleDocument;
  const root = document.getElementById('root') || document.body.appendChild(document.createElement('div'));
  root.innerHTML = `
    <main class="workplace-loading" aria-busy="true">
      <section class="workplace-panel" aria-labelledby="workplace-loading-title">
        <div class="workplace-topline">
          <div class="workplace-brand">
            <img data-brand-logo src="/logo.svg" alt="" />
            <span data-brand-name>IdBridge</span>
          </div>
          <span class="workplace-signal" aria-hidden="true">
            <span></span>
            <span></span>
            <span></span>
          </span>
        </div>
        <div class="workplace-copy">
          <p class="workplace-eyebrow">${copy.eyebrow}</p>
          <h1 class="workplace-title" id="workplace-loading-title">${copy.title}</h1>
          <p class="workplace-detail" data-workplace-detail>${steps[0].detail}</p>
        </div>
        <div class="workplace-progress" aria-hidden="true"><span></span></div>
        <ol class="workplace-steps">
          ${steps.map((step, index) => `
            <li class="workplace-step${index === 0 ? ' is-active' : ''}" data-step="${step.key}">
              <span class="workplace-step-dot" aria-hidden="true"></span>
              <span>${step.label}</span>
            </li>
          `).join('')}
        </ol>
        <div class="workplace-error" data-workplace-error role="alert"></div>
      </section>
    </main>
  `;
}

function setBrand(context) {
  const brandName = context?.entity?.brand_name || context?.entity?.name || context?.application?.name || 'IdBridge';
  const logoUrl = context?.entity?.logo_url || '/logo.svg';
  const brandNode = document.querySelector('[data-brand-name]');
  const logoNode = document.querySelector('[data-brand-logo]');
  if (brandNode) brandNode.textContent = brandName;
  if (logoNode) logoNode.setAttribute('src', logoUrl);
}

function setStep(index, detail = steps[index]?.detail || '') {
  document.querySelectorAll('[data-step]').forEach((node, nodeIndex) => {
    node.classList.toggle('is-done', nodeIndex < index);
    node.classList.toggle('is-active', nodeIndex === index);
  });
  const detailNode = document.querySelector('[data-workplace-detail]');
  if (detailNode) detailNode.textContent = detail;
}

function showError(message) {
  const errorNode = document.querySelector('[data-workplace-error]');
  if (!errorNode) return;
  errorNode.textContent = message;
  errorNode.classList.add('is-visible');
}

function redirectToLogin(returnTo, loginError = 'workplace_not_available') {
  const params = new URLSearchParams();
  params.set('return_to', returnTo || '/portal');
  params.set('workplace', 'feishu');
  params.set('login_error', loginError);
  window.location.replace(`/login?${params.toString()}`);
}

async function runWorkplaceContinue() {
  const params = new URLSearchParams(window.location.search);
  const returnTo = params.get('return_to') || '/portal';
  const oidcClientId = returnToParam(returnTo, 'client_id');
  const workplaceProvider = normalizeWorkplaceProvider(
    params.get('workplace') ||
      params.get('workplace_provider') ||
      params.get('sso_provider') ||
      returnToParam(returnTo, 'workplace') ||
      returnToParam(returnTo, 'workplace_provider') ||
      returnToParam(returnTo, 'sso_provider'),
  );

  if (workplaceProvider !== 'feishu') {
    redirectToLogin(returnTo);
    return;
  }

  try {
    setStep(0);
    const context = await apiRequest(`/api/auth/context${queryString({ path: window.location.pathname, return_to: returnTo })}`);
    setBrand(context);
    const entityRef = context?.entity?.id || context?.entity?.slug || '';
    if (!entityRef || !context?.methods?.includes('feishu')) {
      throw new Error('workplace_not_available');
    }

    setStep(1);
    const providers = await apiRequest(`/api/auth/providers${queryString({ entity_id: entityRef, client_id: oidcClientId })}`);
    const provider = providers.find((item) => item.provider === 'feishu');
    if (!provider?.workplace_exchange_url || !provider.app_id) {
      throw new Error('workplace_not_configured');
    }

    const authCode = queryAuthCode(params, true) || (await feishuAuthCodeFromBridge(provider.app_id));
    setStep(2);
    await apiRequest('/api/auth/feishu/exchange', {
      method: 'POST',
      body: { auth_code: authCode, entity_id: entityRef, client_id: oidcClientId },
    });
    window.location.replace(returnTo);
  } catch (error) {
    showError(copy.error);
    window.setTimeout(() => {
      const message = error instanceof Error ? error.message : '';
      const loginError = message.includes('configured') ? 'workplace_not_configured' : 'workplace_not_available';
      redirectToLogin(returnTo, loginError);
    }, 900);
  }
}

renderShell();
runWorkplaceContinue();
