<script lang='ts'>
  import { autofocus, preventDefaultSubmit } from '../../utils/helpers';
  import Alert from '../Alert.svelte';

  type Props = {
    addError: string;
    closeAddProfile: () => void;
    confirmAddProfile: (name: string, passphrase: string) => void | Promise<void>;
  };

  let {addError, closeAddProfile, confirmAddProfile}: Props = $props();

  let name = $state('');
  let passphrase = $state('');

  const handleSubmit = preventDefaultSubmit(() => confirmAddProfile(name, passphrase));
</script>

<section class='panel'>
  <form onsubmit={handleSubmit}>
    <h1>Create a profile</h1>
    <h2 class='mb-2'>Add a new local profile for your entries.</h2>

    <div>
      <label for='profileName'>Profile name</label>
      <input
        required
        {@attach autofocus}
        placeholder='Username'
        class='input'
        id='profileName'
        bind:value={name}
      />
    </div>

    <div>
      <label for='profilePassphrase' class='mt-2'>Passphrase</label>
      <input
        required
        id='profilePassphrase'
        class='input'
        placeholder='Passphrase'
        type='password'
        bind:value={passphrase}
      />
    </div>

    <Alert type='error' message={addError}/>

    <div class='flex gap-1 mt-2 justify-end'>
      <button type='button' class='btn btn-secondary' onclick={closeAddProfile}>
        Cancel
      </button>

      <button type='submit' class='btn btn-primary'>
        Add profile
      </button>
    </div>
  </form>
</section>
