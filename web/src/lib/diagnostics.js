const SENSITIVE_QUERY_KEYS = new Set(['code', 'auth_code', 'access_token', 'id_token', 'refresh_token', 'token', 'state', 'nonce', 'client_secret', 'cookie']);

function redactedRoute(value) {
  try {
    const url = new URL(value, 'https://diagnostic.invalid');
    for (const key of [...url.searchParams.keys()]) {
      if (SENSITIVE_QUERY_KEYS.has(key.toLowerCase())) url.searchParams.delete(key);
    }
    return `${url.pathname}${url.search}`;
  } catch {
    return '';
  }
}

function safeValue(value) {
  return String(value || '').replace(/[\r\n]/g, ' ').trim();
}

export function diagnosticCode(error, fallback = 'workplace_failed') {
  const candidate = safeValue(error?.code || error?.message || fallback).split(':')[0];
  return /^[a-z0-9_-]{2,80}$/i.test(candidate) ? candidate : fallback;
}

export function formatDiagnostic({ code, stage, route, error, timestamp = new Date().toISOString(), userAgent = globalThis.navigator?.userAgent || '' }) {
  const fields = {
    code: diagnosticCode({ code }, 'workplace_failed'),
    stage: safeValue(stage) || 'unknown',
    route: redactedRoute(route),
    http_status: Number.isInteger(error?.status) ? String(error.status) : '',
    trace_id: safeValue(error?.traceId),
    timestamp: safeValue(timestamp),
    user_agent: safeValue(userAgent),
  };
  return Object.entries(fields)
    .filter(([, value]) => value)
    .map(([key, value]) => `${key}=${value}`)
    .join('\n');
}

export async function copyDiagnostic(input) {
  const text = typeof input === 'string' ? input : formatDiagnostic(input);
  if (globalThis.navigator?.clipboard?.writeText) {
    await globalThis.navigator.clipboard.writeText(text);
    return text;
  }
  const documentRef = globalThis.document;
  if (!documentRef?.body) throw new Error('clipboard_unavailable');
  const textarea = documentRef.createElement('textarea');
  textarea.value = text;
  textarea.setAttribute('readonly', '');
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  documentRef.body.append(textarea);
  textarea.select();
  const copied = documentRef.execCommand('copy');
  textarea.remove();
  if (!copied) throw new Error('clipboard_copy_failed');
  return text;
}
