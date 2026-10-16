<!--
This file is part of eLabFTW Desktop.
@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->
<script lang='ts'>
  import { onMount } from 'svelte';
  import { ListElabftwInstances, ListRemoteEntries } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage, preventDefaultSubmit } from '../../utils/helpers';
  import Modal from '../Modal.svelte';

  type EntityType = 'experiment' | 'resource';
  type Props = {
    profileUuid: string;
    onClose: () => void;
    onImport: (instanceId: number, entityType: EntityType, remoteId: number) => Promise<void>;
  };
  let {profileUuid, onClose, onImport}: Props = $props();

  let instances = $state<main.ElabftwInstance[]>([]);
  let instanceId = $state(0);
  let entityType = $state<EntityType>('experiment');
  let search = $state('');
  let entries = $state<main.RemoteEntryOption[]>([]);
  let remoteId = $state(0);
  let loading = $state(false);
  let importing = $state(false);
  let hasMore = $state(false);
  let error = $state('');
  const pageSize = 50;

  async function loadEntries(reset = false): Promise<void> {
    if (!instanceId || loading || importing) return;
    loading = true;
    error = '';
    if (reset) {
      entries = [];
      remoteId = 0;
      hasMore = false;
    }
    try {
      const page = await ListRemoteEntries(profileUuid, instanceId, entityType, search.trim(), entries.length);
      entries = [...entries, ...page];
      hasMore = page.length === pageSize;
    } catch (e: unknown) {
      error = errorMessage(e);
    } finally {
      loading = false;
    }
  }

  async function loadInstances(): Promise<void> {
    loading = true;
    try {
      instances = await ListElabftwInstances(profileUuid);
      instanceId = instances[0]?.id ?? 0;
    } catch (e: unknown) {
      error = errorMessage(e);
    } finally {
      loading = false;
    }
    if (instanceId) await loadEntries(true);
  }

  async function importSelected(): Promise<void> {
    if (!instanceId || !remoteId || loading || importing) return;
    importing = true;
    error = '';
    try {
      await onImport(instanceId, entityType, remoteId);
    } catch (e: unknown) {
      error = errorMessage(e);
    } finally {
      importing = false;
    }
  }

  const handleSearch = preventDefaultSubmit(() => loadEntries(true));
  onMount(() => { void loadInstances(); });
</script>

<Modal title='Import from eLabFTW' onClose={() => { if (!importing) onClose(); }}>
  <div class='grid gap-1'>
    <p>Select one remote entry to save locally, including its metadata, links and uploads for offline use.</p>
    {#if instances.length === 0 && !loading}
      <p class='description'>No eLabFTW instances configured. Add one in Manage your eLabFTW Instances first.</p>
    {:else}
      <div>
        <label for='import-instance'>eLabFTW instance</label>
        <select id='import-instance' class='input' bind:value={instanceId} disabled={loading || importing}
                onchange={() => void loadEntries(true)}>
          {#each instances as instance (instance.id)}
            <option value={instance.id}>{instance.siteUrl}</option>
          {/each}
        </select>
      </div>
      <div>
        <label for='import-type'>Entry type</label>
        <select id='import-type' class='input' bind:value={entityType} disabled={loading || importing}
                onchange={() => void loadEntries(true)}>
          <option value='experiment'>Experiments</option>
          <option value='resource'>Resources</option>
        </select>
      </div>
      <form class='flex gap-1' onsubmit={handleSearch}>
        <input class='input' aria-label='Search remote entries' placeholder='Search remote entries by title or ID'
               bind:value={search} disabled={loading || importing} />
        <button type='submit' class='btn btn-secondary' disabled={loading || importing}>Search</button>
      </form>
      <div>
        <label for='import-entry'>Remote entry</label>
        <select id='import-entry' class='input' bind:value={remoteId} disabled={loading || importing || entries.length === 0}>
          <option value={0} disabled>Select an entry...</option>
          {#each entries as entry (entry.id)}
            <option value={entry.id}>{entry.title || 'Untitled'} (#{entry.id})</option>
          {/each}
        </select>
        {#if loading}
          <p class='description'>Loading entries...</p>
        {:else if entries.length === 0}
          <p class='description'>No matching entries. You can search by title or remote ID.</p>
        {/if}
        {#if hasMore}
          <button type='button' class='btn btn-secondary mt-2' disabled={loading || importing}
                  onclick={() => void loadEntries()}>Load more entries</button>
        {/if}
      </div>
    {/if}
    {#if error}
      <p class='text-orange' role='alert'>{error}</p>
    {/if}
  </div>
  {#snippet actions()}
    <button type='button' class='btn btn-secondary' disabled={importing} onclick={onClose}>Cancel</button>
    <button type='button' class='btn btn-primary' disabled={!remoteId || loading || importing}
            onclick={importSelected}>{importing ? 'Importing...' : 'Import entry'}</button>
  {/snippet}
</Modal>
