<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import {
    ListEntries,
    GetEntry,
    SaveEntry,
    UpdateEntry,
    DeleteEntry,
    LockProfile,
    PushEntryToElabftw,
    PushAllEntriesToElabftw,
    ListEntryRemoteLinks,
  } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, openExternalURL, preventDefaultSubmit } from '../utils/helpers';
  import InstancesView from './Instances/InstancesView.svelte';
  import InstancesPushModal from './Instances/InstancesPushModal.svelte';
  import MarkdownEditor from "./MarkdownEditor.svelte";
  import { showAlert } from "./stores/alert.svelte";
  import UploadsPanel from './Uploads/UploadsPanel.svelte';

  type Props = {
    profileUuid: string;
    profileName: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor' | 'instances';

  let {profileUuid, profileName, onLogout}: Props = $props();

  let entryTitle = $state('');
  let entryMainText = $state('');
  let entries = $state<main.EntrySummary[]>([]);
  let view = $state<View>('index');
  let loading = $state(false);
  let currentEntryId = $state<number | null>(null); // if not null, Update entry. else Save
  let pushModalOpen = $state(false);
  let pushMode = $state<'single' | 'all'>('single'); // from View of an entry, push a single entry. From list of entries, push all.
  let pushEntryId = $state<number | null>(null); // # currentEntryId. This is for the modal to push to eLab.
  let lastFailedPush = $state<{
    instanceId: number;
    entityType: 'experiment' | 'resource';
  } | null>(null);
  let remoteLinks = $state<main.EntryRemoteLink[]>([]);

  function toRelativeTime(iso: string, locale = 'en'): string {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id: number): Promise<void> {
    showAlert(null);
    currentEntryId = id;
    try {
      const e: main.Entry = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMainText = e.body;
      /* if entry already in eLabFTW, create a link to see it directly */
      remoteLinks = await ListEntryRemoteLinks(profileUuid, id);
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
    showAlert(null);
    currentEntryId = null;
    remoteLinks = [];
  }

  // Save an entry // Update an existing entry
  async function saveOrUpdateEntry(): Promise<void> {
    showAlert({type: 'info', message: currentEntryId ? 'Updating...' : 'Saving...'});
    try {
      if (currentEntryId) {
        await UpdateEntry(profileUuid, currentEntryId, entryTitle, entryMainText);
        remoteLinks = await ListEntryRemoteLinks(profileUuid, currentEntryId);
        showAlert({type: 'success', message: 'Entry updated ✔'});
      } else {
        const id = await SaveEntry(profileUuid, entryTitle, entryMainText);
        currentEntryId = id;
        showAlert({type: 'success', message: `Saved with id ${id} ✔`});
      }

      await refreshEntries();
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    }
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
      {:else} <!-- view === 'instances' -->
        <h1 class='text-ellipsis'>eLabFTW instances</h1>
        <h2>Add the server you want to sync with</h2>
      {/if}
    </div>

    <div class='flex gap-1 items-center'>
      <div class='profile-pill flex items-center' title={profileName}>
        <span class='profile-avatar'>{profileName.slice(0, 2).toUpperCase()}</span>
        <span class='text-strong'>{profileName}</span>
      </div>

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
        <button class='btn btn-primary' onclick={openEditor}><span aria-hidden='true'>+</span> Create entry</button>
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
              <button type='button' class='entry-card' onclick={() => openEntry(e.id)}>
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
    <!-- VIEW EDITOR MODE -->
  {:else if view === 'editor'}
    <section class='panel'>
      <form onsubmit={handleSubmit}>
        <div class='flex justify-between border-bottom mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
          <div class='flex gap-1'>
              {#each remoteLinks as link (`${link.instanceId}-${link.type}-${link.remoteId}`)}
                <button type='button' class='link-button' onclick={() => openExternalURL(link.url)}>
                  See {link.type} #{link.remoteId} at {link.siteUrl}
                </button>
              {/each}
            <button class='btn btn-secondary' type='button' disabled={!currentEntryId}
                    onclick={() => openPushModal('single', currentEntryId)}>
              Push to eLabFTW Instance
            </button>
            <button class='btn btn-primary' type='submit'>Save</button>
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

        <label for='entryMainText' class='mt-2'>Entry main text</label>
        <MarkdownEditor
          value={entryMainText}
          onChange={(next) => entryMainText = next}
        />
        <UploadsPanel
          {profileUuid}
          entryId={currentEntryId}
          {ensureEntrySaved}
        />
      </form>
    </section>
  {/if}
  {#if pushModalOpen}
    <InstancesPushModal
      {profileUuid}
      force={lastFailedPush !== null}
      onClose={closePushModal}
      onPush={confirmPush}
    />
  {/if}
</div>
