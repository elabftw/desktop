<script>
  import { autofocus } from '../../utils/autofocus';

  let { addError, clearProfileSelection, unlock, deleteSelectedProfile } = $props();
  let passphrase = $state('');
</script>

<form class='container-sm' onsubmit={(e) => (e.preventDefault(), unlock(passphrase))}>
  <label for='passphrase' class='label'>Enter your passphrase</label>
  <input
    use:autofocus
    autocomplete='off'
    placeholder='Passphrase'
    bind:value={passphrase}
    class='input'
    id='passphrase'
    type='password'
  />
  <div class='button-row'>
    <button class='btn btn-secondary' type='button' onclick={clearProfileSelection}>Cancel</button>
    <button class='btn btn-primary' type='submit'>Unlock</button>
  </div>
  <br>
  <button class='btn btn-danger' type='button' onclick={deleteSelectedProfile}>Delete profile (dev)</button>

  {#if addError}
    <div class='alert alert-error'>
      <strong>Error:</strong> {addError}
    </div>
  {/if}
</form>
