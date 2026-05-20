<script>
  import ProfileSelector from './components/ProfileSelector/ProfileSelector.svelte';
  import MainApp from './components/MainApp.svelte';
  import logo from './assets/images/elabftw-logo-white-800px.png';

  let appState = $state('select-profile');
  let activeProfile = $state(null);
  let activeProfileName = $state(null);
</script>

<main>
  {#if appState === 'select-profile'}
    <img alt='eLabFTW logo' id='logo' src={logo} style='width: 200px;' />
    <ProfileSelector
      onUnlocked={(uuid, name) => {
        activeProfile = uuid;
        activeProfileName = name;
        appState = 'unlocked';
      }}
    />
  {:else if appState === 'unlocked'}
    <MainApp
      profileUuid={activeProfile}
      profileName={activeProfileName}
      onLogout={() => {
        activeProfile = null;
        appState = 'select-profile';
      }}
    />
  {/if}
</main>
