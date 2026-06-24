<!-- SPDX-License-Identifier: MIT -->

<script lang="ts">
  import { Toast } from '@skeletonlabs/skeleton-svelte';
  import { toaster } from '$lib/toast';
</script>

<Toast.Group {toaster} />
