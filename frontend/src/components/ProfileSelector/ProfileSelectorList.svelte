<script lang='ts'>
  import type { main } from '../../../wailsjs/go/models';

  type Props = {
    profiles: main.ProfileEntry[];
    activeProfile: string | null;
    openAddProfile: () => void;
    selectProfile: (uuid: string) => void;
  };

  let {profiles, activeProfile, openAddProfile, selectProfile}: Props = $props();
</script>

<button class='btn btn-primary' onclick={openAddProfile}>Add profile</button>

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
