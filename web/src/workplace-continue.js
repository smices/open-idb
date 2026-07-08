// SPDX-License-Identifier: MIT

const API_TARGET = import.meta.env.VITE_API_TARGET || import.meta.env.PUBLIC_API_TARGET || '';
const FEISHU_H5_SDK_URLS = [
  import.meta.env.VITE_FEISHU_H5_SDK_URL?.trim(),
  import.meta.env.PUBLIC_FEISHU_H5_SDK_URL?.trim(),
  'https://lf1-cdn-tos.bytegoofy.com/goofy/lark/op/h5-js-sdk-1.5.35.js',
].filter(Boolean);

const steps = [
  { key: 'context', label: '解析访问上下文', detail: '正在确认应用与企业身份' },
  { key: 'feishu', label: '获取飞书授权', detail: '正在连接飞书工作台' },
  { key: 'enter', label: '进入应用', detail: '正在完成登录并跳转' },
];

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

function queryString(params) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params || {})) {
    if (value === undefined || value === null || value === '') continue;
    search.set(key, String(value));
  }
  return search.toString() ? `?${search.toString()}` : '';
}

function queryAuthCode(params) {
  return params.get('auth_code') || params.get('feishu_auth_code') || params.get('code') || '';
}

function errorDetail(error) {
  if (!error) return '';
  if (typeof error === 'string') return error;
  if (error instanceof Error) return error.message;
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
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

function feishuBridgeHandler(bridgeWindow) {
  return (
    bridgeWindow.tt?.requestAuthCode ||
    bridgeWindow.tt?.getAuthCode ||
    bridgeWindow.h5sdk?.requestAuthCode ||
    bridgeWindow.h5sdk?.getAuthCode ||
    bridgeWindow.h5sdk?.biz?.auth?.getAuthCode
  );
}

function hasFeishuBridge(bridgeWindow) {
  return Boolean(feishuBridgeHandler(bridgeWindow) || bridgeWindow.h5sdk?.ready);
}

function loadScript(src) {
  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[data-idb-feishu-sdk="${src}"]`);
    if (existing?.dataset.loaded === 'true') {
      resolve();
      return;
    }
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true });
      existing.addEventListener('error', () => reject(new Error(`failed to load ${src}`)), { once: true });
      return;
    }
    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.dataset.idbFeishuSdk = src;
    script.onload = () => {
      script.dataset.loaded = 'true';
      resolve();
    };
    script.onerror = () => reject(new Error(`failed to load ${src}`));
    document.head.appendChild(script);
  });
}

function waitForFeishuBridge(timeoutMs = 3000) {
  return new Promise((resolve, reject) => {
    const startedAt = Date.now();
    const check = () => {
      if (hasFeishuBridge(window)) {
        resolve();
        return;
      }
      if (Date.now() - startedAt >= timeoutMs) {
        reject(new Error('bridge_unavailable'));
        return;
      }
      window.setTimeout(check, 100);
    };
    check();
  });
}

async function ensureFeishuH5Sdk() {
  if (hasFeishuBridge(window)) return;
  let lastError = null;
  for (const src of FEISHU_H5_SDK_URLS) {
    try {
      await loadScript(src);
      await waitForFeishuBridge();
      return;
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError || new Error('sdk_load_failed');
}

async function feishuAuthCodeFromBridge(appId) {
  await ensureFeishuH5Sdk();
  const requestCode = () =>
    new Promise((resolve, reject) => {
      const handler = feishuBridgeHandler(window);
      if (!handler) {
        reject(new Error('bridge_unavailable'));
        return;
      }
      handler({
        appId,
        success: (response) => {
          const code = response.code || response.auth_code || response.authCode || '';
          if (code) resolve(code);
          else reject(new Error(`auth_code_empty: ${errorDetail(response)}`));
        },
        fail: (error) => reject(new Error(`auth_code_failed: ${errorDetail(error)}`)),
      });
    });

  if (window.h5sdk?.ready) {
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => reject(new Error('bridge_timeout')), 10000);
      window.h5sdk?.error?.((error) => {
        window.clearTimeout(timer);
        reject(new Error(`auth_code_failed: ${errorDetail(error)}`));
      });
      window.h5sdk.ready(() => {
        requestCode()
          .then((code) => {
            window.clearTimeout(timer);
            resolve(code);
          })
          .catch((error) => {
            window.clearTimeout(timer);
            reject(error);
          });
      });
    });
  }
  return requestCode();
}

function injectStyles() {
  const style = document.createElement('style');
  style.textContent = `
    :root {
      color-scheme: light;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: #f5f7f6;
      color: #111827;
    }

    * {
      box-sizing: border-box;
    }

    body {
      margin: 0;
      min-width: 320px;
      min-height: 100dvh;
      background: #f5f7f6;
    }

    .workplace-loading {
      position: relative;
      display: grid;
      min-height: 100dvh;
      overflow: hidden;
      padding: 28px;
      place-items: center;
    }

    .workplace-loading::before,
    .workplace-loading::after {
      position: absolute;
      content: "";
      border: 1px solid rgba(17, 24, 39, 0.08);
      pointer-events: none;
    }

    .workplace-loading::before {
      inset: 24px;
    }

    .workplace-loading::after {
      inset: 36px;
      border-color: rgba(15, 118, 110, 0.16);
    }

    .workplace-panel {
      position: relative;
      z-index: 1;
      width: min(520px, 100%);
      padding: 36px;
      border: 1px solid rgba(17, 24, 39, 0.12);
      border-radius: 8px;
      background: rgba(255, 255, 255, 0.88);
      box-shadow: 0 24px 80px rgba(17, 24, 39, 0.12);
    }

    .workplace-brand {
      display: flex;
      align-items: center;
      gap: 12px;
      min-width: 0;
      color: #111827;
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
      color: #0f766e;
      font-size: 13px;
      font-weight: 700;
    }

    .workplace-title {
      margin: 0;
      color: #111827;
      font-size: 30px;
      font-weight: 760;
      letter-spacing: 0;
      line-height: 1.15;
      text-wrap: balance;
    }

    .workplace-detail {
      min-height: 52px;
      margin: 14px 0 0;
      color: #4b5563;
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
      background: #e5e7eb;
    }

    .workplace-progress span {
      position: absolute;
      inset: 0 auto 0 0;
      width: 38%;
      border-radius: inherit;
      background: #0f766e;
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
      color: #6b7280;
      font-size: 14px;
    }

    .workplace-step-dot {
      display: grid;
      width: 22px;
      height: 22px;
      place-items: center;
      border: 1px solid #d1d5db;
      border-radius: 999px;
      background: #fff;
    }

    .workplace-step-dot::after {
      width: 6px;
      height: 6px;
      border-radius: 999px;
      background: #d1d5db;
      content: "";
    }

    .workplace-step.is-active {
      color: #111827;
      font-weight: 650;
    }

    .workplace-step.is-active .workplace-step-dot {
      border-color: rgba(15, 118, 110, 0.32);
      background: #ecfdf5;
    }

    .workplace-step.is-active .workplace-step-dot::after {
      background: #0f766e;
      animation: idb-workplace-pulse 1.2s ease-in-out infinite;
    }

    .workplace-step.is-done {
      color: #374151;
    }

    .workplace-step.is-done .workplace-step-dot {
      border-color: #0f766e;
      background: #0f766e;
    }

    .workplace-step.is-done .workplace-step-dot::after {
      width: 8px;
      height: 8px;
      border-radius: 0;
      background: transparent;
      border-right: 2px solid #fff;
      border-bottom: 2px solid #fff;
      transform: translateY(-1px) rotate(45deg);
    }

    .workplace-error {
      display: none;
      margin-top: 22px;
      padding: 12px 14px;
      border: 1px solid rgba(217, 119, 6, 0.28);
      border-radius: 8px;
      background: #fffbeb;
      color: #92400e;
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
  const root = document.getElementById('root') || document.body.appendChild(document.createElement('div'));
  root.innerHTML = `
    <main class="workplace-loading" aria-busy="true">
      <section class="workplace-panel" aria-labelledby="workplace-loading-title">
        <div class="workplace-brand">
          <img data-brand-logo src="/logo.svg" alt="" />
          <span data-brand-name>IdBridge</span>
        </div>
        <div class="workplace-copy">
          <p class="workplace-eyebrow">飞书工作台 SSO</p>
          <h1 class="workplace-title" id="workplace-loading-title">正在进入应用</h1>
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

    const authCode = queryAuthCode(params) || (await feishuAuthCodeFromBridge(provider.app_id));
    setStep(2);
    await apiRequest('/api/auth/feishu/exchange', {
      method: 'POST',
      body: { auth_code: authCode, entity_id: entityRef, client_id: oidcClientId },
    });
    window.location.replace(returnTo);
  } catch (error) {
    showError('进入应用失败，正在返回登录页。');
    window.setTimeout(() => {
      const message = error instanceof Error ? error.message : '';
      const loginError = message.includes('configured') ? 'workplace_not_configured' : 'workplace_not_available';
      redirectToLogin(returnTo, loginError);
    }, 900);
  }
}

renderShell();
runWorkplaceContinue();
