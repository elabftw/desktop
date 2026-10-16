<!--
This file is part of eLabFTW lite.

@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { onMount } from 'svelte';
  import {
    ListEntries,
    GetEntry,
    GetPreferredEditor,
    SaveEntry,
    UpdateEntry,
    DeleteEntry,
    LockProfile,
    PullEntryFromElabftw,
    ImportEntryFromElabftw,
    PushEntryToElabftw,
    PushAllEntriesToElabftw,
    ListEntryRemoteLinks,
    UnlinkEntryFromElabftw,
  } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, openExternalURL, preventDefaultSubmit, toRelativeTime } from '../utils/helpers';
  import ProfileSettings from './ProfileSelector/ProfileSettings.svelte';
  import InstancesPushModal from './Instances/InstancesPushModal.svelte';
  import EntryRemoteModal from './Instances/EntryRemoteModal.svelte';
  import EntryImportModal from './Instances/EntryImportModal.svelte';
  import MarkdownEditor from "./MarkdownEditor.svelte";
  import TinyMceEditor from "./TinyMceEditor.svelte";
  import { showAlert } from "./stores/alert.svelte";
  import UploadsPanel from './Uploads/UploadsPanel.svelte';
  import EntryReadonlyPanels from './EntryReadonlyPanels.svelte';
  import EntityTypeIcon from './EntityTypeIcon.svelte';
  import EntityCategoryStatus from './EntityCategoryStatus.svelte';

  type Props = {
    profileUuid: string;
    profileName: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor' | 'profile';
  type PreferredEditor = 'tinymce' | 'markdown';

  let {profileUuid, profileName, onLogout}: Props = $props();

  let preferredEditor = $state<PreferredEditor>('tinymce');
  let preferencesReady = $state(false);
  let entryTitle = $state('');
  let entryMainText = $state('');
  let selectedEntry = $state<main.Entry | null>(null);
  let entryContentType = $state<1 | 2>(1);
  let entries = $state<main.EntrySummary[]>([]);
  let view = $state<View>('index');
  let loading = $state(false);
  let currentEntryId = $state<number | null>(null); // null means a new entry, otherwise update the selected one
  // Prevent editing an existing body with the wrong editor format
  let editorFormatMismatch = $derived(entryContentType !== (preferredEditor === 'markdown' ? 2 : 1));
  let importModalOpen = $state(false);
  let pushModalOpen = $state(false);
  let pushMode = $state<'single' | 'all'>('single'); // from View of an entry, push a single entry. From list of entries, push all.
  let pushEntryId = $state<number | null>(null); // # currentEntryId. This is for the modal to push to eLab.
  let lastFailedPush = $state<{
    instanceId: number;
    entityType: 'experiment' | 'resource';
  } | null>(null);
  let remoteLinks = $state<main.EntryRemoteLink[]>([]);
  // The oldest link determines whether this local entry is an experiment or resource
  let linkedEntityType = $derived(
    remoteLinks[0]?.type === 'experiment' ? 'experiment'
      : remoteLinks[0]?.type === 'resource' ? 'resource' : null
  );
  let remoteModalMode = $state<'open' | 'pull' | 'unlink' | null>(null);
  let remoteActionInProgress = $state(false);
  // Recreate the uploads panel after a pull to load the replacement files
  let uploadsRefreshKey = $state(0);

  // Load the preferred editor from this profile's SQLite settings
  async function loadPreferences(): Promise<void> {
    try {
      const editor = await GetPreferredEditor(profileUuid);
      preferredEditor = editor === 'markdown' ? 'markdown' : 'tinymce';
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      preferencesReady = true;
    }
  }

  async function loadEntry(id: number, requireEditor = false): Promise<void> {
    const entry = await GetEntry(profileUuid, id);
    const links = await ListEntryRemoteLinks(profileUuid, id);
    // Ignore stale responses if the user has opened another entry
    if (currentEntryId !== id || (requireEditor && view !== 'editor')) return;
    entryTitle = entry.title;
    entryMainText = entry.body;
    selectedEntry = entry;
    entryContentType = entry.contentType === 2 ? 2 : 1;
    remoteLinks = links;
  }

  async function openEntry(id: number): Promise<void> {
    showAlert(null);
    currentEntryId = id;
    try {
      await loadEntry(id);
      view = 'editor';
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  async function refreshEntries(): Promise<void> {
    loading = true;
    try {
      entries = await ListEntries(profileUuid);
    } catch (e: unknown) {
      console.error(e);
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  async function openIndex(): Promise<void> {
    await refreshEntries();
    showAlert(null);
    view = 'index';
  }

  async function logout(): Promise<void> {
    try {
      await LockProfile();
      onLogout?.();
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  // Start a blank entry using the profile's preferred editor
  function openEditor(): void {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
    selectedEntry = null;
    entryContentType = preferredEditor === 'markdown' ? 2 : 1;
    showAlert(null);
    currentEntryId = null;
    remoteLinks = [];
  }

  // Save a new entry or update the selected one
  async function saveOrUpdateEntry(): Promise<void> {
    if (editorFormatMismatch) {
      showAlert({type: 'warning', message: 'Change your preferred editor in Profile settings to edit this entry.'});
      return;
    }
    showAlert({type: 'info', message: currentEntryId ? 'Updating...' : 'Saving...'});
    try {
      if (currentEntryId) {
        await UpdateEntry(profileUuid, currentEntryId, entryTitle, entryMainText, entryContentType);
        remoteLinks = await ListEntryRemoteLinks(profileUuid, currentEntryId);
        showAlert({type: 'success', message: 'Entry updated ✔'});
      } else {
        const id = await SaveEntry(profileUuid, entryTitle, entryMainText, entryContentType);
        currentEntryId = id;
        showAlert({type: 'success', message: `Saved with id ${id} ✔`});
      }

      await refreshEntries();
      if (currentEntryId) selectedEntry = await GetEntry(profileUuid, currentEntryId);
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  let profileSettingsReturnView = $state<'index' | 'editor'>('index');

  function openProfileSettings(): void {
    showAlert(null);
    if (view !== 'profile') profileSettingsReturnView = view;
    view = 'profile';
  }

  function closeProfileSettings(): void {
    showAlert(null);
    view = profileSettingsReturnView;
  }

  // Uploads need a saved entry ID before they can be attached
  async function ensureEntrySaved(): Promise<number | null> {
    if (!currentEntryId) await saveOrUpdateEntry();
    return currentEntryId;
  }

  async function deleteEntry(id: number, title: string): Promise<void> {
    showAlert(null);
    const confirmed = window.confirm(`Delete "${title}"? This cannot be undone.`);
    if (!confirmed) {
      return;
    }
    try {
      await DeleteEntry(profileUuid, id);
      await refreshEntries();
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  const handleSubmit = preventDefaultSubmit(saveOrUpdateEntry);

  onMount(() => {
    showAlert(null);
    void refreshEntries();
    void loadPreferences();
  });

  async function confirmImport(instanceId: number, entityType: 'experiment' | 'resource', remoteId: number): Promise<void> {
    const result = await ImportEntryFromElabftw(profileUuid, instanceId, entityType, remoteId);
    importModalOpen = false;
    await refreshEntries();
    const warning = result.warnings?.length ? ` Warning: ${result.warnings.join(' ')}` : '';
    showAlert({
      type: result.warnings?.length ? 'warning' : 'success',
      message: `Imported ${result.type} #${result.remoteId} as local entry #${result.localId} ✔${warning}`,
    });
  }

  // modal helpers
  function openPushModal(mode: 'single' | 'all', entryId: number | null = null): void {
    pushMode = mode;
    pushEntryId = entryId;
    showAlert(null);
    pushModalOpen = true;
  }

  function closePushModal(): void {
    pushModalOpen = false;
    lastFailedPush = null;
  }

  function openRemoteModal(mode: 'open' | 'pull' | 'unlink'): void {
    if (remoteLinks.length === 0) return;
    showAlert(null);
    if (mode === 'open' && remoteLinks.length === 1) {
      openExternalURL(remoteLinks[0].url);
      return;
    }
    remoteModalMode = mode;
  }

  function closeRemoteModal(): void {
    if (remoteActionInProgress) return;
    remoteModalMode = null;
  }

  async function confirmPull(link: main.EntryRemoteLink): Promise<void> {
    const entryId = currentEntryId;
    if (entryId === null) {
      showAlert({type: 'error', message: 'No entry selected.'});
      return;
    }
    if (remoteActionInProgress) return;
    remoteActionInProgress = true;

    try {
      const result = await PullEntryFromElabftw(
        profileUuid,
        entryId,
        link.instanceId,
        link.type,
      );

      // Ignore a pull result if another entry is now selected
      if (currentEntryId !== entryId || view !== 'editor') {
        remoteModalMode = null;
        return;
      }

      await loadEntry(entryId, true);
      if (currentEntryId !== entryId || view !== 'editor') {
        remoteModalMode = null;
        return;
      }
      uploadsRefreshKey += 1;
      await refreshEntries();

      const warning = result.warnings?.length ? ` Warning: ${result.warnings.join(' ')}` : '';
      showAlert({
        type: result.warnings?.length ? 'warning' : 'success',
        message: `Pulled ${result.type} #${result.remoteId} and replaced the local entry ✔${warning}`,
      });
      remoteModalMode = null;
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      remoteActionInProgress = false;
    }
  }

  async function confirmUnlink(link: main.EntryRemoteLink): Promise<void> {
    const entryId = currentEntryId;
    if (entryId === null || remoteActionInProgress) return;
    remoteActionInProgress = true;

    try {
      await UnlinkEntryFromElabftw(profileUuid, entryId, link.instanceId, link.remoteId, link.type);
      remoteModalMode = null;
      if (currentEntryId !== entryId || view !== 'editor') return;

      remoteLinks = await ListEntryRemoteLinks(profileUuid, entryId);
      selectedEntry = await GetEntry(profileUuid, entryId);
      await refreshEntries();
      showAlert({type: 'success', message: 'Entry unlinked from eLabFTW ✔'});
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      remoteActionInProgress = false;
    }
  }

  async function confirmPush(
    instanceId: number,
    entityType: 'experiment' | 'resource',
    force = false,
  ): Promise<void> {
    try {
      if (pushMode === 'all') {
        const results = await PushAllEntriesToElabftw(profileUuid, instanceId, entityType, force);
        showAlert({type: 'success', message: `Pushed ${results.length} entries ✔`});
      } else {
        if (!pushEntryId) {
          showAlert({type: 'error', message: 'No entry selected.'});
          return;
        }

        const result = await PushEntryToElabftw(profileUuid, pushEntryId, instanceId, entityType, force);
        remoteLinks = await ListEntryRemoteLinks(profileUuid, currentEntryId);
        const warning = result.uploadWarnings?.length ? ` Upload warning: ${result.uploadWarnings.join(' ')}` : '';

        showAlert({
          type: result.uploadWarnings?.length ? 'warning' : 'success',
          message: `Entry ${result.action} as ${result.type} #${result.remoteId} ✔${warning}`,
        });
      }

      lastFailedPush = null;
      pushModalOpen = false;
    } catch (e: unknown) {
      const message = errorMessage(e);
      // warning if remote data is more recent than lite
      if (message.includes('was modified after your last sync')) {
        lastFailedPush = { instanceId, entityType };
        showAlert({ type: 'warning', message });
      } else {
        lastFailedPush = null;
        showAlert({ type: 'error', message });
      }
    }
  }
</script>

<!-- when using editor, the container becomes larger (side panel with uploads/metadata appears) -->
<div class='container' class:editor-container={view === 'editor'}>
  <header class='flex justify-between items-center mb-2' class:border-bottom={view !== 'editor'}>
    <div class='w-100 text-ellipsis'>
      {#if view === 'index'}
        <h1>My Entries</h1>
        <h2>Manage your saved entries.</h2>
      {:else if view === 'editor'}
        <div class='flex items-center gap-1'>
          {#if linkedEntityType}
            <h1 class='text-info'><span aria-hidden='true'><EntityTypeIcon type={linkedEntityType} /></span></h1>
          {/if}
          <div class='grid'>
            <h1 class='text-ellipsis'>{entryTitle.trim() || 'Untitled entry'}</h1>
            <EntityCategoryStatus snapshot={selectedEntry?.remoteDetails} />
          </div>
        </div>
      {:else} <!-- view === 'profile' -->
        <h1>Profile settings</h1>
        <h2>Manage preferences for {profileName}.</h2>
      {/if}
    </div>

    <div class='flex gap-1 items-center'>
      <div class='profile-pill flex items-center' title={profileName}>
        <span class='profile-avatar'>{profileName.slice(0, 2).toUpperCase()}</span>
        <span class='text-strong'>{profileName}</span>
      </div>

      <button
        class='btn btn-secondary'
        type='button'
        onclick={openProfileSettings}
        disabled={!preferencesReady || view === 'profile'}
        aria-label='Profile settings'
        title='Profile settings'
      >
        <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M10.5 2h3l.55 2.6c.63.18 1.24.43 1.8.75l2.23-1.5 2.13 2.13-1.5 2.23c.32.56.57 1.17.75 1.8L22 10.5v3l-2.6.55a8 8 0 0 1-.75 1.8l1.5 2.23-2.13 2.13-2.23-1.5a8 8 0 0 1-1.8.75L13.5 22h-3l-.55-2.6a8 8 0 0 1-1.8-.75l-2.23 1.5-2.13-2.13 1.5-2.23a8 8 0 0 1-.75-1.8L2 13.5v-3l2.6-.55c.18-.63.43-1.24.75-1.8l-1.5-2.23 2.13-2.13 2.23 1.5c.56-.32 1.17-.57 1.8-.75z'/><circle cx='12' cy='12' r='3'/></svg>
      </button>
      <button class='btn btn-danger' onclick={logout}>Logout</button>
    </div>
  </header>

  <!-- VIEW MODE -->
  {#if view === 'index'}
    <section class='panel' aria-labelledby='entries-title'>
      <div class='flex justify-between items-center mb-2 border-bottom'>
        <div class='flex items-center gap-1'>
          <div class='icon' aria-hidden='true'>&#x2756;</div>
          <div>
            <h3 id='entries-title'>Saved entries</h3>
            <span class='description'>
              {entries.length === 1 ? '1 entry' : `${entries.length} entries`}
            </span>
          </div>
        </div>
        <div class='flex gap-1'>
          <button class='btn btn-secondary' type='button' onclick={() => { showAlert(null); importModalOpen = true; }}>
            Import from eLabFTW
          </button>
          <button class='btn btn-primary' onclick={openEditor} disabled={!preferencesReady}><span aria-hidden='true'>+</span> Create entry</button>
        </div>
      </div>

      {#if loading}
        <div class='empty-state'>
          Loading entries...
        </div>
      {:else if entries.length === 0}
        <div class='empty-state'>
          <div class='icon' aria-hidden='true'>✎</div>
          <h3>No entries yet</h3>
          <p>Create your first entry to start writing.</p>
        </div>
      {:else}
        <div class='grid gap-1'>
          {#each entries as e (e.id)}
            <div class='flex gap-1'>
              <button type='button' class='entry-card' onclick={() => openEntry(e.id)} disabled={!preferencesReady}>
                <span class='icon-sm' aria-hidden='true'><EntityTypeIcon type={e.remoteType} /></span>
                <span class='grid gap-03'>
                <span class='text-ellipsis text-strong text-big'>
                  {e.title || 'Untitled entry'}
                </span>
                <span class='description'>
                  Last edited {toRelativeTime(e.modifiedAt)}
                </span>
                <EntityCategoryStatus snapshot={e.remoteDetails} />
              </span>

                <span class='text-orange text-strong' aria-hidden='true'>
                Open &#8594;
              </span>
              </button>
              <button type='button' class='btn btn-danger ' onclick={() => deleteEntry(e.id, e.title)}
                      aria-label={`Delete ${e.title}`}>
                <span aria-hidden='true'>&#128465;</span>
              </button>
            </div>
          {/each}
        </div>
      {/if}
      <div class='flex justify-end mt-2 gap-1'>
        {#if entries.length !== 0}
          <button class='btn btn-secondary' onclick={() => openPushModal('all')}>Push all entries to eLabFTW</button>
        {/if}
      </div>
    </section>
    <!-- PROFILE SETTINGS -->
  {:else if view === 'profile'}
    <ProfileSettings
      {profileUuid}
      {preferredEditor}
      onBack={closeProfileSettings}
      onSaved={(editor) => {
        if (currentEntryId === null && entryMainText.trim() === '') {
          entryContentType = editor === 'markdown' ? 2 : 1;
        }
        preferredEditor = editor;
      }}
    />
    <!-- VIEW EDITOR MODE -->
  {:else if view === 'editor'}
    <div>
      <div class='panel flex justify-between mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
          <div class='flex gap-1'>
              {#if remoteLinks.length > 0}
                <div class='flex gap-1'>
                  <button
                    type='button'
                    class='btn btn-secondary btn-external-link'
                    title={remoteLinks.length === 1
                      ? `Open ${remoteLinks[0].type} #${remoteLinks[0].remoteId} on ${remoteLinks[0].siteUrl}`
                      : 'Choose an eLabFTW instance'}
                    onclick={() => openRemoteModal('open')}
                  >
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M14 3h7v7M10 14 21 3M21 13v7a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1h7'/></svg>
                    Open in eLabFTW
                  </button>
                  <button type='button' class='btn btn-secondary' onclick={() => openRemoteModal('pull')}>
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'><path fill='currentColor' d='M169.4 502.6c12.5 12.5 32.8 12.5 45.3 0l160-160c12.5-12.5 12.5-32.8 0-45.3s-32.8-12.5-45.3 0L224 402.7V32c0-17.7-14.3-32-32-32s-32 14.3-32 32v370.7L54.6 297.3c-12.5-12.5-32.8-12.5-45.3 0s-12.5 32.8 0 45.3l160 160z'/></svg>
                    Pull
                  </button>
                  <button type='button' class='btn btn-secondary btn-quiet'
                          title='Remove the local sync link without deleting either entry'
                          onclick={() => openRemoteModal('unlink')}>
                    Unlink
                  </button>
                </div>
              {/if}
            <button class='btn btn-secondary' type='button' disabled={!currentEntryId}
                    onclick={() => openPushModal('single', currentEntryId)}>
              <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'><path fill='currentColor' d='M214.6 9.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 109.3V480c0 17.7 14.3 32 32 32s32-14.3 32-32V109.3l105.4 105.4c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z'/></svg>
              Push
            </button>
            <button class='btn btn-primary' type='submit' form='entry-editor-form' disabled={editorFormatMismatch}>
              <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2'><path d='M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z'/><path d='M17 21v-8H7v8M7 3v5h8'/></svg>
              Save
            </button>
          </div>
      </div>
      <div class='editor-layout'>
        <form id='entry-editor-form' class='panel' onsubmit={handleSubmit}>
        <div>
          <label for='entryTitle'>Title</label>
          <input
            {@attach autofocus}
            required
            id='entryTitle'
            type='text'
            class='input text-strong text-big'
            bind:value={entryTitle}
            placeholder='Title of your entry'
          />
        </div>

        <div class='mt-2 mb-1'>
          <label id='entryMainTextLabel'>Content</label>
        </div>
        {#if editorFormatMismatch}
          <div class='grid gap-1 mb-2'>
            <p class='description'>
              {currentEntryId === null ? 'This draft contains' : 'This entry was saved as'}
              {entryContentType === 1 ? 'HTML (TinyMCE)' : 'Markdown'} content.
              Select that editor in Profile settings to edit it without losing formatting.
            </p>
            <div>
              <button type='button' class='btn btn-secondary' onclick={openProfileSettings}>Open profile settings</button>
            </div>
          </div>
        {:else if preferredEditor === 'tinymce'}
          <TinyMceEditor
            value={entryMainText}
            onChange={(next) => { entryMainText = next; }}
            label='entryMainTextLabel'
          />
        {:else}
          <MarkdownEditor
            value={entryMainText}
            onChange={(next) => { entryMainText = next; }}
            label='entryMainTextLabel'
          />
        {/if}
        </form>
        <aside class='editor-sidebar'>
          <section class='panel'>
            <h3 class='border-bottom'>Entry information</h3>
            <div class='entry-info-row'><span>Type</span><span class='flex items-center gap-03'><EntityTypeIcon type={linkedEntityType} /> {linkedEntityType === 'resource' ? 'Resource' : linkedEntityType === 'experiment' ? 'Experiment' : 'Local entry'}</span></div>
            <div class='entry-info-row'><span>Category / Status</span><EntityCategoryStatus snapshot={selectedEntry?.remoteDetails} emptyLabel={remoteLinks.length > 0 ? 'Pull to load' : 'Not linked'} /></div>
            <div class='entry-info-row'><span>Created</span><span>{selectedEntry ? new Intl.DateTimeFormat(undefined, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(selectedEntry.createdAt)) : 'Not saved yet'}</span></div>
            <div class='entry-info-row'><span>Modified</span><span>{selectedEntry ? new Intl.DateTimeFormat(undefined, {dateStyle: 'medium', timeStyle: 'short'}).format(new Date(selectedEntry.modifiedAt)) : 'Not saved yet'}</span></div>
            <div class='entry-info-row'><span>Content type</span><span>{entryContentType === 1 ? 'HTML (TinyMCE)' : 'Markdown'}</span></div>
          </section>
          {#key uploadsRefreshKey}
            <UploadsPanel
              {profileUuid}
              entryId={currentEntryId}
              {ensureEntrySaved}
              compact
            />
          {/key}
          <EntryReadonlyPanels
            snapshot={selectedEntry?.remoteDetails}
            source={selectedEntry?.remoteSource}
            linked={remoteLinks.length > 0}
          />
        </aside>
      </div>
    </div>
  {/if}
  {#if importModalOpen}
    <EntryImportModal
      {profileUuid}
      onClose={() => { importModalOpen = false; }}
      onImport={confirmImport}
    />
  {/if}
  {#if pushModalOpen}
    <InstancesPushModal
      {profileUuid}
      force={lastFailedPush !== null}
      lockedEntityType={pushMode === 'single' ? linkedEntityType : null}
      onClose={closePushModal}
      onPush={confirmPush}
    />
  {/if}
  {#if remoteModalMode}
    <EntryRemoteModal
      links={remoteLinks}
      mode={remoteModalMode}
      onClose={closeRemoteModal}
      onOpen={(link) => {
        openExternalURL(link.url);
        closeRemoteModal();
      }}
      onPull={confirmPull}
      onUnlink={confirmUnlink}
    />
  {/if}
</div>
