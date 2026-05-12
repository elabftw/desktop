<script>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, SaveEntry, GetEntry, LockProfile } from '../../wailsjs/go/main/App.d.ts';

  let { profileUuid, onLogout } = $props();
  let entryTitle = $state('');
  let entryMainText = $state('');
  let status = $state('');
  let entries = $state([]);
  let listStatus = $state('');
  let view = $state('index'); // 'index' | 'editor'
  let addError = $state('');


  function toRelativeTime(iso, locale = 'en') {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id) {
    addError = '';
    try {
      const e = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMainText = e.body;
      view = 'editor';
    } catch (e) {
      addError = e?.message ?? String(e);
    }
  }

  async function refreshEntries() {
    listStatus = 'Loading...';
    try {
      entries = await ListEntries(profileUuid);
      listStatus = '';
    } catch (e) {
      console.error(e);
      listStatus = e?.message ?? String(e);
    }
  }

  async function openIndex() {
    await refreshEntries();
    status = '';
    addError = '';
    view = 'index';
  }

  async function logout() {
    try {
      await LockProfile();
      onLogout?.();
    } catch (e) {
      status = '';
      addError = e?.message ?? String(e);
    }
  }

  function openEditor() {
    view = 'editor';
    entryTitle = '';
    entryMainText = '';
  }

  async function saveEntry() {
    status = 'Saving...';
    addError = '';
    try {
      const id = await SaveEntry(profileUuid, entryTitle, entryMainText);
      status = `Saved with id ${id}`;
      await refreshEntries();
    } catch (e) {
      status = '';
      addError = e?.message ?? String(e);
    }
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
  <form class='container-md' onsubmit={(e) => (e.preventDefault(), saveEntry())}>
    <div class='input-box'>
      <label for='entryTitle'>Entry title</label>
      <input
        id='entryTitle'
        type='text'
        class='input'
        bind:value={entryTitle}
        placeholder='Type something...'
      />
    </div>

    <div class='input-box'>
      <label for='entryMainText'>Entry main text</label>
      <textarea
        id='entryMainText'
        bind:value={entryMainText}
        placeholder='Your main content...'
      ></textarea>
    </div>

    <div class='button-row'>
      <button class='btn btn-secondary' type='button' onclick={openIndex}>Back</button>
      <button class='btn btn-primary' type='submit'>Save</button>
    </div>

    {#if status}
      <div class='alert alert-success'>
        <p>{status}</p>
      </div>
    {/if}
    {#if addError}
      <div class='alert alert-error'>
        <strong>Error:</strong> {addError}
      </div>
    {/if}
  </form>
{/if}

<style>
  ul {
    list-style: none;
    padding: 0;
  }

  .title {
    color: #fab95b;
    background: none;
    border: 0;
    padding: 0;
    font: inherit;
    cursor: pointer;
    text-align: left;
    font-size: 1.5rem;
  }

  .title:hover {
    text-decoration: underline;
  }
</style>
