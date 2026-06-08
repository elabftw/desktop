<!--
This file is part of eLabFTW Desktop.

@author Nicolas <Deltablot>
@author Moustapha <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import type { main } from '../../../wailsjs/go/models';
  import ProfileSelectorUnlockForm from './ProfileSelectorUnlockForm.svelte';

  type Props = {
    profiles: main.ProfileEntry[];
    activeProfile: string | null;
    addError: string;
    openAddProfile: () => void;
    selectProfile: (uuid: string, name: string) => void;
    clearProfileSelection: () => void;
    unlock: (passphrase: string) => void | Promise<void>;
    deleteSelectedProfile: (passphrase: string) => void | Promise<void>;
  };

  let {
    profiles,
    activeProfile,
    addError,
    openAddProfile,
    selectProfile,
    clearProfileSelection,
    unlock,
    deleteSelectedProfile
  }: Props = $props();

  function initials(profile: main.ProfileEntry): string {
    return profile.display_name?.trim().slice(0, 2).toUpperCase();
  }
</script>

<section>
  <p class='header-subtitle'>Welcome back</p>
  <header class='flex justify-between items-center mb-2'>
    <div>
      <h1>Select a profile</h1>
      <h2>Choose a profile to unlock your saved entries.</h2>
    </div>
    <button class='btn btn-primary flex-auto' onclick={openAddProfile}>
      <span aria-hidden='true'>+</span>Add profile
    </button>
  </header>

  <section class='panel' aria-labelledby='profiles-title'>
    <div class='flex justify-between items-center border-bottom mb-2'>
      <div class='flex items-center gap-1'>
        <div class='icon' aria-hidden='true'>●</div>
        <div>
          <h3 id='profiles-title'>Profiles</h3>
          <p class='description'>{profiles.length === 1 ? '1 profile available' : `${profiles.length} profiles available`}</p>
        </div>
      </div>
    </div>

    {#if profiles.length > 0}
      <div class='grid gap-1'>
        {#each profiles as profile (profile.uuid)}
          <button
            type='button'
            class='entry-card'
            class:active={activeProfile === profile.uuid}
            class:masked={activeProfile !== null && activeProfile !== profile.uuid}
            onclick={() => selectProfile(profile.uuid, profile.display_name)}
            aria-pressed={activeProfile === profile.uuid}
          >
            <span class='profile-avatar'>{initials(profile)}</span>
            <span class='grid gap-03'>
              <span class='text-strong text-white text-ellipsis text-big'>{profile.display_name?.trim() || profile.uuid}</span>
              <span class='text-gray'>{activeProfile === profile.uuid ? 'Selected' : 'Click to unlock'}</span>
            </span>

            <span class='text-orange text-strong' aria-hidden='true'>
              {activeProfile === profile.uuid ? 'Unlock ↓' : 'Select →'}
            </span>
          </button>
          {#if activeProfile === profile.uuid}
            <div class='profile-inline-unlock'>
              <ProfileSelectorUnlockForm
                {addError}
                {clearProfileSelection}
                {unlock}
                {deleteSelectedProfile}
              />
            </div>
          {/if}
        {/each}
      </div>
    {:else}
      <div class='empty-state'>
        <div class='icon-sm' aria-hidden='true'>+</div>
        <h3>No profiles yet</h3>
        <p>Create a profile to start saving entries.</p>
        <button class='btn btn-primary' onclick={openAddProfile}>
          Create your first profile
        </button>
      </div>
    {/if}
  </section>
</section>
