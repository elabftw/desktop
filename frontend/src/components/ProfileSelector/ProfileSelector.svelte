<script>
  import { onMount } from 'svelte';
  import { GetProfileIndex, AddProfile, UnlockProfile, DeleteProfile } from '../../../wailsjs/go/main/App';
  import ProfileSelectorList from './ProfileSelectorList.svelte';
  import ProfileSelectorCreateForm from './ProfileSelectorCreateForm.svelte';
  import ProfileSelectorUnlockForm from './ProfileSelectorUnlockForm.svelte';

  let showAddProfile = $state(false);
  let profiles = $state([]);
  let activeProfile = $state(null);
  let index = null;
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
    addError = '';
    showAddProfile = true;
    activeProfile = null;
  }

  function closeAddProfile() {
    showAddProfile = false;
    addError = '';
  }

  async function confirmAddProfile(name, passphrase) {
    name = name.trim();
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

  async function deleteSelectedProfile(passphrase) {
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
    addError = '';
  }

  async function unlock(passphrase) {
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
  {#if !showAddProfile}
    <ProfileSelectorList {profiles} {activeProfile} {openAddProfile} {selectProfile}/>
  {/if}

  {#if showAddProfile}
    <ProfileSelectorCreateForm {addError} {closeAddProfile} {confirmAddProfile}/>
  {/if}

  {#if activeProfile}
    <ProfileSelectorUnlockForm {addError} {clearProfileSelection} {unlock} {deleteSelectedProfile}/>
  {/if}
</div>
