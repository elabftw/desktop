<script lang='ts'>
  import { onMount } from 'svelte';
  import {
    GetProfileIndex,
    AddProfile,
    UnlockProfile,
    DeleteProfile, ForceDeleteProfile
  } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage } from "../../utils/helpers";

  import ProfileSelectorList from './ProfileSelectorList.svelte';
  import ProfileSelectorCreateForm from './ProfileSelectorCreateForm.svelte';
  import ProfileSelectorUnlockForm from './ProfileSelectorUnlockForm.svelte';

  type Props = {
    onUnlocked?: (profileUuid: string) => void;
  };

  let showAddProfile = $state(false);
  let profiles = $state<main.ProfileEntry[]>([]);
  let activeProfile = $state<string | null>(null);
  let index = $state<main.ProfileIndex | null>(null);
  let addError = $state('');

  let {onUnlocked}: Props = $props();

  async function refreshIndex(): Promise<void> {
    addError = '';
    try {
      index = await GetProfileIndex();
      profiles = index?.profiles ?? [];
    } catch (e: unknown) {
      profiles = [];
      addError = errorMessage(e);
    }
  }

  function openAddProfile(): void {
    addError = '';
    showAddProfile = true;
    activeProfile = null;
  }

  function closeAddProfile(): void {
    showAddProfile = false;
    addError = '';
  }

  async function confirmAddProfile(name: string, passphrase: string): Promise<void> {
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
    } catch (e: unknown) {
      addError = errorMessage(e);
    }
  }

  function selectProfile(uuid: string): void {
    showAddProfile = false;
    addError = '';
    activeProfile = uuid;
  }

  async function forceDeleteProfile() {
    const ok = confirm('Force delete this profile and all local entries? This cannot be undone.');
    if (!ok) return;

    try {
      index = await ForceDeleteProfile(activeProfile)
      profiles = index?.profiles ?? [];
      clearProfileSelection();
    } catch (err) {
      console.error('Force delete failed:', err)
      throw err
    }
  }

  async function deleteSelectedProfile(passphrase: string): Promise<void> {
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
      index = await DeleteProfile(activeProfile, passphrase);
      profiles = index?.profiles ?? [];
      clearProfileSelection();
    } catch (e: unknown) {
      console.error('Delete failed:', e);
      addError = errorMessage(e);
    }
  }

  function clearProfileSelection(): void {
    activeProfile = null;
    addError = '';
  }

  async function unlock(passphrase: string): Promise<void> {
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
    } catch (e: unknown) {
      addError = errorMessage(e);
    }
  }

  onMount(() => {
    void refreshIndex();
  });
</script>

<div class='container'>
  {#if !showAddProfile}
    <ProfileSelectorList {profiles} {activeProfile} {openAddProfile} {selectProfile}/>
  {/if}

  {#if showAddProfile}
    <ProfileSelectorCreateForm {addError} {closeAddProfile} {confirmAddProfile}/>
  {/if}

  {#if activeProfile}
    <ProfileSelectorUnlockForm
      {addError}
      {clearProfileSelection}
      {unlock}
      {deleteSelectedProfile}
      {forceDeleteProfile}
    />
  {/if}
</div>
