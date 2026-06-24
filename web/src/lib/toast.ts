// SPDX-License-Identifier: MIT

import { createToaster } from '@skeletonlabs/skeleton-svelte';
import { writable, type Writable } from 'svelte/store';

type ToastType = 'success' | 'error' | 'info' | 'warning';

export type IdToast = {
  id: string;
  type: ToastType;
  title: string;
};

type ToastGlobal = {
  toaster?: ReturnType<typeof createToaster>;
  messages?: Writable<IdToast[]>;
};

const toastGlobal = globalThis as typeof globalThis & { __idbridgeToast?: ToastGlobal };
toastGlobal.__idbridgeToast ??= {};

export const toaster =
  toastGlobal.__idbridgeToast.toaster ??
  (toastGlobal.__idbridgeToast.toaster = createToaster({
    placement: 'top',
    duration: 2600,
    gap: 8,
  }));

export const toastMessages = toastGlobal.__idbridgeToast.messages ?? (toastGlobal.__idbridgeToast.messages = writable<IdToast[]>([]));

export const dismissToast = (id: string) => {
  toastMessages.update((items) => items.filter((item) => item.id !== id));
  toaster.remove(id);
};

const createToast = (type: ToastType, message: string) => {
  if (!message) return;
  const id = toaster.create({ type, title: message, closable: true });
  toastMessages.update((items) => [{ id, type, title: message }, ...items].slice(0, 4));
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent<IdToast>('idbridge-toast', { detail: { id, type, title: message } }));
    window.setTimeout(() => dismissToast(id), 3600);
  }
};

export const notifySuccess = (message: string) => {
  createToast('success', message);
};

export const notifyError = (message: string) => {
  createToast('error', message);
};
