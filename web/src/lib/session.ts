// SPDX-License-Identifier: MIT

import { browser } from '$app/environment';

export function redirectToLogin(): void {
  if (browser) {
    window.location.replace('/login');
  }
}

export function redirectToPath(path: string): void {
  if (browser) {
    window.location.href = path;
  }
}

export function redirectToCurrentPath(currentPath: string, search: string): void {
  if (browser) {
    window.location.assign(`${currentPath}${search}`);
  }
}
