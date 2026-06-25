<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { t } from '$lib/i18n';
  import { ArrowRight, Building2, FileCheck2, KeyRound, ShieldCheck } from 'lucide-svelte';

  const metrics = [
    { value: 'SSO', label: 'login.metric.sso' },
    { value: 'RBAC', label: 'login.metric.access' },
    { value: 'Audit', label: 'login.metric.audit' },
  ];

  const assurances = [
    { icon: ShieldCheck, title: 'login.assurance.sso', description: 'login.assurance.ssoDescription' },
    { icon: KeyRound, title: 'login.assurance.access', description: 'login.assurance.accessDescription' },
    { icon: FileCheck2, title: 'login.assurance.audit', description: 'login.assurance.auditDescription' },
  ];
</script>

<svelte:head>
  <title>{t('login.homeTitle')}</title>
</svelte:head>

<main class="glass-page-dark min-h-dvh px-5 py-6 text-surface-950-50 lg:px-12">
  <section class="mx-auto flex min-h-[calc(100dvh-3rem)] max-w-7xl flex-col gap-8">
    <header class="flex items-center justify-between gap-4">
      <a class="inline-flex items-center gap-3 text-surface-950-50 no-underline" href="/" aria-label={t('app.title')}>
        <span class="preset-glass-primary inline-flex size-10 items-center justify-center rounded-container">
          <img class="size-7" src="/logo.svg" alt="" aria-hidden="true" />
        </span>
        <span class="text-sm font-semibold tracking-wide">{t('app.title')}</span>
      </a>
    </header>

    <div class="grid flex-1 items-center gap-10 lg:grid-cols-[minmax(0,1fr)_27rem]">
      <section class="max-w-3xl py-8">
        <p class="text-sm font-semibold text-primary-200">{t('login.eyebrow')}</p>
        <h1 class="mt-5 text-4xl font-semibold leading-tight text-balance sm:text-5xl lg:text-6xl">{t('login.homeTitle')}</h1>
        <p class="mt-5 max-w-2xl text-base leading-8 text-pretty text-surface-700-300">{t('login.homeSubtitle')}</p>

        <div class="mt-8 flex flex-wrap gap-3">
          <a class="btn preset-filled-primary-500 h-12 justify-center gap-2 rounded-container px-5 text-base font-semibold" href="/login">
            {t('login.primaryCta')}
            <ArrowRight size={18} aria-hidden="true" />
          </a>
          <a class="btn h-12 justify-center rounded-container border border-surface-300-700 bg-surface-50-950 px-5 text-base font-semibold text-surface-950-50 hover:border-primary-400" href="/admin/login">
            {t('login.consoleCta')}
          </a>
        </div>

        <div class="mt-8 grid max-w-2xl grid-cols-3 gap-3">
          {#each metrics as metric}
            <div class="rounded-container border border-surface-300-700 bg-surface-50-950 px-4 py-3">
              <strong class="block text-2xl font-semibold tabular-nums">{metric.value}</strong>
              <span class="mt-1 block text-xs leading-5 text-surface-600-400">{t(metric.label)}</span>
            </div>
          {/each}
        </div>
      </section>

      <aside class="rounded-container border border-surface-300-700 bg-surface-50-950 p-5 shadow-2xl shadow-black/20">
        <div class="mb-5 flex items-center gap-3">
          <span class="preset-glass-primary inline-flex size-11 items-center justify-center rounded-container text-primary-200">
            <Building2 size={22} aria-hidden="true" />
          </span>
          <div>
            <p class="text-xs font-semibold uppercase text-primary-300">{t('login.assuranceEyebrow')}</p>
            <h2 class="mt-1 text-xl font-semibold">{t('login.assuranceTitle')}</h2>
          </div>
        </div>

        <div class="grid gap-3">
          {#each assurances as item}
            <article class="grid grid-cols-[2.25rem_minmax(0,1fr)] gap-3 rounded-container border border-surface-200-800 bg-surface-100-900 p-4">
              <span class="inline-flex size-9 items-center justify-center rounded-container bg-primary-500/14 text-primary-300">
                <item.icon size={18} aria-hidden="true" />
              </span>
              <div>
                <h3 class="text-sm font-semibold">{t(item.title)}</h3>
                <p class="mt-1 text-sm leading-6 text-surface-600-400">{t(item.description)}</p>
              </div>
            </article>
          {/each}
        </div>
      </aside>
    </div>
  </section>
</main>
