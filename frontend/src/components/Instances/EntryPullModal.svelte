<!--
This file is part of eLabFTW Desktop.
@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->
<script lang='ts'>
  import type { main } from '../../../wailsjs/go/models';
  import Modal from '../Modal.svelte';

  type Props = {
    link: main.EntryRemoteLink;
    onClose: () => void;
    onPull: () => Promise<void>;
  };

  let {link, onClose, onPull}: Props = $props();
  let pulling = $state(false);

  async function confirmPull(): Promise<void> {
    pulling = true;
    try {
      await onPull();
    } finally {
      pulling = false;
    }
  }
</script>

<Modal title='Pull from eLabFTW' onClose={onClose}>
  <div class='grid gap-1 text-white'>
    <p>Pull <span class='text-strong'>{link.type}</span> with <span class='text-success'>ID #{link.remoteId}</span> from <span class='text-orange'>{link.siteUrl}</span>?</p>
    <p>
      This will overwrite the local title, body and uploads with the remote version.
      Local changes that are not present online will be lost. <span class='text-strong text-orange'>This cannot be undone.</span>
    </p>
  </div>

  <svelte:fragment slot='actions'>
    <button class='btn btn-secondary' type='button' disabled={pulling} onclick={onClose}>
      Cancel
    </button>
    <button class='btn btn-danger' type='button' disabled={pulling} onclick={confirmPull}>
      {pulling ? 'Pulling...' : 'Pull and overwrite'}
    </button>
  </svelte:fragment>
</Modal>
