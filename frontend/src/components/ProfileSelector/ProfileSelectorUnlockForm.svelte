<script lang='ts'>
  import { autofocus, preventDefaultSubmit } from '../../utils/helpers';
  import Alert from '../Alert.svelte';

  type Props = {
    addError: string;
    clearProfileSelection: () => void;
    unlock: (passphrase: string) => void | Promise<void>;
    deleteSelectedProfile: (passphrase: string) => void | Promise<void>;
    forceDeleteProfile: () => void | Promise<void>;
  };

  let {
    addError,
    clearProfileSelection,
    unlock,
    deleteSelectedProfile,
    forceDeleteProfile
  }: Props = $props();

  let passphrase = $state('');

  const handleSubmit = preventDefaultSubmit(() => unlock(passphrase));
</script>

<form class='container-sm' onsubmit={handleSubmit}>
  <label for='unlockPassphrase' class='label'>Enter your passphrase</label>

  <input
    use:autofocus
    autocomplete='off'
    placeholder='Passphrase'
    bind:value={passphrase}
    class='input'
    id='unlockPassphrase'
    type='password'
  />

  <div class='button-row'>
    <button class='btn btn-secondary' type='button' onclick={clearProfileSelection}>Cancel</button>
    <button class='btn btn-primary' type='submit'>Unlock</button>
  </div>
  <br/>
  <button
    class='btn btn-danger'
    type='button'
    onclick={() => deleteSelectedProfile(passphrase)}
  >
    Delete profile (dev)
  </button>
  <button class='btn btn-danger' type='button' onclick={forceDeleteProfile}>Force delete</button>

  <Alert type='error' message={addError}/>
</form>
