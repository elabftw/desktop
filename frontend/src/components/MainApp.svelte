<script lang='ts'>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, SaveEntry, GetEntry, LockProfile } from '../../wailsjs/go/main/App';
  import { autofocus } from '../utils/autofocus';
  import type { main } from '../../wailsjs/go/models';
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

  function errorMessage(e: unknown): string {
    return e instanceof Error ? e.message : String(e);
  }

  function toRelativeTime(iso: string, locale = 'en'): string {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id: number): Promise<void> {
    addError = '';
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
    listStatus = 'Loading...';
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

  function handleSubmit(e: SubmitEvent): void {
    e.preventDefault();
    void saveEntry();
  }

  onMount(refreshEntries);
</script>

<h1>Index</h1>
<button class='btn btn-danger' onclick={logout}>Logout</button>

<p>
  Profile unlocked: {profileUuid}
</p>

{#if view === 'index'}
  <h2>Saved entries</h2>

  {#if listStatus}
    <p>{listStatus}</p>
  {:else if entries.length === 0}
    <p>No entries yet</p>
  {:else}
    <ul>
      {#each entries as e (e.id)}
        <li>
          <button
            type='button'
            class='title'
            onclick={() => openEntry(e.id)}
          >
            {e.title}
          </button>
          <div>Last modification: {toRelativeTime(e.updatedAt)}</div>
        </li>
      {/each}
    </ul>
  {/if}

  <button class='btn btn-primary' onclick={openEditor}>Create entry</button>

{:else if view === 'editor'}
  <form class='container-md' onsubmit={handleSubmit}>
    <div class='input-box'>
      <label for='entryTitle'>Entry title</label>
      <input
        use:autofocus
        id='entryTitle'
        type='text'
        class='input'
        bind:value={entryTitle}
        placeholder='Title of your entry'
      />
    </div>

    <div class='input-box'>
      <label for='entryMainText'>Entry main text</label>
      <textarea
        id='entryMainText'
        bind:value={entryMainText}
        placeholder='The main text...'
      ></textarea>
    </div>

    <div class='button-row'>
      <button class='btn btn-secondary' type='button' onclick={openIndex}>Back</button>
      <button class='btn btn-primary' type='submit'>Save</button>
    </div>

    {#if status}
      <Alert type='success' message={status}/>
    {/if}
    <Alert type='error' message={addError}/>
  </form>
{/if}
