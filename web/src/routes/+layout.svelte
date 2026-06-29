<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/state';
  import { browser } from '$app/environment';
  import { api } from '$lib/api';
  import {
    authLoading,
    authUser,
    platformBranding,
    sidebarCollapsed,
    initLocaleFromStorage,
    setPlatformBranding,
    toggleSidebar,
    type UserSummary,
  } from '$lib/stores';
  import { onMount } from 'svelte';
  import { t, initThemeFromStorage } from '$lib/i18n';
  import UserMenu from '$lib/components/layout/UserMenu.svelte';
  import IdToastProvider from '$lib/components/ui/IdToastProvider.svelte';
  import { redirectToPath } from '$lib/session';
  import {
    AppWindow,
    Building2,
    ChevronLeft,
    ChevronRight,
    FileSearch,
    Gauge,
    GitBranch,
    Network,
    RefreshCw,
    Settings,
    ShieldCheck,
    UserCog,
    UserRound,
    UsersRound,
  } from 'lucide-svelte';
  import '../app.css';

  let { children } = $props();
  let hasIdentitySource = $state(false);

  const brandName = $derived($platformBranding.platform_name || t('app.title'));
  const brandLogoUrl = $derived($platformBranding.logo_url || '/logo.svg');

  const portalItems = [
    { id: 'portal', path: '/portal', label: t('nav.portal.apps'), title: t('portal.title'), icon: AppWindow },
    { id: 'profile', path: '/portal/profile', label: t('nav.portal.profile'), title: t('profile.title'), icon: UserRound },
  ];

  const adminGroups = [
    {
      id: 'overview',
      label: t('nav.admin.overview'),
      items: [{ id: 'dashboard', path: '/admin', label: t('nav.admin.dashboard'), title: t('nav.admin.dashboardTitle'), description: t('nav.admin.dashboardDescription'), icon: Gauge }],
    },
    {
      id: 'identity',
      label: t('nav.admin.identity'),
      items: [
        { id: 'sources', path: '/admin/identity-sources', label: t('identitySources.title'), title: t('identitySources.title'), description: t('nav.admin.identitySourcesDescription'), icon: GitBranch },
        { id: 'syncJobs', path: '/admin/sync-jobs', label: t('syncJobs.title'), title: t('syncJobs.title'), description: t('nav.admin.syncJobsDescription'), icon: RefreshCw },
      ],
    },
    {
      id: 'governance',
      label: t('nav.admin.governance'),
      items: [
        { id: 'organization', path: '/admin/organization', label: t('organization.title'), title: t('organization.title'), description: t('nav.admin.organizationDescription'), icon: Network },
        { id: 'users', path: '/admin/users', label: t('users.accountManagement'), title: t('users.accountManagement'), description: t('nav.admin.usersDescription'), icon: UsersRound },
        { id: 'applications', path: '/admin/applications', label: t('applications.title'), title: t('applications.title'), description: t('nav.admin.applicationsDescription'), icon: AppWindow },
        { id: 'roles', path: '/admin/roles', label: t('roles.title'), title: t('roles.title'), description: t('nav.admin.rolesDescription'), icon: ShieldCheck },
      ],
    },
    {
      id: 'system',
      label: t('nav.admin.system'),
      items: [
        { id: 'entities', path: '/admin/entities', label: t('entities.title'), title: t('entities.title'), description: t('nav.admin.entitiesDescription'), icon: Building2 },
        { id: 'platform', path: '/admin/platform', label: t('platform.title'), title: t('platform.title'), description: t('nav.admin.platformDescription'), icon: Settings },
        { id: 'adminUsers', path: '/admin/admin-users', label: t('adminUsers.title'), title: t('adminUsers.title'), description: t('nav.admin.adminUsersDescription'), icon: UserCog },
        { id: 'audit', path: '/admin/audit', label: t('audit.title'), title: t('audit.title'), description: t('nav.admin.auditDescription'), icon: FileSearch },
      ],
    },
  ];
  const adminItems = [
    ...adminGroups.flatMap((group) => group.items),
    { id: 'profile', path: '/admin/profile', label: t('profile.title'), title: t('profile.title'), description: t('nav.admin.profileDescription'), icon: UserRound },
  ];

  const isLoginPath = (pathname: string) =>
    pathname === '/' ||
    pathname === '/login' ||
    pathname === '/admin/login' ||
    pathname === '/auth/continue';

  const isAdminPath = (pathname: string) => pathname === '/admin' || pathname.startsWith('/admin/');
  const isPortalPath = (pathname: string) => pathname === '/portal' || pathname.startsWith('/portal/');
  const findActiveItem = <T extends { path: string }>(items: T[], pathname: string, fallback: T) =>
    [...items]
      .sort((a, b) => b.path.length - a.path.length)
      .find((item) => pathname === item.path || pathname.startsWith(`${item.path}/`)) || fallback;

  const activePortalItem = $derived(findActiveItem(portalItems, page.url.pathname, portalItems[0]));
  const activeAdminItem = $derived(findActiveItem(adminItems, page.url.pathname, adminItems[0]));
  const visibleAdminGroups = $derived(
    adminGroups
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => (item.id !== 'adminUsers' && item.id !== 'platform') || $authUser?.capabilities?.includes('system')),
      }))
      .filter((group) => group.items.length > 0),
  );
  const isLoginPage = $derived(isLoginPath(page.url.pathname));
  const isAdminShell = $derived(isAdminPath(page.url.pathname) && page.url.pathname !== '/admin/login');
  const isPortalShell = $derived(isPortalPath(page.url.pathname));

  onMount(() => {
    initLocaleFromStorage();
    initThemeFromStorage();

    if (!browser) return;

    void api
      .getPlatformBranding()
      .then(setPlatformBranding)
      .catch(() => undefined);

    const initSession = async () => {
      if (isLoginPath(page.url.pathname)) {
        authLoading.set(false);
        return;
      }

      try {
        if (isAdminPath(page.url.pathname)) {
          const admin = await api.adminMe();
          const sourceData = await api.listIdentitySources({ limit: 1 }).catch(() => ({ items: [], sources: [] }));
          hasIdentitySource = Boolean((sourceData.items || sourceData.sources || []).length);
          authUser.set({
            id: admin.admin_id || admin.id,
            entity_id: admin.entity_id || '',
            username: admin.username,
            display_name: admin.display_name || admin.username,
            locale: 'zh-CN',
            console_scope: admin.role === 'platform_admin' ? 'enterprise_admin' : 'enterprise_admin',
            capabilities: admin.role === 'platform_admin' ? ['user', 'enterprise', 'system'] : ['user', 'enterprise'],
          } as UserSummary);
        } else {
          const user = await api.me();
          authUser.set(user as UserSummary);
        }
      } catch {
        const returnTo = `${page.url.pathname}${page.url.search}`;
        const loginPath = isAdminPath(page.url.pathname) ? '/admin/login' : '/login';
        redirectToPath(`${loginPath}?return_to=${encodeURIComponent(returnTo)}`);
      } finally {
        authLoading.set(false);
      }
    };

    initSession();
  });

  const handleLogout = () => {
    if (browser) {
      document.cookie = 'idb_session=; Max-Age=0; Path=/;';
      document.cookie = 'idb_admin_session=; Max-Age=0; Path=/;';
    }
    const target = isAdminShell ? '/admin/login' : '/login';
    authUser.set(null);
    redirectToPath(target);
  };
</script>

<svelte:head>
  <meta name="application-name" content={brandName} />
  {#if $platformBranding.favicon_url}
    <link rel="icon" href={$platformBranding.favicon_url} />
  {/if}
</svelte:head>

<div class="glass-page min-h-dvh text-surface-950-50">
  {#if isLoginPage}
    <main class="w-full min-h-dvh">
      {@render children()}
    </main>
  {:else if $authLoading}
    <main class="grid min-h-dvh place-items-center px-6">
      <div class="card w-full max-w-xl space-y-3 bg-surface-50-950 border border-surface-200-800 p-4" aria-label={t('common.loading')}>
        <div class="h-4 w-36 rounded bg-surface-200-800"></div>
        <div class="h-4 rounded bg-surface-200-800"></div>
        <div class="h-4 w-5/6 rounded bg-surface-200-800"></div>
        <div class="h-4 w-2/3 rounded bg-surface-200-800"></div>
      </div>
    </main>
  {:else if !$authUser}
    <main class="w-full min-h-dvh">
      {@render children()}
    </main>
  {:else if isPortalShell}
    <main class="portal-shell min-h-dvh">
      <header class="portal-topbar">
        <div class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-5 py-3 lg:px-8">
          <a class="inline-flex items-center gap-3 no-underline" href="/portal" aria-label={brandName}>
            <span class="portal-brand-mark">
              <img class="size-7" src={brandLogoUrl} alt="" aria-hidden="true" />
            </span>
            <span class="text-sm font-semibold">{brandName}</span>
          </a>
          <nav class="portal-nav" aria-label="Portal">
            {#each portalItems as item}
              <a
                href={item.path}
                class="portal-nav-item {activePortalItem.id === item.id ? 'active' : ''}"
                aria-current={activePortalItem.id === item.id ? 'page' : undefined}
              >
                <item.icon size={16} aria-hidden="true" />
                <span>{item.label}</span>
              </a>
            {/each}
          </nav>
          <div class="flex items-center gap-2">
            <UserMenu user={$authUser} onlogout={handleLogout} />
          </div>
        </div>
      </header>
      <div class="mx-auto max-w-7xl px-5 py-6 lg:px-8">
        {@render children()}
      </div>
    </main>
  {:else}
    <div class={$sidebarCollapsed ? 'app-shell collapsed' : 'app-shell'}>
      <aside class="sidebar" aria-label="Admin">
        <div class={$sidebarCollapsed ? 'brand collapsed' : 'brand'}>
          <div class="brand-mark">
            <img src={brandLogoUrl} alt="" aria-hidden="true" />
          </div>
          {#if !$sidebarCollapsed}
            <div>
              <strong>{brandName}</strong>
              <span>Admin Console</span>
            </div>
          {/if}
        </div>

        <nav class="nav-list">
          {#each visibleAdminGroups as group}
            <div class="nav-section">
              {#if !$sidebarCollapsed}
                <div class="nav-section-label">{group.label}</div>
              {/if}
              {#each group.items as item}
                {@const lockedByIdentitySource = item.id === 'organization' && !hasIdentitySource}
                <a
                  href={lockedByIdentitySource ? '/admin/identity-sources' : item.path}
                  class="nav-item {activeAdminItem.id === item.id ? 'active' : ''} {lockedByIdentitySource ? 'muted' : ''} {$sidebarCollapsed ? 'collapsed' : ''}"
                  aria-label={item.label}
                  title={lockedByIdentitySource ? t('nav.admin.identitySourceRequired') : $sidebarCollapsed ? item.label : undefined}
                  aria-current={activeAdminItem.id === item.id ? 'page' : undefined}
                >
                  <item.icon size={18} aria-hidden="true" />
                  {#if !$sidebarCollapsed}
                    <span>{item.label}</span>
                  {/if}
                </a>
              {/each}
            </div>
          {/each}
        </nav>

        <div class={$sidebarCollapsed ? 'sidebar-footer collapsed' : 'sidebar-footer'}>
          <button
            class="btn preset-outlined-surface-500 nav-collapse-button"
            type="button"
            onclick={toggleSidebar}
            aria-label={t('layout.toggleSidebar')}
            title={t('layout.toggleSidebar')}
          >
            {#if $sidebarCollapsed}
              <ChevronRight size={18} aria-hidden="true" />
            {:else}
              <ChevronLeft size={18} aria-hidden="true" />
            {/if}
          </button>
        </div>
      </aside>

      <main class="main">
        <header class="topbar">
          <div class="topbar-title">
            <h1>{activeAdminItem.title}</h1>
            <p>{activeAdminItem.description}</p>
          </div>
          <div class="topbar-actions">
            <UserMenu user={$authUser} onlogout={handleLogout} />
          </div>
        </header>

        <div class="main-content">
          {@render children()}
        </div>
      </main>
    </div>
  {/if}
</div>
<IdToastProvider />
