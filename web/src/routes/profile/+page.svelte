<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, type CurrentUser } from '$lib/api';
  import { authUser } from '$lib/stores';

  let currentPassword = '';
  let newPassword = '';
  let confirmPassword = '';
  let user: CurrentUser | null = null;
  let submitting = false;

  let error = '';
  let success = '';

  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));
  const maskedIdentifier = (value?: string): string => {
    if (!value) return '-';
    if (value.length <= 8) return value;
    return `${value.slice(0, 4)}...${value.slice(-4)}`;
  };

  const savePassword = async () => {
    error = '';
    success = '';

    if (!currentPassword) {
      error = t('profile.currentPasswordRequired');
      return;
    }

    if (!newPassword) {
      error = t('profile.newPasswordRequired');
      return;
    }

    if (!confirmPassword) {
      error = t('profile.confirmPasswordRequired');
      return;
    }

    if (newPassword.length < 8) {
      error = t('profile.passwordTooShort');
      return;
    }

    if (newPassword !== confirmPassword) {
      error = t('profile.passwordMismatch');
      return;
    }

    submitting = true;
    try {
      await api.updatePassword({ current_password: currentPassword, new_password: newPassword });
      success = t('profile.updateSuccess');
      currentPassword = '';
      newPassword = '';
      confirmPassword = '';
    } catch {
      error = t('profile.currentPasswordError');
    } finally {
      submitting = false;
    }
  };

  $: user = $authUser;
</script>

<svelte:head>
  <title>{t('profile.title')}</title>
</svelte:head>

<section class="space-y-4">
  <div class="grid gap-4 xl:grid-cols-2">
    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3">
      <h2 class="text-lg font-semibold">{t('profile.userInfo')}</h2>
      {#if user}
        {#if user.weak_password}
          <aside class="alert preset-tonal-warning" role="alert"><p>{t('profile.weakPasswordWarning')}</p></aside>
        {/if}
        <dl class="grid gap-3 text-sm">
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('profile.userId')}</dt>
            <dd class="font-medium">{maskedIdentifier(user.id)}</dd>
          </div>
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('profile.entityId')}</dt>
            <dd class="font-medium">{maskedIdentifier(user.entity_id)}</dd>
          </div>
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('users.username')}</dt>
            <dd class="font-medium">{user.username}</dd>
          </div>
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('users.displayName')}</dt>
            <dd class="font-medium">{user.display_name || '-'}</dd>
          </div>
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('users.email')}</dt>
            <dd class="font-medium">{user.email || '-'}</dd>
          </div>
          <div class="flex items-center justify-between gap-4 border-b border-surface-200-800 pb-2">
            <dt class="text-surface-500">{t('users.phone')}</dt>
            <dd class="font-medium">{user.phone || '-'}</dd>
          </div>
          <div class="flex items-center justify-between gap-4">
            <dt class="text-surface-500">{t('users.locale')}</dt>
            <dd class="font-medium">{localeLabel(user.locale)}</dd>
          </div>
        </dl>
      {:else}
        <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
      {/if}
    </section>

    <form class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3" on:submit|preventDefault={savePassword}>
      <div>
        <h2 class="text-lg font-semibold">{t('profile.changePassword')}</h2>
        <p class="mt-1 text-sm text-surface-500">{t('profile.passwordPolicyHint')}</p>
      </div>

      {#if error}
        <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
      {/if}
      {#if success}
        <aside class="alert preset-tonal-primary" role="status"><p>{success}</p></aside>
      {/if}

      <label class="block">
        <span class="text-sm text-surface-500">{t('profile.currentPassword')}</span>
        <input class="input w-full" type="password" bind:value={currentPassword} autocomplete="current-password" required />
      </label>
      <label class="block">
        <span class="text-sm text-surface-500">{t('profile.newPassword')}</span>
        <input class="input w-full" type="password" bind:value={newPassword} autocomplete="new-password" required />
      </label>
      <label class="block">
        <span class="text-sm text-surface-500">{t('profile.confirmPassword')}</span>
        <input class="input w-full" type="password" bind:value={confirmPassword} autocomplete="new-password" required />
      </label>

      <button class="btn preset-filled-primary-500" type="submit" disabled={submitting}>
        {submitting ? t('common.loading') : t('profile.updatePassword')}
      </button>
    </form>

    <section class="card bg-surface-50-950 border border-surface-200-800 p-4 space-y-3 xl:col-span-2">
      <h2 class="text-lg font-semibold">{t('profile.securityState')}</h2>
      <div class="grid gap-3 md:grid-cols-3">
        <article class="card bg-surface-50-950 border border-surface-200-800 p-3">
          <p class="text-xs text-surface-500">{t('profile.passwordState')}</p>
          <p class="mt-1 font-semibold">{user?.weak_password ? t('profile.passwordWeak') : t('profile.passwordNormal')}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-3">
          <p class="text-xs text-surface-500">{t('users.email')}</p>
          <p class="mt-1 break-all font-semibold">{user?.email || '-'}</p>
        </article>
        <article class="card bg-surface-50-950 border border-surface-200-800 p-3">
          <p class="text-xs text-surface-500">{t('users.phone')}</p>
          <p class="mt-1 break-all font-semibold">{user?.phone || '-'}</p>
        </article>
      </div>
    </section>
  </div>
</section>
