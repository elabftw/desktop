<script>
import { onMount } from 'svelte';
import ProfileSelector from './ProfileSelector.svelte';
	import MainApp from './MainApp.svelte';
  import logo from './assets/images/elabftw-logo-white-800px.png'
  import {Login, GetProfileIndex, GetHash} from '../wailsjs/go/main/App.js'

let appState = 'select-profile';
  let passphrase;
let profiles = [];
  let activeProfile = null;

  async function login() {
    await Login(passphrase);
console.log(GetHash());

  }
function selectProfile(uuid) {
		activeProfile = uuid;
	}
async function getProfiles() {
  const index = await GetProfileIndex();
  profiles = index?.profiles ?? [];
  console.log(index);
}
function unlock() {
	appState = 'unlocked';
}
	onMount(() => {
		getProfiles();
	});
</script>
<main>
{#if appState === 'select-profile'}
  <img alt='eLabFTW logo' id='logo' src='{logo}'>
	<ProfileSelector
		on:unlocked={(e) => {
			activeProfile = e.detail.uuid;
			appState = 'unlocked';
		}}
	/>
{:else if appState === 'unlocked'}
	<MainApp profileUuid={activeProfile} />
{/if}

</main>
