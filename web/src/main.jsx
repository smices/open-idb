import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import {
  App as AntApp,
  Alert,
  Button,
  Card,
  Checkbox,
  ConfigProvider,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Layout,
  Menu,
  Modal,
  Popconfirm,
  Select,
  Segmented,
  Skeleton,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  theme as antdTheme,
  Tree,
  Typography,
} from 'antd';
import {
  AppWindow,
  Building2,
  ChevronLeft,
  ChevronRight,
  FileSearch,
  Gauge,
  GitBranch,
  KeyRound,
  Network,
  Plus,
  RefreshCw,
  Save,
  Search,
  Settings,
  ShieldCheck,
  UserCog,
  UserRound,
  UsersRound,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { api } from './lib/api.ts';
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

  const handleLogout = () => {
    document.cookie = 'idb_session=; Max-Age=0; Path=/;';
    document.cookie = 'idb_admin_session=; Max-Age=0; Path=/;';
    setUser(null);
    message.success(t('layout.logout'));
    navigate(kind === 'admin' ? '/admin/login' : '/login');
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

const FEISHU_H5_SDK_URLS = [
  import.meta.env.VITE_FEISHU_H5_SDK_URL?.trim(),
  import.meta.env.PUBLIC_FEISHU_H5_SDK_URL?.trim(),
  'https://lf1-cdn-tos.bytegoofy.com/goofy/lark/op/h5-js-sdk-1.5.35.js',
].filter(Boolean);

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

function normalizeWorkplaceProvider(provider) {
  const value = (provider || '').trim().toLowerCase();
  if (value === 'lark') return 'feishu';
  return value === 'feishu' ? value : '';
}

function returnToParam(returnToValue, name) {
  try {
    return new URL(returnToValue, window.location.origin).searchParams.get(name) || '';
  } catch {
    return '';
  }
}

function queryAuthCode(params, isWorkplaceLogin) {
  return params.get('auth_code') || params.get('feishu_auth_code') || (isWorkplaceLogin ? params.get('code') || '' : '');
}

function errorDetail(error) {
  if (!error) return '';
  if (typeof error === 'string') return error;
  if (error instanceof Error) return error.message;
  try {
    return JSON.stringify(error);
  } catch {
    return String(error);
  }
}

function feishuBridgeHandler(bridgeWindow) {
  return (
    bridgeWindow.tt?.requestAuthCode ||
    bridgeWindow.tt?.getAuthCode ||
    bridgeWindow.h5sdk?.requestAuthCode ||
    bridgeWindow.h5sdk?.getAuthCode ||
    bridgeWindow.h5sdk?.biz?.auth?.getAuthCode
  );
}

function hasFeishuBridge(bridgeWindow) {
  return Boolean(feishuBridgeHandler(bridgeWindow) || bridgeWindow.h5sdk?.ready);
}

function loadScript(src) {
  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[data-idb-feishu-sdk="${src}"]`);
    if (existing?.dataset.loaded === 'true') {
      resolve();
      return;
    }
    if (existing) {
      existing.addEventListener('load', () => resolve(), { once: true });
      existing.addEventListener('error', () => reject(new Error(`failed to load ${src}`)), { once: true });
      return;
    }
    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    script.dataset.idbFeishuSdk = src;
    script.onload = () => {
      script.dataset.loaded = 'true';
      resolve();
    };
    script.onerror = () => reject(new Error(`failed to load ${src}`));
    document.head.appendChild(script);
  });
}

function waitForFeishuBridge(timeoutMs = 3000) {
  return new Promise((resolve, reject) => {
    const startedAt = Date.now();
    const check = () => {
      if (hasFeishuBridge(window)) {
        resolve();
        return;
      }
      if (Date.now() - startedAt >= timeoutMs) {
        reject(new Error('bridge_unavailable'));
        return;
      }
      window.setTimeout(check, 100);
    };
    check();
  });
}

async function ensureFeishuH5Sdk() {
  if (hasFeishuBridge(window)) return;
  let lastError = null;
  for (const src of FEISHU_H5_SDK_URLS) {
    try {
      await loadScript(src);
      await waitForFeishuBridge();
      return;
    } catch (error) {
      lastError = error;
    }
  }
  throw lastError || new Error('sdk_load_failed');
}

async function feishuAuthCodeFromBridge(appId) {
  await ensureFeishuH5Sdk();
  const requestCode = () =>
    new Promise((resolve, reject) => {
      const handler = feishuBridgeHandler(window);
      if (!handler) {
        reject(new Error('bridge_unavailable'));
        return;
      }
      handler({
        appId,
        success: (response) => {
          const code = response.code || response.auth_code || response.authCode || '';
          if (code) resolve(code);
          else reject(new Error(`auth_code_empty: ${errorDetail(response)}`));
        },
        fail: (error) => reject(new Error(`auth_code_failed: ${errorDetail(error)}`)),
      });
    });

  if (window.h5sdk?.ready) {
    return new Promise((resolve, reject) => {
      const timer = window.setTimeout(() => reject(new Error('bridge_timeout')), 10000);
      window.h5sdk?.error?.((error) => {
        window.clearTimeout(timer);
        reject(new Error(`auth_code_failed: ${errorDetail(error)}`));
      });
      window.h5sdk.ready(() => {
        requestCode()
          .then((code) => {
            window.clearTimeout(timer);
            resolve(code);
          })
          .catch((error) => {
            window.clearTimeout(timer);
            reject(error);
          });
      });
    });
  }
  return requestCode();
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

  const label = (key, fallback) => {
    const value = t(key);
    return value === key ? fallback : value;
  };

  const loginErrorText = (key) => {
    if (!key) return '';
    return t('login.error.generic');
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
        const loginError = loginErrorText(params.get('login_error'));
        if (loginError) setError(loginError);
        if (!loginError && next?.auto_redirect_url && !isWorkplaceLogin) {
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

        if (!isAdminAccountLogin && isWorkplaceLogin && ref) {
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
        const loginError = loginErrorText(params.get('login_error'));
        if (loginError) setError(loginError);
      } finally {
        setLoading(false);
        setProvidersLoading(false);
      }
    }
    load();
  }, [isAdminAccountPath, isAdminAccountLogin, pathname, returnTo, isWorkplaceLogin, oidcClientId, t]);

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
          <AdminRouter user={user} />
        </Content>
      </Layout>
    </Layout>
  );
}

function AdminRouter({ user }) {
  const pathname = window.location.pathname;
  if (pathname === '/admin') return <DashboardPage />;
  if (pathname === '/admin/entities') return <EntitiesPage />;
  if (pathname === '/admin/identity-sources') return <IdentitySourcesPage />;
  if (pathname === '/admin/sync-jobs') return <SyncJobsPage />;
  if (pathname === '/admin/organization') return <OrganizationPage />;
  if (pathname === '/admin/users') return <UsersPage />;
  if (/^\/admin\/users\/[^/]+$/.test(pathname)) return <UserDetailPage id={decodeURIComponent(pathname.split('/').pop())} />;
  if (pathname === '/admin/applications') return <ApplicationsPage />;
  if (pathname === '/admin/roles') return <RolesPage />;
  if (pathname === '/admin/audit') return <AuditPage />;
  if (pathname === '/admin/platform') return <PlatformPage />;
  if (pathname === '/admin/admin-users') return <AdminUsersPage />;
  if (pathname === '/admin/profile') return <ProfilePage user={user} admin />;
  return <Empty description={t('common.notFound')} />;
}

function PageCard({ title, extra, children }) {
  return <Card className="data-card" title={title} extra={extra}>{children}</Card>;
}

function useLoader(loader, deps = []) {
  const { message } = AntApp.useApp();
  const [state, setState] = useState({ loading: true, data: null, error: '' });
  const reload = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: '' }));
    try {
      const data = await loader();
      setState({ loading: false, data, error: '' });
      return data;
    } catch (err) {
      const msg = errorMessage(err);
      setState({ loading: false, data: null, error: msg });
      message.error(msg);
      return null;
    }
  }, deps);
  useEffect(() => {
    reload();
  }, [reload]);
  return { ...state, reload };
}

function DashboardPage() {
  const { t } = useTranslation();
  const { loading, data } = useLoader(() => api.dashboardSummary(), []);
  const metrics = [
    ['users', t('dashboard.users')],
    ['active_users', t('dashboard.activeUsers')],
    ['new_users', t('dashboard.newUsers')],
    ['admin_users', t('dashboard.adminUsers')],
    ['application_activity', t('dashboard.applicationActivity')],
    ['pending_authorization', t('dashboard.pendingAuthorization')],
  ];
  return (
    <div className="page-stack">
      <div className="metric-grid">
        {metrics.map(([key, label]) => <div className="metric-card" key={key}><span>{label}</span><strong>{loading ? '-' : data?.[key] ?? 0}</strong></div>)}
      </div>
      <PageCard title={t('dashboard.title')}>
        {loading ? <Skeleton active /> : <Descriptions column={2} bordered size="small" items={[
          { key: 'sync', label: t('dashboard.syncHealth'), children: data?.sync_health ? t(`dashboard.syncStatus.${data.sync_health}`, data.sync_health) : '-' },
          { key: 'users', label: t('dashboard.totalUsers'), children: data?.users ?? 0 },
        ]} />}
      </PageCard>
    </div>
  );
}

function EntitiesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(() => api.listEntities({ limit: 100 }), []);
  const items = data?.items || [];
  const save = async () => {
    const values = await form.validateFields();
    try {
      if (editing) await api.updateEntity(editing.id, values);
      else await api.createEntity(values);
      message.success(t(editing ? 'common.updateSuccess' : 'common.createSuccess'));
      setOpen(false);
      reload();
    } catch (err) {
      message.error(errorMessage(err));
    }
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setEditing(null); form.resetFields(); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={items} columns={[
        { title: t('entities.name'), dataIndex: 'name' },
        { title: t('entities.slug'), dataIndex: 'slug' },
        { title: t('entities.status'), dataIndex: 'status', render: (v) => <Tag>{t(`entities.status.${v}`, v)}</Tag> },
        { title: t('entities.defaultLocale'), dataIndex: 'default_locale' },
        { title: t('entities.brandName'), dataIndex: 'brand_name' },
        { title: t('entities.createdAt'), dataIndex: 'created_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Button size="small" onClick={() => { setEditing(row); form.setFieldsValue(row); setOpen(true); }}>{t('common.edit')}</Button> },
      ]} />
      <Modal title={editing ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)} destroyOnHidden>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('entities.name')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="slug" label={t('entities.slug')} rules={[{ required: !editing }]}><Input disabled={Boolean(editing)} /></Form.Item>
          <Form.Item name="status" label={t('entities.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`entities.status.${value}`, value) }))} /></Form.Item>
          <Form.Item name="default_locale" label={t('entities.defaultLocale')}><Select options={['zh-CN', 'en-US'].map((value) => ({ value, label: value }))} /></Form.Item>
          <Form.Item name="brand_name" label={t('entities.brandName')}><Input /></Form.Item>
          <Form.Item name="logo_url" label={t('entities.logoUrl')}><Input /></Form.Item>
          <Form.Item name="login_message" label={t('entities.loginMessage')}><Input.TextArea rows={3} /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function IdentitySourcesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [configOpen, setConfigOpen] = useState(false);
  const [configForm] = Form.useForm();
  const { loading, data, reload } = useLoader(() => api.listIdentitySources({ limit: 100 }), []);
  const sources = pickItems(data, ['items', 'sources']);
  const loadConfig = async () => {
    setConfigOpen(true);
    try {
      const cfg = await api.getFeishuIdentitySourceConfig();
      configForm.setFieldsValue({ ...cfg, ...(cfg.config || {}) });
    } catch {
      configForm.resetFields();
    }
  };
  const saveConfig = async () => {
    const values = await configForm.validateFields();
    await api.upsertFeishuIdentitySourceConfig({
      display_name: values.display_name || 'Feishu',
      status: values.status || 'active',
      sync_enabled: Boolean(values.sync_enabled),
      config: {
        app_id: values.app_id || '',
        app_secret: values.app_secret || '',
        encrypt_key: values.encrypt_key || '',
        verification_token: values.verification_token || '',
      },
    });
    message.success(t('common.updateSuccess'));
    setConfigOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar">
        <div />
        <Space>
          <Button icon={<Settings size={16} />} onClick={loadConfig}>{t('identitySources.feishuConfigTitle')}</Button>
          <Button type="primary" icon={<Plus size={16} />} onClick={async () => { await api.createIdentitySource({ type: 'feishu', name: 'Feishu', sync_enabled: true }); message.success(t('common.createSuccess')); reload(); }}>{t('identitySources.createFeishuSource')}</Button>
        </Space>
      </div>
      <Table rowKey="id" loading={loading} dataSource={sources} columns={[
        { title: t('entities.name'), dataIndex: 'name' },
        { title: t('identitySources.type'), dataIndex: 'type', render: (v) => t(`identitySources.type.${v}`, v) },
        { title: t('identitySources.status'), dataIndex: 'status', render: (v) => <Tag>{t(`identitySources.status.${v}`, v)}</Tag> },
        { title: t('identitySources.syncEnabled'), dataIndex: 'sync_enabled', render: (v) => <Tag color={v ? 'green' : undefined}>{t(Boolean(v) ? 'common.yes' : 'common.no')}</Tag> },
        { title: t('users.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={async () => { await api.triggerSourceSync(row.id, 'incremental'); message.success(t('identitySources.incrementalSyncStarted')); }}>{t('identitySources.triggerIncremental')}</Button>
          <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteIdentitySource(row.id); message.success(t('common.deleteSuccess')); reload(); }}><Button size="small" danger>{t('common.delete')}</Button></Popconfirm>
        </Space> },
      ]} />
      <Modal title={t('identitySources.feishuConfigTitle')} open={configOpen} onOk={saveConfig} onCancel={() => setConfigOpen(false)} destroyOnHidden>
        <Form form={configForm} layout="vertical">
          <Form.Item name="display_name" label={t('profile.displayName')}><Input /></Form.Item>
          <Form.Item name="status" label={t('identitySources.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`identitySources.status.${value}`, value) }))} /></Form.Item>
          <Form.Item name="sync_enabled" label={t('identitySources.enableSync')} valuePropName="checked"><Switch /></Form.Item>
          <Form.Item name="app_id" label={t('identitySources.appId')}><Input /></Form.Item>
          <Form.Item name="app_secret" label={t('identitySources.appSecret')}><Input.Password /></Form.Item>
          <Form.Item name="encrypt_key" label={t('identitySources.encryptKey')}><Input.Password /></Form.Item>
          <Form.Item name="verification_token" label={t('identitySources.verificationToken')}><Input.Password /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function SyncJobsPage() {
  const { t } = useTranslation();
  const { loading, data, reload } = useLoader(() => api.listSyncJobs({ limit: 100 }), []);
  return <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
    { title: t('syncJobs.provider'), dataIndex: 'provider' },
    { title: t('syncJobs.type'), dataIndex: 'type' },
    { title: t('syncJobs.status'), dataIndex: 'status', render: (v) => <Tag>{t(`syncJobs.status.${v}`, v)}</Tag> },
    { title: t('syncJobs.traceId'), dataIndex: 'trace_id', ellipsis: true },
    { title: t('syncJobs.startedAt'), dataIndex: 'started_at', render: formatDate },
    { title: t('syncJobs.finishedAt'), dataIndex: 'finished_at', render: formatDate },
    { title: t('syncJobs.error'), dataIndex: 'error_message', ellipsis: true },
  ]} title={() => <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>} />;
}

function OrganizationPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [treeData, setTreeData] = useState([]);
  const [selected, setSelected] = useState(null);
  const [loading, setLoading] = useState(true);
  const [query, setQuery] = useState('');
  const toNode = (node) => ({ key: `${node.kind}:${node.id}`, title: `${node.name} · ${node.kind}`, raw: node, isLeaf: !node.has_children });
  const loadRoot = async () => {
    setLoading(true);
    try {
      const root = await api.getOrganizationTreeRoot({ limit: 100 });
      setTreeData([toNode(root.root), ...root.children.map(toNode)]);
    } catch (err) {
      message.error(errorMessage(err));
    } finally {
      setLoading(false);
    }
  };
  useEffect(() => { loadRoot(); }, []);
  const onLoadData = async (node) => {
    const raw = node.raw;
    if (!raw || node.children?.length) return;
    const res = await api.listOrganizationTreeChildren({ kind: raw.kind, id: raw.id, limit: 100 });
    node.children = (res.items || []).map(toNode);
    setTreeData([...treeData]);
  };
  const search = async () => {
    if (!query.trim()) return loadRoot();
    const res = await api.searchOrganizationTree({ q: query.trim(), limit: 100 });
    setTreeData((res.items || []).map(toNode));
  };
  return (
    <div className="two-column">
      <Card title={t('organization.organizationTree')} extra={<Space.Compact><Input value={query} onChange={(e) => setQuery(e.target.value)} onPressEnter={search} placeholder={t('organization.search')} /><Button icon={<Search size={16} />} onClick={search} aria-label={t('organization.search')} /></Space.Compact>}>
        {loading ? <Skeleton active /> : <Tree showLine loadData={onLoadData} treeData={treeData} onSelect={(_, info) => setSelected(info.node.raw)} />}
      </Card>
      <Card title={t('directory.details')}>
        {selected ? <Descriptions bordered size="small" column={1} items={Object.entries(selected).map(([key, value]) => ({ key, label: key, children: typeof value === 'boolean' ? String(value) : value || '-' }))} /> : <Empty />}
      </Card>
    </div>
  );
}

function UsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [status, setStatus] = useState('');
  const { loading, data, reload } = useLoader(() => api.listUsers({ status, limit: 100 }), [status]);
  return (
    <div className="page-stack">
      <div className="toolbar">
        <Select allowClear placeholder={t('users.status')} style={{ width: 180 }} value={status || undefined} onChange={(v) => setStatus(v || '')} options={['active', 'disabled', 'inactive'].map((value) => ({ value, label: t(`users.status.${value}`, value) }))} />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
        { title: t('users.username'), dataIndex: 'username' },
        { title: t('users.displayName'), dataIndex: 'display_name' },
        { title: t('users.email'), dataIndex: 'email' },
        { title: t('users.status'), dataIndex: 'lifecycle_status', render: (v) => <Tag>{t(`users.status.${v}`, v)}</Tag> },
        { title: t('users.type'), dataIndex: 'user_type', render: (v) => t(`users.type.${v}`, v) },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={() => navigate(`/admin/users/${encodeURIComponent(row.id)}`)}>{t('common.details')}</Button>
          <Button size="small" onClick={async () => { await (row.lifecycle_status === 'active' ? api.disableUser(row.id) : api.enableUser(row.id)); message.success(t('common.updateSuccess')); reload(); }}>{row.lifecycle_status === 'active' ? t('common.disable') : t('common.enable')}</Button>
        </Space> },
      ]} />
    </div>
  );
}

function UserDetailPage({ id }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [editOpen, setEditOpen] = useState(false);
  const [bindingOpen, setBindingOpen] = useState(false);
  const [form] = Form.useForm();
  const [bindingForm] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [user, roles, allRoles, sessions, bindings, sources] = await Promise.all([
      api.getUser(id),
      api.getUserRoles(id).catch(() => []),
      api.listRoles({ limit: 200 }).catch(() => ({ items: [] })),
      api.listUserSessions(id, { limit: 50 }).catch(() => ({ items: [] })),
      api.listUserBindings(id).catch(() => []),
      api.listIdentitySources({ limit: 100 }).catch(() => ({ items: [], sources: [] })),
    ]);
    return { user, roles, allRoles: allRoles.items || [], sessions: sessions.items || [], bindings, sources: pickItems(sources, ['items', 'sources']) };
  }, [id]);
  if (loading || !data) return <Skeleton active paragraph={{ rows: 8 }} />;
  const saveUser = async () => {
    const values = await form.validateFields();
    await api.updateUser(id, values);
    message.success(t('common.updateSuccess'));
    setEditOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <PageCard title={data.user.display_name || data.user.username} extra={<Button onClick={() => { form.setFieldsValue(data.user); setEditOpen(true); }}>{t('common.edit')}</Button>}>
        <Descriptions bordered size="small" column={2} items={[
          { key: 'username', label: t('users.username'), children: data.user.username },
          { key: 'email', label: t('users.email'), children: data.user.email || '-' },
          { key: 'phone', label: t('users.phone'), children: data.user.phone || '-' },
          { key: 'status', label: t('users.status'), children: <Tag>{t(`users.status.${data.user.lifecycle_status}`, data.user.lifecycle_status)}</Tag> },
          { key: 'locale', label: t('users.locale'), children: data.user.locale },
          { key: 'updated', label: t('users.updatedAt'), children: formatDate(data.user.updated_at) },
        ]} />
      </PageCard>
      <PageCard title={t('users.roles')} extra={<Select placeholder={t('users.assignRole')} style={{ width: 220 }} options={data.allRoles.map((r) => ({ value: r.id, label: `${r.name} (${r.code})` }))} onChange={async (roleId) => { await api.assignRoleToUser(id, roleId); message.success(t('users.roleAssigned')); reload(); }} />}>
        <Space wrap>{data.roles.map((role) => <Tag key={role.id} closable onClose={(e) => { e.preventDefault(); api.removeRoleFromUser(id, role.id).then(reload); }}>{role.name}</Tag>)}</Space>
      </PageCard>
      <Table title={() => <Space>{t('users.bindings')}<Button size="small" icon={<Plus size={14} />} onClick={() => setBindingOpen(true)}>{t('common.create')}</Button></Space>} rowKey="id" dataSource={data.bindings} columns={[
        { title: t('users.source'), dataIndex: 'source_name' },
        { title: t('users.providerUid'), dataIndex: 'provider_uid' },
        { title: t('users.primaryBinding'), dataIndex: 'is_primary', render: (v) => t(Boolean(v) ? 'common.yes' : 'common.no') },
        { title: t('users.boundAt'), dataIndex: 'bound_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteUserBinding(id, row.id); reload(); }}><Button danger size="small">{t('common.delete')}</Button></Popconfirm> },
      ]} />
      <Table title={() => t('users.sessions')} rowKey="id" dataSource={data.sessions} columns={[
        { title: t('users.loginMethod'), dataIndex: 'login_method' },
        { title: t('users.status'), dataIndex: 'status', render: (v) => <Tag>{t(`users.sessionStatus.${v}`, v)}</Tag> },
        { title: t('users.ip'), dataIndex: 'ip' },
        { title: t('users.createdAt'), dataIndex: 'created_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Button size="small" onClick={async () => { await api.revokeSession(row.id); reload(); }}>{t('users.revokeSession')}</Button> },
      ]} />
      <Modal title={t('common.edit')} open={editOpen} onOk={saveUser} onCancel={() => setEditOpen(false)}>
        <Form form={form} layout="vertical">
          <Form.Item name="display_name" label={t('users.displayName')}><Input /></Form.Item>
          <Form.Item name="email" label={t('users.email')}><Input /></Form.Item>
          <Form.Item name="phone" label={t('users.phone')}><Input /></Form.Item>
          <Form.Item name="locale" label={t('users.locale')}><Select options={['zh-CN', 'en-US'].map((value) => ({ value, label: value }))} /></Form.Item>
        </Form>
      </Modal>
      <Modal title={t('users.createBinding')} open={bindingOpen} onOk={async () => { const values = await bindingForm.validateFields(); await api.createUserBinding(id, values); setBindingOpen(false); reload(); }} onCancel={() => setBindingOpen(false)}>
        <Form form={bindingForm} layout="vertical">
          <Form.Item name="source_id" label={t('users.source')} rules={[{ required: true }]}><Select options={data.sources.map((s) => ({ value: s.id, label: s.name }))} /></Form.Item>
          <Form.Item name="directory_user_id" label={t('users.directoryUserId')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="provider_uid" label={t('users.providerUid')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="provider_union_id" label={t('users.providerUnionId')}><Input /></Form.Item>
          <Form.Item name="is_primary" label={t('users.primaryBinding')} valuePropName="checked"><Checkbox /></Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function ApplicationsPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState(null);
  const [drawer, setDrawer] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [apps, roles, clients] = await Promise.all([api.listApplications({ limit: 100 }), api.listRoles({ limit: 200 }).catch(() => ({ items: [] })), api.listOIDCClients({ limit: 200 }).catch(() => ({ clients: [] }))]);
    return { apps: apps.applications || [], roles: roles.items || [], clients: clients.clients || [] };
  }, []);
  const save = async () => {
    const values = await form.validateFields();
    if (selected) await api.updateApplication(selected.id, { name: values.name, status: values.status });
    else await api.createApplication({ name: values.name, type: values.type || 'oidc' });
    message.success(t('common.updateSuccess'));
    setOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setSelected(null); form.resetFields(); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={data?.apps || []} columns={[
        { title: t('applications.name'), dataIndex: 'name' },
        { title: t('applications.type'), dataIndex: 'type', render: (v) => t(`applications.type.${v}`, v) },
        { title: t('applications.status'), dataIndex: 'status', render: (v) => <Tag>{t(`applications.status.${v}`, v)}</Tag> },
        { title: t('applications.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space>
          <Button size="small" onClick={() => { setSelected(row); form.setFieldsValue(row); setOpen(true); }}>{t('common.edit')}</Button>
          <Button size="small" onClick={() => setDrawer(row)}>{t('applications.accessOidc')}</Button>
          <Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteApplication(row.id); reload(); }}><Button danger size="small">{t('common.delete')}</Button></Popconfirm>
        </Space> },
      ]} />
      <Modal title={selected ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical"><Form.Item name="name" label={t('applications.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="type" label={t('applications.type')}><Select disabled={Boolean(selected)} options={['oidc', 'saml', 'custom'].map((value) => ({ value, label: t(`applications.type.${value}`, value) }))} /></Form.Item><Form.Item name="status" label={t('applications.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`applications.status.${value}`, value) }))} /></Form.Item></Form>
      </Modal>
      <ApplicationDrawer app={drawer} roles={data?.roles || []} client={data?.clients?.find((c) => c.application_id === drawer?.id)} onClose={() => setDrawer(null)} onDone={reload} />
    </div>
  );
}

function ApplicationDrawer({ app, roles, client, onClose, onDone }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [roleIds, setRoleIds] = useState([]);
  const [form] = Form.useForm();
  useEffect(() => {
    async function load() {
      if (!app) return;
      const assignments = await api.listApplicationRoleAssignments(app.id).catch(() => ({ items: [], roles: [] }));
      setRoleIds(pickItems(assignments, ['items', 'roles']).map((item) => item.role_id));
      form.setFieldsValue(client || { redirect_uris: [], allowed_scopes: ['openid', 'profile'], grant_types: ['authorization_code'], response_types: ['code'], pkce_required: true });
    }
    load();
  }, [app, client, form]);
  if (!app) return null;
  const saveAccess = async () => {
    await api.setApplicationRoleAssignments(app.id, roleIds);
    message.success(t('applications.saveSuccess'));
    onDone?.();
  };
  const saveClient = async () => {
    const values = await form.validateFields();
    const payload = { ...values, application_id: app.id };
    if (client?.id) await api.updateOIDCClient(client.id, values);
    else {
      const res = await api.createOIDCClient(payload);
      Modal.info({ title: t('applications.clientSecret'), content: <pre className="json-box">{res.client_secret}</pre> });
    }
    message.success(t('applications.saveSuccess'));
    onDone?.();
  };
  return (
    <Drawer title={app.name} width={560} open={Boolean(app)} onClose={onClose}>
      <Space direction="vertical" size={18} style={{ width: '100%' }}>
        <Card title={t('applications.accessRoles')} extra={<Button onClick={saveAccess}>{t('common.save')}</Button>}>
          <Select mode="multiple" style={{ width: '100%' }} value={roleIds} onChange={setRoleIds} options={roles.map((r) => ({ value: r.id, label: `${r.name} (${r.code})` }))} />
        </Card>
        <Card title={t('applications.oidcClient')} extra={<Button onClick={saveClient}>{t('common.save')}</Button>}>
          <Form form={form} layout="vertical">
            <Form.Item name="client_id" label={t('applications.clientId')}><Input disabled={Boolean(client?.id)} placeholder={t('applications.autoWhenEmpty')} /></Form.Item>
            <Form.Item name="redirect_uris" label={t('applications.redirectUris')}><Select mode="tags" /></Form.Item>
            <Form.Item name="allowed_scopes" label={t('applications.scopes')}><Select mode="tags" /></Form.Item>
            <Form.Item name="grant_types" label={t('applications.grantTypes')}><Select mode="tags" /></Form.Item>
            <Form.Item name="response_types" label={t('applications.responseTypes')}><Select mode="tags" /></Form.Item>
            <Form.Item name="pkce_required" label={t('applications.pkce')} valuePropName="checked"><Switch /></Form.Item>
            <Form.Item name="workplace_provider" label={t('applications.workplaceProvider')}><Input /></Form.Item>
            <Form.Item name="workplace_app_id" label={t('applications.workplaceAppId')}><Input /></Form.Item>
            <Form.Item name="workplace_app_secret" label={t('applications.workplaceAppSecret')}><Input.Password /></Form.Item>
          </Form>
        </Card>
      </Space>
    </Drawer>
  );
}

function RolesPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [roleOpen, setRoleOpen] = useState(false);
  const [permOpen, setPermOpen] = useState(false);
  const [selectedRole, setSelectedRole] = useState(null);
  const [roleForm] = Form.useForm();
  const [permForm] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [roles, permissions] = await Promise.all([api.listRoles({ limit: 200 }), api.listPermissions({ limit: 200 })]);
    return { roles: roles.items || [], permissions: permissions.items || [] };
  }, []);
  const saveRole = async () => {
    const values = await roleForm.validateFields();
    if (selectedRole) await api.updateRole(selectedRole.id, values);
    else await api.createRole(values);
    message.success(t('common.updateSuccess'));
    setRoleOpen(false);
    reload();
  };
  const savePermission = async () => {
    const values = await permForm.validateFields();
    await api.createPermission(values);
    message.success(t('common.createSuccess'));
    setPermOpen(false);
    reload();
  };
  return (
    <div className="two-column">
      <Card title={t('roles.title')} extra={<Button icon={<Plus size={16} />} onClick={() => { setSelectedRole(null); roleForm.resetFields(); setRoleOpen(true); }}>{t('common.create')}</Button>}>
        <Table rowKey="id" loading={loading} dataSource={data?.roles || []} pagination={false} columns={[
          { title: t('roles.name'), dataIndex: 'name' },
          { title: t('roles.code'), dataIndex: 'code' },
          { title: t('common.actions'), render: (_, row) => <Space><Button size="small" onClick={() => { setSelectedRole(row); roleForm.setFieldsValue(row); setRoleOpen(true); }}>{t('common.edit')}</Button><RolePermissionEditor role={row} permissions={data?.permissions || []} /></Space> },
        ]} />
      </Card>
      <Card title={t('roles.permissions')} extra={<Button icon={<Plus size={16} />} onClick={() => setPermOpen(true)}>{t('common.create')}</Button>}>
        <Table rowKey="id" loading={loading} dataSource={data?.permissions || []} pagination={{ pageSize: 8 }} columns={[{ title: t('roles.code'), dataIndex: 'code' }, { title: t('roles.name'), dataIndex: 'name' }, { title: t('roles.type'), dataIndex: 'type' }]} />
      </Card>
      <Modal title={t('roles.role')} open={roleOpen} onOk={saveRole} onCancel={() => setRoleOpen(false)}><Form form={roleForm} layout="vertical"><Form.Item name="name" label={t('roles.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="code" label={t('roles.code')} rules={[{ required: !selectedRole }]}><Input disabled={Boolean(selectedRole)} /></Form.Item><Form.Item name="description" label={t('roles.description')}><Input.TextArea /></Form.Item></Form></Modal>
      <Modal title={t('roles.permission')} open={permOpen} onOk={savePermission} onCancel={() => setPermOpen(false)}><Form form={permForm} layout="vertical"><Form.Item name="code" label={t('roles.code')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="name" label={t('roles.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="type" label={t('roles.type')} rules={[{ required: true }]}><Select options={['api', 'menu', 'data'].map((value) => ({ value, label: t(`roles.permissionType.${value}`, value) }))} /></Form.Item></Form></Modal>
    </div>
  );
}

function RolePermissionEditor({ role, permissions }) {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState([]);
  const load = async () => {
    const current = await api.listRolePermissions(role.id).catch(() => []);
    setSelected(current.map((p) => p.id));
    setOpen(true);
  };
  const save = async () => {
    const current = await api.listRolePermissions(role.id).catch(() => []);
    const currentIds = new Set(current.map((p) => p.id));
    const nextIds = new Set(selected);
    await Promise.all([
      ...selected.filter((id) => !currentIds.has(id)).map((id) => api.assignPermissionToRole(role.id, id)),
      ...current.filter((p) => !nextIds.has(p.id)).map((p) => api.removePermissionFromRole(role.id, p.id)),
    ]);
    message.success(t('roles.permissionSaveSuccess'));
    setOpen(false);
  };
  return <><Button size="small" onClick={load}>{t('roles.permissions')}</Button><Modal title={t('roles.permissionDialogTitle', { name: role.name })} open={open} onOk={save} onCancel={() => setOpen(false)}><Select mode="multiple" style={{ width: '100%' }} value={selected} onChange={setSelected} options={permissions.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))} /></Modal></>;
}

function AuditPage() {
  const { t } = useTranslation();
  const [filters, setFilters] = useState({});
  const [selected, setSelected] = useState(null);
  const { loading, data, reload } = useLoader(() => api.listAuditLogs({ ...filters, limit: 100 }), [JSON.stringify(filters)]);
  return (
    <div className="page-stack">
      <div className="toolbar-left">
        <Input placeholder={t('audit.action')} style={{ width: 180 }} onChange={(e) => setFilters((f) => ({ ...f, action: e.target.value || undefined }))} />
        <Input placeholder={t('audit.resourceType')} style={{ width: 180 }} onChange={(e) => setFilters((f) => ({ ...f, resource_type: e.target.value || undefined }))} />
        <Button icon={<RefreshCw size={16} />} onClick={reload}>{t('common.refresh')}</Button>
      </div>
      <Table rowKey="id" loading={loading} dataSource={data?.items || []} columns={[
        { title: t('audit.action'), dataIndex: 'action', render: (v) => t(`audit.action.${v}`, v) },
        { title: t('audit.actor'), dataIndex: 'actor_type', render: (v) => t(`audit.actor.${v}`, v) },
        { title: t('audit.resourceType'), render: (_, row) => `${t(`audit.resource.${row.resource_type}`, row.resource_type)}/${row.resource_id}` },
        { title: t('audit.ip'), dataIndex: 'ip' },
        { title: t('audit.time'), dataIndex: 'created_at', render: formatDate },
        { title: t('audit.traceId'), dataIndex: 'trace_id', ellipsis: true },
        { title: t('audit.details'), render: (_, row) => <Button size="small" onClick={() => setSelected(row)}>JSON</Button> },
      ]} />
      <Drawer width={640} open={Boolean(selected)} onClose={() => setSelected(null)} title={t('audit.detailTitle')}><pre className="json-box">{JSON.stringify(selected, null, 2)}</pre></Drawer>
    </div>
  );
}

function PlatformPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [form] = Form.useForm();
  const { loading } = useLoader(async () => {
    const branding = await api.getAdminPlatformBranding();
    form.setFieldsValue(branding);
    return branding;
  }, []);
  const save = async () => {
    const values = await form.validateFields();
    await api.updatePlatformBranding(values);
    message.success(t('common.updateSuccess'));
  };
  return <Card className="data-card" title={t('platform.title')} loading={loading} extra={<Button type="primary" icon={<Save size={16} />} onClick={save}>{t('common.save')}</Button>}><Form form={form} layout="vertical"><Form.Item name="platform_name" label={t('platform.name')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="logo_url" label={t('platform.logoUrl')}><Input /></Form.Item><Form.Item name="favicon_url" label={t('platform.faviconUrl')}><Input /></Form.Item><Form.Item name="title_suffix" label={t('platform.titleSuffix')}><Input /></Form.Item></Form></Card>;
}

function AdminUsersPage() {
  const { t } = useTranslation();
  const { message } = AntApp.useApp();
  const [open, setOpen] = useState(false);
  const [passwordMode, setPasswordMode] = useState(false);
  const [editing, setEditing] = useState(null);
  const [form] = Form.useForm();
  const { loading, data, reload } = useLoader(async () => {
    const [admins, roles, entities] = await Promise.all([api.listAdminUsers(), api.listAdminRoles(), api.listEntities({ limit: 200 }).catch(() => ({ items: [] }))]);
    return { admins: admins.items || [], roles, entities: entities.items || [] };
  }, []);
  const save = async () => {
    const values = await form.validateFields();
    if (passwordMode) await api.setAdminUserPassword(editing.id, values.password);
    else if (editing) await api.updateAdminUser(editing.id, values);
    else await api.createAdminUser(values);
    message.success(t('common.updateSuccess'));
    setOpen(false);
    reload();
  };
  return (
    <div className="page-stack">
      <div className="toolbar"><div /><Button type="primary" icon={<Plus size={16} />} onClick={() => { setEditing(null); setPasswordMode(false); form.resetFields(); setOpen(true); }}>{t('common.create')}</Button></div>
      <Table rowKey="id" loading={loading} dataSource={data?.admins || []} columns={[
        { title: t('adminUsers.account'), dataIndex: 'username' },
        { title: t('adminUsers.displayName'), dataIndex: 'display_name' },
        { title: t('adminUsers.role'), dataIndex: 'role', render: (v) => <Tag>{v}</Tag> },
        { title: t('adminUsers.status'), dataIndex: 'status', render: (v) => <Tag>{t(`adminUsers.status.${v}`, v)}</Tag> },
        { title: t('adminUsers.updatedAt'), dataIndex: 'updated_at', render: formatDate },
        { title: t('common.actions'), render: (_, row) => <Space><Button size="small" onClick={() => { setEditing(row); setPasswordMode(false); form.setFieldsValue(row); setOpen(true); }}>{t('common.edit')}</Button><Button size="small" onClick={() => { setEditing(row); setPasswordMode(true); form.resetFields(); setOpen(true); }}>{t('adminUsers.changePassword')}</Button><Popconfirm title={t('common.delete')} onConfirm={async () => { await api.deleteAdminUser(row.id); reload(); }}><Button size="small" danger>{t('common.delete')}</Button></Popconfirm></Space> },
      ]} />
      <Modal title={passwordMode ? t('adminUsers.changePassword') : editing ? t('common.edit') : t('common.create')} open={open} onOk={save} onCancel={() => setOpen(false)}>
        <Form form={form} layout="vertical">
          {!passwordMode ? <><Form.Item name="username" label={t('adminUsers.loginAccount')} rules={[{ required: !editing }]}><Input disabled={Boolean(editing)} /></Form.Item><Form.Item name="display_name" label={t('adminUsers.displayName')} rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="email" label={t('adminUsers.email')}><Input /></Form.Item><Form.Item name="role" label={t('adminUsers.role')} rules={[{ required: true }]}><Select options={(data?.roles || []).map((r) => ({ value: r.value, label: r.label }))} /></Form.Item><Form.Item name="entity_id" label={t('adminUsers.company')}><Select allowClear options={(data?.entities || []).map((e) => ({ value: e.id, label: e.name }))} /></Form.Item><Form.Item name="status" label={t('adminUsers.status')}><Select options={['active', 'disabled'].map((value) => ({ value, label: t(`adminUsers.status.${value}`, value) }))} /></Form.Item></> : null}
          {(!editing || passwordMode) ? <Form.Item name="password" label={t('login.password')} rules={[{ required: true, min: 8 }]}><Input.Password /></Form.Item> : null}
        </Form>
      </Modal>
    </div>
  );
}

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
