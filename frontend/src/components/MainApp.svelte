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
  import {
    ListEntries,
    GetEntry,
    GetPreferredEditor,
    SaveEntry,
    UpdateEntry,
    DeleteEntry,
    LockProfile,
    PullEntryFromElabftw,
    PushEntryToElabftw,
    PushAllEntriesToElabftw,
    ListEntryRemoteLinks,
  } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, openExternalURL, preventDefaultSubmit, toRelativeTime } from '../utils/helpers';
  import InstancesView from './Instances/InstancesView.svelte';
  import ProfileSettings from './ProfileSelector/ProfileSettings.svelte';
  import InstancesPushModal from './Instances/InstancesPushModal.svelte';
  import EntryPullModal from './Instances/EntryPullModal.svelte';
  import MarkdownEditor from "./MarkdownEditor.svelte";
  import TinyMceEditor from "./TinyMceEditor.svelte";
  import { showAlert } from "./stores/alert.svelte";
  import UploadsPanel from './Uploads/UploadsPanel.svelte';

  type Props = {
    profileUuid: string;
    profileName: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor' | 'instances' | 'profile';
  type PreferredEditor = 'tinymce' | 'markdown';

  let {profileUuid, profileName, onLogout}: Props = $props();

  let preferredEditor = $state<PreferredEditor>('tinymce');
  let preferencesReady = $state(false);
  let entryTitle = $state('');
  let entryMainText = $state('');
  let entryContentType = $state<1 | 2>(1);
  let entries = $state<main.EntrySummary[]>([]);
  let view = $state<View>('index');
  let loading = $state(false);
  let currentEntryId = $state<number | null>(null); // if not null, Update entry. else Save
  // An existing body must not be interpreted as another format without conversion.
  let editorFormatMismatch = $derived(entryContentType !== (preferredEditor === 'markdown' ? 2 : 1));
  let pushModalOpen = $state(false);
  let pushMode = $state<'single' | 'all'>('single'); // from View of an entry, push a single entry. From list of entries, push all.
  let pushEntryId = $state<number | null>(null); // # currentEntryId. This is for the modal to push to eLab.
  let lastFailedPush = $state<{
    instanceId: number;
    entityType: 'experiment' | 'resource';
  } | null>(null);
  let remoteLinks = $state<main.EntryRemoteLink[]>([]);
  // Links are ordered oldest first, so the first push defines the remote type.
  let linkedEntityType = $derived(
    remoteLinks[0]?.type === 'experiment' ? 'experiment'
      : remoteLinks[0]?.type === 'resource' ? 'resource' : null
  );
  let pullLink = $state<main.EntryRemoteLink | null>(null);
  // reactive to trigger an uploads refresh after a pull
  let uploadsRefreshKey = $state(0);

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

  async function loadEntry(id: number): Promise<void> {
    const entry = await GetEntry(profileUuid, id);
    const links = await ListEntryRemoteLinks(profileUuid, id);
    entryTitle = entry.title;
    entryMainText = entry.body;
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

  function openEditor(): void {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
    entryContentType = preferredEditor === 'markdown' ? 2 : 1;
    showAlert(null);
    currentEntryId = null;
    remoteLinks = [];
  }

  // Save an entry // Update an existing entry
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
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
  }

  let profileSettingsReturnView = $state<'index' | 'editor' | 'instances'>('index');

  function openProfileSettings(): void {
    showAlert(null);
    if (view !== 'profile') profileSettingsReturnView = view;
    view = 'profile';
  }

  function closeProfileSettings(): void {
    showAlert(null);
    view = profileSettingsReturnView;
  }

  // use for Uploads to check the entry Id. Pass it to UploadsPAnel so that we dont need to check and warn
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

  function openInstances(): void {
    showAlert(null);
    view = 'instances';
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

  function openPullModal(link: main.EntryRemoteLink): void {
    showAlert(null);
    pullLink = link;
  }

  function closePullModal(): void {
    pullLink = null;
  }

  async function confirmPull(): Promise<void> {
    if (!currentEntryId || !pullLink) {
      showAlert({type: 'error', message: 'No remote entry selected.'});
      return;
    }

    try {
      const result = await PullEntryFromElabftw(
        profileUuid,
        currentEntryId,
        pullLink.instanceId,
        pullLink.type,
      );

      await loadEntry(currentEntryId);
      uploadsRefreshKey += 1;
      await refreshEntries();

      const warning = result.warnings?.length ? ` Warning: ${result.warnings.join(' ')}` : '';
      showAlert({
        type: result.warnings?.length ? 'warning' : 'success',
        message: `Pulled ${result.type} #${result.remoteId} and replaced the local entry ✔${warning}`,
      });
      pullLink = null;
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
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
      // warning if remote data is more recent than desktop
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

<div class='container'>
  <header class='flex justify-between items-center mb-2 border-bottom'>
    <div class='w-100 text-ellipsis'>
      {#if view === 'index'}
        <h1>My Entries</h1>
        <h2>Manage your saved entries.</h2>
      {:else if view === 'editor'}
        <h1 class='text-ellipsis'>{entryTitle.trim() || 'Untitled entry'}</h1>
        <h2>Write, edit, and save your entry.</h2>
      {:else if view === 'instances'}
        <h1 class='text-ellipsis'>eLabFTW instances</h1>
        <h2>Add the server you want to sync with</h2>
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
        <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'>
          <path d='M10.5 2h3l.55 2.6c.63.18 1.24.43 1.8.75l2.23-1.5 2.13 2.13-1.5 2.23c.32.56.57 1.17.75 1.8L22 10.5v3l-2.6.55a8 8 0 0 1-.75 1.8l1.5 2.23-2.13 2.13-2.23-1.5a8 8 0 0 1-1.8.75L13.5 22h-3l-.55-2.6a8 8 0 0 1-1.8-.75l-2.23 1.5-2.13-2.13 1.5-2.23a8 8 0 0 1-.75-1.8L2 13.5v-3l2.6-.55c.18-.63.43-1.24.75-1.8l-1.5-2.23 2.13-2.13 2.23 1.5c.56-.32 1.17-.57 1.8-.75z'/>
          <circle cx='12' cy='12' r='3'/>
        </svg>
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
        <button class='btn btn-primary' onclick={openEditor} disabled={!preferencesReady}><span aria-hidden='true'>+</span> Create entry</button>
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
                <span class='icon-sm' aria-hidden='true'>▤</span>
                <span class='grid gap-03'>
                <span class='text-ellipsis text-white text-strong text-big'>
                  {e.title || 'Untitled entry'}
                </span>
                <span class='description'>
                  Last edited {toRelativeTime(e.modifiedAt)}
                </span>
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
        <!-- TODO next version fetch entries: discuss how we handle it -->
        <!--        <button class='btn btn-secondary' disabled>Fetch entries from eLabFTW (next version)</button>-->
        <button class='btn btn-secondary' onclick={openInstances}>
          Manage your eLabFTW Instances
        </button>

      </div>
    </section>
    <!-- VIEW ELABFTW INSTANCES -->
  {:else if view === 'instances'}
    <InstancesView
      {profileUuid}
      onBack={openIndex}
    />
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
    <section class='panel'>
      <form onsubmit={handleSubmit}>
        <div class='flex justify-between border-bottom mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
          <div class='flex gap-1'>
              {#each remoteLinks as link (`${link.instanceId}-${link.type}-${link.remoteId}`)}
                <div class='flex gap-03'>
                  <button
                    type='button'
                    class='link-button flex items-center gap-03'
                    title={`Open ${link.type} #${link.remoteId} on ${link.siteUrl}`}
                    onclick={() => openExternalURL(link.url)}
                  >
                    {#if link.type === 'resource'}
                      <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 512 512'>
                        <path fill='currentColor' d='M192 32c0-17.7 14.3-32 32-32h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32V32zm32 352h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zm192 0h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zM320 192h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zm-182.6-3.9c12.5-12.5 32.8-12.5 45.3 0l45.3 45.3c12.5 12.5 12.5 32.8 0 45.3l-45.3 45.3c-12.5 12.5-32.8 12.5-45.3 0l-45.3-45.4c-12.5-12.5-12.5-32.8 0-45.3l45.3-45.3zM32 384h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32H32c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32z'/>
                      </svg>
                    {:else}
                      <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 448 512'>
                        <path fill='currentColor' d='M288 0H128c-17.7 0-32 14.3-32 32s14.3 32 32 32v151.5L7.5 426.3C2.6 435 0 444.7 0 454.7 0 486.4 25.6 512 57.3 512h333.4c31.6 0 57.3-25.6 57.3-57.3 0-10-2.6-19.8-7.5-28.4L320 215.5V64c17.7 0 32-14.3 32-32S337.7 0 320 0h-32zM192 215.5V64h64v151.5c0 11.1 2.9 22.1 8.4 31.8L306 320H142l41.6-72.7c5.5-9.7 8.4-20.6 8.4-31.8z'/>
                      </svg>
                    {/if}
                    Open {link.type} in eLabFTW
                    <svg aria-hidden='true' width='0.85em' height='0.85em' viewBox='0 0 512 512'>
                      <path fill='currentColor' d='M320 0c-17.7 0-32 14.3-32 32s14.3 32 32 32h82.7L201.3 265.4c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L448 109.3V192c0 17.7 14.3 32 32 32s32-14.3 32-32V32c0-17.7-14.3-32-32-32H320zM80 96c-44.2 0-80 35.8-80 80v256c0 44.2 35.8 80 80 80h256c44.2 0 80-35.8 80-80v-80c0-17.7-14.3-32-32-32s-32 14.3-32 32v80c0 8.8-7.2 16-16 16H80c-8.8 0-16-7.2-16-16V176c0-8.8 7.2-16 16-16h80c17.7 0 32-14.3 32-32s-14.3-32-32-32H80z'/>
                    </svg>
                  </button>
                  <button type='button' class='btn btn-secondary' onclick={() => openPullModal(link)}>
                    <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'>
                      <path fill='currentColor' d='M169.4 502.6c12.5 12.5 32.8 12.5 45.3 0l160-160c12.5-12.5 12.5-32.8 0-45.3s-32.8-12.5-45.3 0L224 402.7V32c0-17.7-14.3-32-32-32s-32 14.3-32 32v370.7L54.6 297.3c-12.5-12.5-32.8-12.5-45.3 0s-12.5 32.8 0 45.3l160 160z'/>
                    </svg>
                    Pull
                  </button>
                </div>
              {/each}
            <button class='btn btn-secondary' type='button' disabled={!currentEntryId}
                    onclick={() => openPushModal('single', currentEntryId)}>
              <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 384 512'>
                <path fill='currentColor' d='M214.6 9.4c-12.5-12.5-32.8-12.5-45.3 0l-160 160c-12.5 12.5-12.5 32.8 0 45.3s32.8 12.5 45.3 0L160 109.3V480c0 17.7 14.3 32 32 32s32-14.3 32-32V109.3l105.4 105.4c12.5 12.5 32.8 12.5 45.3 0s12.5-32.8 0-45.3l-160-160z'/>
              </svg>
              Push
            </button>
            <button class='btn btn-primary' type='submit' disabled={editorFormatMismatch}>Save</button>
          </div>
        </div>

        <div>
          <label for='entryTitle'>Entry title</label>
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
          <label id='entryMainTextLabel'>Entry main text</label>
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
        {#key uploadsRefreshKey}
          <UploadsPanel
            {profileUuid}
            entryId={currentEntryId}
            {ensureEntrySaved}
          />
        {/key}
      </form>
    </section>
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
  {#if pullLink}
    <EntryPullModal
      link={pullLink}
      onClose={closePullModal}
      onPull={confirmPull}
    />
  {/if}
</div>
