<script lang='ts'>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import {
    ListEntries,
    GetEntry,
    SaveEntry,
    DeleteEntry,
    LockProfile,
    ListElabftwInstances,
    AddElabftwInstance,
    UpdateElabftwInstance,
    DeleteElabftwInstance,
    FetchElabftwInfo,
    PushEntryToElabftw,
    PushAllEntriesToElabftw
  } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, preventDefaultSubmit } from '../utils/helpers';
  import Alert from './Alert.svelte';
  import type { AlertState } from './Alert.svelte';
  import Modal from './Modal.svelte';

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
  let alert = $state<AlertState | null>(null);
  // elabftw instances
  let instances = $state<main.ElabftwInstance[]>([]);
  let instanceSiteUrl = $state('');
  let instanceApiKey = $state('');
  let instanceVerifyTls = $state(true);
  // test info endpoint
  let elabftwInfoOutput = $state('');
  // update elabftw instance
  let editingInstanceId = $state<number | null>(null);
  // push entries to elabftw
  // TODO: separate into components because main app is handling everything right now
  let currentEntryId = $state<number | null>(null);
  let pushModalOpen = $state(false);
  let pushMode = $state<'single' | 'all'>('single');
  let pushEntityType = $state<'experiment' | 'resource'>('experiment');
  let pushInstanceId = $state<number | null>(null);
  let pushEntryId = $state<number | null>(null);

  function toRelativeTime(iso: string, locale = 'en'): string {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id: number): Promise<void> {
    alert = null;
    currentEntryId = id;
    try {
      const e: main.Entry = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMainText = e.body;
      view = 'editor';
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  async function refreshEntries(): Promise<void> {
    loading = true;
    try {
      entries = await ListEntries(profileUuid);
    } catch (e: unknown) {
      console.error(e);
      alert = { type: 'error', message: errorMessage(e) };
    } finally {
      loading = false;
    }
  }

  async function openIndex(): Promise<void> {
    await refreshEntries();
    alert = null;
    view = 'index';
  }

  async function logout(): Promise<void> {
    try {
      await LockProfile();
      onLogout?.();
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  function openEditor(): void {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
    alert = null;
    currentEntryId = null;
  }

  async function saveEntry(): Promise<void> {
    alert = { type: 'info', message: 'Saving...' };
    try {
      const id = await SaveEntry(profileUuid, entryTitle, entryMainText);
      alert = { type: 'success', message: `Saved with id ${id} ✔` };
      await refreshEntries();
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  async function deleteEntry(id: number, title: string): Promise<void> {
    alert = null;
    const confirmed = window.confirm(`Delete "${title}"? This cannot be undone.`);
    if (!confirmed) {
      return;
    }
    try {
      await DeleteEntry(profileUuid, id);
      await refreshEntries();
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  const handleSubmit = preventDefaultSubmit(saveEntry);

  onMount(() => {
    void refreshEntries();
  });

  // elabftw instances

  async function refreshInstances(): Promise<void> {
    loading = true;
    try {
      instances = await ListElabftwInstances(profileUuid);
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    } finally {
      loading = false;
    }
  }

  async function openInstances(): Promise<void> {
    alert = null;
    await refreshInstances();
    view = 'instances';
  }

  async function saveInstance(): Promise<void> {
    alert = { type: 'info', message: editingInstanceId ? 'Updating instance...' : 'Adding instance...' };

    try {
      if (editingInstanceId) {
        await UpdateElabftwInstance(
          profileUuid,
          editingInstanceId,
          instanceSiteUrl,
          instanceApiKey,
          instanceVerifyTls,
        );

        alert = { type: 'success', message: 'eLabFTW instance updated ✔' };
      } else {
        await AddElabftwInstance(
          profileUuid,
          instanceSiteUrl,
          instanceApiKey,
          instanceVerifyTls,
        );

        alert = { type: 'success', message: 'eLabFTW instance added ✔' };
      }

      editingInstanceId = null;
      instanceSiteUrl = '';
      instanceApiKey = '';
      instanceVerifyTls = true;

      await refreshInstances();
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  const handleInstanceSubmit = preventDefaultSubmit(saveInstance);

  async function deleteInstance(id: number, siteUrl: string): Promise<void> {
    const confirmed = window.confirm(`Delete eLabFTW instance "${siteUrl}"?`);
    if (!confirmed) return;

    try {
      await DeleteElabftwInstance(profileUuid, id);
      await refreshInstances();
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  // const handleInstanceSubmit = preventDefaultSubmit(addInstance);

  // test elabftw instances

  async function testInstance(id: number): Promise<void> {
    alert = { type: 'info', message: 'Fetching eLabFTW /info...' };
    elabftwInfoOutput = '';

    try {
      const info = await FetchElabftwInfo(profileUuid, id);
      elabftwInfoOutput = JSON.stringify(info.raw, null, 2);
      alert = { type: 'success', message: 'Connected to eLabFTW ✔' };
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }

  // TODO: move all instance related to another component
  function editInstance(instance: main.ElabftwInstance): void {
    editingInstanceId = instance.id;
    instanceSiteUrl = instance.siteUrl;
    instanceApiKey = '';
    instanceVerifyTls = instance.verifyTls;
    alert = {
      type: 'info',
      message: 'Editing instance. Leave API key empty to keep the current key.',
    };
  }

  function cancelEditInstance(): void {
    editingInstanceId = null;
    instanceSiteUrl = '';
    instanceApiKey = '';
    instanceVerifyTls = true;
    alert = null;
  }

  // modal helpers
  async function openPushModal(mode: 'single' | 'all', entryId: number | null = null): Promise<void> {
    pushMode = mode;
    pushEntryId = entryId;
    pushEntityType = 'experiment';
    alert = null;

    await refreshInstances();

    pushInstanceId = instances.length === 1 ? instances[0].id : null;
    pushModalOpen = true;
  }

  function closePushModal(): void {
    pushModalOpen = false;
  }

  async function confirmPush(): Promise<void> {
    if (!pushInstanceId) {
      alert = { type: 'error', message: 'Select an eLabFTW instance.' };
      return;
    }

    try {
      if (pushMode === 'all') {
        const results = await PushAllEntriesToElabftw(profileUuid, pushInstanceId, pushEntityType);
        alert = { type: 'success', message: `Pushed ${results.length} entries ✔` };
      } else {
        if (!pushEntryId) {
          alert = { type: 'error', message: 'No entry selected.' };
          return;
        }

        const result = await PushEntryToElabftw(profileUuid, pushEntryId, pushInstanceId, pushEntityType);
        alert = { type: 'success', message: `Entry ${result.action} as ${result.type} #${result.remoteId} ✔` };
      }

      pushModalOpen = false;
    } catch (e: unknown) {
      alert = { type: 'error', message: errorMessage(e) };
    }
  }
</script>

<div class='container'>
  <header class='flex justify-between items-center mb-2 border-bottom'>
    <div class='w-100 text-ellipsis'>
      <p class='header-subtitle'>Unlocked profile</p>
      {#if view === 'index'}
        <h1>My Entries</h1>
        <h2>Manage your saved entries.</h2>
      {:else if view === 'editor'}
        <h1 class='text-ellipsis'>{entryTitle.trim() || 'Untitled entry'}</h1>
        <h2>Write, edit, and save your entry.</h2>
      {:else}
        <h1 class='text-ellipsis'>eLabFTW instances</h1>
        <h2>Add the server you want to sync with</h2>
      {/if}
    </div>

    <div class='flex gap-1 items-center'>
      <div class='profile-pill flex items-center gap-1' title={profileName}>
        <span class='profile-avatar'>{profileName.slice(0, 2).toUpperCase()}</span>
          <span class='text-strong'>{profileName}</span>
      </div>

      <button class='btn btn-danger' onclick={logout}>
        Logout
      </button>
    </div>
  </header>

  <!-- VIEW MODE -->
  {#if view === 'index'}
    <section class='panel' aria-labelledby='entries-title'>
      <div class='flex justify-between items-center mb-2 border-bottom'>
        <div class='flex items-center gap-1'>
          <div class='icon' aria-hidden='true'>▣</div>
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
                  Last edited {toRelativeTime(e.updatedAt)}
                </span>
              </span>

              <span class='text-orange text-strong' aria-hidden='true'>
                Open &#8594;
              </span>
            </button>
            <button type='button' class='btn btn-danger ' onclick={() => deleteEntry(e.id, e.title)} aria-label={`Delete ${e.title}`}>
              <span aria-hidden='true'>&#128465;</span>
            </button>
            </div>
          {/each}
        </div>
      {/if}
<!--        TODO -->
        <div class='flex justify-end mt-2 gap-1'>
          {#if entries.length !== 0}
          <button class='btn btn-secondary' onclick={() => openPushModal('all')}>Push all entries to eLabFTW</button>
          {/if}
          <button class='btn btn-secondary'>Fetch entries from eLabFTW</button>
          <button class='btn btn-secondary' onclick={openInstances}>
            See eLabFTW Instances
          </button>

        </div>
    </section>
    <!-- VIEW ELABFTW INSTANCES -->
  {:else if view === 'instances'}
    <section class='panel'>
      <div class='flex justify-between border-bottom mb-2 items-center'>
        <span>
          Add the site URL and your API key to allow communication between the desktop app and your eLabFTW instance.
          <br>
          See the <a href='https://doc.elabftw.net/docs/usage/api'>Documentation</a> to learn how to create a new API key.
        </span>
        <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
      </div>

      <form onsubmit={handleInstanceSubmit} class='grid gap-1'>
        <div>
          <label for='instanceSiteUrl'>Site URL</label>
          <input
            id='instanceSiteUrl'
            type='url'
            class='input'
            bind:value={instanceSiteUrl}
            placeholder='https://elab.example.org'
          />
        </div>

        <div>
          <label for='instanceApiKey'>API key</label>
          <input
            id='instanceApiKey'
            type='password'
            class='input'
            bind:value={instanceApiKey}
            placeholder={editingInstanceId ? 'Leave empty to keep current API key' : 'Your eLabFTW API key'}
          />
        </div>

        <label class='checkbox-row flex items-center gap-1'>
          <input type='checkbox' bind:checked={instanceVerifyTls} />
          <span class='checkbox-box'></span>
          <span>Verify TLS certificates</span>
        </label>

        <div class='flex justify-end gap-1'>
          {#if editingInstanceId}
            <button class='btn btn-secondary' type='button' onclick={cancelEditInstance}>
              Cancel
            </button>
          {/if}

          <button class='btn btn-primary' type='submit'>
            {editingInstanceId ? 'Update instance' : 'Add instance'}
          </button>
        </div>
      </form>

      <div class='border-bottom mt-2 mb-2'></div>

      {#if loading}
        <div class='empty-state'>Loading instances...</div>
      {:else if instances.length === 0}
        <div class='empty-state'>
          <h3>No eLabFTW instances yet</h3>
          <p>Add one above before pushing entries.</p>
        </div>
      {:else}
        <div class='grid gap-1'>
          {#each instances as instance (instance.id)}
            <div class='flex justify-between items-center gap-1'>
              <div class='grid gap-03'>
                <span class='text-white text-strong'>{instance.siteUrl}</span>
                <span class='description'>
                TLS verification: {instance.verifyTls ? 'enabled' : 'disabled'}
              </span>
              </div>

              <div>

              <button
                type='button'
                class='btn btn-danger'
                onclick={() => deleteInstance(instance.id, instance.siteUrl)}
              >
                Delete
              </button>
              <button
                type='button'
                class='btn btn-secondary'
                onclick={() => editInstance(instance)}
              >
                Edit
              </button>
                <button type='button' class='btn btn-secondary' onclick={() => testInstance(instance.id)}>
                  Test
                </button>
              </div>
            </div>
          {/each}
          {#if elabftwInfoOutput}
            <div class='mt-2'>
              <h3>eLabFTW /info response</h3>
              <pre class='panel'>{elabftwInfoOutput}</pre>
            </div>
          {/if}
        </div>
      {/if}

    </section>
    <!-- VIEW EDITOR MODE -->
  {:else if view === 'editor'}
    <section class='panel'>
      <form onsubmit={handleSubmit}>
        <div class='flex justify-between border-bottom mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
          <div class='flex gap-1'>
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
            id='entryTitle'
            type='text'
            class='input text-strong text-big'
            bind:value={entryTitle}
            placeholder='Title of your entry'
          />
        </div>

        <label for='entryMainText' class='mt-2'>Entry main text</label>
        <textarea id='entryMainText' bind:value={entryMainText} placeholder='The main text...'></textarea>
      </form>
    </section>
  {/if}
  {#if pushModalOpen}
    <Modal title='Push to eLabFTW' onClose={closePushModal}>
      {#if instances.length > 1}
        <label for='pushInstance'>Instance</label>
        <select id='pushInstance' class='input' bind:value={pushInstanceId}>
          <option value={null}>Select instance...</option>
          {#each instances as instance (instance.id)}
            <option value={instance.id}>{instance.siteUrl}</option>
          {/each}
        </select>
      {:else if instances.length === 1}
        <p class='description'>Instance: {instances[0].siteUrl}</p>
      {:else}
        <p class='description'>No eLabFTW instances configured.</p>
      {/if}

      <label for='pushEntityType' class='mt-2'>Remote type</label>
      <select id='pushEntityType' class='input' bind:value={pushEntityType}>
        <option value='experiment'>Experiment</option>
        <option value='resource'>Resource</option>
      </select>

      <svelte:fragment slot='actions'>
        <button class='btn btn-secondary' type='button' onclick={closePushModal}>Cancel</button>
        <button class='btn btn-primary' type='button' onclick={confirmPush}>Push</button>
      </svelte:fragment>
    </Modal>
  {/if}
  {#if alert}
    <Alert type={alert.type} message={alert.message} />
  {/if}
</div>
