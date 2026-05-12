<script>
  let {addError, closeAddProfile, confirmAddProfile} = $props();
  let name = $state('');
  let passphrase = $state('');
</script>

<form class='container-sm' onsubmit={(e) => (e.preventDefault(), confirmAddProfile(name, passphrase))}>
  <label for='profileName'>Profile name</label>
  <!-- svelte-ignore a11y_autofocus : todo implement autoFocusOnLoad -->
  <input autofocus placeholder='Your username' class='input' id='profileName' bind:value={name}/>
  <label for='passphrase'>Passphrase</label>
  <input
    id='passphrase'
    class='input'
    placeholder='Your passphrase'
    type='password'
    autocomplete='new-password'
    bind:value={passphrase}
  />
  {#if addError}
    <div class='alert alert-error'>
      <strong>addError:</strong> {addError}
    </div>
  {/if}
  <div class='button-row'>
    <button type='button' class='btn btn-secondary' onclick={closeAddProfile}>Cancel</button>
    <button type='submit' class='btn btn-primary'>Add</button>
  </div>
</form>
