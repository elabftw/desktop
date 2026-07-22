<script lang='ts'>
  import {
    DeleteUpload,
    DownloadUpload,
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
    ensureEntrySaved: () => Promise<number | null>;
    onAlert: (alert: AlertState | null) => void;
  };

  let {profileUuid, entryId, ensureEntrySaved, onAlert}: Props = $props();

  let uploads = $state<main.StoredUpload[]>([]);
  let loading = $state(false);
  let busyUploadId = $state<number | null>(null); // state when upload is being downloaded or deleted, modal popup

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
    const uploadEntryId = await ensureEntrySaved();
    if (!uploadEntryId) return;

    try {
      const path = await SelectFile();
      if (!path) return;
      const upload = await ImportUpload(profileUuid, uploadEntryId, path);
      onAlert({type: 'success', message: `Imported ${upload.realName} ✔`});
      await refreshUploads();
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    }
  }

  async function downloadUpload(upload: main.StoredUpload): Promise<void> {
    if (!entryId) return;

    busyUploadId = upload.id;

    try {
      const destination = await DownloadUpload(
        profileUuid,
        entryId,
        upload.id
      );

      // Empty means the save dialog was cancelled.
      if (!destination) return;

      onAlert({
        type: 'success',
        message: `Saved ${upload.realName} ✔`
      });
    } catch (e: unknown) {
      onAlert({
        type: 'error',
        message: errorMessage(e)
      });
    } finally {
      busyUploadId = null;
    }
  }

  async function deleteUpload(upload: main.StoredUpload): Promise<void> {
    if (!entryId) return;

    const confirmed = window.confirm(
      `Delete "${upload.realName}" from this entry?`
    );

    if (!confirmed) return;

    busyUploadId = upload.id;

    try {
      await DeleteUpload(profileUuid, entryId, upload.id);

      // Update immediately rather than doing another database read.
      uploads = uploads.filter((item) => item.id !== upload.id);

      onAlert({
        type: 'success',
        message: `Deleted ${upload.realName} ✔`
      });
    } catch (e: unknown) {
      onAlert({
        type: 'error',
        message: errorMessage(e)
      });
    } finally {
      busyUploadId = null;
    }
  }

  // Display file sizes in a user-friendly format instead of raw bytes.
  // Currently supports B, KB and MB.
  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }

  function formatDate(timestamp: string): string {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(timestamp));
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
          <div class="description upload-date">
            Added {formatDate(upload.createdAt)}
          </div>
          <div class='flex gap-03'>
            <button aria-label='Download file' class='btn btn-secondary' type='button' disabled={busyUploadId === upload.id} onclick={() => downloadUpload(upload)}>
              <!-- Down-underlined arrow for download-->
              <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v12" /><path d="m7 10 5 5 5-5" /><path d="M5 21h14" /></svg>
            </button>
            <!-- bin icon for delete -->
            <button type='button' aria-label='Delete file' class='btn btn-danger' disabled={busyUploadId === upload.id} onclick={() => deleteUpload(upload)}>
              <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18" /><path d="M8 6V4h8v2" /><path d="M19 6 18 21H6L5 6" /><path d="M10 11v6" /><path d="M14 11v6" /></svg>
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</section>
