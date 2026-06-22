<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { UserSummary } from '$lib/stores';
  import { t } from '$lib/i18n';
  import { Avatar, Popover } from '@skeletonlabs/skeleton-svelte';
  import { AutoAvatar } from 'open-avatar';
  import { LogOut } from 'lucide-svelte';

  let {
    user = null,
    onlogout,
  }: {
    user: UserSummary | null;
    onlogout: () => void;
  } = $props();

  const displayName = $derived(user?.display_name || user?.username || t('layout.profile'));
  const avatarAlt = $derived(displayName || 'User');
  const avatarSrc = $derived(
    AutoAvatar(user?.id || user?.username || displayName, user?.avatar_url, {
      shape: 'circle',
      size: 64,
      title: avatarAlt,
    }).src,
  );
</script>

<Popover positioning={{ placement: 'bottom-end' }} closeOnInteractOutside={true} closeOnEscape={true}>
  <Popover.Trigger class="btn preset-outlined-surface-500 rounded-full p-1" type="button" aria-label={t('layout.profile')}>
    <Avatar class="user-avatar">
      <Avatar.Image src={avatarSrc} alt={avatarAlt} />
      <Avatar.Fallback>{displayName.slice(0, 1).toUpperCase()}</Avatar.Fallback>
    </Avatar>
  </Popover.Trigger>
  <Popover.Positioner class="theme-picker-positioner">
    <Popover.Content class="card bg-surface-50-950 border border-surface-200-800 p-3 space-y-3 shadow-xl z-50 w-72" role="menu" aria-label={t('layout.profile')}>
      <div class="user-menu-copy">
        <strong>{displayName}</strong>
        {#if user}
          <p>
            {user.username}
            <br />
            {user.email || user.phone || user.entity_id}
          </p>
        {/if}
      </div>
      <button class="btn btn-sm preset-outlined-surface-500 w-full justify-start" type="button" role="menuitem" onclick={onlogout}>
        <LogOut size={16} aria-hidden="true" />
        {t('layout.logout')}
      </button>
    </Popover.Content>
  </Popover.Positioner>
</Popover>
