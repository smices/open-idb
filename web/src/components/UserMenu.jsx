import React from 'react';
import { AutoAvatar } from 'open-avatar/react';
import { Languages, LogOut, Monitor, Moon, Sun, UserRound } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button, Dropdown, Segmented, Space, Typography } from 'antd';

export function UserMenu({ user, themeMode, language, onThemeChange, onLanguageChange, onLogout, profileHref = '/portal/profile' }) {
  const { t } = useTranslation();
  const name = user?.display_name || user?.username || 'IdBridge';

  const themeItems = [
    { value: 'light', label: <Sun size={15} aria-label={t('layout.lightTheme')} /> },
    { value: 'dark', label: <Moon size={15} aria-label={t('layout.darkTheme')} /> },
    { value: 'system', label: <Monitor size={15} aria-label={t('layout.systemTheme')} /> },
  ];
  const languageItems = [
    { value: 'zh-CN', label: <span className="user-segment-label"><Languages size={14} />中</span> },
    { value: 'en-US', label: <span className="user-segment-label"><Languages size={14} />EN</span> },
  ];
  const menuItems = [
    {
      key: 'identity',
      disabled: true,
      label: (
        <Space size={10} className="user-menu-identity">
          <AutoAvatar userIdOrName={name} avatar={user?.avatar_url || ''} shape="circle" size={34} />
          <span>
            <Typography.Text strong>{name}</Typography.Text>
            <Typography.Text type="secondary" className="block-text">{user?.username || user?.email || ''}</Typography.Text>
          </span>
        </Space>
      ),
    },
    { type: 'divider' },
    {
      key: 'preferences',
      label: (
        <div className="user-menu-control-row" onClick={(event) => event.stopPropagation()}>
          <Segmented size="small" value={themeMode} options={themeItems} onChange={onThemeChange} aria-label={t('layout.theme')} />
          <Segmented size="small" value={language} options={languageItems} onChange={onLanguageChange} aria-label={t('layout.language')} />
        </div>
      ),
    },
    { type: 'divider' },
    {
      key: 'profile',
      icon: <UserRound size={15} />,
      label: <a href={profileHref}>{t('layout.profile')}</a>,
    },
    {
      key: 'logout',
      danger: true,
      icon: <LogOut size={15} />,
      label: t('layout.logout'),
      onClick: onLogout,
    },
  ];

  return (
    <Dropdown trigger={['click']} placement="bottomRight" arrow menu={{ items: menuItems }}>
      <Button type="text" className="avatar-trigger" aria-label={t('layout.profile')}>
        <AutoAvatar userIdOrName={name} avatar={user?.avatar_url || ''} shape="circle" size={36} />
      </Button>
    </Dropdown>
  );
}
