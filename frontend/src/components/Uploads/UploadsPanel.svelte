<script lang='ts'>
  import {
    ImportUpload,
    ListEntryUploads,
    SelectFile
  } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage } from '../../utils/helpers';
  import type { AlertState } from '../Alert.svelte';

  type Props = {
    profileUuid: string;
    entryId: number | null;
    onAlert: (alert: AlertState | null) => void;
  };

  let {profileUuid, onAlert, entryId}: Props = $props();

  let uploads = $state<main.StoredUpload[]>([]);
  let loading = $state(false);

  async function refreshUploads(): Promise<void> {
    if (!entryId) {
      uploads = [];
      return;
    }

    loading = true;

    try {
      uploads = await ListEntryUploads(profileUuid, entryId);
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  async function importUpload(): Promise<void> {
    if (!entryId) {
      onAlert({type: 'warning', message: 'Save the entry before adding uploads.'});
      return;
    }

    try {
      const path = await SelectFile();
      if (!path) return;

      const upload = await ImportUpload(profileUuid, entryId, path);

      onAlert({type: 'success', message: `Imported ${upload.realName} ✔`});
      await refreshUploads();
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    }
  }

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }

  $effect(() => {
    if (entryId) {
      void refreshUploads();
    } else {
      uploads = [];
    }
  });
</script>

<section class='panel mt-2'>
  <div class='flex justify-between items-center border-bottom mb-2'>
    <div>
      <h3>Uploads</h3>
      <p class='description'>Encrypted local uploads for this profile.</p>
    </div>

    <button class='btn btn-secondary' type='button' onclick={importUpload}>
      Import upload
    </button>
  </div>

  {#if loading}
    <div class='empty-state'>Loading uploads...</div>
  {:else if uploads.length === 0}
    <div class='empty-state'>
      <h3>No uploads yet</h3>
      <p>Import a file to store it encrypted in this profile.</p>
    </div>
  {:else}
    <div class='grid gap-1'>
      {#each uploads as upload (upload.id)}
        <div class='flex justify-between items-center gap-1'>
          <div class='grid gap-03'>
            <span class='text-white text-strong'>{upload.longName}</span>
            <span class='description'>
              {formatFileSize(upload.filesize)} · {upload.hashAlgorithm} · {upload.state}
            </span>
          </div>

          <span class='description text-ellipsis' title={upload.hash}>
            {upload.hash.slice(0, 12)}
          </span>
        </div>
      {/each}
    </div>
  {/if}
</section>
