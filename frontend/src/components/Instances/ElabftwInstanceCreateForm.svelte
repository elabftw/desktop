<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later

This file is the centralized form for creating the relationship between
an eLabFTW instance and the desktop App.
It is used in the Instances View page and in the Modal called from the
"Push to eLabFTW" buttons
-->
<script lang='ts'>
  import { AddElabftwInstance } from '../../../wailsjs/go/main/App';
  import { errorMessage, preventDefaultSubmit } from '../../utils/helpers';
  import PasswordInput from '../PasswordInput.svelte';
  import { showAlert } from "../stores/alert.svelte";
  type Props = {
    profileUuid: string;
    onCreated: () => void | Promise<void>;
    submitLabel?: string;
  };

  let {profileUuid, onCreated, submitLabel = 'Add instance'}: Props = $props();

  let siteUrl = $state('');
  let apiKey = $state('');
  let verifyTls = $state(true);
  let saving = $state(false);

  async function createInstance(): Promise<void> {
    if (saving) {
      return;
    }

    saving = true;
    showAlert({type: 'info', message: 'Adding instance...'});

    try {
      await AddElabftwInstance(profileUuid, siteUrl.trim(), apiKey, verifyTls);

      showAlert({
        type: 'success',
        message: 'eLabFTW instance added ✔',
      });

      siteUrl = '';
      apiKey = '';
      verifyTls = true;

      await onCreated();
    } catch (error: unknown) {
      showAlert({
        type: 'error',
        message: errorMessage(error),
      });
    } finally {
      saving = false;
    }
  }

  const handleSubmit = preventDefaultSubmit(createInstance);
</script>

<form onsubmit={handleSubmit} class='grid gap-1'>
  <div>
    <label for='new-instance-site-url'>Site URL</label>
    <input
      required
      id='new-instance-site-url'
      type='url'
      class='input'
      bind:value={siteUrl}
      placeholder='https://elab.example.org'
      disabled={saving}
    />
  </div>

  <div>
    <label for='new-instance-api-key'>API key</label>
    <PasswordInput
      required
      id='new-instance-api-key'
      bind:value={apiKey}
      placeholder='Your eLabFTW API key'
      disabled={saving}
    />
  </div>

  <label class='checkbox-row flex items-center gap-1'>
    <input
      type='checkbox'
      bind:checked={verifyTls}
      disabled={saving}
    />
    <span class='checkbox-box'></span>
    <span>Verify TLS certificates</span>
  </label>

  <div class='flex justify-end'>
    <button
      class='btn btn-primary'
      type='submit'
      disabled={saving}
    >
      {saving ? 'Adding...' : submitLabel}
    </button>
  </div>
</form>
