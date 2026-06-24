<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n';
  import { api, type Entity } from '$lib/api';
  import IdModal from '$lib/components/ui/IdModal.svelte';
  import { notifyError, notifySuccess } from '$lib/toast';
  import { Pencil } from 'lucide-svelte';

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
  let error = '';

  const statusLabel = (value: string): string => t(`entities.status.${value}`, value);
  const localeLabel = (value: string): string => (value === 'zh-CN' ? t('layout.chinese') : t('layout.english'));

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
    error = '';
    formOpen = true;
  };

  const closeForm = () => {
    formOpen = false;
  };

  const saveForm = async () => {
    saving = true;
    error = '';
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

      notifySuccess(t(editing ? 'common.updateSuccess' : 'common.createSuccess'));
      formOpen = false;
      await fetchEntities();
    } catch {
      notifyError(t('entities.saveFailed'));
    } finally {
      saving = false;
    }
  };

  onMount(fetchEntities);
</script>

<svelte:head>
  <title>{t('entities.title')}</title>
</svelte:head>

<section class="space-y-4">
  <div class="flex items-center justify-end gap-3">
    <button class="btn btn-sm preset-filled-primary-500" type="button" on:click={openCreate}>{t('entities.create')}</button>
  </div>

  {#if error}
    <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
  {/if}

  <section class="card bg-surface-50-950 border border-surface-200-800 overflow-hidden">
    {#if loading}
      <div class="p-6 text-center text-sm text-surface-500">{t('common.loading')}</div>
    {:else if entities.length === 0}
      <div class="p-6 text-sm text-surface-500">{t('entities.noData')}</div>
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
            {#each entities as entity}
              <tr>
                <td>
                  <p class="font-medium">{entity.name}</p>
                </td>
                <td class="font-mono text-sm">{entity.slug}</td>
                <td>
                  <span class={`badge ${entity.status === 'active' ? 'preset-tonal-success' : 'preset-tonal-warning'}`}>{statusLabel(entity.status)}</span>
                </td>
                <td>{localeLabel(entity.default_locale)}</td>
                <td class="whitespace-nowrap">{new Date(entity.created_at).toLocaleString()}</td>
                <td class="text-right">
                  <button class="btn btn-xs preset-outlined-surface-500 inline-grid size-7 min-h-0 min-w-0 place-items-center p-0" type="button" on:click={() => openEdit(entity)} aria-label={t('common.edit')} title={t('common.edit')}>
                    <Pencil class="size-4" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </section>
</section>

<IdModal
  open={formOpen}
  title={editing ? t('entities.editTitle') : t('entities.createTitle')}
  subtitle={editing ? formSlug : t('entities.slug')}
  maxWidth="max-w-xl"
  onClose={closeForm}
>
  <form id="entity-dialog-form" class="space-y-4" on:submit|preventDefault={saveForm}>
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
  </form>

  {#snippet footer()}
    <button class="btn btn-sm preset-outlined-surface-500" type="button" on:click={closeForm}>{t('common.cancel')}</button>
    <button class="btn btn-sm preset-filled-primary-500" type="submit" form="entity-dialog-form" disabled={saving}>{saving ? t('common.loading') : t('common.save')}</button>
  {/snippet}
</IdModal>
