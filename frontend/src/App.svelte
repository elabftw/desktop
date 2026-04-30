<script>
  import { onMount } from 'svelte';
  import ProfileSelector from './ProfileSelector.svelte';
  import MainApp from './MainApp.svelte';
  import logo from './assets/images/elabftw-logo-white-800px.png';
  import { Login, GetProfileIndex } from '../wailsjs/go/main/App.js';

  let appState = 'select-profile';
  let passphrase;
  // let profiles = [];
  let activeProfile = null;

  async function login() {
    passphrase = '';
    await Login(passphrase);
  }
  function selectProfile(uuid) {
    activeProfile = uuid;
  }
  async function getProfiles() {
    const index = await GetProfileIndex();
    // profiles = index?.profiles ?? [];
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
    <img alt="eLabFTW logo" id="logo" src={logo} />
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
