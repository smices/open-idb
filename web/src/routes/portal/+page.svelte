<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type UserAccessSummary } from '$lib/api';
  import { authUser } from '$lib/stores';
  import { AppWindow, ShieldCheck } from 'lucide-svelte';

  let access: UserAccessSummary | null = null;
  let loading = true;
  let error = '';

  onMount(() => {
    api.myAccess()
      .then((data) => {
        access = data;
      })
      .catch((e) => {
        error = String(e || '加载应用失败');
      })
      .finally(() => {
        loading = false;
      });
  });

  $: apps = access?.applications.filter((app) => app.has_access) || [];
</script>

<svelte:head>
  <title>用户门户</title>
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
      <h1 class="text-2xl font-semibold tracking-normal text-surface-950-50">可访问应用</h1>
      <p class="mt-3 max-w-2xl text-sm leading-6 text-surface-600-400">
        {$authUser?.display_name || $authUser?.username} 当前可访问的应用会显示在这里。
      </p>
    </section>

    {#if error}
      <aside class="alert preset-tonal-error" role="alert"><p>{error}</p></aside>
    {/if}

    {#if apps.length}
      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {#each apps as app}
          <article class="card bg-surface-50-950 border border-surface-200-800 p-4">
            <div class="flex items-start justify-between gap-3">
              <span class="inline-flex size-11 items-center justify-center rounded bg-surface-100-900 text-primary-600-400">
                <AppWindow size={21} aria-hidden="true" />
              </span>
              <span class={`badge ${app.application_type === 'oidc_client' ? 'preset-tonal-primary' : 'preset-outlined-surface-500'}`}>{app.application_type}</span>
            </div>
            <h2 class="mt-4 text-lg font-semibold">{app.application_name}</h2>
            <p class="mt-2 text-sm leading-6 text-surface-600-400">已授权访问，包含 {app.roles.length} 个角色上下文。</p>
            <div class="mt-4 flex items-center gap-2 text-sm font-medium text-success-600-400">
              <ShieldCheck size={16} aria-hidden="true" />
              <span>可访问</span>
            </div>
          </article>
        {/each}
      </section>
    {:else}
      <section class="card bg-surface-50-950 border border-surface-200-800 p-6 text-center text-sm text-surface-500">
        当前账号还没有可访问应用。
      </section>
    {/if}
  </div>
{/if}
