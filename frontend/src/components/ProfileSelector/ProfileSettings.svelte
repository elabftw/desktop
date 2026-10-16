<!--
This file is part of eLabFTW Desktop.

@author Nicolas CARPi / Deltablot
@author Moustapha Camara / Deltablot
@copyright 2026 Deltablot
@see https://www.elabftw.net Official website
SPDX-License-Identifier: GPL-3.0-or-later
-->

<script lang='ts'>
  import { SetPreferredEditor } from '../../../wailsjs/go/main/App';
  import { errorMessage, preventDefaultSubmit } from '../../utils/helpers';
  import { showAlert } from '../stores/alert.svelte';

  type PreferredEditor = 'tinymce' | 'markdown';
  type Props = {
    profileUuid: string;
    preferredEditor: PreferredEditor;
    onBack: () => void;
    onSaved: (editor: PreferredEditor) => void;
  };

  let {profileUuid, preferredEditor, onBack, onSaved}: Props = $props();
  let selectedEditor = $state<PreferredEditor>(preferredEditor);
  let saving = $state(false);

  async function savePreference(): Promise<void> {
    saving = true;
    try {
      await SetPreferredEditor(profileUuid, selectedEditor);
      onSaved(selectedEditor);
      showAlert({type: 'success', message: 'Editor preference saved ✔'});
    } catch (e: unknown) {
      showAlert({type: 'error', message: errorMessage(e)});
    } finally {
      saving = false;
    }
  }
</script>

<section class='panel'>
  <div class='border-bottom mb-2'>
    <button type='button' class='btn btn-secondary' onclick={onBack}>← Back</button>
  </div>
  <form onsubmit={preventDefaultSubmit(savePreference)} class='grid gap-1'>
    <div>
      <label for='preferredEditor'>Preferred editor</label>
      <select id='preferredEditor' class='input' bind:value={selectedEditor}>
        <option value='tinymce'>TinyMCE (rich text)</option>
        <option value='markdown'>Markdown</option>
      </select>
    </div>
    <p>
      The preference applies to new entries. Entries saved in another format
      can be edited by selecting their original editor here. Content is never automatically converted.
    </p>
    <div class='flex justify-end'>
      <button type='submit' class='btn btn-primary' disabled={saving || selectedEditor === preferredEditor}>
        {saving ? 'Saving...' : 'Save preference'}
      </button>
    </div>
  </form>
</section>
