<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/stores';
  import type { UserSummary } from '$lib/stores';
  import { getLocale, setTheme, themeMode } from '$lib/stores';
  import { setCurrentLocale, t, type ThemeMode } from '$lib/i18n';
  import { redirectToCurrentPath } from '$lib/session';
  import { Avatar, Popover } from '@skeletonlabs/skeleton-svelte';
  import { AutoAvatar } from 'open-avatar';
  import { KeyRound, LogOut, Moon, Sun, UserRound } from 'lucide-svelte';

  let {
    user = null,
    onlogout,
  }: {
    user: UserSummary | null;
    onlogout: () => void;
  } = $props();

  const displayName = $derived(user?.display_name || user?.username || t('layout.profile'));
  const avatarAlt = $derived(displayName || 'User');
  const isAdmin = $derived($page.url.pathname === '/admin' || $page.url.pathname.startsWith('/admin/'));
  const profileHref = $derived(isAdmin ? '/admin/profile' : '/portal/profile');
  const securityHref = $derived(isAdmin ? '/admin/profile#password' : '/portal/profile#password');
  let currentLocale = $state(getLocale());

  const modeOptions = [
    { value: 'light' as ThemeMode, labelKey: 'layout.lightTheme', icon: Sun },
    { value: 'dark' as ThemeMode, labelKey: 'layout.darkTheme', icon: Moon },
  ];

  const avatarSrc = $derived(
    AutoAvatar(user?.id || user?.username || displayName, user?.avatar_url, {
      shape: 'circle',
      size: 64,
      title: avatarAlt,
    }).src,
  );

  const changeThemeMode = (mode: ThemeMode) => {
    setTheme(mode);
  };

  const changeLanguage = (locale: 'en-US' | 'zh-CN') => {
    currentLocale = locale;
    setCurrentLocale(locale);
    redirectToCurrentPath($page.url.pathname, $page.url.search);
  };
</script>

<Popover positioning={{ placement: 'bottom-end' }} closeOnInteractOutside={true} closeOnEscape={true}>
  <Popover.Trigger class="avatar-menu-trigger" type="button" aria-label={t('layout.profile')} title={displayName}>
    <Avatar class="user-avatar">
      <Avatar.Image src={avatarSrc} alt={avatarAlt} />
      <Avatar.Fallback>{displayName.slice(0, 1).toUpperCase()}</Avatar.Fallback>
    </Avatar>
  </Popover.Trigger>
  <Popover.Positioner class="theme-picker-positioner">
    <Popover.Content class="card w-72 bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl z-50" role="menu" aria-label={t('layout.profile')}>
      <div class="border-b border-surface-200-800 pb-3">
        <strong class="block truncate text-sm">{displayName}</strong>
        {#if user}
          <p class="mt-1 truncate text-xs text-surface-600-400">{user.email || user.phone || user.username}</p>
        {/if}
      </div>

      <section class="border-b border-surface-200-800 py-2" aria-label={`${t('layout.theme')} / ${t('layout.language')}`}>
        <div class="compact-pill-row">
          {#each modeOptions as option}
            <button
              class="compact-pill {$themeMode === option.value ? 'active' : ''}"
              type="button"
              onclick={() => changeThemeMode(option.value)}
              aria-pressed={$themeMode === option.value}
              title={t(option.labelKey)}
            >
              <option.icon size={13} aria-hidden="true" />
              <span>{t(option.labelKey)}</span>
            </button>
          {/each}
          <button
            class="compact-pill {currentLocale === 'en-US' ? 'active' : ''}"
            type="button"
            onclick={() => changeLanguage('en-US')}
            aria-pressed={currentLocale === 'en-US'}
          >
            EN
          </button>
          <button
            class="compact-pill {currentLocale === 'zh-CN' ? 'active' : ''}"
            type="button"
            onclick={() => changeLanguage('zh-CN')}
            aria-pressed={currentLocale === 'zh-CN'}
          >
            中
          </button>
        </div>
      </section>

      <div class="py-2">
        <a class="menu-item-link" href={profileHref} role="menuitem">
          <UserRound size={16} aria-hidden="true" />
          <span>{t('layout.profile')}</span>
        </a>
        <a class="menu-item-link" href={securityHref} role="menuitem">
          <KeyRound size={16} aria-hidden="true" />
          <span>{t('profile.changePassword')}</span>
        </a>
      </div>

      <button class="menu-item-link w-full border-t border-surface-200-800 pt-2" type="button" role="menuitem" onclick={onlogout}>
        <LogOut size={16} aria-hidden="true" />
        <span>{t('layout.logout')}</span>
      </button>
    </Popover.Content>
  </Popover.Positioner>
</Popover>
