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

<section class='auth-panel'>
  <form class='auth-card' onsubmit={handleSubmit}>
    <div class='form-header'>
      <h2>Create profile</h2>
      <p>Add a new local profile for your entries.</p>
    </div>

    <div class='input-box'>
      <label for='profileName'>Profile name</label>
      <input
        use:autofocus
        placeholder='Username'
        class='input'
        id='profileName'
        bind:value={name}
      />
    </div>

    <div class='input-box'>
      <label for='profilePassphrase'>Passphrase</label>
      <input
        id='profilePassphrase'
        class='input'
        placeholder='Passphrase'
        type='password'
        bind:value={passphrase}
      />
    </div>

    <Alert type='error' message={addError}/>

    <div class='button-row form-actions'>
      <button type='button' class='btn btn-secondary' onclick={closeAddProfile}>
        Cancel
      </button>

      <button type='submit' class='btn btn-primary'>
        Add profile
      </button>
    </div>
  </form>
</section>
