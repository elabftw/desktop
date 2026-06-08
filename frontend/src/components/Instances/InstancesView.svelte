<!--
This file contains the list of instances. It allows you to see instances,
log new instances, edit existing and delete. You can test the API as well
(info endpoint).
-->
<script lang='ts'>
  import {
    ListElabftwInstances,
    AddElabftwInstance,
    UpdateElabftwInstance,
    DeleteElabftwInstance,
    FetchElabftwInfo,
  } from '../../../wailsjs/go/main/App';

  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage, preventDefaultSubmit, openExternalURL } from '../../utils/helpers';
  import type { AlertState } from '../Alert.svelte';

  type Props = {
    profileUuid: string;
    onBack: () => void;
    onAlert: (alert: AlertState | null) => void;
  };

  let {profileUuid, onBack, onAlert}: Props = $props();

  let loading = $state(false);
  let instances = $state<main.ElabftwInstance[]>([]);
  let instanceSiteUrl = $state('');
  let instanceApiKey = $state('');
  let instanceVerifyTls = $state(true);
  let editingInstanceId = $state<number | null>(null);
  let elabftwInfoOutput = $state('');

  async function refreshInstances(): Promise<void> {
    loading = true;
    try {
      instances = await ListElabftwInstances(profileUuid);
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  // save or update an instance
  async function saveInstance(): Promise<void> {
    onAlert({
      type: 'info',
      message: editingInstanceId ? 'Updating instance...' : 'Adding instance...',
    });

    try {
      if (editingInstanceId) {
        await UpdateElabftwInstance(profileUuid, editingInstanceId, instanceSiteUrl, instanceApiKey, instanceVerifyTls);
        onAlert({type: 'success', message: 'eLabFTW instance updated ✔'});
      } else {
        await AddElabftwInstance(profileUuid, instanceSiteUrl, instanceApiKey, instanceVerifyTls);
        onAlert({type: 'success', message: 'eLabFTW instance added ✔'});
      }

      resetForm();
      await refreshInstances();
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    }
  }

  async function deleteInstance(id: number, siteUrl: string): Promise<void> {
    const confirmed = window.confirm(`Delete eLabFTW instance "${siteUrl}"?`);
    if (!confirmed) return;
    try {
      await DeleteElabftwInstance(profileUuid, id);
      await refreshInstances();
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    }
  }

  /* send a GET request to info endpoint and ensure the connection is ok */
  async function testInstance(id: number): Promise<void> {
    onAlert({type: 'info', message: 'Fetching eLabFTW /info...'});
    elabftwInfoOutput = '';
    try {
      const info = await FetchElabftwInfo(profileUuid, id);
      elabftwInfoOutput = JSON.stringify(info.raw, null, 2);
      onAlert({type: 'success', message: 'Connected to eLabFTW ✔'});
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    }
  }

  /* switch the view to edit mode for an instance */
  function editInstance(instance: main.ElabftwInstance): void {
    editingInstanceId = instance.id;
    instanceSiteUrl = instance.siteUrl;
    instanceApiKey = '';
    instanceVerifyTls = instance.verifyTls;
    onAlert({type: 'info', message: 'Editing instance. Leave API key empty to keep the current key.'});
  }

  function cancelEditInstance(): void {
    resetForm();
    onAlert(null);
  }

  function resetForm(): void {
    editingInstanceId = null;
    instanceSiteUrl = '';
    instanceApiKey = '';
    instanceVerifyTls = true;
  }

  const handleInstanceSubmit = preventDefaultSubmit(saveInstance);

  $effect(() => void refreshInstances());
</script>

<section class='panel'>
  <div class='flex justify-between border-bottom mb-2 items-center'>
    <span>
      Add the site URL and your API key to allow communication between the desktop app and your eLabFTW instance.
      <br/>
      See the
      <button type='button' class='link-button'
              onclick={() => openExternalURL('https://doc.elabftw.net/docs/usage/api')}>Documentation</button>
      to learn how to create a new API key.
    </span>
    <button class='btn btn-secondary' type='button' onclick={onBack}>← Back</button>
  </div>

  <form onsubmit={handleInstanceSubmit} class='grid gap-1'>
    <div>
      <label for='instanceSiteUrl'>Site URL</label>
      <input
        required
        id='instanceSiteUrl'
        type='url'
        class='input'
        bind:value={instanceSiteUrl}
        placeholder='https://elab.example.org'
      />
    </div>

    <div>
      <label for='instanceApiKey'>API key</label>
      <input
        required
        id='instanceApiKey'
        type='password'
        class='input'
        bind:value={instanceApiKey}
        placeholder={editingInstanceId ? 'Leave empty to keep current API key' : 'Your eLabFTW API key'}
      />
    </div>

    <label class='checkbox-row flex items-center gap-1'>
      <input type='checkbox' bind:checked={instanceVerifyTls}/>
      <span class='checkbox-box'></span>
      <span>Verify TLS certificates</span>
    </label>

    <div class='flex justify-end gap-1'>
      {#if editingInstanceId}
        <button class='btn btn-secondary' type='button' onclick={cancelEditInstance}>Cancel</button>
      {/if}

      <button class='btn btn-primary' type='submit'>
        {editingInstanceId ? 'Update instance' : 'Add instance'}
      </button>
    </div>
  </form>

  <div class='border-bottom mb-2'></div>

  {#if loading}
    <div class='empty-state'>Loading instances...</div>
  {:else if instances.length === 0}
    <div class='empty-state'>
      <h3>No eLabFTW instances yet</h3>
      <p>Add one above before pushing entries.</p>
    </div>
  {:else}
    <div class='grid gap-1'>
      {#each instances as instance (instance.id)}
        <div class='flex justify-between items-center gap-1'>
          <div class='grid gap-03'>
            <span class='text-white'>{instance.siteUrl}</span>
            <span class={instance.verifyTls ? 'text-success' : 'text-orange'}>
              TLS verification: {instance.verifyTls ? 'enabled' : 'disabled'}
            </span>
          </div>

          <div class='flex gap-1'>
            <button type='button' class='btn btn-danger' onclick={() => deleteInstance(instance.id, instance.siteUrl)}>
              Delete
            </button>
            <button type='button' class='btn btn-secondary' onclick={() => editInstance(instance)}>Edit</button>
            <button type='button' class='btn btn-secondary' onclick={() => testInstance(instance.id)}>Test</button>
          </div>
        </div>
      {/each}
      <!-- output of the Info GET response -->
      {#if elabftwInfoOutput}
        <div class='mt-2'>
          <h3>eLabFTW /info response</h3>
          <pre class='panel'>{elabftwInfoOutput}</pre>
        </div>
      {/if}
    </div>
  {/if}
</section>
