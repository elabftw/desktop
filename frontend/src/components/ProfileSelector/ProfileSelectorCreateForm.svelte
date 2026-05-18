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

<form class='container-sm' onsubmit={handleSubmit}>
  <label for='profileName'>Profile name</label>
  <input
    use:autofocus
    placeholder='Username'
    class='input'
    id='profileName'
    bind:value={name}
  />

  <label for='profilePassphrase'>Passphrase</label>
  <input
    id='profilePassphrase'
    class='input'
    placeholder='Passphrase'
    type='password'
    bind:value={passphrase}
  />

  <Alert type='error' message={addError}/>

  <div class='button-row'>
    <button type='button' class='btn btn-secondary' onclick={closeAddProfile}>Cancel</button>
    <button type='submit' class='btn btn-primary'>Add</button>
  </div>
</form>
