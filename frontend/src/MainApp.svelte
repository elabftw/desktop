<script>
  import { onMount } from 'svelte';
  import { DateTime } from 'luxon';

  import { ListEntries, SaveEntry, GetEntry } from '../wailsjs/go/main/App';
  export let profileUuid;
  let showEditor = false;
  let entryTitle = '';
  let entryMaintext = '';
  let status = '';
  let entries = [];
  let listStatus = '';

  function toRelativeTime(iso, locale = 'en') {
    return DateTime.fromISO(iso).setLocale(locale).toRelative() ?? 'now';
  }

  async function openEntry(id) {
    status = '';
    try {
      const e = await GetEntry(profileUuid, id);
      entryTitle = e.title;
      entryMaintext = e.body;
      showEditor = true;
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

  function createEntry() {
    showEditor = true;
    showIndex = false;
  }
  function index() {
    showEditor = false;
    showIndex = true;
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
  onMount(() => {
    refreshEntries();
  });
</script>

<h1>Index</h1>
<button on:click={refreshEntries}> Refresh </button>

<p>
  Profile unlocked: {profileUuid}
</p>

<h2>Saved entries</h2>

{#if listStatus}
  <p>{listStatus}</p>
{:else if entries.length === 0}
  <p>No entries yet</p>
{:else}
  <ul>
    {#each entries as e (e.id)}
      <li>
        <a
          class="title"
          href="#"
          on:click|preventDefault={() => openEntry(e.id)}
        >
          {e.title}
        </a>
        <div>Last modification: {toRelativeTime(e.updatedAt)}</div>
      </li>
    {/each}
  </ul>
{/if}

{#if !showEditor}
  <button on:click={createEntry}> Create entry </button>
{:else}
  <div>
    <label for="entryTitle">Entry title</label>
    <input
      id="entryTitle"
      type="text"
      bind:value={entryTitle}
      placeholder="Type something..."
    />
  </div>
  <div>
    <label for="entryMaintext">Entry main text</label>
    <textarea
      id="entryMaintext"
      bind:value={entryMaintext}
      placeholder="Your main content..."
    ></textarea>
  </div>
  <button on:click={saveEntry}> Save </button>
  {#if status}
    <p>{status}</p>
  {/if}
{/if}

<style>
  ul {
    list-style: none;
  }
  a {
    text-decoration: none;
    color: #fab95b;
  }
  a:hover {
    text-decoration: underline;
  }
  .title {
    font-size: 1.5rem;
  }
  input,
  button {
    padding: 0.5rem 0.75rem;
  }
</style>
