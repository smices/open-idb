// SPDX-License-Identifier: MIT

export function safeReturnTo(value, fallback = '/portal', origin = globalThis.window?.location?.origin) {
  const candidate = String(value || '').trim();
  if (!candidate.startsWith('/') || candidate.startsWith('//') || !origin) return fallback;

  try {
    const base = new URL(origin);
    const resolved = new URL(candidate, base);
    if (resolved.origin !== base.origin) return fallback;
    const normalized = `${resolved.pathname}${resolved.search}`;
    if (!normalized.startsWith('/') || normalized.startsWith('//')) return fallback;
    return normalized;
  } catch {
    return fallback;
  }
}
