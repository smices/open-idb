<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type Entity } from '$lib/api';

  const localeOptions = ['en-US', 'zh-CN'];
  const statusOptions = ['active', 'disabled'];

  let entities: Entity[] = [];
  let loading = false;
  let formOpen = false;
  let editing: Entity | null = null;
  let formName = '';
  let formSlug = '';
  let formStatus = 'active';
  let formDefaultLocale = 'en-US';
  let formBrandName = '';
  let formLogoUrl = '';
  let formLoginMessage = '';
  let saving = false;
  let message = '';
  let error = '';
  let entitySearch = '';
  let statusFilter = 'all';

  const includesQuery = (value: unknown, query: string): boolean =>
    String(value ?? '').toLowerCase().includes(query.trim().toLowerCase());

  const statusLabel = (value: string): string => t(`entities.status.${value}`, value);
  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));

  const matchesEntitySearch = (item: Entity, query: string): boolean => {
    if (!query.trim()) return true;
    return [item.id, item.name, item.slug, item.status, statusLabel(item.status), item.default_locale, localeLabel(item.default_locale)].some((value) =>
      includesQuery(value, query),
    );
  };

  const fetchEntities = async () => {
    loading = true;
    error = '';
    try {
      const data = await api.listEntities({ limit: 200 });
      entities = data.items || [];
    } catch {
      error = t('entities.fetchFailed');
    } finally {
      loading = false;
    }
  };

  const openCreate = () => {
    editing = null;
    formName = '';
    formSlug = '';
    formStatus = 'active';
    formDefaultLocale = 'en-US';
    formBrandName = '';
    formLogoUrl = '';
    formLoginMessage = '';
    message = '';
    error = '';
    formOpen = true;
  };

  const openEdit = (entity: Entity) => {
    editing = entity;
    formName = entity.name;
    formSlug = entity.slug;
    formStatus = entity.status;
    formDefaultLocale = entity.default_locale;
    formBrandName = entity.brand_name || '';
    formLogoUrl = entity.logo_url || '';
    formLoginMessage = entity.login_message || '';
    message = '';
    error = '';
    formOpen = true;
  };

  const closeForm = () => {
    formOpen = false;
  };

  const resetFilters = () => {
    entitySearch = '';
    statusFilter = 'all';
  };

  const handleDialogKeydown = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && formOpen) {
      closeForm();
    }
  };

  const saveForm = async () => {
    saving = true;
    error = '';
    message = '';
    try {
      if (editing) {
        await api.updateEntity(editing.id, {
          name: formName,
          status: formStatus,
          default_locale: formDefaultLocale,
          brand_name: formBrandName,
          logo_url: formLogoUrl,
          login_message: formLoginMessage,
        });
      } else {
        await api.createEntity({
          name: formName,
          slug: formSlug,
          default_locale: formDefaultLocale,
          brand_name: formBrandName,
          logo_url: formLogoUrl,
          login_message: formLoginMessage,
        });
      }

      message = t(editing ? 'common.updateSuccess' : 'common.createSuccess');
      formOpen = false;
      await fetchEntities();
    } catch {
      error = t('entities.saveFailed');
    } finally {
      saving = false;
    }
  };

  onMount(fetchEntities);

  $: filteredEntities = entities.filter((item) => {
    const searchMatches = matchesEntitySearch(item, entitySearch);
    const statusMatches = statusFilter === 'all' || item.status === statusFilter;
    return searchMatches && statusMatches;
  });
  $: activeEntityCount = entities.filter((item) => item.status === 'active').length;
  $: disabledEntityCount = entities.filter((item) => item.status === 'disabled').length;
  $: englishEntityCount = entities.filter((item) => item.default_locale === 'en-US').length;
  $: chineseEntityCount = entities.filter((item) => item.default_locale === 'zh-CN').length;
</script>

<svelte:head>
  <title>{t('entities.title')}</title>
</svelte:head>

<svelte:window on:keydown={handleDialogKeydown} />

<section class="space-y-4">
  <div class="flex items-center justify-end gap-3">
    <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreate}>{t('entities.create')}</button>
  </div>

  <form class="card bg-surface-50-950 border border-surface-200-800 grid gap-3 p-4 md:grid-cols-[minmax(0,1fr)_minmax(0,16rem)_auto]" on:submit|preventDefault>
    <label class="flex flex-col gap-2">
      <span class="text-sm leading-5 text-surface-500">{t('entities.search')}</span>
      <input class="input w-full" type="search" bind:value={entitySearch} placeholder={t('entities.searchPlaceholder')} />
    </label>
    <label class="flex flex-col gap-2">
      <span class="text-sm leading-5 text-surface-500">{t('entities.status')}</span>
      <select class="input w-full" bind:value={statusFilter}>
        <option value="all">{t('common.all')}</option>
        {#each statusOptions as status}
          <option value={status}>{statusLabel(status)}</option>
        {/each}
      </select>
    </label>
    <div class="flex flex-wrap items-end gap-2">
      <button class="btn preset-filled-primary-500" type="submit">{t('common.filter')}</button>
      <button class="btn preset-outlined-surface-500" type="button" on:click={resetFilters}>{t('common.reset')}</button>
    </div>
  </form>

  {#if message}
    <aside class="alert preset-tonal-primary" role="status"><p>{message}</p></aside>
  {/if}
  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    <div class="grid gap-3 border-b border-surface-200-800 p-4 text-sm sm:grid-cols-2 lg:grid-cols-5">
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('entities.visibleRows')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{`${filteredEntities.length} / ${entities.length}`}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('entities.activeEntities')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{activeEntityCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('entities.disabledEntities')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{disabledEntityCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('entities.defaultEnglish')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{englishEntityCount}</p></article>
      <article class="card bg-surface-50-950 border border-surface-200-800 p-4"><p class="text-xs text-surface-500">{t('entities.defaultChinese')}</p><p class="mt-2 text-2xl font-semibold tabular-nums">{chineseEntityCount}</p></article>
    </div>
    {#if loading}
      <div class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if filteredEntities.length === 0}
      <div class="p-6 text-sm text-surface-500">{entities.length === 0 ? t('entities.noData') : t('entities.noSearchResults')}</div>
    {:else}
      <div class="overflow-x-auto">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th scope="col">{t('entities.name')}</th>
              <th scope="col">{t('entities.slug')}</th>
              <th scope="col">{t('entities.status')}</th>
              <th scope="col">{t('entities.defaultLocale')}</th>
              <th scope="col">{t('entities.createdAt')}</th>
              <th scope="col" class="text-right">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {#each filteredEntities as entity}
              <tr>
                <td>
                  <p class="font-medium">{entity.name}</p>
                  <p class="text-xs text-surface-500 break-all">{entity.id}</p>
                </td>
                <td class="font-mono text-sm">{entity.slug}</td>
                <td>
                  <span class={`badge ${entity.status === 'active' ? 'preset-tonal-success' : 'preset-tonal-warning'}`}>{statusLabel(entity.status)}</span>
                </td>
                <td>{localeLabel(entity.default_locale)}</td>
                <td class="whitespace-nowrap">{new Date(entity.created_at).toLocaleString()}</td>
                <td class="text-right">
                  <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={() => openEdit(entity)}>{t('common.edit')}</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</section>

{#if formOpen}
  <div class="fixed inset-0 z-40 bg-surface-950/55 backdrop-blur-sm" aria-hidden="true" on:click={closeForm}></div>
  <div class="fixed inset-y-0 right-0 z-50 flex w-full justify-end" role="dialog" aria-modal="true" aria-labelledby="entity-form-title" tabindex="-1">
    <form
      class="flex h-full w-full max-w-xl flex-col border-l border-surface-200-800 bg-surface-50-950 text-surface-950-50 shadow-2xl"
      on:submit|preventDefault={saveForm}
    >
      <header class="flex items-center justify-between gap-3 border-b border-surface-200-800 px-5 py-4">
        <div>
          <h2 id="entity-form-title" class="text-lg font-semibold">{editing ? t('entities.editTitle') : t('entities.createTitle')}</h2>
          <p class="mt-1 text-sm text-surface-500">{editing ? formSlug : t('entities.slug')}</p>
        </div>
        <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeForm} aria-label={t('common.close')}>{t('common.close')}</button>
      </header>

      <div class="flex-1 space-y-4 overflow-y-auto px-5 py-5">
        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.name')}</span>
          <input class="input w-full bg-surface-50-950" type="text" bind:value={formName} required />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.slug')}</span>
          <input class="input w-full bg-surface-50-950 font-mono" type="text" bind:value={formSlug} required disabled={!!editing} />
        </label>

        {#if editing}
          <label class="block">
            <span class="text-sm text-surface-500">{t('entities.status')}</span>
            <select class="input w-full bg-surface-50-950" bind:value={formStatus}>
              {#each statusOptions as status}
                <option value={status}>{statusLabel(status)}</option>
              {/each}
            </select>
          </label>
        {/if}

        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.defaultLocale')}</span>
          <select class="input w-full bg-surface-50-950" bind:value={formDefaultLocale}>
            {#each localeOptions as locale}
              <option value={locale}>{localeLabel(locale)}</option>
            {/each}
          </select>
        </label>

        <div class="border-t border-surface-200-800 pt-4">
          <h3 class="text-sm font-semibold">{t('entities.branding')}</h3>
          <p class="mt-1 text-sm text-surface-500">{t('entities.brandingHelp')}</p>
        </div>

        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.brandName')}</span>
          <input class="input w-full bg-surface-50-950" type="text" bind:value={formBrandName} placeholder={formName} />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.logoUrl')}</span>
          <input class="input w-full bg-surface-50-950" type="url" bind:value={formLogoUrl} placeholder="https://example.com/logo.png" />
        </label>

        <label class="block">
          <span class="text-sm text-surface-500">{t('entities.loginMessage')}</span>
          <textarea class="textarea w-full bg-surface-50-950" rows="3" bind:value={formLoginMessage} placeholder={t('entities.loginMessagePlaceholder')}></textarea>
        </label>
      </div>

      <footer class="flex justify-end gap-2 border-t border-surface-200-800 bg-surface-100-900 px-5 py-4">
        <button class="btn preset-outlined-surface-500" type="button" on:click={closeForm}>{t('common.cancel')}</button>
        <button class="btn preset-filled-primary-500" type="submit" disabled={saving}>{saving ? t('common.loading') : t('common.save')}</button>
      </footer>
    </form>
  </div>
{/if}
