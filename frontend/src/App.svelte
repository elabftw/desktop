<script>
  import ProfileSelector from './ProfileSelector.svelte';
  import MainApp from './MainApp.svelte';
  import logo from './assets/images/elabftw-logo-white-800px.png';

  let appState = $state('select-profile');
  let activeProfile = $state(null);
</script>

<main>
  {#if appState === 'select-profile'}
    <img alt='eLabFTW logo' id='logo' src={logo} style='width: 200px;' />
    <ProfileSelector
      onUnlocked={(uuid) => {
        activeProfile = uuid;
        appState = 'unlocked';
      }}
    />
  {:else if appState === 'unlocked'}
    <MainApp
      profileUuid={activeProfile}
      onLogout={() => {
        activeProfile = null;
        appState = 'select-profile';
      }}
    />
  {/if}
</main>
