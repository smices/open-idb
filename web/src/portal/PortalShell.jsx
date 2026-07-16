import { Segmented, Space } from 'antd';
import { AppWindow, UserRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { UserMenu } from '../components/UserMenu.jsx';
import { PortalHomePage } from './PortalHomePage.jsx';
import { ProfilePage } from './ProfilePage.jsx';

function navigate(path) {
  window.location.href = path;
}

function portalItems(t) {
  return [
    { key: '/portal', path: '/portal', label: t('nav.portal.apps'), icon: <AppWindow size={16} /> },
    { key: '/portal/profile', path: '/portal/profile', label: t('nav.portal.profile'), icon: <UserRound size={16} /> },
  ];
}

export function PortalShell({ user, branding, themeState, onLogout }) {
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
        {pathname === '/portal/profile' ? <ProfilePage user={user} /> : <PortalHomePage />}
      </main>
    </div>
  );
}
