<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
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
  let closing = $state(false);

  /* allow having the Alert at the top left of every component (not depend of its parent) */
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return {
      destroy() { node.remove() }
    };
  }

  // Re-show the alert when the parent provides a new message.
  $effect(() => {
    if (message) {
      visible = true;
      closing = false;
    }
  });

  function close() {
    closing = true;
  }

  function onAnimationEnd() {
    if (closing) {
      visible = false;
      closing = false;
    }
  }
</script>

{#if message && visible}
  <div
    use:portal
    class={`alert alert-${type} alert-floating ${closing ? 'alert-closing' : ''}`}
    onanimationend={onAnimationEnd}
  >
    <strong>{message}</strong>
    <button class='alert-close' type='button' aria-label='Close alert' onclick={close}>&#x2717;</button>
  </div>
{/if}
