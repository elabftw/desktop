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
  const maxUploadSizeMb = 100;

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
    // uploads need to be attached to an entry. Before clicking save, the entry doesn't have an id yet.
    if (!entryId) {
      onAlert({type: 'warning', message: 'Save the entry before adding the first upload.'});
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

  // Display file sizes in a user-friendly format instead of raw bytes.
  // Currently supports B, KB and MB.
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
  <div class='border-bottom mb-2'>
    <h3>Uploads</h3>
    <p class='description'>File size limit: {maxUploadSizeMb} MB</p>
  </div>

  {#if loading}
    <div class='empty-state'>Loading uploads...</div>
  {:else}
    <div class='uploads-grid'>
      <!-- Always visible import button -->
      <button type='button' class='upload-card upload-card-add' onclick={importUpload}>
        <span class='upload-icon' aria-hidden='true'>+</span>
        <span class='text-strong'>Import a file</span>
      </button>

      {#each uploads as upload (upload.id)}
        <div class='upload-card'>
          <div class='upload-name'>{upload.realName}</div>
          <div class='description'>
            {formatFileSize(upload.filesize)}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</section>
