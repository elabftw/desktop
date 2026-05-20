<script lang='ts'>
  import { autofocus, preventDefaultSubmit } from '../../utils/helpers';
  import Alert from '../Alert.svelte';

  type Props = {
    addError: string;
    clearProfileSelection: () => void;
    unlock: (passphrase: string) => void | Promise<void>;
    deleteSelectedProfile: (passphrase: string) => void | Promise<void>;
  };

  let {
    addError,
    clearProfileSelection,
    unlock,
    deleteSelectedProfile
  }: Props = $props();

  let passphrase = $state('');

  const handleSubmit = preventDefaultSubmit(() => unlock(passphrase));
</script>

<section>
  <form class='auth-unlock-card' onsubmit={handleSubmit}>
    <p class='description'>Enter your passphrase to continue.</p>
    <div>
      <label for='unlockPassphrase' class='label'>Passphrase</label>
      <input
        {@attach autofocus}
        autocomplete='off'
        placeholder='Enter passphrase'
        bind:value={passphrase}
        class='input'
        id='unlockPassphrase'
        type='password'
      />
    </div>

    <Alert type='error' message={addError}/>

    <div class='flex-row-center form-actions'>
      <button class='btn btn-secondary' type='button' onclick={clearProfileSelection}>
        Cancel
      </button>

      <button class='btn btn-primary' type='submit'>
        Unlock
      </button>
    </div>

    <details class='danger-zone'>
      <summary>Danger zone</summary>

      <div class='danger-actions'>
        <button class='btn btn-danger btn-sm' type='button' onclick={() => deleteSelectedProfile(passphrase)}>
          Delete profile
        </button>
      </div>
    </details>
  </form>
</section>
