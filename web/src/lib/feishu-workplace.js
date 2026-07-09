// SPDX-License-Identifier: MIT

const FEISHU_H5_SDK_URLS = [
  import.meta.env.VITE_FEISHU_H5_SDK_URL?.trim(),
  import.meta.env.PUBLIC_FEISHU_H5_SDK_URL?.trim(),
  'https://lf-scm-cn.feishucdn.com/lark/op/h5-js-sdk-1.5.44.js',
  'https://lf1-cdn-tos.bytegoofy.com/goofy/lark/op/h5-js-sdk-1.5.35.js',
].filter(Boolean);

export function normalizeWorkplaceProvider(provider) {
  const value = (provider || '').trim().toLowerCase();
  if (value === 'lark') return 'feishu';
  return value === 'feishu' ? value : '';
}

export function returnToParam(returnToValue, name) {
  try {
    return new URL(returnToValue, window.location.origin).searchParams.get(name) || '';
  } catch {
    return '';
  }
}

export function queryAuthCode(params, includeOAuthCode = false) {
  return params.get('auth_code') || params.get('feishu_auth_code') || (includeOAuthCode ? params.get('code') || '' : '');
}

export function isFeishuClient() {
  const userAgent = window.navigator?.userAgent?.toLowerCase() || '';
  return userAgent.includes('feishu') || userAgent.includes('lark');
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

function feishuBridgeHandlers(bridgeWindow) {
  const handlers = [];
  const tt = bridgeWindow.tt;
  const h5sdk = bridgeWindow.h5sdk;

  if (tt?.requestAccess) {
    handlers.push({
      kind: 'requestAccess',
      handler: tt.requestAccess,
      owner: tt,
      options: (appId) => ({ appID: appId, scopeList: [] }),
    });
  }

  for (const entry of [
    [tt, tt?.requestAuthCode],
    [tt, tt?.getAuthCode],
    [h5sdk, h5sdk?.requestAuthCode],
    [h5sdk, h5sdk?.getAuthCode],
    [h5sdk?.biz?.auth, h5sdk?.biz?.auth?.getAuthCode],
  ]) {
    const [owner, handler] = entry;
    if (!handler) continue;
    handlers.push({
      kind: 'requestAuthCode',
      handler,
      owner,
      options: (appId) => ({ appId }),
    });
  }

  return handlers;
}

function hasFeishuBridge(bridgeWindow) {
  return feishuBridgeHandlers(bridgeWindow).length > 0;
}

function shouldFallbackFromRequestAccess(error) {
  return Number(error?.errno) === 103 || /not support|unsupported/i.test(errorDetail(error));
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
  if (!isFeishuClient()) throw new Error('bridge_unavailable');
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

export async function feishuAuthCodeFromBridge(appId) {
  await ensureFeishuH5Sdk();
  const requestCode = (handlerIndex = 0) =>
    new Promise((resolve, reject) => {
      const handlers = feishuBridgeHandlers(window);
      const bridge = handlers[handlerIndex];
      if (!bridge) {
        reject(new Error('bridge_unavailable'));
        return;
      }
      try {
        bridge.handler.call(bridge.owner, {
          ...bridge.options(appId),
          success: (response) => {
            const code = response.code || response.auth_code || response.authCode || '';
            if (code) resolve(code);
            else reject(new Error(`auth_code_empty: ${errorDetail(response)}`));
          },
          fail: (error) => {
            if (bridge.kind === 'requestAccess' && handlers[handlerIndex + 1] && shouldFallbackFromRequestAccess(error)) {
              requestCode(handlerIndex + 1).then(resolve).catch(reject);
              return;
            }
            reject(new Error(`auth_code_failed: ${errorDetail(error)}`));
          },
        });
      } catch (error) {
        if (bridge.kind === 'requestAccess' && handlers[handlerIndex + 1] && shouldFallbackFromRequestAccess(error)) {
          requestCode(handlerIndex + 1).then(resolve).catch(reject);
          return;
        }
        reject(error);
      }
    });

  if (window.h5sdk?.ready) {
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => reject(new Error('bridge_timeout')), 10000);
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
