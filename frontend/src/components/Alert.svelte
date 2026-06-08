<!--
This file is part of eLabFTW Desktop.

@author Nicolas <Deltablot>
@author Moustapha <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  export type AlertState = {
    type: 'success' | 'error' | 'warning' | 'info';
    message: string;
  };

  type AlertProps = Partial<AlertState>;

  let { type = 'success', message = '' }: AlertProps = $props();
  let visible = $state(true);

  // Re-show the alert when the parent provides a new message.
  $effect(() => {
    if (message) visible = true;
  });
</script>

{#if message && visible}
  <div class={`alert alert-${type} flex justify-between items-center`}>
    <strong>{message}</strong>
    <button class='alert-close' type='button' aria-label='Close alert' onclick={() => visible = false}>&#x2717;</button>
  </div>
{/if}
