<script>
  let { profiles, activeProfile, onAddProfile, onSelectProfile } = $props();
</script>

<button class='btn btn-primary' onclick={onAddProfile}>Add profile</button>
<h1>Select a profile</h1>
<div class='profiles'>
  {#each profiles as profile (profile.uuid)}
    <button
      class='profile-box'
      class:active={activeProfile === profile.uuid}
      class:masked={activeProfile !== null && activeProfile !== profile.uuid}
      onclick={() => onSelectProfile(profile.uuid)}
    >
      {profile.display_name || profile.uuid}
    </button>
  {/each}
</div>
