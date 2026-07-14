import React, { Suspense, useCallback, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  App as AntApp,
  Alert,
  Button,
  Card,
  ConfigProvider,
  Empty,
  Form,
  Input,
  Layout,
  Menu,
  Segmented,
  Skeleton,
  Space,
  Statistic,
  Table,
  Tag,
  theme as antdTheme,
  Typography,
} from 'antd';
import {
  Archive,
  AppWindow,
  Building2,
  ChevronLeft,
  ChevronRight,
  FileSearch,
  Gauge,
  GitBranch,
  KeyRound,
  Mail,
  Network,
  RefreshCw,
  Settings,
  ShieldCheck,
  UserCog,
  UserRound,
  UsersRound,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api } from './lib/api.ts';
import { feishuAuthCodeFromBridge, isFeishuClient, normalizeWorkplaceProvider, queryAuthCode, returnToParam } from './lib/feishu-workplace.js';
import { UserMenu } from './components/UserMenu.jsx';
import './i18n/index.js';
import 'antd/dist/reset.css';
import './styles.css';

const { Content, Header, Sider } = Layout;
const THEME_KEY = 'idb-theme-mode';
const LOCALE_KEY = 'idb-language';
const SIDEBAR_KEY = 'idb-sidebar-collapsed';
const defaultBranding = { platform_name: 'IdBridge', logo_url: '', favicon_url: '', title_suffix: '' };

function currentPath() {
  return `${window.location.pathname}${window.location.search}`;
}

function navigate(path) {
  window.location.href = path;
}

function formatDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
}

function errorMessage(error, fallback = 'Request failed') {
  return error instanceof Error ? error.message : fallback;
}

function pickItems(payload, keys = ['items']) {
  for (const key of keys) {
    if (Array.isArray(payload?.[key])) return payload[key];
  }
  return Array.isArray(payload) ? payload : [];
}

function pathKind(pathname) {
  if (pathname === '/' || pathname === '/login' || pathname === '/admin/login' || pathname === '/auth/continue' || /^\/t\/[^/]+\/admin\/login$/.test(pathname)) return 'public';
  if (pathname === '/portal' || pathname.startsWith('/portal/')) return 'portal';
  if (pathname === '/admin' || pathname.startsWith('/admin/')) return 'admin';
  return 'public';
}

function isAdminPath(pathname) {
  return pathname === '/admin' || pathname.startsWith('/admin/');
}

function routeTitle(pathname, t) {
  const items = adminItems(t).flatMap((group) => group.children);
  const portal = portalItems(t);
  const active = [...items, ...portal].sort((a, b) => b.path.length - a.path.length).find((item) => pathname === item.path || pathname.startsWith(`${item.path}/`));
  return active || { label: t('app.title'), description: '' };
}

function useThemeLanguage() {
  const { i18n } = useTranslation();
  const [themeMode, setThemeMode] = useState(() => localStorage.getItem(THEME_KEY) || 'system');
  const [resolvedTheme, setResolvedTheme] = useState(() => {
    const stored = localStorage.getItem(THEME_KEY);
    if (stored === 'dark' || stored === 'light') return stored;
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  });
  const [language, setLanguage] = useState(() => localStorage.getItem(LOCALE_KEY) || i18n.language || 'zh-CN');

  useEffect(() => {
    localStorage.setItem(LOCALE_KEY, language);
    i18n.changeLanguage(language);
    document.documentElement.lang = language;
  }, [language, i18n]);

  useEffect(() => {
    const media = window.matchMedia('(prefers-color-scheme: dark)');
    const apply = () => setResolvedTheme(themeMode === 'system' ? (media.matches ? 'dark' : 'light') : themeMode);
    localStorage.setItem(THEME_KEY, themeMode);
    apply();
    media.addEventListener('change', apply);
    return () => media.removeEventListener('change', apply);
  }, [themeMode]);

  return { themeMode, resolvedTheme, language, setThemeMode, setLanguage };
}

function adminItems(t) {
  return [
    {
      key: 'overview',
      label: t('nav.admin.overview'),
      children: [{ key: '/admin', path: '/admin', label: t('layout.menu.dashboard'), icon: <Gauge size={17} />, description: t('nav.admin.dashboardDescription') }],
    },
    {
      key: 'identity',
      label: t('nav.admin.identity'),
      children: [
        { key: '/admin/identity-sources', path: '/admin/identity-sources', label: t('layout.menu.sources'), icon: <GitBranch size={17} />, description: t('nav.admin.identitySourcesDescription') },
        { key: '/admin/sync-jobs', path: '/admin/sync-jobs', label: t('layout.menu.syncJobs'), icon: <RefreshCw size={17} />, description: t('nav.admin.syncJobsDescription') },
      ],
    },
    {
      key: 'governance',
      label: t('nav.admin.governance'),
      children: [
        { key: '/admin/organization', path: '/admin/organization', label: t('layout.menu.organization'), icon: <Network size={17} />, description: t('nav.admin.organizationDescription') },
        { key: '/admin/users', path: '/admin/users', label: t('layout.menu.users'), icon: <UsersRound size={17} />, description: t('nav.admin.usersDescription') },
        { key: '/admin/archived-users', path: '/admin/archived-users', label: t('archivedUsers.title'), icon: <Archive size={17} />, description: t('nav.admin.archivedUsersDescription') },
        { key: '/admin/applications', path: '/admin/applications', label: t('layout.menu.applications'), icon: <AppWindow size={17} />, description: t('nav.admin.applicationsDescription') },
        { key: '/admin/roles', path: '/admin/roles', label: t('layout.menu.roles'), icon: <ShieldCheck size={17} />, description: t('nav.admin.rolesDescription') },
      ],
    },
    {
      key: 'system',
      label: t('nav.admin.system'),
      children: [
        { key: '/admin/entities', path: '/admin/entities', label: t('layout.menu.entities'), icon: <Building2 size={17} />, description: t('nav.admin.entitiesDescription') },
        { key: '/admin/platform', path: '/admin/platform', label: t('platform.title'), icon: <Settings size={17} />, description: t('nav.admin.platformDescription') },
        { key: '/admin/admin-users', path: '/admin/admin-users', label: t('adminUsers.title'), icon: <UserCog size={17} />, description: t('nav.admin.adminUsersDescription') },
        { key: '/admin/audit', path: '/admin/audit', label: t('layout.menu.audit'), icon: <FileSearch size={17} />, description: t('nav.admin.auditDescription') },
        { key: '/admin/profile', path: '/admin/profile', label: t('layout.menu.profile'), icon: <UserRound size={17} />, description: t('nav.admin.profileDescription') },
      ],
    },
  ];
}

function portalItems(t) {
  return [
    { key: '/portal', path: '/portal', label: t('nav.portal.apps'), icon: <AppWindow size={16} /> },
    { key: '/portal/profile', path: '/portal/profile', label: t('nav.portal.profile'), icon: <UserRound size={16} /> },
  ];
}

function Root() {
  const themeState = useThemeLanguage();
  const antThemeConfig = useMemo(() => ({
    cssVar: { key: 'idbridge', prefix: 'ant' },
    algorithm: [
      themeState.resolvedTheme === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
      antdTheme.compactAlgorithm,
    ],
    token: {
      borderRadius: 8,
      colorPrimary: '#1f7a8c',
      colorInfo: '#1f7a8c',
      fontFamily: 'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    },
    components: {
      Button: { controlHeight: 34 },
      Input: { controlHeight: 34 },
      Select: { controlHeight: 34 },
      Table: { headerBg: 'var(--ant-color-fill-quaternary)' },
    },
  }), [themeState.resolvedTheme]);

  return (
    <ConfigProvider theme={antThemeConfig}>
      <AntApp>
        <IdBridgeApp themeState={themeState} />
      </AntApp>
    </ConfigProvider>
  );
}

function IdBridgeApp({ themeState }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [pathname, setPathname] = useState(window.location.pathname);
  const [branding, setBranding] = useState(defaultBranding);
  const [user, setUser] = useState(null);
  const [authLoading, setAuthLoading] = useState(true);
  const kind = pathKind(pathname);

  const loadBranding = useCallback(async () => {
    try {
      const next = await api.getPlatformBranding();
      setBranding({ ...defaultBranding, ...next });
    } catch {
      setBranding(defaultBranding);
    }
  }, []);

  const loadSession = useCallback(async () => {
    if (kind === 'public') {
      setAuthLoading(false);
      return;
    }
    setAuthLoading(true);
    try {
      if (kind === 'admin') {
        const admin = await api.adminMe();
        setUser({
          id: admin.admin_id || admin.id,
          entity_id: admin.entity_id || '',
          username: admin.username,
          display_name: admin.display_name || admin.username,
          locale: 'zh-CN',
          capabilities: admin.role === 'platform_admin' ? ['user', 'enterprise', 'system'] : ['user', 'enterprise'],
        });
      } else {
        setUser(await api.me());
      }
    } catch {
      const loginPath = kind === 'admin' ? '/admin/login' : '/login';
      navigate(`${loginPath}?return_to=${encodeURIComponent(currentPath())}`);
    } finally {
      setAuthLoading(false);
    }
  }, [kind]);

  useEffect(() => {
    const onPop = () => setPathname(window.location.pathname);
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, []);

  useEffect(() => {
    loadBranding();
  }, [loadBranding]);

  useEffect(() => {
    loadSession();
  }, [loadSession]);

  useEffect(() => {
    const brandName = branding.platform_name || t('app.title');
    document.title = branding.title_suffix ? `${brandName} · ${branding.title_suffix}` : brandName;
    if (branding.favicon_url) {
      let icon = document.querySelector('link[rel="icon"]');
      if (!icon) {
        icon = document.createElement('link');
        icon.rel = 'icon';
        document.head.appendChild(icon);
      }
      icon.href = branding.favicon_url;
    }
  }, [branding, t]);

  const handleLogout = async () => {
    const endpoint = kind === 'admin' ? '/sapi/logout' : '/api/auth/logout';
    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        credentials: 'same-origin',
      });
      if (!response.ok && response.status !== 401 && response.status !== 404) {
        throw new Error(`logout failed with status ${response.status}`);
      }
      setUser(null);
      message.success(t('layout.logout'));
      navigate(kind === 'admin' ? '/admin/login' : '/login');
    } catch {
      message.error(t('layout.logoutFailed'));
    }
  };

  if (kind === 'public') {
    return (
      <div className="app-root">
        <PublicRouter branding={branding} />
      </div>
    );
  }

  if (authLoading) {
    return (
      <main className="app-root content">
        <Skeleton active paragraph={{ rows: 8 }} />
      </main>
    );
  }

  if (!user) return null;

  if (kind === 'portal') {
    return <PortalShell user={user} branding={branding} themeState={themeState} onLogout={handleLogout} />;
  }

  return <AdminShell user={user} branding={branding} themeState={themeState} onLogout={handleLogout} />;
}

function PublicRouter({ branding }) {
  const pathname = window.location.pathname;
  if (pathname === '/') return <HomePage branding={branding} />;
  return <LoginPage branding={branding} />;
}

function HomePage({ branding }) {
  const { t } = useTranslation();
  const brandName = branding.platform_name || t('app.title');
  const logo = branding.logo_url || '/logo.svg';
  return (
    <main className="auth-page">
      <section className="auth-copy">
        <a className="auth-brand" href="/" aria-label={brandName}>
          <img src={logo} alt="" />
          <span>{brandName}</span>
        </a>
        <h1>{t('login.homeTitle')}</h1>
        <p>{t('login.homeSubtitle')}</p>
        <Space size={10} style={{ marginTop: 28 }}>
          <Button type="primary" size="large" href="/login">{t('login.primaryCta')}</Button>
        </Space>
      </section>
      <Card className="auth-card">
        <Space direction="vertical" size={16}>
          <Statistic title="SSO" value="Feishu" />
          <Statistic title={t('login.metric.access')} value="RBAC" />
          <Statistic title={t('login.metric.audit')} value={t('dashboard.syncStatus.ready')} />
        </Space>
      </Card>
    </main>
  );
}

function modeFromLoginPath(pathname) {
  if (pathname === '/auth/continue') return 'app';
  if (pathname === '/admin/login') return 'admin';
  if (/^\/t\/[^/]+\/admin\/login$/.test(pathname)) return 'entity_admin';
  return 'user';
}

function entitySlugFromPath(pathname) {
  return pathname.match(/^\/t\/([^/]+)\/admin\/login$/)?.[1] || '';
}

function brandFromSlug(slug) {
  return slug
    .split(/[-_]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function FeishuMark() {
  return (
    <span className="feishu-mark" aria-hidden="true">
      <span />
      <span />
      <span />
      <span />
    </span>
  );
}

function LoginPage({ branding }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [context, setContext] = useState(null);
  const [providers, setProviders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [providersLoading, setProvidersLoading] = useState(false);
  const [providersLoaded, setProvidersLoaded] = useState(false);
  const [autoRedirecting, setAutoRedirecting] = useState(false);
  const [error, setError] = useState('');
  const pathname = window.location.pathname;
  const params = new URLSearchParams(window.location.search);
  const pathMode = modeFromLoginPath(pathname);
  const isAdminAccountPath = pathMode === 'admin' || pathMode === 'entity_admin';
  const returnTo = params.get('return_to') || (isAdminAccountPath ? '/admin' : '/portal');
  const oidcClientId = returnToParam(returnTo, 'client_id');
  const workplaceProvider = normalizeWorkplaceProvider(
    params.get('workplace') ||
      params.get('workplace_provider') ||
      params.get('sso_provider') ||
      returnToParam(returnTo, 'workplace') ||
      returnToParam(returnTo, 'workplace_provider') ||
      returnToParam(returnTo, 'sso_provider'),
  );
  const isWorkplaceLogin = workplaceProvider === 'feishu';
  const canUseWorkplaceBridge = isWorkplaceLogin && isFeishuClient();
  const loginErrorParam = params.get('login_error') || '';
  const loginTraceID = params.get('trace_id') || '';
  const mode = context?.mode || pathMode;
  const isAdminAccountLogin = mode === 'admin' || mode === 'entity_admin';
  const loginAction = isAdminAccountLogin ? '/sapi/login/account' : '/api/login/account';
  const pathEntitySlug = entitySlugFromPath(pathname);
  const entityRef = context?.entity?.id || context?.entity?.slug || pathEntitySlug;
  const entityBrand = context?.entity?.brand_name || context?.entity?.name || brandFromSlug(pathEntitySlug);
  const entityLabel = entityBrand || context?.entity?.slug || context?.entity?.id || '';
  const applicationName = context?.application?.name || (mode === 'app' ? 'Demo App' : branding.platform_name || t('app.title'));
  const brandName = branding.platform_name || t('app.title');
  const logo = context?.entity?.logo_url || branding.logo_url || '/logo.svg';
  const isEnterpriseEntrance = mode === 'app' || Boolean(entityRef);
  const currentHost = window.location.host;

  const label = (key, fallback) => {
    const value = t(key);
    return value === key ? fallback : value;
  };

  const loginErrorText = (key, traceID = '') => {
    if (!key) return '';
    const translationKey = `login.error.${key}`;
    const translated = t(translationKey);
    const base = translated === translationKey ? t('login.error.generic') : translated;
    if (!traceID) return base;
    return `${base} ${label('login.error.traceId', '追踪 ID：{traceId}').replace('{traceId}', traceID)}`;
  };

  useEffect(() => {
    async function load() {
      setLoading(true);
      setProvidersLoading(false);
      setProvidersLoaded(false);
      setAutoRedirecting(false);
      setError('');
      try {
        const next = isAdminAccountPath
          ? await api.getAdminLoginContext({ path: pathname, return_to: returnTo })
          : await api.getLoginContext({ path: pathname, return_to: returnTo });
        setContext(next);
        const loginError = loginErrorText(loginErrorParam, loginTraceID);
        if (loginError) setError(loginError);
        if (!loginError && next?.auto_redirect_url && !canUseWorkplaceBridge) {
          setAutoRedirecting(true);
          navigate(next.auto_redirect_url);
          return;
        }
        const ref = next?.entity?.id || next?.entity?.slug || '';
        let providerList = [];
        if (!isAdminAccountLogin && ref && next?.methods?.includes('feishu')) {
          setProvidersLoading(true);
          providerList = await api.listLoginProviders(ref, oidcClientId);
          setProviders(providerList);
          setProvidersLoaded(true);
          setProvidersLoading(false);
        }

        if (!loginError && !isAdminAccountLogin && canUseWorkplaceBridge && ref) {
          const provider = providerList.find((item) => item.provider === 'feishu');
          if (!provider?.workplace_exchange_url || !provider.app_id) {
            setError(t('login.error.generic'));
          } else {
            setAutoRedirecting(true);
            const authCode = queryAuthCode(params, isWorkplaceLogin) || (await feishuAuthCodeFromBridge(provider.app_id));
            await api.exchangeFeishuAppCode({ auth_code: authCode, entity_id: ref, client_id: oidcClientId });
            navigate(returnTo);
          }
        }
      } catch (err) {
        const loginError = loginErrorText(loginErrorParam, loginTraceID);
        setError(loginError || t('login.error.generic'));
      } finally {
        setLoading(false);
        setProvidersLoading(false);
      }
    }
    load();
  }, [isAdminAccountPath, isAdminAccountLogin, pathname, returnTo, isWorkplaceLogin, oidcClientId, loginErrorParam, loginTraceID, t]);

  const feishu = providers.find((item) => item.provider === 'feishu' && (item.oauth_url || item.workplace_exchange_url));
  const providerLoginUrl = context?.auto_redirect_url || feishu?.oauth_url || getFeishuLoginUrl(entityRef, returnTo);
  const primaryActionLabel = mode === 'app'
    ? label('login.enterprise.feishuForApp', `使用飞书继续登录 ${applicationName}`).replace('{app}', applicationName)
    : t('login.enterprise.feishuPrimary');
  const title = mode === 'app'
    ? label('login.enterprise.titleForApp', `登录 ${applicationName}`).replace('{app}', applicationName)
    : mode === 'entity_admin'
      ? label('login.entity_admin.titleWithBrand', `登录 ${entityLabel || brandName}`).replace('{brand}', entityLabel || brandName)
      : isAdminAccountLogin
        ? label('login.admin.title', '管理员登录')
        : t('login.title');
  const subtitle = isAdminAccountLogin
    ? label(`login.${mode}.subtitleSafe`, '使用独立账号登录。')
    : context?.entity?.login_message || (isEnterpriseEntrance
    ? label('login.enterprise.subtitle', '使用飞书完成企业员工身份登录。').replace('{entity}', entityLabel || label('login.enterprise.defaultEntity', '企业'))
    : t('login.feishuSubtitle'));
  const adminEyebrow = label(`login.${mode}.eyebrow`, label('login.admin.eyebrow', '管理员登录'));
  const adminFormEyebrow = label(`login.${mode}.formEyebrow`, label('login.admin.formEyebrow', '管理员'));
  const adminFormTitle = label(`login.${mode}.formTitle`, label('login.admin.formTitle', '管理平台'));

  return (
    <main className="auth-page">
      <section className={`auth-copy${isAdminAccountLogin ? ' auth-copy-admin' : ''}`}>
        <a className="auth-brand" href="/" aria-label={brandName}>
          <img src={logo} alt="" />
          <span>{context?.entity?.brand_name || context?.entity?.name || brandName}</span>
        </a>
        <p className="auth-eyebrow">{isAdminAccountLogin ? adminEyebrow : label('login.enterprise.eyebrow', '企业 SSO')}</p>
        <h1>{title}</h1>
        <p>{subtitle}</p>
        {isAdminAccountLogin ? (
          <div className="auth-trust-panel" aria-label={t('login.trust.ariaLabel')}>
            <div>
              <ShieldCheck size={17} />
              <span>{label('login.trust.authorizedOnly', '仅限授权管理员访问')}</span>
            </div>
            <div>
              <Network size={17} />
              <span>{label('login.trust.currentDomain', '当前域名：{host}').replace('{host}', currentHost)}</span>
            </div>
            <div>
              <KeyRound size={17} />
              <span>{label('login.trust.noDownloads', '本页不会要求安装软件或下载文件')}</span>
            </div>
            <div>
              <Mail size={17} />
              <span>
                {label('login.trust.support', '支持邮箱：')}
                <a href="mailto:smices@gmail.com">smices@gmail.com</a>
              </span>
            </div>
          </div>
        ) : null}
      </section>
      <Card className="auth-card login-card">
        {loading || autoRedirecting ? <Skeleton active paragraph={{ rows: 4 }} /> : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {error ? <Alert type="error" showIcon message={error} /> : null}
            <div className="login-card-heading">
              <Typography.Text type="secondary">{isAdminAccountLogin ? adminFormEyebrow : label('login.enterprise.formEyebrow', '安全登录')}</Typography.Text>
              <Typography.Title level={3}>{isAdminAccountLogin ? adminFormTitle : label('login.enterprise.formTitle', '使用企业身份继续')}</Typography.Title>
            </div>
            {!isAdminAccountLogin ? (
              providersLoading ? (
                <Skeleton.Button active block style={{ height: 46 }} />
              ) : (
                <a className="feishu-login-button" href={providerLoginUrl}>
                  <FeishuMark />
                  <span>{isWorkplaceLogin ? label('login.workplace.retry', '重试飞书工作台登录') : primaryActionLabel}</span>
                </a>
              )
            ) : null}
            {providersLoaded && entityRef && !feishu && !isAdminAccountLogin ? <Alert type="warning" showIcon message={t('login.error.generic')} /> : null}
            {isAdminAccountLogin ? (
              <form method="post" action={loginAction}>
                <input type="hidden" name="return_to" value={returnTo} />
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <label>
                    <Typography.Text>{t(`login.${mode}.accountLabel`)}</Typography.Text>
                    <Input name="account" autoComplete="username" required prefix={<KeyRound size={15} />} placeholder={t(`login.${mode}.accountPlaceholder`)} />
                  </label>
                  <label>
                    <Typography.Text>{t('login.password')}</Typography.Text>
                    <Input.Password name="password" autoComplete="current-password" required placeholder={t('login.passwordPlaceholder')} />
                  </label>
                  <Button htmlType="submit" type="primary" block onClick={() => message.destroy()}>{t(`login.${mode}.submit`)}</Button>
                  <Typography.Text className="login-origin-note" type="secondary">
                    {label('login.trust.originNote', '提交前请确认浏览器地址为 {host}。').replace('{host}', currentHost)}
                  </Typography.Text>
                </Space>
              </form>
            ) : null}
          </Space>
        )}
      </Card>
    </main>
  );
}

function getFeishuLoginUrl(entityRef, returnTo) {
  const params = new URLSearchParams();
  if (entityRef) params.set('entity_id', entityRef);
  params.set('return_to', returnTo);
  return `/api/auth/feishu/login?${params.toString()}`;
}

function PortalShell({ user, branding, themeState, onLogout }) {
  const { t } = useTranslation();
  const pathname = window.location.pathname;
  const brandName = branding.platform_name || t('app.title');
  const items = portalItems(t);
  const activePortalPath = [...items].sort((a, b) => b.path.length - a.path.length).find((item) => pathname === item.path || pathname.startsWith(`${item.path}/`))?.path || '/portal';
  return (
    <div className="portal-shell">
      <header className="portal-topbar">
        <div className="portal-topbar-inner">
          <a className="brand-mark" href="/portal" aria-label={brandName}><img src={branding.logo_url || '/logo.svg'} alt="" /> <strong>{brandName}</strong></a>
          <Segmented
            value={activePortalPath}
            options={items.map((item) => ({ value: item.path, label: <Space>{item.icon}{item.label}</Space> }))}
            onChange={(path) => navigate(path)}
          />
          <UserMenu user={user} {...themeState} onThemeChange={themeState.setThemeMode} onLanguageChange={themeState.setLanguage} onLogout={onLogout} profileHref="/portal/profile" />
        </div>
      </header>
      <main className="portal-content">
        {pathname === '/portal/profile' ? <ProfilePage user={user} admin={false} /> : <PortalPage />}
      </main>
    </div>
  );
}

function AdminShell({ user, branding, themeState, onLogout }) {
  const { t } = useTranslation();
  const pathname = window.location.pathname;
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(SIDEBAR_KEY) === '1');
  const active = routeTitle(pathname, t);
  const selectedKey = active.path || '/admin';
  const brandName = branding.platform_name || t('app.title');
  const groups = adminItems(t);
  const menuItems = groups.map((group) => ({
    key: group.key,
    label: group.label,
    type: 'group',
    children: group.children
      .filter((item) => !['/admin/platform', '/admin/admin-users'].includes(item.path) || user?.capabilities?.includes('system'))
      .map((item) => ({ key: item.path, icon: item.icon, label: item.label })),
  }));

  const toggle = () => {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(SIDEBAR_KEY, next ? '1' : '0');
  };

  return (
    <Layout className="shell">
      <Sider className="sidebar" width={260} collapsedWidth={76} collapsible collapsed={collapsed} trigger={null}>
        <a className="brand" href="/admin" aria-label={brandName}>
          <span className="brand-mark"><img src={branding.logo_url || '/logo.svg'} alt="" /></span>
          {!collapsed ? <span><strong>{brandName}</strong><span>{t('layout.adminConsole')}</span></span> : null}
        </a>
        <Button
          className="sidebar-rail-toggle"
          icon={collapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
          onClick={toggle}
          title={collapsed ? t('layout.expandSidebar') : t('layout.collapseSidebar')}
          aria-label={collapsed ? t('layout.expandSidebar') : t('layout.collapseSidebar')}
        />
        <Menu className="nav-menu" mode="inline" selectedKeys={[selectedKey]} items={menuItems} onClick={({ key }) => navigate(key)} />
      </Sider>
      <Layout>
        <Header className="topbar">
          <div className="topbar-title">
            <h1>{active.label}</h1>
            <p>{active.description}</p>
          </div>
          <UserMenu user={user} {...themeState} onThemeChange={themeState.setThemeMode} onLanguageChange={themeState.setLanguage} onLogout={onLogout} profileHref="/admin/profile" />
        </Header>
        <Content className="content">
          {pathname === '/admin/profile' ? <ProfilePage user={user} admin /> : (
            <Suspense fallback={<Skeleton active paragraph={{ rows: 8 }} />}>
              <AdminPagesRouter />
            </Suspense>
          )}
        </Content>
      </Layout>
    </Layout>
  );
}

const AdminPagesRouter = React.lazy(() => import('./admin-pages.jsx').then((module) => ({ default: module.AdminRouter })));

function ProfilePage({ user, admin }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  useEffect(() => {
    profileForm.setFieldsValue(user);
  }, [profileForm, user]);
  const saveProfile = async () => {
    const values = await profileForm.validateFields();
    await (admin ? api.updateAdminMe({ display_name: values.display_name }) : api.updateMe({ display_name: values.display_name }));
    message.success(t('common.updateSuccess'));
  };
  const savePassword = async () => {
    const values = await passwordForm.validateFields();
    await (admin ? api.updateAdminPassword(values) : api.updatePassword(values));
    passwordForm.resetFields();
    message.success(t('common.updateSuccess'));
  };
  return (
    <div className="two-column">
      <Card title={t('profile.title')} extra={<Button onClick={saveProfile}>{t('common.save')}</Button>}>
        <Form form={profileForm} layout="vertical"><Form.Item name="display_name" label={t('profile.displayName')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="username" label={t('users.username')}><Input disabled /></Form.Item><Form.Item name="email" label={t('users.email')}><Input disabled /></Form.Item></Form>
      </Card>
      <Card title={t('profile.updatePassword')} extra={<Button onClick={savePassword}>{t('common.save')}</Button>}>
        <Form form={passwordForm} layout="vertical"><Form.Item name="current_password" label={t('profile.currentPassword')} rules={[{ required: true }]}><Input.Password /></Form.Item><Form.Item name="new_password" label={t('profile.newPassword')} rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item></Form>
      </Card>
    </div>
  );
}

function PortalPage() {
  const { t } = useTranslation();
  const { loading, data } = useLoader(() => api.myAccess(), []);
  const apps = data?.applications || [];
  return <Card title={t('portal.title')} loading={loading}>{apps.length ? <Table rowKey="application_id" dataSource={apps} columns={[{ title: t('portal.application'), dataIndex: 'application_name' }, { title: t('portal.type'), dataIndex: 'application_type' }, { title: t('portal.access'), dataIndex: 'has_access', render: (v) => <Tag color={v ? 'green' : 'red'}>{t(v ? 'portal.accessible' : 'portal.inaccessible')}</Tag> }, { title: t('users.roles'), dataIndex: 'roles', render: (roles) => roles?.map((r) => <Tag key={r.role_id}>{r.role_code}</Tag>) }]} /> : <Empty description={t('common.empty')} />}</Card>;
}

createRoot(document.getElementById('root')).render(<Root />);
