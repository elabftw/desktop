<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, SaveEntry, GetEntry, LockProfile } from '../wailsjs/go/main/App';

  const dispatch = createEventDispatcher();

  export let profileUuid;
  let entryTitle = '';
  let entryMaintext = '';
  let status = '';
  let entries = [];
  let listStatus = '';
  let view = 'index'; // 'index' | 'editor'
  let addError = '';


  function toRelativeTime(iso, locale = 'en') {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id) {
    status = '';
    addError = '';
    try {
      const e = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMaintext = e.body;
      view = 'editor';
      status = `Loaded entry ${e.id}`;
    } catch (e) {
      status = '';
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
    view = 'index';
  }

  async function logout() {
    await LockProfile();
    dispatch('logout');
  }

  function openEditor() {
    view = 'editor';
    entryTitle = '';
    entryMaintext = '';
  }

  async function saveEntry() {
    status = 'Saving...';
    addError = '';
    try {
      const id = await SaveEntry(profileUuid, entryTitle, entryMaintext);
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
<button class='btn btn-danger' on:click={logout}>Logout</button>

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
            on:click|preventDefault={() => openEntry(e.id)}
          >
            {e.title}
          </button>
          <div>Last modification: {toRelativeTime(e.updatedAt)}</div>
        </li>
      {/each}
    </ul>
  {/if}

  <button class='btn btn-primary' on:click={openEditor}>Create entry</button>

{:else if view === 'editor'}
  <div class='container-md'>
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
      <label for='entryMaintext'>Entry main text</label>
      <textarea
        id='entryMaintext'
        bind:value={entryMaintext}
        placeholder='Your main content...'
      ></textarea>
    </div>

    <div class='button-row'>
      <button class='btn btn-secondary' on:click={openIndex}>Back</button>
      <button class='btn btn-primary' on:click={saveEntry}>Save</button>
    </div>

    {#if status}
      <p>{status}</p>
    {/if}
    {#if addError}
      <div class='alert alert-error'>
        <strong>Error:</strong> {addError}
      </div>
    {/if}
  </div>
{/if}

<style>
  ul {
    list-style: none;
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
