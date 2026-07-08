// SPDX-License-Identifier: MIT

const FEISHU_H5_SDK_URLS = [
  import.meta.env.VITE_FEISHU_H5_SDK_URL?.trim(),
  import.meta.env.PUBLIC_FEISHU_H5_SDK_URL?.trim(),
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

export async function feishuAuthCodeFromBridge(appId) {
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
