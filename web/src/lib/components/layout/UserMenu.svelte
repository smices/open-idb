<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/stores';
  import type { UserSummary } from '$lib/stores';
  import { getLocale, setTheme, themeMode } from '$lib/stores';
  import { setCurrentLocale, t, type ThemeMode } from '$lib/i18n';
  import { redirectToCurrentPath } from '$lib/session';
  import { Menu, SegmentedControl } from '@skeletonlabs/skeleton-svelte';
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

<Menu positioning={{ placement: 'bottom-end', gutter: 8 }}>
  <Menu.Trigger class="avatar-menu-trigger" aria-label={t('layout.profile')} title={displayName}>
    <span class="user-avatar">
      <img src={avatarSrc} alt={avatarAlt} />
      <span>{displayName.slice(0, 1).toUpperCase()}</span>
    </span>
  </Menu.Trigger>

  <Menu.Positioner class="z-50">
    <Menu.Content class="card w-72 bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl">
      <div class="border-b border-surface-200-800 pb-3">
        <strong class="block truncate text-sm">{displayName}</strong>
        {#if user}
          <p class="mt-1 truncate text-xs text-surface-600-400">{user.email || user.phone || user.username}</p>
        {/if}
      </div>

      <section class="border-b border-surface-200-800 py-2" aria-label={`${t('layout.theme')} / ${t('layout.language')}`}>
        <div class="grid gap-2">
          <SegmentedControl
            class="compact-segmented-control"
            value={$themeMode}
            onValueChange={(details) => changeThemeMode(details.value as ThemeMode)}
            aria-label={t('layout.theme')}
          >
            <SegmentedControl.Control class="compact-segmented-control-track">
              {#each modeOptions as option}
                <SegmentedControl.Item class="compact-segmented-control-item" value={option.value} title={t(option.labelKey)}>
                  <SegmentedControl.ItemHiddenInput />
                  <SegmentedControl.ItemText class="inline-flex items-center gap-1.5">
                    <option.icon size={13} aria-hidden="true" />
                    <span>{t(option.labelKey)}</span>
                  </SegmentedControl.ItemText>
                </SegmentedControl.Item>
              {/each}
            </SegmentedControl.Control>
          </SegmentedControl>

          <SegmentedControl
            class="compact-segmented-control"
            value={currentLocale}
            onValueChange={(details) => changeLanguage(details.value as 'en-US' | 'zh-CN')}
            aria-label={t('layout.language')}
          >
            <SegmentedControl.Control class="compact-segmented-control-track">
              <SegmentedControl.Item class="compact-segmented-control-item" value="en-US">
                <SegmentedControl.ItemHiddenInput />
                <SegmentedControl.ItemText>{t('layout.languageShort.en')}</SegmentedControl.ItemText>
              </SegmentedControl.Item>
              <SegmentedControl.Item class="compact-segmented-control-item" value="zh-CN">
                <SegmentedControl.ItemHiddenInput />
                <SegmentedControl.ItemText>{t('layout.languageShort.zh')}</SegmentedControl.ItemText>
              </SegmentedControl.Item>
            </SegmentedControl.Control>
          </SegmentedControl>
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
    </Menu.Content>
  </Menu.Positioner>
</Menu>
