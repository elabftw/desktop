<!--
This file is part of eLabFTW Desktop.
@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->
<script lang='ts'>
  import type { main } from '../../../wailsjs/go/models';
  import Modal from '../Modal.svelte';

  type Props = {
    links: main.EntryRemoteLink[];
    mode: 'open' | 'pull';
    onClose: () => void;
    onOpen: (link: main.EntryRemoteLink) => void;
    onPull: (link: main.EntryRemoteLink) => Promise<void>;
  };

  let {links, mode, onClose, onOpen, onPull}: Props = $props();
  let selectedIndex = $state(links.length === 1 ? 0 : -1);
  let selectedLink = $derived(links[selectedIndex] ?? null);
  let pulling = $state(false);

  async function confirmAction(): Promise<void> {
    if (!selectedLink) return;
    if (mode === 'open') {
      onOpen(selectedLink);
      return;
    }

    pulling = true;
    try {
      await onPull(selectedLink);
    } finally {
      pulling = false;
    }
  }
</script>

<Modal title={mode === 'pull' ? 'Pull from eLabFTW' : 'Open in eLabFTW'} onClose={() => { if (!pulling) onClose(); }}>
  <div class='grid gap-1 text-white'>
    {#if links.length > 1}
      <div>
        <label for='remote-instance'>eLabFTW instance</label>
        <select id='remote-instance' class='input' bind:value={selectedIndex}>
          <option value={-1} disabled>Select an instance...</option>
          {#each links as link, index (`${link.instanceId}-${link.type}-${link.remoteId}`)}
            <option value={index}>{link.siteUrl} ({link.type} #{link.remoteId})</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if selectedLink}
      <p>Instance: <span class='text-orange'>{selectedLink.siteUrl}</span></p>
      <p>Remote entry: <span class='text-strong'>{selectedLink.type === 'experiment' ? 'Experiment' : 'Resource'} #{selectedLink.remoteId}</span></p>
      {#if mode === 'pull'}
        <p>
          This will overwrite the local title, body and uploads with the remote version.
          Local changes that are not present online will be lost. <span class='text-strong text-orange'>This cannot be undone.</span>
        </p>
      {/if}
    {:else}
      <p class='description'>Select an instance to {mode === 'pull' ? 'pull from' : 'open'}.</p>
    {/if}
  </div>

  <svelte:fragment slot='actions'>
    <button class='btn btn-secondary' type='button' disabled={pulling} onclick={onClose}>
      Cancel
    </button>
    <button class={mode === 'pull' ? 'btn btn-danger' : 'btn btn-primary'} type='button'
            disabled={pulling || !selectedLink} onclick={confirmAction}>
      {pulling ? 'Pulling...' : mode === 'pull' ? 'Pull and overwrite' : 'Open in browser'}
    </button>
  </svelte:fragment>
</Modal>
