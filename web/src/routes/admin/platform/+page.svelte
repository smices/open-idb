<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { t } from '$lib/i18n';
  import { platformBranding, setPlatformBranding } from '$lib/stores';
  import { notifyError, notifySuccess } from '$lib/toast';

  let platformName = '';
  let logoUrl = '';
  let faviconUrl = '';
  let titleSuffix = '';
  let loading = true;
  let saving = false;

  const fillForm = () => {
    platformName = $platformBranding.platform_name || 'IdBridge';
    logoUrl = $platformBranding.logo_url || '';
    faviconUrl = $platformBranding.favicon_url || '';
    titleSuffix = $platformBranding.title_suffix || '';
  };

  const loadBranding = async () => {
    loading = true;
    try {
      const branding = await api.getAdminPlatformBranding();
      setPlatformBranding(branding);
      fillForm();
    } catch {
      notifyError(t('platform.fetchFailed'));
    } finally {
      loading = false;
    }
  };

  const saveBranding = async () => {
    if (!platformName.trim()) {
      notifyError(t('platform.nameRequired'));
      return;
    }
    saving = true;
    try {
      const branding = await api.updatePlatformBranding({
        platform_name: platformName.trim(),
        logo_url: logoUrl.trim(),
        favicon_url: faviconUrl.trim(),
        title_suffix: titleSuffix.trim(),
      });
      setPlatformBranding(branding);
      fillForm();
      notifySuccess(t('common.updateSuccess'));
    } catch {
      notifyError(t('platform.saveFailed'));
    } finally {
      saving = false;
    }
  };

  onMount(loadBranding);
</script>

<svelte:head>
  <title>{t('platform.title')}</title>
</svelte:head>

<section class="max-w-3xl space-y-4">
  <form class="card bg-surface-50-950 border border-surface-200-800 p-4" on:submit|preventDefault={saveBranding}>
    {#if loading}
      <div class="space-y-3" aria-label={t('common.loading')}>
        <div class="h-9 w-48 rounded bg-surface-200-800"></div>
        <div class="h-10 rounded bg-surface-200-800"></div>
        <div class="h-10 rounded bg-surface-200-800"></div>
        <div class="h-10 rounded bg-surface-200-800"></div>
      </div>
    {:else}
      <div class="grid gap-4">
        <label class="block">
          <span class="text-sm text-surface-500">{t('platform.name')}</span>
          <input class="input w-full bg-surface-50-950" type="text" bind:value={platformName} required />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('platform.logoUrl')}</span>
          <input class="input w-full bg-surface-50-950" type="url" bind:value={logoUrl} placeholder="https://example.com/logo.svg" />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('platform.faviconUrl')}</span>
          <input class="input w-full bg-surface-50-950" type="url" bind:value={faviconUrl} placeholder="https://example.com/favicon.ico" />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('platform.titleSuffix')}</span>
          <input class="input w-full bg-surface-50-950" type="text" bind:value={titleSuffix} placeholder={t('platform.titleSuffixPlaceholder')} />
        </label>
      </div>

      <div class="mt-5 flex justify-end">
        <button class="btn btn-sm preset-filled-primary-500" type="submit" disabled={saving}>{saving ? t('common.loading') : t('common.save')}</button>
      </div>
    {/if}
  </form>

  <section class="card bg-surface-50-950 border border-surface-200-800 p-4">
    <h2 class="text-sm font-semibold">{t('platform.preview')}</h2>
    <div class="mt-4 flex items-center gap-3 rounded-container border border-surface-200-800 bg-surface-100-900 p-3">
      <span class="inline-flex size-10 items-center justify-center rounded-container border border-surface-200-800 bg-surface-50-950">
        <img class="size-7 object-contain" src={logoUrl || '/logo.svg'} alt="" aria-hidden="true" />
      </span>
      <div class="min-w-0">
        <p class="truncate text-sm font-semibold">{platformName || 'IdBridge'}</p>
        <p class="truncate text-xs text-surface-500">{titleSuffix || t('platform.defaultTitleSuffix')}</p>
      </div>
    </div>
  </section>
</section>
