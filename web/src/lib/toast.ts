// SPDX-License-Identifier: MIT

import { createToaster } from '@skeletonlabs/skeleton-svelte';

export const toaster = createToaster({
  placement: 'top',
  duration: 2600,
  gap: 8,
});

export const notifySuccess = (description: string) => {
  if (!description) return;
  toaster.create({ type: 'success', description, closable: true });
};

export const notifyError = (description: string) => {
  if (!description) return;
  toaster.create({ type: 'error', description, closable: true });
};
