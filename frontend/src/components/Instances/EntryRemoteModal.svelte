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
    mode: 'open' | 'pull' | 'unlink';
    onClose: () => void;
    onOpen: (link: main.EntryRemoteLink) => void;
    onPull: (link: main.EntryRemoteLink) => Promise<void>;
    onUnlink: (link: main.EntryRemoteLink) => Promise<void>;
  };

  let {links, mode, onClose, onOpen, onPull, onUnlink}: Props = $props();
  let selectedIndex = $state(links.length === 1 ? 0 : -1);
  let selectedLink = $derived(links[selectedIndex] ?? null);
  let working = $state(false);

  async function confirmAction(): Promise<void> {
    if (!selectedLink) return;
    if (mode === 'open') {
      onOpen(selectedLink);
      return;
    }

    working = true;
    try {
      if (mode === 'pull') {
        await onPull(selectedLink);
      } else {
        await onUnlink(selectedLink);
      }
    } finally {
      working = false;
    }
  }
</script>

<Modal title={mode === 'pull' ? 'Pull from eLabFTW' : mode === 'unlink' ? 'Unlink from eLabFTW' : 'Open in eLabFTW'} onClose={() => { if (!working) onClose(); }}>
  <div class='grid gap-1'>
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
      {:else if mode === 'unlink'}
        <p>
          Remove the local synchronization link? Both the local and remote entries will remain unchanged.
          Cached remote details and upload mappings for this link will be cleared.
        </p>
      {/if}
    {:else}
      <p class='description'>Select an instance to {mode === 'pull' ? 'pull from' : mode === 'unlink' ? 'unlink from' : 'open'}.</p>
    {/if}
  </div>

  {#snippet actions()}
    <button class='btn btn-secondary' type='button' disabled={working} onclick={onClose}>
      Cancel
    </button>
    <button class={mode === 'open' ? 'btn btn-primary' : 'btn btn-danger'} type='button'
            disabled={working || !selectedLink} onclick={confirmAction}>
      {working ? (mode === 'pull' ? 'Pulling...' : 'Unlinking...') : mode === 'pull' ? 'Pull and overwrite' : mode === 'unlink' ? 'Unlink entry' : 'Open in browser'}
    </button>
  {/snippet}
</Modal>
