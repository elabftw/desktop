<script>
  import { autofocus } from '../../utils/autofocus';
  import Alert from '../Alert.svelte';

  let {addError, closeAddProfile, confirmAddProfile} = $props();
  let name = $state('');
  let passphrase = $state('');
</script>

<form class='container-sm' onsubmit={(e) => (e.preventDefault(), confirmAddProfile(name, passphrase))}>
  <label for='profileName'>Profile name</label>
  <input use:autofocus placeholder='Username' class='input' id='profileName' bind:value={name}/>
  <label for='passphrase'>Passphrase</label>
  <input
    id='passphrase'
    class='input'
    placeholder='Passphrase'
    type='password'
    autocomplete='new-password'
    bind:value={passphrase}
  />
  <Alert type='error' message={addError} />
  <div class='button-row'>
    <button type='button' class='btn btn-secondary' onclick={closeAddProfile}>Cancel</button>
    <button type='submit' class='btn btn-primary'>Add</button>
  </div>
</form>
