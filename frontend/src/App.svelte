<script>
  import ProfileSelector from './ProfileSelector.svelte';
  import MainApp from './MainApp.svelte';
  import logo from './assets/images/elabftw-logo-white-800px.png';

  let appState = 'select-profile';
  let activeProfile = null;
</script>

<main>
  {#if appState === 'select-profile'}
    <img alt='eLabFTW logo' id='logo' src={logo} style='width: 200px;' />
    <ProfileSelector
      on:unlocked={(e) => {
        activeProfile = e.detail.uuid;
        appState = 'unlocked';
      }}
    />
  {:else if appState === 'unlocked'}
    <MainApp profileUuid={activeProfile}/>
  {/if}
</main>
