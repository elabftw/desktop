<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { GetProfileIndex, AddProfile, UnlockProfile, DeleteProfile } from '../wailsjs/go/main/App';

  const dispatch = createEventDispatcher();

  let showAddProfile = false;
  let profiles = [];
  let activeProfile = null;
  let passphrase = '';

  let index = null;

  async function refreshIndex() {
    index = await GetProfileIndex();
    profiles = index?.profiles ?? [];
  }

  let newProfileName = '';
  let newProfilePassphrase = '';
  let addError = '';

  function openAddProfile() {
    newProfileName = '';
    newProfilePassphrase = '';
    addError = '';
    showAddProfile = true;
  }

  function closeAddProfile() {
    showAddProfile = false;
    addError = '';
  }

  async function confirmAddProfile() {
    const name = newProfileName.trim();
    const passphrase = newProfilePassphrase.trim();

    if (!name) {
      addError = 'Please enter a profile name.';
      return;
    }

    if (!passphrase) {
      addError = 'Please enter a passphrase.';
      return;
    }

    try {
      await AddProfile(name, passphrase);
      closeAddProfile();
      await refreshIndex();
    } catch (e) {
      addError = e?.message ?? String(e);
    }
  }

  function onModalKeydown(e) {
    if (e.key === 'Escape') closeAddProfile();
    if (e.key === 'Enter') confirmAddProfile();
  }

  function selectProfile(uuid) {
    activeProfile = uuid;
  }

  async function deleteSelectedProfile() {
    if (!activeProfile) {
      addError = 'Please select a profile to delete.';
      return;
    }

    const ok = confirm('Delete this profile and all local entries? This cannot be undone.');
    if (!ok) return;

    try {
      const index = await DeleteProfile(activeProfile);
      profiles = index?.profiles ?? [];
      activeProfile = null;
      passphrase = '';
    } catch (e) {
      addError = e?.message ?? String(e);
    }
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
      dispatch('unlocked', {uuid: activeProfile});
    } catch (e) {
      addError = e?.message ?? String(e);
    }
  }

  onMount(refreshIndex);
</script>

<h1>Select a profile</h1>

<div class='profiles'>
  {#each profiles as profile (profile.uuid)}
    <button
      class='profile-box'
      class:active={activeProfile === profile.uuid}
      class:masked={activeProfile !== null && activeProfile !== profile.uuid}
      on:click={() => selectProfile(profile.uuid)}
    >
      {profile.display_name || profile.uuid}
    </button>
  {/each}
</div>

<div style='margin-top: 12px;'>
  <button class='btn btn-primary' on:click={openAddProfile}>Add profile</button>
</div>

{#if showAddProfile}
  <div
    class='modal'
    role='dialog'
    aria-modal='true'
    on:keydown={onModalKeydown}
  >
    <div class='modal-title'>Add profile</div>
    <div class='modal-body'>
      <label class='modal-label' for='profileName'>Profile name</label>
      <!-- svelte-ignore a11y-autofocus : not going to have a full js function for every input that needs autofocus... -->
      <input
        autofocus
        id='profileName'
        class='modal-input'
        bind:value={newProfileName}
      />

      <label class='modal-label' for='profilePassphrase'>Passphrase</label>
      <input
        id='profilePassphrase'
        class='modal-input'
        type='password'
        autocomplete='new-password'
        bind:value={newProfilePassphrase}
      />
      {#if addError}
        <div class='modal-error'>{addError}</div>
      {/if}
    </div>

    <div class='modal-actions'>
      <button class='btn btn-primary' on:click={closeAddProfile}>Cancel</button>
      <button class='btn btn-primary' on:click={confirmAddProfile}>Add</button>
    </div>
  </div>
{/if}

{#if activeProfile}
  <div class='input-box' id='input'>
    <label for='passphrase'>Enter your passphrase</label>
      <input
        autocomplete='off'
        placeholder='Your passphrase...'
        bind:value={passphrase}
        class='input'
        id='passphrase'
        type='password'
      />
      <div>
        <button class='btn btn-primary' on:click={unlock}>
          Unlock
        </button>
        <button class='btn btn-danger' on:click={deleteSelectedProfile}>
          Delete profile (dev)
        </button>
      </div>
    </div>
    {#if addError}
      <div class='alert alert-error'>
        <strong>Error:</strong> {addError}
      </div>
    {/if}
{/if}

<style>
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
