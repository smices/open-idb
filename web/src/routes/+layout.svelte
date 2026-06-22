<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/state';
  import { browser } from '$app/environment';
  import { api } from '$lib/api';
  import {
    authLoading,
    authUser,
    sidebarCollapsed,
    initLocaleFromStorage,
    toggleSidebar,
    type UserSummary,
  } from '$lib/stores';
  import { onMount } from 'svelte';
  import { t, initThemeFromStorage } from '$lib/i18n';
  import ThemeBackground from '$lib/components/layout/ThemeBackground.svelte';
  import ThemeLanguagePanel from '$lib/components/layout/ThemeLanguagePanel.svelte';
  import UserMenu from '$lib/components/layout/UserMenu.svelte';
  import { redirectToLogin, redirectToPath } from '$lib/session';
  import {
    Activity,
    AppWindow,
    Building2,
    ChevronLeft,
    ChevronRight,
    CircleUserRound,
    FileSearch,
    Gauge,
    GitBranch,
    KeyRound,
    Link2,
    Network,
    RefreshCw,
    ShieldCheck,
    UsersRound,
    Workflow,
  } from 'lucide-svelte';
  import '../app.css';

  let { children } = $props();

  const menuItems = [
    { id: 'dashboardUser', scope: 'user', path: '/dashboard?scope=user', label: 'layout.menu.userDashboard', title: 'dashboard.userTitle', icon: Gauge },
    { id: 'profile', scope: 'user', path: '/profile', label: 'layout.menu.profile', title: 'profile.title', icon: Activity },
    { id: 'dashboardEnterprise', scope: 'enterprise', path: '/dashboard', label: 'layout.menu.enterpriseDashboard', title: 'dashboard.enterpriseTitle', icon: Gauge },
    { id: 'users', scope: 'enterprise', path: '/users', label: 'layout.menu.users', title: 'users.title', icon: UsersRound },
    { id: 'applications', scope: 'enterprise', path: '/applications', label: 'layout.menu.applications', title: 'applications.title', icon: AppWindow },
    { id: 'assignments', scope: 'enterprise', path: '/application-assignments', label: 'layout.menu.assignments', title: 'assignments.title', icon: ShieldCheck },
    { id: 'sources', scope: 'enterprise', path: '/identity-sources', label: 'layout.menu.sources', title: 'identitySources.title', icon: GitBranch },
    { id: 'directory', scope: 'enterprise', path: '/directory-users', label: 'layout.menu.directory', title: 'directory.title', icon: CircleUserRound },
    { id: 'organization', scope: 'enterprise', path: '/organization', label: 'layout.menu.organization', title: 'organization.title', icon: Network },
    { id: 'integrations', scope: 'enterprise', path: '/integrations', label: 'layout.menu.integrations', title: 'integrations.title', icon: Link2 },
    { id: 'syncJobs', scope: 'enterprise', path: '/sync-jobs', label: 'layout.menu.syncJobs', title: 'syncJobs.title', icon: RefreshCw },
    { id: 'roles', scope: 'enterprise', path: '/roles', label: 'layout.menu.roles', title: 'roles.title', icon: ShieldCheck },
    { id: 'audit', scope: 'enterprise', path: '/audit', label: 'layout.menu.audit', title: 'audit.title', icon: FileSearch },
    { id: 'dashboardSystem', scope: 'system', path: '/dashboard?scope=system', label: 'layout.menu.systemDashboard', title: 'dashboard.systemTitle', icon: Gauge },
    { id: 'entities', scope: 'system', path: '/entities', label: 'layout.menu.entities', title: 'entities.title', icon: Building2 },
    { id: 'resourceScopes', scope: 'system', path: '/resource-scopes', label: 'layout.menu.resourceScopes', title: 'resourceScopes.title', icon: KeyRound },
    { id: 'mcp', scope: 'system', path: '/mcp', label: 'layout.menu.mcp', title: 'mcp.title', icon: Workflow },
  ];
  const menuGroups = [
    { id: 'user', label: 'layout.scope.user', items: menuItems.filter((item) => item.scope === 'user') },
    { id: 'enterprise', label: 'layout.scope.enterprise', items: menuItems.filter((item) => item.scope === 'enterprise') },
    { id: 'system', label: 'layout.scope.system', items: menuItems.filter((item) => item.scope === 'system') },
  ];
  const isLoginPath = (pathname: string) =>
    pathname === '/' ||
    pathname === '/login' ||
    pathname === '/auth/continue' ||
    /^\/t\/[^/]+\/admin\/login$/.test(pathname);

  onMount(() => {
    initLocaleFromStorage();
    initThemeFromStorage();

    if (!browser) {
      return;
    }

    const initSession = async () => {
      if (isLoginPath(page.url.pathname)) {
        authLoading.set(false);
        return;
      }

      try {
        const user = await api.me();
        authUser.set(user as UserSummary);
      } catch {
        redirectToLogin();
      } finally {
        authLoading.set(false);
      }
    };

    initSession();
  });

  const handleLogout = () => {
    if (browser) {
      document.cookie = 'idb_session=; Max-Age=0; Path=/;';
    }
    redirectToPath('/');
    authUser.set(null);
  };

  const canShowMenuItem = (item: (typeof menuItems)[number]) => {
    const capabilities = $authUser?.capabilities;
    const consoleScope = $authUser?.console_scope;
    if (!capabilities?.length && !consoleScope) return true;
    if (item.scope === 'user') return true;
    if (item.scope === 'enterprise') return capabilities?.includes('enterprise') || consoleScope === 'enterprise_admin';
    if (item.scope === 'system') return capabilities?.includes('system') || consoleScope === 'enterprise_admin';
    return false;
  };
  const defaultDashboardId = () => {
    const enterpriseDashboard = menuItems.find((item) => item.id === 'dashboardEnterprise');
    if (enterpriseDashboard && canShowMenuItem(enterpriseDashboard)) return 'dashboardEnterprise';
    return 'dashboardUser';
  };
  const resolveActiveNav = () => {
    const pathname = page.url?.pathname || '/';
    if (pathname === '/' || pathname === '/dashboard') {
      const scope = page.url?.searchParams.get('scope');
      if (scope === 'user') return 'dashboardUser';
      if (scope === 'system') {
        const systemDashboard = menuItems.find((item) => item.id === 'dashboardSystem');
        return systemDashboard && canShowMenuItem(systemDashboard) ? 'dashboardSystem' : defaultDashboardId();
      }
      return defaultDashboardId();
    }
    const match = menuItems.find((item) => {
      const itemPath = item.path.split('?')[0];
      return pathname === itemPath || pathname.startsWith(`${itemPath}/`);
    });
    return match?.id || defaultDashboardId();
  };

  const isLoginPage = $derived(
    isLoginPath(page.url.pathname),
  );
  const activeNav = $derived(resolveActiveNav());
  const activeMenuItem = $derived(menuItems.find((item) => item.id === activeNav));

  $effect(() => {
    if (browser && $authUser && activeMenuItem && !canShowMenuItem(activeMenuItem)) {
      redirectToPath('/dashboard');
    }
  });
</script>

<div class="glass-page min-h-screen text-surface-950-50">
  <ThemeBackground intensity="quiet" />
  {#if isLoginPage}
    <main class="relative z-10 w-full min-h-screen">
      {@render children()}
    </main>
  {:else}
    {#if $authLoading}
      <div class="app-shell">
        <aside class="sidebar" aria-label={t('layout.menu.dashboard')}>
          <div class="brand">
            <div class="brand-mark">
              <img src="/logo.svg" alt="" aria-hidden="true" />
            </div>
            <div>
              <strong>{t('app.title')}</strong>
              <span>Admin Console</span>
            </div>
          </div>
          <nav class="nav-list" aria-hidden="true">
            {#each menuItems.slice(0, 8) as item}
              <span class="nav-item">
                <item.icon size={18} aria-hidden="true" />
                <span>{t(item.label)}</span>
              </span>
            {/each}
          </nav>
        </aside>
        <main class="main">
          <header class="topbar"></header>
          <div class="main-content">
            <div class="skeleton-grid" aria-label={t('common.loading')}>
              <div></div>
              <div></div>
              <div></div>
            </div>
          </div>
        </main>
      </div>
    {:else if !$authUser}
      <main class="relative z-10 w-full min-h-screen">
        {@render children()}
      </main>
    {:else}
      <div class={$sidebarCollapsed ? 'app-shell collapsed' : 'app-shell'}>
        <aside class="sidebar" aria-label={t('layout.menu.dashboard')}>
          <div class={$sidebarCollapsed ? 'brand collapsed' : 'brand'}>
            <div class="brand-mark">
              <img src="/logo.svg" alt="" aria-hidden="true" />
            </div>
            {#if !$sidebarCollapsed}
              <div>
                <strong>{t('app.title')}</strong>
                <span>Admin Console</span>
              </div>
            {/if}
          </div>

          <nav class="nav-list">
            {#each menuGroups as group}
              {@const visibleItems = group.items.filter((item) => canShowMenuItem(item))}
              {#if visibleItems.length}
              <div class="nav-section">
                {#if !$sidebarCollapsed}
                  <div class="nav-section-label">{t(group.label)}</div>
                {/if}
                {#each visibleItems as item}
                  <a
                    href={item.path}
                    class="nav-item {activeNav === item.id ? 'active' : ''} {$sidebarCollapsed ? 'collapsed' : ''}"
                    aria-label={t(item.label)}
                    title={$sidebarCollapsed ? t(item.label) : undefined}
                    aria-current={activeNav === item.id ? 'page' : undefined}
                  >
                    <item.icon size={18} aria-hidden="true" />
                    {#if !$sidebarCollapsed}
                      <span>{t(item.label)}</span>
                    {/if}
                  </a>
                {/each}
              </div>
              {/if}
            {/each}
          </nav>

          <div class={$sidebarCollapsed ? 'sidebar-footer collapsed' : 'sidebar-footer'}>
            <button
              class="btn btn-icon nav-collapse-button preset-outlined-surface-500"
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
              <h1>{activeMenuItem ? t(activeMenuItem.title) : t('app.title')}</h1>
            </div>
            <div class="topbar-actions">
              <ThemeLanguagePanel />
              <UserMenu user={$authUser} onlogout={handleLogout} />
            </div>
          </header>

          <div class="main-content">
            {@render children()}
          </div>

        </main>
      </div>
    {/if}
  {/if}
</div>
