<script lang='ts'>
  import { ListElabftwInstances } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage } from '../../utils/helpers';
  import Modal from '../Modal.svelte';
  import type { AlertState } from '../Alert.svelte';

  type EntityType = 'experiment' | 'resource';

  type Props = {
    profileUuid: string;
    onClose: () => void;
    onAlert: (alert: AlertState | null) => void;
    onPush: (instanceId: number, entityType: EntityType) => Promise<void>;
  };

  let {profileUuid, onClose, onAlert, onPush}: Props = $props();

  let loading = $state(false);
  let instances = $state<main.ElabftwInstance[]>([]);
  let selectedInstanceId = $state<number | null>(null);
  let entityType = $state<EntityType>('experiment');

  async function refreshInstances(): Promise<void> {
    loading = true;
    try {
      instances = await ListElabftwInstances(profileUuid);
      selectedInstanceId = instances.length === 1 ? instances[0].id : null;
    } catch (e: unknown) {
      onAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  async function confirmPush(): Promise<void> {
    if (!selectedInstanceId) {
      onAlert({type: 'error', message: 'Select an eLabFTW instance.'});
      return;
    }
    await onPush(selectedInstanceId, entityType);
  }

  $effect(() => void refreshInstances());
</script>

<Modal title='Push to eLabFTW' onClose={onClose}>
  {#if loading}
    <p class='description'>Loading eLabFTW instances...</p>
  {:else if instances.length > 1}
    <label for='pushInstance'>Instance</label>
    <select id='pushInstance' class='input' bind:value={selectedInstanceId}>
      <option value={null}>Select instance...</option>
      {#each instances as instance (instance.id)}
        <option value={instance.id}>{instance.siteUrl}</option>
      {/each}
    </select>
  {:else if instances.length === 1}
    <p class='description'>Instance: {instances[0].siteUrl}</p>
  {:else}
    <p class='description'>No eLabFTW instances configured.</p>
  {/if}

  <label for='pushEntityType' class='mt-2'>Remote type</label>
  <select id='pushEntityType' class='input' bind:value={entityType}>
    <option value='experiment'>Experiment</option>
    <option value='resource'>Resource</option>
  </select>

  <svelte:fragment slot='actions'>
    <button class='btn btn-secondary' type='button' onclick={onClose}>Cancel</button>
    <button class='btn btn-primary' type='button' disabled={loading || instances.length === 0} onclick={confirmPush}>
      Push
    </button>
  </svelte:fragment>
</Modal>

