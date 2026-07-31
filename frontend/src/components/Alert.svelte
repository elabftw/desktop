<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { alert, showAlert } from "./stores/alert.svelte";

  let closing = $state(false);

  /* allow having the Alert at the top left of every component (not depend of its parent) */
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return {
      destroy() { node.remove() }
    };
  }

  $effect(() => {
    if (alert.current) closing = false;
  });

  function close() {
    closing = true;
  }

  function onAnimationEnd() {
    if (closing) {
      closing = false;
      showAlert(null);
    }
  }
</script>

{#if alert.current}
  <div
    use:portal
    class={`alert alert-${alert.current.type} alert-floating ${closing ? 'alert-closing' : ''}`}
    onanimationend={onAnimationEnd}
  >
    <strong>{alert.current.message}</strong>
    <button class='alert-close' type='button' aria-label='Close alert' onclick={close}>&#x2717;</button>
  </div>
{/if}
