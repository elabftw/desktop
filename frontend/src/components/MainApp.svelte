<script lang='ts'>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, GetEntry, SaveEntry, DeleteEntry, LockProfile } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, preventDefaultSubmit } from '../utils/helpers';
  import Alert from './Alert.svelte';
  import type { AlertState } from './Alert.svelte';

  type Props = {
    profileUuid: string;
    profileName: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor';

  let {profileUuid, profileName, onLogout}: Props = $props();

  let entryTitle = $state('');
  let entryMainText = $state('');
  let entries = $state<main.EntrySummary[]>([]);
  let view = $state<View>('index');
  let loading = $state(false);
  let alert = $state<AlertState | null>(null);

  function toRelativeTime(iso: string, locale = 'en'): string {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id: number): Promise<void> {
    alert = null;
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
</script>

<div class='container'>
  <header class='flex justify-between items-center mb-2 border-bottom'>
    <div class='w-100'>
      <p class='header-subtitle'>Unlocked profile</p>
      {#if view === 'index'}
        <h1>My Entries</h1>
        <h2>Manage your saved entries.</h2>
      {:else}
        <h1 class='text-ellipsis'>{entryTitle.trim() || 'Untitled entry'}</h1>
        <h2>Write, edit, and save your entry.</h2>
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
    </section>
  {:else if view === 'editor'}
    <section class='panel'>
      <form onsubmit={handleSubmit}>
        <div class='flex justify-between border-bottom mb-2'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>← Back</button>
          <button class='btn btn-primary' type='submit'>Save</button>
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
        {#if alert}
          <Alert type={alert.type} message={alert.message} />
        {/if}
      </form>
    </section>
  {/if}
</div>
