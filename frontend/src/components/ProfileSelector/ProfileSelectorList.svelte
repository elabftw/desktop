<script>
  import AlertError from '../AlertError.svelte';

  let {addError, profiles, activeProfile, openAddProfile, selectProfile} = $props();
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
<AlertError message={addError}/>
