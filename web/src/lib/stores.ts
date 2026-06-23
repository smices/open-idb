// SPDX-License-Identifier: MIT

import { browser } from '$app/environment';
import { writable, type Writable } from 'svelte/store';
import { initLocaleFromStorage as initLocaleInternal, setThemeMode as setThemeModeInternal, getCurrentLocale, getThemeModeFromStorage, type ThemeMode } from './i18n';

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
