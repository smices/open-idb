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
  import UserMenu from '$lib/components/layout/UserMenu.svelte';
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
    ShieldCheck,
    UserRound,
    UsersRound,
  } from 'lucide-svelte';
  import '../app.css';

  let { children } = $props();
  let hasIdentitySource = $state(false);

  const portalItems = [
    { id: 'portal', path: '/portal', label: '应用', title: '用户门户', icon: AppWindow },
    { id: 'profile', path: '/portal/profile', label: '资料', title: '个人资料', icon: UserRound },
  ];

  const adminGroups = [
    {
      id: 'overview',
      label: '总览',
      items: [{ id: 'dashboard', path: '/admin', label: '概览', title: '管理概览', description: '查看身份接入、账号、应用授权和同步任务的整体状态。', icon: Gauge }],
    },
    {
      id: 'identity',
      label: '身份接入',
      items: [
        { id: 'sources', path: '/admin/identity-sources', label: '身份源接入', title: '身份源接入', description: '配置企业的主身份目录，例如飞书；通讯录和组织架构都从这里同步。', icon: GitBranch },
        { id: 'syncJobs', path: '/admin/sync-jobs', label: '同步任务', title: '同步任务', description: '查看身份源同步、增量更新和失败重试记录。', icon: RefreshCw },
      ],
    },
    {
      id: 'governance',
      label: '身份治理',
      items: [
        { id: 'organization', path: '/admin/organization', label: '组织架构', title: '组织架构', description: '基于已接入身份源生成公司、部门和用户分组；未配置身份源前不应维护组织架构。', icon: Network },
        { id: 'users', path: '/admin/users', label: '账号管理', title: '账号管理', description: '维护系统内可登录、可授权、可分配角色和应用权限的账号。', icon: UsersRound },
        { id: 'applications', path: '/admin/applications', label: '应用', title: '应用管理', description: '管理接入 IdBridge 的业务应用，维护 OIDC 接入配置和客户端凭据。', icon: AppWindow },
        { id: 'roles', path: '/admin/roles', label: '角色权限', title: '角色权限', description: '定义角色、权限和资源范围，并分配给账号或组织对象。', icon: ShieldCheck },
      ],
    },
    {
      id: 'system',
      label: '系统管理',
      items: [
        { id: 'entities', path: '/admin/entities', label: '公司管理', title: '公司管理', description: '维护平台下的公司、登录品牌和默认语言。', icon: Building2 },
        { id: 'audit', path: '/admin/audit', label: '审计日志', title: '审计日志', description: '追踪登录、同步、授权和管理员操作记录。', icon: FileSearch },
      ],
    },
  ];
  const adminItems = [
    ...adminGroups.flatMap((group) => group.items),
    { id: 'profile', path: '/admin/profile', label: '个人资料', title: '个人资料', description: '维护当前管理员账号资料和登录密码。', icon: UserRound },
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
  const isLoginPage = $derived(isLoginPath(page.url.pathname));
  const isAdminShell = $derived(isAdminPath(page.url.pathname) && page.url.pathname !== '/admin/login');
  const isPortalShell = $derived(isPortalPath(page.url.pathname));

  onMount(() => {
    initLocaleFromStorage();
    initThemeFromStorage();

    if (!browser) return;

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
          <a class="inline-flex items-center gap-3 no-underline" href="/portal" aria-label={t('app.title')}>
            <span class="portal-brand-mark">
              <img class="size-7" src="/logo.svg" alt="" aria-hidden="true" />
            </span>
            <span class="text-sm font-semibold">{t('app.title')}</span>
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
          {#each adminGroups as group}
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
                  title={lockedByIdentitySource ? '请先配置身份源' : $sidebarCollapsed ? item.label : undefined}
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
