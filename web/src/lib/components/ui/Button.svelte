<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import type { Snippet } from 'svelte';

  type Variant = 'primary' | 'secondary' | 'ghost' | 'danger';
  type Size = 'xs' | 'sm' | 'md' | 'icon';

  let {
    href,
    type = 'button',
    variant = 'secondary',
    size = 'md',
    disabled = false,
    ariaLabel,
    title,
    class: className = '',
    onclick,
    children,
  }: {
    href?: string;
    type?: 'button' | 'submit' | 'reset';
    variant?: Variant;
    size?: Size;
    disabled?: boolean;
    ariaLabel?: string;
    title?: string;
    class?: string;
    onclick?: (event: MouseEvent) => void;
    children?: Snippet;
  } = $props();

  const variantClass = $derived.by(() => {
    if (variant === 'primary') return 'preset-filled-primary-500';
    if (variant === 'danger') return 'preset-tonal-error';
    if (variant === 'ghost') return 'preset-outlined-surface-200-800';
    return 'preset-outlined-surface-500';
  });

  const sizeClass = $derived.by(() => {
    if (size === 'xs') return 'btn-xs';
    if (size === 'sm') return 'btn-sm';
    if (size === 'icon') return 'btn-icon idb-icon-button';
    return '';
  });

  const classes = $derived(`btn ${variantClass} ${sizeClass} ${className}`.trim());
</script>

{#if href}
  <a class={classes} href={href} aria-label={ariaLabel} title={title}>
    {@render children?.()}
  </a>
{:else}
  <button class={classes} type={type} disabled={disabled} aria-label={ariaLabel} title={title} onclick={onclick}>
    {@render children?.()}
  </button>
{/if}
