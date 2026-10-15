<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import type { Snippet } from 'svelte';

  type Props = {
    title: string;
    onClose: () => void;
    children?: Snippet;
    actions?: Snippet;
  };

  let {title, onClose, children, actions}: Props = $props();

  function stopClick(event: MouseEvent): void {
    event.stopPropagation();
  }
</script>

<div class='modal-backdrop' role='presentation' onclick={onClose}>
  <dialog class='modal panel' aria-labelledby='modal-title' open onclick={stopClick}>
    <header class='flex justify-between items-center border-bottom mb-2'>
      <h1 id='modal-title'>{title}</h1>
      <button class='btn btn-secondary' type='button' onclick={onClose} aria-label='Close modal'>&#x2717;</button>
    </header>

    <div>
      {@render children?.()}
    </div>

    <footer class='flex justify-end gap-1 mt-2'>
      {@render actions?.()}
    </footer>
  </dialog>
</div>
