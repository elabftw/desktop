<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi <Deltablot>
@author Moustapha Camara <Deltablot>
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<!--
This file contains the modal to push informations to eLabFTW. It
is adapted to be scalable, so later on we'll just replace eLabFTW
with "Instance Name" for cross platform data share.
-->
<script lang='ts'>
  import { ListElabftwInstances } from '../../../wailsjs/go/main/App';
  import type { main } from '../../../wailsjs/go/models';
  import { errorMessage } from '../../utils/helpers';
  import Modal from '../Modal.svelte';
  import { showAlert } from "../stores/alert.svelte";
  import ElabftwInstanceCreateForm from './ElabftwInstanceCreateForm.svelte';

  type EntityType = 'experiment' | 'resource';

  type Props = {
    profileUuid: string;
    force?: boolean;
    onClose: () => void;
    onPush: (
      instanceId: number,
      entityType: EntityType,
      force?: boolean,
    ) => Promise<void>;
  };

  let {
    profileUuid,
    force = false,
    onClose,
    onPush,
  }: Props = $props();

  let loading = $state(true);
  let pushing = $state(false);
  let showCreateForm = $state(false);
  let instances = $state<main.ElabftwInstance[]>([]);
  let selectedInstanceId = $state<number | null>(null);
  let entityType = $state<EntityType>('experiment');

  async function refreshInstances(previousIds?: Set<number>): Promise<void> {
    loading = true;
    try {
      instances = await ListElabftwInstances(profileUuid);
      if (previousIds) {
        const created = instances.find(instance => !previousIds.has(instance.id));
        if (created) {
          selectedInstanceId = created.id;
          return;
        }
      }
      if (selectedInstanceId !== null && instances.some(instance => instance.id === selectedInstanceId)) return;
      selectedInstanceId = instances.length === 1 ? instances[0].id : null;
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      loading = false;
    }
  }

  async function createInstanceFromModal(): Promise<void> {
    const previousIds = new Set(instances.map(instance => instance.id));
    await refreshInstances(previousIds);
    showCreateForm = false;
  }

  async function confirmPush(): Promise<void> {
    if (selectedInstanceId === null) {
      showAlert({type: 'error', message: 'Select an eLabFTW instance.'});
      return;
    }
    pushing = true;

    try {
      await onPush(selectedInstanceId, entityType, force);
    } finally {
      pushing = false;
    }
  }

  $effect(() => void refreshInstances());
</script>

<Modal title='Push to eLabFTW' onClose={onClose}>
  {#if loading}
    <p class='description'>
      Loading eLabFTW instances...
    </p>

  {:else if instances.length === 0}
    <div class='grid gap-1'>
      <p class='description'>
        No eLabFTW instance is configured.
        Add one to continue.
      </p>

      <ElabftwInstanceCreateForm
        {profileUuid}
        onCreated={createInstanceFromModal}
        submitLabel='Add and continue'
      />
    </div>

  {:else}
    <div class='grid gap-1'>
      {#if instances.length === 1}
        <div>
          <h1>Instance</h1>
          <p class='text-white'>{instances[0].siteUrl}</p>
        </div>
      {:else}
        <div>
          <label for='push-instance'>Instance</label>
          <select id='push-instance' class='input' bind:value={selectedInstanceId}>
            <option value={null}>Select instance...</option>
            {#each instances as instance (instance.id)}
              <option value={instance.id}>
                {instance.siteUrl}
              </option>
            {/each}
          </select>
        </div>
      {/if}

      {#if showCreateForm}
        <ElabftwInstanceCreateForm
          {profileUuid}
          onCreated={createInstanceFromModal}
          submitLabel='Add and select'
        />
      {/if}

      <div>
        <label for='push-entity-type'>Remote type</label>
        <select id='push-entity-type' class='input' bind:value={entityType}>
          <option value='experiment'>Experiment</option>
          <option value='resource'>Resource</option>
        </select>
      </div>
    </div>
  {/if}

  <svelte:fragment slot='actions'>
    <button
      class='btn btn-secondary'
      type='button'
      disabled={pushing}
      onclick={onClose}
    >
      Cancel
    </button>

    {#if instances.length > 0}
      <button
        class={`btn ${force ? 'btn-danger' : 'btn-primary'}`}
        type='button'
        disabled={loading || pushing || selectedInstanceId === null || showCreateForm}
        onclick={confirmPush}
      >
        {pushing ? 'Pushing...' : force ? 'Push anyway' : 'Push'}
      </button>
    {/if}
  </svelte:fragment>
</Modal>
