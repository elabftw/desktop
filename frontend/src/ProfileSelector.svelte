<script>
  import { onMount } from 'svelte';
  import { GetProfileIndex, AddProfile, UnlockProfile, DeleteProfile } from '../wailsjs/go/main/App';

  let showAddProfile = $state(false);
  let profiles = $state([]);
  let activeProfile = $state(null);
  let passphrase = $state('');
  let index = null;
  let newProfileName = $state('');
  let newProfilePassphrase = $state('');
  let addError = $state('');
  let {onUnlocked} = $props();

  async function refreshIndex() {
    addError = '';
    try {
      index = await GetProfileIndex();
      profiles = index?.profiles ?? [];
    } catch (e) {
      profiles = [];
      addError = e?.message ?? String(e);
    }
  }

  function openAddProfile() {
    newProfileName = '';
    newProfilePassphrase = '';
    addError = '';
    showAddProfile = true;
    activeProfile = null;
  }

  function closeAddProfile() {
    showAddProfile = false;
    addError = '';
  }

  async function confirmAddProfile() {
    const name = newProfileName.trim();
    const passphrase = newProfilePassphrase;

    if (!name) {
      addError = 'Please enter a profile name.';
      return;
    }

    if (!passphrase.trim()) {
      addError = 'Please enter a passphrase.';
      return;
    }

    try {
      await AddProfile(name, passphrase);
      await refreshIndex();
      closeAddProfile();
    } catch (e) {
      addError = e?.message ?? String(e);
    }
  }

  function selectProfile(uuid) {
    showAddProfile = false;
    addError = '';
    activeProfile = uuid;
  }

  async function deleteSelectedProfile() {
    if (!activeProfile) {
      addError = 'Please select a profile to delete.';
      return;
    }

    if (!passphrase.trim()) {
      addError = 'Please enter the profile passphrase before deleting.';
      return;
    }

    const ok = confirm('Delete this profile and all local entries? This cannot be undone.');
    if (!ok) return;

    try {
      const index = await DeleteProfile(activeProfile, passphrase);
      profiles = index?.profiles ?? [];
      clearProfileSelection();
    } catch (e) {
      console.error('Delete failed:', e);
      addError = e?.message || e?.toString?.() || String(e);
    }
  }

  function clearProfileSelection() {
    activeProfile = null;
    passphrase = '';
    addError = '';
  }


  async function unlock() {
    if (!activeProfile) {
      addError = 'Please select a profile.';
      return;
    }

    if (!passphrase.trim()) {
      addError = 'Please enter your passphrase.';
      return;
    }

    try {
      await UnlockProfile(activeProfile, passphrase);
      onUnlocked?.(activeProfile);
    } catch (e) {
      addError = e?.message ?? String(e);
    }
  }

  onMount(refreshIndex);
</script>

<div class='container'>
  <h1>Select a profile</h1>
  <div class='profiles'>
    {#each profiles as profile (profile.uuid)}
      <button
        class='profile-box'
        class:active={activeProfile === profile.uuid}
        class:masked={activeProfile !== null && activeProfile !== profile.uuid}
        onclick={() => selectProfile(profile.uuid)}
      >
        {profile.display_name || profile.uuid}
      </button>
    {/each}
  </div>

  <button class='btn btn-primary' onclick={openAddProfile}>Add profile</button>

  {#if showAddProfile}
    <div class='container-sm'>
      <label for='profileName'>Profile name</label>
      <!-- svelte-ignore a11y_autofocus : not going to have a full js function for every input that needs autofocus... -->
      <input autofocus placeholder='your profile name...' class='input' id='profileName' bind:value={newProfileName}/>

      <label for='profilePassphrase'>Passphrase</label>
      <input
        id='profilePassphrase'
        class='input'
        placeholder='your passphrase...'
        type='password'
        autocomplete='new-password'
        bind:value={newProfilePassphrase}
      />
      {#if addError}
        <div class='alert alert-error'>
          <strong>Error:</strong> {addError}
        </div>
      {/if}
      <div class='button-row'>
        <button class='btn btn-secondary' onclick={closeAddProfile}>Cancel</button>
        <button class='btn btn-primary' onclick={confirmAddProfile}>Add</button>
      </div>
    </div>
  {/if}

  {#if activeProfile}
    <div class='container-sm'>
      <label for='passphrase'>Enter your passphrase</label>
      <input
        autocomplete='off'
        placeholder='Your passphrase...'
        bind:value={passphrase}
        class='input'
        id='passphrase'
        type='password'
      />
      <div class='button-row'>
        <button class='btn btn-secondary' onclick={clearProfileSelection}>Cancel</button>
        <button class='btn btn-primary' onclick={unlock}>Unlock</button>
      </div>
      <br>
      <button class='btn btn-danger' onclick={deleteSelectedProfile}>Delete profile (dev)</button>

      {#if addError}
        <div class='alert alert-error'>
          <strong>Error:</strong> {addError}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .profiles {
    margin-top: 12px;
  }

  .profile-box {
    cursor: pointer;
    width: 20vw;
    margin: 10px auto;
    padding: 0.5rem;
    border: 1px solid white;
  }

  .profile-box.active {
    background: white;
    color: black;
  }

  .profile-box.masked {
    opacity: 0.25;
    filter: blur(1px);
  }

  .profile-box:hover {
    background-color: white;
    color: black;
    cursor: pointer;
  }
</style>
