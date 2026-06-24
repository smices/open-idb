<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { onMount } from 'svelte';
  import { X } from 'lucide-svelte';
  import { t } from '$lib/i18n';
  import { dismissToast, type IdToast } from '$lib/toast';

  let toasts = $state<IdToast[]>([]);

  const removeToast = (id: string) => {
    toasts = toasts.filter((toast) => toast.id !== id);
    dismissToast(id);
  };

  const showToast = (toast: IdToast) => {
    toasts = [toast, ...toasts.filter((item) => item.id !== toast.id)].slice(0, 4);
    window.setTimeout(() => {
      toasts = toasts.filter((item) => item.id !== toast.id);
    }, 3600);
  };

  onMount(() => {
    const handleToast = (event: Event) => {
      showToast((event as CustomEvent<IdToast>).detail);
    };
    window.addEventListener('idbridge-toast', handleToast);
    return () => window.removeEventListener('idbridge-toast', handleToast);
  });
</script>

<span class="sr-only" data-idbridge-toast-provider="mounted"></span>

{#if toasts.length}
  <section
    class="pointer-events-none fixed inset-x-0 top-4 z-[2147483647] flex flex-col items-center gap-2 px-4"
    aria-live="polite"
    aria-relevant="additions text"
  >
    {#each toasts as toast (toast.id)}
      <article
        class={`card pointer-events-auto flex min-h-10 w-fit max-w-[min(28rem,calc(100vw-2rem))] items-center gap-3 border px-3 py-2 text-sm shadow-xl ${
          toast.type === 'error'
            ? 'border-error-500/30 bg-error-50-950 text-error-950-50'
            : toast.type === 'success'
              ? 'border-success-500/30 bg-success-50-950 text-success-950-50'
              : 'border-surface-200-800 bg-surface-50-950 text-surface-950-50'
        }`}
        role="status"
        data-scope="toast"
      >
        <p class="min-w-0 leading-5">{toast.title}</p>
        <button
          class="inline-grid size-6 shrink-0 place-items-center rounded-sm text-current/65 hover:bg-current/10 hover:text-current"
          type="button"
          aria-label={t('common.close')}
          title={t('common.close')}
          onclick={() => removeToast(toast.id)}
        >
          <X class="size-4" aria-hidden="true" />
        </button>
      </article>
    {/each}
  </section>
{/if}
