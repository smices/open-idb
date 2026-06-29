// SPDX-License-Identifier: MIT

import { browser } from '$app/environment';
import { writable, type Writable } from 'svelte/store';
import { initLocaleFromStorage as initLocaleInternal, setThemeMode as setThemeModeInternal, getCurrentLocale, getThemeModeFromStorage, type ThemeMode } from './i18n';
import type { PlatformBranding } from './api';

export interface UserSummary {
  id: string;
  entity_id: string;
  username: string;
  display_name: string;
  email?: string;
  phone?: string;
  avatar_url?: string;
  locale: string;
  weak_password?: boolean;
  console_scope?: 'user' | 'enterprise_admin';
  capabilities?: Array<'user' | 'enterprise' | 'system'>;
}

export const authLoading: Writable<boolean> = writable(true);
export const authUser: Writable<UserSummary | null> = writable(null);
export const platformBranding: Writable<PlatformBranding> = writable({
  platform_name: 'IdBridge',
  logo_url: '',
  favicon_url: '',
  title_suffix: '',
});
export const sidebarCollapsed: Writable<boolean> = writable(
  browser ? localStorage.getItem('idb-sidebar-collapsed') === '1' : false,
);
export const themeMode: Writable<ThemeMode> = writable(browser ? getThemeModeFromStorage() : 'light');

themeMode.subscribe((mode) => {
  if (browser) {
    setThemeModeInternal(mode);
  }
});

export function initLocaleFromStorage(): void {
  if (browser) {
    initLocaleInternal();
  }
}

export function toggleSidebar(): void {
  sidebarCollapsed.update((value) => {
    const next = !value;
    localStorage.setItem('idb-sidebar-collapsed', next ? '1' : '0');
    return next;
  });
}

export function getLocale(): 'en-US' | 'zh-CN' {
  return getCurrentLocale();
}

export function setTheme(mode: ThemeMode): void {
  themeMode.set(mode);
}

export function setPlatformBranding(value: PlatformBranding): void {
  platformBranding.set({
    platform_name: value.platform_name || 'IdBridge',
    logo_url: value.logo_url || '',
    favicon_url: value.favicon_url || '',
    title_suffix: value.title_suffix || '',
    updated_at: value.updated_at,
  });
}
