<script>
  import { createEventDispatcher, onMount } from 'svelte';
  import { DateTime } from 'luxon';
  import { ListEntries, SaveEntry, GetEntry } from '../wailsjs/go/main/App';

  const dispatch = createEventDispatcher();

  export let profileUuid;
  let entryTitle = '';
  let entryMaintext = '';
  let status = '';
  let entries = [];
  let listStatus = '';
  let view = 'index'; // 'index' | 'editor'

  function toRelativeTime(iso, locale = 'en') {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id) {
    status = '';
    try {
      const e = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMaintext = e.body;
      view = 'editor';
      status = `Loaded entry ${e.id}`;
    } catch (e) {
      console.error(e);
      status = e?.message ?? String(e);
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

  function openIndex() {
    view = 'index';
  }

  function logout() {
    dispatch('logout');
  }

  function openEditor() {
    view = 'editor';
    entryTitle = '';
    entryMaintext = '';
  }

  async function saveEntry() {
    status = 'Saving...';
    try {
      const id = await SaveEntry(profileUuid, entryTitle, entryMaintext);
      status = `Saved with id ${id}`;
    } catch (e) {
      status = e?.message ?? String(e);
    }
  }

  onMount(refreshEntries);
</script>

<h1>Index</h1>
<button on:click={refreshEntries}> Refresh</button>

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

  <button on:click={openEditor}>Create entry</button>
  <button on:click={logout}>Logout</button>

{:else if view === 'editor'}
  <div>
    <label for='entryTitle'>Entry title</label>
    <input
      id='entryTitle'
      type='text'
      bind:value={entryTitle}
      placeholder='Type something...'
    />
  </div>

  <div>
    <label for='entryMaintext'>Entry main text</label>
    <textarea
      id='entryMaintext'
      bind:value={entryMaintext}
      placeholder='Your main content...'
    ></textarea>
  </div>

  <button on:click={saveEntry}>Save</button>
  <button on:click={openIndex}>Back</button>

  {#if status}
    <p>{status}</p>
  {/if}
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

  input,
  button {
    padding: 0.5rem 0.75rem;
  }
</style>
