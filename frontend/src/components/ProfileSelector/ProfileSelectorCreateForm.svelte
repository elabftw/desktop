<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { autofocus, preventDefaultSubmit } from '../../utils/helpers';
  import Alert from '../Alert.svelte';
  import PasswordInput from "../PasswordInput.svelte";

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
      <PasswordInput
        id='profilePassphrase'
        bind:value={passphrase}
        required
        placeholder='Passphrase'
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
