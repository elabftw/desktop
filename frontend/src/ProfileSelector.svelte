<script>
  import { onMount, createEventDispatcher } from 'svelte';
  import { GetProfileIndex, AddProfile, UnlockProfile } from '../wailsjs/go/main/App';

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
  let addError = '';

  function openAddProfile() {
    newProfileName = '';
    addError = '';
    showAddProfile = true;
  }

  function closeAddProfile() {
    showAddProfile = false;
    addError = '';
  }

  async function confirmAddProfile() {
    const name = newProfileName.trim();
    if (!name) {
      addError = 'Please enter a profile name.';
      return;
    }

    try {
      // If your AddProfile returns updated index, assign it to your state here
      await AddProfile(name);

      closeAddProfile();

      // Refresh your index/list if needed:
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

  async function unlock() {
    // dispatch('unlocked', {uuid: activeProfile});
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
    } catch(e) {
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
  <button on:click={openAddProfile}>Add profile</button>
</div>

{#if showAddProfile}
  <!-- Backdrop -->
  <button class='modal-backdrop' on:click={closeAddProfile}>x</button>

  <!-- Modal -->
  <div
    class='modal'
    role='dialog'
    aria-modal='true'
    on:keydown={onModalKeydown}
  >
    <div class='modal-title'>Add profile</div>
    <div class='modal-body'>
      <label class='modal-label' for='profileName'>Profile name</label>
      <input
        id='profileName'
        class='modal-input'
        bind:value={newProfileName}
      />

      {#if addError}
        <div class='modal-error'>{addError}</div>
      {/if}
    </div>

    <div class='modal-actions'>
      <button on:click={closeAddProfile}>Cancel</button>
      <button on:click={confirmAddProfile}>Add</button>
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
    <button class='btn' on:click={unlock}>Unlock</button>
  </div>
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
