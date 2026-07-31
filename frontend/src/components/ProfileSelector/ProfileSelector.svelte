<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { onMount } from 'svelte';
  import {
    GetProfileIndex,
    AddProfile,
    UnlockProfile,
    DeleteProfile
  } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { showAlert } from "../stores/alert.svelte";

  import ProfileSelectorList from './ProfileSelectorList.svelte';
  import ProfileSelectorCreateForm from './ProfileSelectorCreateForm.svelte';

  type Props = {
    onUnlocked?: (profileUuid: string, profileName: string) => void;
  };

  let showAddProfile = $state(false);
  let profiles = $state<main.ProfileEntry[]>([]);
  let activeProfile = $state<string | null>(null);
  let activeProfileName = $state<string | null>(null);
  let index = $state<main.ProfileIndex | null>(null);

  let {onUnlocked}: Props = $props();

  async function refreshIndex(): Promise<void> {
    showAlert(null);

    try {
      index = await GetProfileIndex();
      profiles = index?.profiles ?? [];
    } catch (e: unknown) {
      profiles = [];
      showAlert({type: 'error', message: String(e)});
    }
  }

  function openAddProfile(): void {
    showAlert(null);
    showAddProfile = true;
    activeProfile = null;
  }

  function closeAddProfile(): void {
    showAddProfile = false;
    showAlert(null);
  }

  async function confirmAddProfile(name: string, passphrase: string): Promise<void> {
    name = name.trim();

    if (!name) {
      showAlert({type: 'error', message: String('Please enter a profile name.')});
      return;
    }

    if (!passphrase.trim()) {
      showAlert({type: 'error', message: String('Please enter a passphrase.')});
      return;
    }

    try {
      await AddProfile(name, passphrase);
      await refreshIndex();
      closeAddProfile();
    } catch (e: unknown) {
      showAlert({type: 'error', message: String(e)});
    }
  }

  function selectProfile(uuid: string, name: string): void {
    showAddProfile = false;
    showAlert(null);
    activeProfile = uuid;
    activeProfileName = name;
  }

  async function deleteSelectedProfile(passphrase: string): Promise<void> {
    if (!activeProfile) {
      showAlert({type: 'error', message: String('Please select a profile to delete.')});
      return;
    }

    if (!passphrase.trim()) {
      showAlert({type: 'error', message: String('Please enter the profile passphrase before deleting.')});
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
      showAlert({type: 'error', message: String(e)});
    }
  }

  function clearProfileSelection(): void {
    activeProfile = null;
    showAlert(null);
  }

  async function unlock(passphrase: string): Promise<void> {
    if (!activeProfile) {
      showAlert({type: 'error', message: 'Please select a profile.'});
      return;
    }

    if (!passphrase.trim()) {
      showAlert({type: 'error', message: 'Please enter your passphrase.'});
      return;
    }

    try {
      await UnlockProfile(activeProfile, passphrase);
      onUnlocked?.(activeProfile, activeProfileName);
    } catch (e: unknown) {
      showAlert({type: 'error', message: String(e)});
    }
  }

  onMount(() => {
    void refreshIndex();
  });
</script>

<div class='container'>
  {#if !showAddProfile}
    <ProfileSelectorList
      {profiles}
      {activeProfile}
      {openAddProfile}
      {selectProfile}
      {clearProfileSelection}
      {unlock}
      {deleteSelectedProfile}
    />
  {/if}

  {#if showAddProfile}
    <ProfileSelectorCreateForm
      {closeAddProfile}
      {confirmAddProfile}
    />
  {/if}
</div>
