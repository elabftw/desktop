<script lang='ts'>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, SaveEntry, GetEntry, LockProfile } from '../../wailsjs/go/main/App';
  import type { main } from '../../wailsjs/go/models';
  import { autofocus, errorMessage, preventDefaultSubmit } from '../utils/helpers';
  import Alert from './Alert.svelte';

  type Props = {
    profileUuid: string;
    onLogout?: () => void;
  };

  type View = 'index' | 'editor';

  let {profileUuid, onLogout}: Props = $props();

  let entryTitle = $state('');
  let entryMainText = $state('');
  let status = $state('');
  let entries = $state<main.EntrySummary[]>([]);
  let listStatus = $state('');
  let view = $state<View>('index');
  let addError = $state('');

  function toRelativeTime(iso: string, locale = 'en'): string {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id: number): Promise<void> {
    addError = '';
    status = '';

    try {
      const e: main.Entry = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMainText = e.body;
      view = 'editor';
    } catch (e: unknown) {
      addError = errorMessage(e);
    }
  }

  async function refreshEntries(): Promise<void> {
    listStatus = 'Loading entries...';
    try {
      entries = await ListEntries(profileUuid);
      listStatus = '';
    } catch (e: unknown) {
      console.error(e);
      listStatus = errorMessage(e);
    }
  }

  async function openIndex(): Promise<void> {
    await refreshEntries();
    status = '';
    addError = '';
    view = 'index';
  }

  async function logout(): Promise<void> {
    try {
      await LockProfile();
      onLogout?.();
    } catch (e: unknown) {
      status = '';
      addError = errorMessage(e);
    }
  }

  function openEditor(): void {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
    status = '';
    addError = '';
  }

  async function saveEntry(): Promise<void> {
    status = 'Saving...';
    addError = '';

    try {
      const id = await SaveEntry(profileUuid, entryTitle, entryMainText);
      status = `Saved with id ${id}`;
      await refreshEntries();
    } catch (e: unknown) {
      status = '';
      addError = errorMessage(e);
    }
  }

  const handleSubmit = preventDefaultSubmit(saveEntry);

  onMount(() => {
    void refreshEntries();
  });
</script>

<div class='app-shell'>
  <header class='app-header'>
    <div>
      <p class='eyebrow'>Unlocked profile</p>

      {#if view === 'index'}
        <h1>My Entries</h1>
        <p class='app-subtitle'>Manage your saved entries.</p>
      {:else}
        <h1>{entryTitle.trim() || 'Untitled entry'}</h1>
        <p class='app-subtitle'>Write, edit, and save your entry.</p>
      {/if}
    </div>

    <div class='account-bar'>
      <div class='profile-pill' title={profileUuid}>
        <span class='profile-avatar-small'>{profileUuid.slice(0, 2).toUpperCase()}</span>
        <span class='profile-pill-text'>
          <span class='profile-name-small'>Profile</span>
          <span class='profile-id-small'>{profileUuid.slice(0, 8)}...</span>
        </span>
      </div>

      <button class='btn btn-danger btn-logout' onclick={logout}>
        Logout
      </button>
    </div>
  </header>

  {#if addError}
    <div class='app-alert'>
      <Alert type='error' message={addError}/>
    </div>
  {/if}

  {#if view === 'index'}
    <section class='entries-panel' aria-labelledby='entries-title'>
      <div class='entries-panel-header'>
        <div class='entries-title-block'>
          <div class='entries-icon' aria-hidden='true'>▣</div>
          <div>
            <h2 id='entries-title'>Saved entries</h2>
            <p>
              {entries.length === 1 ? '1 entry' : `${entries.length} entries`}
            </p>
          </div>
        </div>

        <button class='btn btn-primary' onclick={openEditor}>
          <span aria-hidden='true'>+</span>
          Create entry
        </button>
      </div>

      {#if listStatus}
        <div class='loading-state'>
          {listStatus}
        </div>
      {:else if entries.length === 0}
        <div class='empty-state'>
          <div class='empty-icon' aria-hidden='true'>✎</div>
          <h3>No entries yet</h3>
          <p>Create your first entry to start writing.</p>

          <button class='btn btn-primary' onclick={openEditor}>
            Create entry
          </button>
        </div>
      {:else}
        <div class='entry-list'>
          {#each entries as e (e.id)}
            <button
              type='button'
              class='entry-card'
              onclick={() => openEntry(e.id)}
            >
              <span class='entry-card-icon' aria-hidden='true'>▤</span>

              <span class='entry-card-content'>
                <span class='entry-card-title'>
                  {e.title || 'Untitled entry'}
                </span>

                <span class='entry-card-meta'>
                  Last edited {toRelativeTime(e.updatedAt)}
                </span>
              </span>

              <span class='entry-card-action' aria-hidden='true'>
                Open &#8594;
              </span>
            </button>
          {/each}
        </div>
      {/if}
    </section>
  {:else if view === 'editor'}
    <section class='editor-panel'>
      <form class='editor-form' onsubmit={handleSubmit}>
        <div class='editor-toolbar'>
          <button class='btn btn-secondary' type='button' onclick={openIndex}>
            ← Back
          </button>

          <button class='btn btn-primary' type='submit'>
            Save
          </button>
        </div>

        <div class='input-box'>
          <label for='entryTitle'>Entry title</label>
          <input
            use:autofocus
            id='entryTitle'
            type='text'
            class='input entry-title-input'
            bind:value={entryTitle}
            placeholder='Title of your entry'
          />
        </div>

        <div class='input-box'>
          <label for='entryMainText'>Entry main text</label>
          <textarea
            id='entryMainText'
            class='entry-textarea'
            bind:value={entryMainText}
            placeholder='The main text...'
          ></textarea>
        </div>

        {#if status}
          <Alert type='success' message={status}/>
        {/if}

        <Alert type='error' message={addError}/>
      </form>
    </section>
  {/if}
</div>
