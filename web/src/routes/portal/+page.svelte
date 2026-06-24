<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type UserAccessSummary } from '$lib/api';
  import { t, tf } from '$lib/i18n';
  import { authUser } from '$lib/stores';

  let access: UserAccessSummary | null = null;
  let loading = true;
  let error = '';

  onMount(() => {
    api.myAccess()
      .then((data) => {
        access = data;
      })
      .catch((e) => {
        error = e instanceof Error ? e.message : t('portal.fetchFailed');
      })
      .finally(() => {
        loading = false;
      });
  });

  $: apps = access?.applications.filter((app) => app.has_access) || [];
</script>

<svelte:head>
  <title>{t('portal.title')}</title>
</svelte:head>

{#if loading}
  <section class="card bg-surface-50-950 border border-surface-200-800 space-y-3 p-5" aria-busy="true">
    <div class="h-5 w-32 rounded bg-surface-200-800"></div>
    <div class="h-4 w-full max-w-xl rounded bg-surface-200-800"></div>
    <div class="grid gap-4 pt-4 md:grid-cols-2 xl:grid-cols-3">
      <div class="h-36 rounded border border-surface-200-800 bg-surface-100-900"></div>
      <div class="h-36 rounded border border-surface-200-800 bg-surface-100-900"></div>
      <div class="h-36 rounded border border-surface-200-800 bg-surface-100-900"></div>
    </div>
  </section>
{:else}
  <div class="space-y-6">
    <section>
      <h1 class="text-2xl font-semibold tracking-normal text-surface-950-50">{t('portal.title')}</h1>
      <p class="mt-3 max-w-2xl text-sm leading-6 text-surface-600-400">
        {tf('portal.description', { name: $authUser?.display_name || $authUser?.username || '-' })}
      </p>
    </section>

    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}

    {#if apps.length}
      <section class="card overflow-hidden border border-surface-200-800 bg-surface-50-950">
        <table class="table min-w-full">
          <thead>
            <tr>
              <th scope="col">{t('portal.application')}</th>
              <th scope="col">{t('portal.type')}</th>
              <th scope="col" class="w-28 !text-center">{t('portal.access')}</th>
            </tr>
          </thead>
          <tbody>
            {#each apps as app}
              <tr>
                <td>
                  <p class="font-medium">{app.application_name}</p>
                  <p class="text-xs text-surface-500">{tf('portal.roleCount', { count: String(app.roles.length) })}</p>
                </td>
                <td class="whitespace-nowrap text-sm text-surface-600-400">{app.application_type}</td>
                <td class="w-28 !text-center">
                  <span class="mx-auto inline-flex items-center justify-center gap-1.5 text-sm text-success-600-400">
                    <span class="size-2 rounded-full bg-success-500" aria-hidden="true"></span>
                    {t('portal.accessible')}
                  </span>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>
    {:else}
      <section class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">
        {t('portal.noApplications')}
      </section>
    {/if}
  </div>
{/if}
