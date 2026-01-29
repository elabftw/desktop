<script>
	import { onMount, createEventDispatcher } from 'svelte';
	import { GetProfileIndex } from '../wailsjs/go/main/App';

	const dispatch = createEventDispatcher();

	let profiles = [];
	let activeProfile = null;
	let passphrase = '';

	async function loadProfiles() {
		const index = await GetProfileIndex();
		profiles = index?.profiles ?? [];
	}

	function selectProfile(uuid) {
		activeProfile = uuid;
	}

	function unlock() {
		// encryption skipped for now
		dispatch('unlocked', { uuid: activeProfile });
	}

	onMount(loadProfiles);
</script>

<h1>Select a profile</h1>

<div class="profiles">
	{#each profiles as profile (profile.uuid)}
		<div
			class="profile-box"
			class:active={activeProfile === profile.uuid}
			class:masked={activeProfile !== null && activeProfile !== profile.uuid}
			on:click={() => selectProfile(profile.uuid)}
		>
			{profile.display_name || profile.uuid}
		</div>
	{/each}
</div>

{#if activeProfile}
  <div class='input-box' id='input'>
<label>Enter your passphrase</label>
    <input autocomplete='off' placeholder='Your passphrase...' bind:value={passphrase} class='input' id='name' type='password'/>
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
