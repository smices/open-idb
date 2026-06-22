<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { page } from '$app/stores';
  import { getThemeNameFromStorage, setCurrentLocale, setThemeName, t, type ThemeMode } from '$lib/i18n';
  import { getLocale, setTheme, themeMode } from '$lib/stores';
  import { redirectToCurrentPath } from '$lib/session';
  import { Popover } from '@skeletonlabs/skeleton-svelte';
  import { Globe2, Monitor, Moon, Palette, Sun } from 'lucide-svelte';

  const themes = [
    { name: 'catppuccin', emoji: 'Cat' },
    { name: 'cerberus', emoji: 'Wolf' },
    { name: 'concord', emoji: 'Bot' },
    { name: 'crimson', emoji: 'Red' },
    { name: 'fennec', emoji: 'Fox' },
    { name: 'hamlindigo', emoji: 'Suit' },
    { name: 'legacy', emoji: 'Skull' },
    { name: 'mint', emoji: 'Mint' },
    { name: 'modern', emoji: 'Mod' },
    { name: 'mona', emoji: 'Mona' },
    { name: 'nosh', emoji: 'Nosh' },
    { name: 'nouveau', emoji: 'Crown' },
    { name: 'pine', emoji: 'Pine' },
    { name: 'reign', emoji: 'Book' },
    { name: 'rocket', emoji: 'Rocket' },
    { name: 'rose', emoji: 'Rose' },
    { name: 'sahara', emoji: 'Desert' },
    { name: 'seafoam', emoji: 'Sea' },
    { name: 'terminus', emoji: 'Term' },
    { name: 'vintage', emoji: 'Retro' },
    { name: 'vox', emoji: 'Vox' },
    { name: 'wintry', emoji: 'Snow' },
  ];

  const modeOptions = [
    { value: 'system' as ThemeMode, labelKey: 'layout.systemTheme', icon: Monitor },
    { value: 'light' as ThemeMode, labelKey: 'layout.lightTheme', icon: Sun },
    { value: 'dark' as ThemeMode, labelKey: 'layout.darkTheme', icon: Moon },
  ];

  let themePopoverOpen = false;
  let languagePopoverOpen = false;
  let themeName = getThemeNameFromStorage();
  let committedTheme: string | null = null;

  const applyPreviewTheme = (name: string) => {
    if (typeof document !== 'undefined') {
      document.documentElement.dataset.theme = name;
    }
  };

  const restoreTheme = () => {
    applyPreviewTheme(committedTheme ?? themeName);
    committedTheme = null;
  };

  const changeThemeName = (name: string) => {
    themeName = name;
    committedTheme = name;
    setThemeName(name);
    applyPreviewTheme(name);
    themePopoverOpen = false;
  };

  const changeThemeMode = (mode: ThemeMode) => {
    setTheme(mode);
  };

  const changeLanguage = (locale: 'en-US' | 'zh-CN') => {
    setCurrentLocale(locale);
    languagePopoverOpen = false;
    redirectToCurrentPath($page.url.pathname, $page.url.search);
  };
</script>

<div class="theme-control" aria-label={t('layout.theme')}>
  <div class="theme-menu">
    <Popover
      open={themePopoverOpen}
      onOpenChange={(event) => {
        themePopoverOpen = event.open;
        if (!event.open) restoreTheme();
      }}
      positioning={{ placement: 'bottom-end' }}
      closeOnInteractOutside={true}
      closeOnEscape={true}
    >
      <Popover.Trigger
        class={`btn btn-icon header-icon-button ${themePopoverOpen ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-label={t('layout.theme')}
        title={t('layout.theme')}
      >
        <Palette size={16} aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Positioner class="theme-picker-positioner">
        <Popover.Content class="theme-picker-card card bg-surface-50-950 border border-surface-200-800 p-3 space-y-4 shadow-xl max-h-[75vh] lg:max-h-none overflow-y-auto z-50" role="dialog" aria-label={t('layout.theme')}>
          <div class="theme-mode-row">
            <div class="mb-2 flex flex-wrap gap-2">
              {#each modeOptions as option}
                <button
                  class={`btn ${$themeMode === option.value ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'} btn-sm`}
                  type="button"
                  onclick={() => changeThemeMode(option.value)}
                  aria-label={t(option.labelKey)}
                  title={t(option.labelKey)}
                >
                  <option.icon size={16} aria-hidden="true" />
                  <span>{t(option.labelKey)}</span>
                </button>
              {/each}
            </div>
          </div>

          <div class="grid grid-cols-1 gap-2 lg:grid-cols-3" role="listbox">
            {#each themes as theme}
              <button
                data-theme={theme.name}
                class="theme-swatch-card bg-surface-50-950 p-3 preset-outlined-surface-100-900 hover:preset-outlined-surface-950-50 rounded-container grid grid-cols-[auto_1fr_auto] items-center gap-4 {themeName === theme.name ? 'preset-tonal-primary' : ''}"
                type="button"
                role="option"
                aria-selected={theme.name === themeName}
                aria-label={theme.name}
                onclick={() => changeThemeName(theme.name)}
                onmouseenter={() => applyPreviewTheme(theme.name)}
                onmouseleave={restoreTheme}
                onfocus={() => applyPreviewTheme(theme.name)}
                onblur={restoreTheme}
              >
                <span class="text-xs text-surface-500">{theme.emoji}</span>
                <h3 class="text-sm capitalize font-bold text-left">{theme.name}</h3>
                <div class="flex items-center justify-center -space-x-1.5">
                  <div class="aspect-square w-4 rounded-full border-[1px] border-black/10 bg-primary-500"></div>
                  <div class="aspect-square w-4 rounded-full border-[1px] border-black/10 bg-secondary-500"></div>
                  <div class="aspect-square w-4 rounded-full border-[1px] border-black/10 bg-tertiary-500"></div>
                </div>
              </button>
            {/each}
          </div>
        </Popover.Content>
      </Popover.Positioner>
    </Popover>

    <Popover
      open={languagePopoverOpen}
      onOpenChange={(event) => (languagePopoverOpen = event.open)}
      positioning={{ placement: 'bottom-end' }}
      closeOnInteractOutside={true}
      closeOnEscape={true}
    >
      <Popover.Trigger
        class={`btn btn-icon header-icon-button ${languagePopoverOpen ? 'preset-filled-primary-500' : 'preset-outlined-surface-500'}`}
        type="button"
        aria-label={t('layout.language')}
        title={t('layout.language')}
      >
        <Globe2 size={16} aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Positioner class="theme-picker-positioner">
        <Popover.Content class="card bg-surface-50-950 border border-surface-200-800 p-2 shadow-xl z-50 min-w-44" role="menu" aria-label={t('layout.language')}>
          <button class="btn btn-sm preset-outlined-surface-500 w-full justify-start" type="button" role="menuitem" onclick={() => changeLanguage('en-US')}>{t('layout.english')}</button>
          <button class="btn btn-sm preset-outlined-surface-500 mt-1 w-full justify-start" type="button" role="menuitem" onclick={() => changeLanguage('zh-CN')}>{t('layout.chinese')}</button>
        </Popover.Content>
      </Popover.Positioner>
    </Popover>
  </div>
</div>
