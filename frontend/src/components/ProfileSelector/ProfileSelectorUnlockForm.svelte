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
    <div>
      <label for='unlockPassphrase' class='text-big mt-2'>Passphrase</label>
      <input
        {@attach autofocus}
        required
        autocomplete='off'
        placeholder='Enter your passphrase to continue.'
        bind:value={passphrase}
        class='input'
        id='unlockPassphrase'
        type='password'
      />
    </div>

   <Alert type='error' message={addError}/>

    <div class='flex gap-1 mt-2 justify-end'>
      <button class='btn btn-secondary' type='button' onclick={clearProfileSelection}>
        Cancel
      </button>

      <button class='btn btn-primary' type='submit'>
        Unlock
      </button>
    </div>

    <details class='border-top mt-2'>
      <summary class='text-orange'>Danger zone</summary>

      <div class='mt-2'>
        <button class='btn btn-danger' type='button' onclick={() => deleteSelectedProfile(passphrase)}>
          Delete profile
        </button>
      </div>
    </details>
  </form>
</section>
