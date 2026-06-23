<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/stores';
  import { api, type LoginContext, type LoginMode, type LoginProvider } from '$lib/api';
  import { t, tf } from '$lib/i18n';
  import { redirectToPath } from '$lib/session';
  import { AppWindow, Building2, KeyRound, ShieldCheck } from 'lucide-svelte';
  import { onMount } from 'svelte';

  let account = import.meta.env.DEV ? 'admin' : '';
  let password = import.meta.env.DEV ? 'admin123' : '';
  let context: LoginContext | null = null;
  let providers: LoginProvider[] = [];
  let loadingContext = true;
  let providersLoading = false;
  let providersLoaded = false;
  let busyError = '';
  let autoRedirecting = false;

  $: query = $page.url.searchParams;
  $: isAdminLogin = $page.url.pathname === '/admin/login';
  $: returnTo = query.get('return_to') || (isAdminLogin ? '/admin' : '/portal');
  $: loginAction = isAdminLogin ? '/sapi/login/account' : '/api/login/account';
  $: loginErrorKey = query.get('login_error');
  $: mode = context?.mode || modeFromPath($page.url.pathname);
  $: pathEntitySlug = entitySlugFromPath($page.url.pathname);
  $: entityRef = context?.entity?.id || context?.entity?.slug || pathEntitySlug;
  $: entityBrand = context?.entity?.brand_name || context?.entity?.name || brandFromSlug(pathEntitySlug);
  $: entityLabel = entityBrand || context?.entity?.slug || context?.entity?.id || '';
  $: entityLogoUrl = context?.entity?.logo_url || '';
  $: entityLoginMessage = context?.entity?.login_message || '';
  $: canLoadProviders = Boolean(entityRef && context?.methods.includes('feishu'));
  $: feishuProvider = providers.find((provider) => provider.provider === 'feishu' && provider.oauth_url);
  $: isEnterpriseEntrance = mode === 'app' || Boolean(entityRef);
  $: applicationName = context?.application?.name || (mode === 'app' ? 'Demo App' : 'IdBridge');
  $: identityLogoUrl = entityLogoUrl || '/logo.svg';
  $: pageTitle =
    mode === 'app'
      ? tf('login.enterprise.titleForApp', { app: applicationName })
      : mode === 'entity_admin' && entityBrand
        ? tf('login.entity_admin.titleWithBrand', { brand: entityBrand })
        : isEnterpriseEntrance
          ? t('login.enterprise.title')
          : t(`login.${mode}.title`);
  $: pageSubtitle =
    mode === 'app'
      ? tf('login.enterprise.subtitleForApp', { entity: entityLabel || t('login.enterprise.defaultEntity') })
      : mode === 'entity_admin' && entityLoginMessage
        ? entityLoginMessage
        : isEnterpriseEntrance
          ? tf('login.enterprise.subtitle', { entity: entityLabel || t('login.enterprise.defaultEntity') })
          : t(`login.${mode}.subtitle`);
  $: primaryActionLabel = mode === 'app' ? tf('login.enterprise.feishuForApp', { app: applicationName }) : t('login.enterprise.feishuPrimary');
  $: contextStatus =
    mode === 'app' && context?.application
      ? t('login.enterprise.contextReady')
      : isEnterpriseEntrance
        ? t('login.enterprise.identityReady')
        : t('login.enterprise.contextPending');

  function modeFromPath(pathname: string): LoginMode {
    if (pathname === '/auth/continue') return 'app';
    if (pathname === '/admin/login' || /^\/t\/[^/]+\/admin\/login$/.test(pathname)) return 'entity_admin';
    return 'user';
  }

  function entitySlugFromPath(pathname: string): string {
    const match = pathname.match(/^\/t\/([^/]+)\/admin\/login$/);
    return match?.[1] || '';
  }

  function brandFromSlug(slug: string): string {
    return slug
      .split(/[-_]/)
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
      .join(' ');
  }

  const loginErrorText = (key: string | null): string => {
    if (!key) return '';
    const map: Record<string, string> = {
      invalid_credentials: 'login.error.invalid_credentials',
      missing_params: 'login.error.missing_params',
      invalid_state: 'login.error.invalid_state',
      no_feishu_source: 'login.error.no_feishu_source',
      exchange_failed: 'login.error.exchange_failed',
      user_disabled: 'login.error.user_disabled',
    };
    return map[key] ? t(map[key]) : t('login.error.missing_params');
  };

  const getFeishuLoginUrl = (): string => {
    const params = new URLSearchParams();
    if (entityRef) params.set('entity_id', entityRef);
    params.set('return_to', returnTo);
    return `/api/auth/feishu/login?${params.toString()}`;
  };

  const getProviderLoginUrl = (): string => context?.auto_redirect_url || getFeishuLoginUrl();

  const loadProviders = async (entityValue = entityRef, methods = context?.methods || []) => {
    providersLoaded = false;
    providers = [];
    if (!entityValue || !methods.includes('feishu')) return;
    providersLoading = true;
    try {
      providers = await api.listLoginProviders(entityValue);
      providersLoaded = true;
    } catch {
      busyError = t('login.providersFailed');
    } finally {
      providersLoading = false;
    }
  };

  const loadContext = async () => {
    loadingContext = true;
    try {
      context = isAdminLogin
        ? await api.getAdminLoginContext({ path: $page.url.pathname, return_to: returnTo })
        : await api.getLoginContext({ path: $page.url.pathname, return_to: returnTo });
      busyError = loginErrorText(loginErrorKey);
      if (!busyError && context?.auto_redirect_url) {
        autoRedirecting = true;
        redirectToPath(context.auto_redirect_url);
        return;
      }
      const resolvedEntityRef = context?.entity?.id || context?.entity?.slug || '';
      await loadProviders(resolvedEntityRef, context?.methods || []);
    } catch {
      context = null;
      busyError = t('login.contextFailed');
    } finally {
      loadingContext = false;
    }
  };

  onMount(() => {
    void loadContext();
  });
</script>

<svelte:head>
  <title>{pageTitle}</title>
</svelte:head>

<main class="glass-page-dark min-h-dvh px-5 py-8 text-surface-950-50 lg:px-16">
  <section class="mx-auto grid min-h-[calc(100dvh-4rem)] max-w-6xl items-center gap-8 lg:grid-cols-[minmax(0,1fr)_28rem]">
    <div class="max-w-2xl">
      <div class="mb-8 flex items-center gap-4">
        <div class="preset-glass-primary flex size-12 shrink-0 items-center justify-center rounded-container text-primary-200">
          {#if identityLogoUrl}
            <img class="size-8 rounded-sm object-contain" src={identityLogoUrl} alt="" />
          {:else if mode === 'entity_admin'}
            <Building2 size={24} />
          {:else if mode === 'app'}
            <AppWindow size={24} />
          {/if}
        </div>
        <div class="min-w-0">
          <p class="text-sm font-semibold text-primary-200">{isEnterpriseEntrance ? t('login.enterprise.eyebrow') : t(`login.${mode}.eyebrow`)}</p>
          <p class="mt-1 truncate text-sm text-surface-600-400">{contextStatus}</p>
        </div>
      </div>

      <h1 class="max-w-2xl text-4xl font-semibold leading-tight text-balance sm:text-5xl">
        {pageTitle}
      </h1>
      <p class="mt-5 max-w-xl text-base leading-7 text-pretty text-surface-700-300">
        {pageSubtitle}
      </p>

      <dl class="mt-8 grid gap-3 sm:grid-cols-3">
        <div class="border-l border-white/18 pl-4">
          <dt class="text-xs text-surface-600-400">{t('login.enterprise.scopeLabel')}</dt>
          <dd class="mt-2 text-sm font-semibold">{mode === 'app' ? applicationName : t('login.enterprise.scopeValue')}</dd>
        </div>
        <div class="border-l border-white/18 pl-4">
          <dt class="text-xs text-surface-600-400">{t('login.enterprise.entityLabel')}</dt>
          <dd class="mt-2 break-words text-sm font-semibold">{entityLabel || t('login.enterprise.defaultEntity')}</dd>
        </div>
        <div class="border-l border-white/18 pl-4">
          <dt class="text-xs text-surface-600-400">{t('login.enterprise.methodLabel')}</dt>
          <dd class="mt-2 text-sm font-semibold">{t('login.enterprise.methodValue')}</dd>
        </div>
      </dl>
    </div>

    <section class="w-full rounded-container border border-surface-300-700 bg-surface-50-950 p-6 shadow-2xl shadow-black/24">
      <div class="mb-6">
        <p class="text-sm font-medium text-primary-200">{isEnterpriseEntrance ? t('login.enterprise.formEyebrow') : t(`login.${mode}.formEyebrow`)}</p>
        <h2 class="mt-2 text-2xl font-semibold">
          {isEnterpriseEntrance
            ? t('login.enterprise.formTitle')
            : mode === 'entity_admin' && entityBrand
              ? tf('login.entity_admin.formTitleWithBrand', { brand: entityBrand })
              : t(`login.${mode}.formTitle`)}
        </h2>
      </div>

      {#if busyError}
        <aside class="alert preset-tonal-error mb-5" role="alert"><p>{busyError}</p></aside>
      {/if}

      {#if loadingContext || autoRedirecting}
        <div class="space-y-3" aria-label={t('common.loading')}>
          <div class="preset-glass-surface-soft h-12 rounded-container"></div>
          <div class="preset-glass-surface-soft h-12 rounded-container"></div>
          <div class="preset-glass-surface-soft h-12 rounded-container"></div>
        </div>
      {:else}
        {#if entityRef}
          <div class="mb-5 flex min-h-12 items-center gap-3 rounded-container border border-surface-200-800 bg-surface-100-900 px-4 py-3 text-sm">
            <ShieldCheck class="shrink-0 text-primary-300" size={18} />
            <div class="min-w-0">
              <p class="text-xs text-surface-600-400">{t('login.enterprise.verifiedContext')}</p>
              <p class="truncate font-semibold">{entityLabel || entityRef}</p>
            </div>
          </div>
        {/if}

        {#if canLoadProviders}
          <div class="space-y-3">
            {#if providersLoading}
              <p class="rounded-container border border-surface-200-800 bg-surface-100-900 px-4 py-3 text-sm text-surface-600-400">{t('common.loading')}</p>
            {:else if feishuProvider}
              <a class="btn h-12 w-full justify-center gap-2 rounded-container bg-primary-500 text-base font-semibold text-white hover:bg-primary-600 focus-visible:ring-4 focus-visible:ring-primary-400/30" href={getProviderLoginUrl()}>
                <span class="feishu-mark" aria-hidden="true">
                  <span></span><span></span><span></span><span></span>
                </span>
                {primaryActionLabel}
              </a>
            {:else if providersLoaded}
              <p class="rounded-container border border-surface-200-800 bg-surface-100-900 px-4 py-3 text-sm text-surface-600-400">{t('login.noProviders')}</p>
            {/if}
          </div>

          <div class="my-6 flex items-center gap-3 text-xs text-surface-500">
            <span class="h-px flex-1 bg-surface-200-800"></span>
            {t('login.enterprise.accountFallback')}
            <span class="h-px flex-1 bg-surface-200-800"></span>
          </div>
        {/if}

        <form method="post" action={loginAction} class="space-y-4">
          <input type="hidden" name="return_to" value={returnTo} />
          {#if entityRef}
            <input type="hidden" name="entity_id" value={entityRef} />
          {/if}

          <label class="block">
            <span class="mb-2 flex items-center gap-2 text-sm text-surface-700-300"><KeyRound size={15} />{t(`login.${mode}.accountLabel`)}</span>
            <input
              name="account"
              class="h-12 w-full rounded-container border border-surface-300-700 bg-surface-50-950 px-4 text-base text-surface-950-50 outline-none transition placeholder:text-surface-500 focus:border-primary-400 focus:ring-4 focus:ring-primary-400/20"
              type="text"
              bind:value={account}
              placeholder={t(`login.${mode}.accountPlaceholder`)}
              autocomplete="username"
              required
            />
          </label>

          <label class="block">
            <span class="mb-2 block text-sm text-surface-700-300">{t('login.password')}</span>
            <input
              name="password"
              class="h-12 w-full rounded-container border border-surface-300-700 bg-surface-50-950 px-4 text-base text-surface-950-50 outline-none transition placeholder:text-surface-500 focus:border-primary-400 focus:ring-4 focus:ring-primary-400/20"
              type="password"
              bind:value={password}
              placeholder={t('login.passwordPlaceholder')}
              autocomplete="current-password"
              required
            />
          </label>

          <button class="btn h-12 w-full justify-center gap-2 rounded-container border border-surface-300-700 bg-surface-100-900 text-base font-semibold text-surface-950-50 hover:border-primary-400 focus-visible:ring-4 focus-visible:ring-primary-400/20" type="submit">
            {t(`login.${mode}.submit`)}
          </button>
        </form>
      {/if}
    </section>
  </section>
</main>
