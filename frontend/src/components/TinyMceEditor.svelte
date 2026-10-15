<script lang="ts">
  import { onMount } from 'svelte';
  import tinymce from 'tinymce/tinymce';
  import type { Editor } from 'tinymce/tinymce';
  import 'tinymce/models/dom';
  import 'tinymce/icons/default';
  import 'tinymce/themes/silver';
  import 'tinymce/plugins/accordion';
  import 'tinymce/plugins/code';
  import 'tinymce/plugins/link';
  import 'tinymce/plugins/lists';
  import 'tinymce/plugins/table';
  import 'tinymce/skins/ui/oxide-dark/skin.min.css';

  type Props = {
    value: string;
    onChange?: (value: string) => void;
    label: string;
  };

  let { value, onChange, label }: Props = $props();

  let textarea: HTMLTextAreaElement;
  let editor = $state<Editor | null>(null);
  let ready = $state(false);
  // Avoid sending changes back to Svelte while applying external editor content
  let syncingFromProps = false;

  onMount(() => {
    let destroyed = false;

    void tinymce.init({
      target: textarea,
      menubar: false,
      plugins: 'lists link table code accordion',
      toolbar: 'undo redo | blocks | bold italic | bullist numlist | link table accordion | code',
      height: 480,
      branding: false,
      promotion: false,
      skin: false,
      content_css: false,
      content_style: 'html, body { background: #1b2636; color: #fff; font-family: Nunito, sans-serif; } a { color: #9ed9f4; }',
      setup: (instance) => {
        instance.on('init', () => {
          if (destroyed) return;

          syncingFromProps = true;
          instance.setContent(value);
          syncingFromProps = false;
        });

        instance.on('input change undo redo', () => {
          if (!destroyed && !syncingFromProps) {
            onChange?.(instance.getContent());
          }
        });
      },
    }).then((editors) => {
      const instance = editors[0];
      if (!instance) return;

      if (destroyed) {
        instance.remove();
        return;
      }

      editor = instance;
      ready = true;
    }).catch((error: unknown) => {
      if (!destroyed) {
        console.error('TinyMCE initialization failed:', error);
      }
    });

    // Remove the TinyMCE instance when the component is unmounted
    return () => {
      destroyed = true;
      editor?.remove();
      editor = null;
    };
  });

  // Apply incoming values only when they differ from the editor's current content
  $effect(() => {
    if (!editor || editor.getContent() === value) return;

    syncingFromProps = true;
    editor.setContent(value);
    syncingFromProps = false;
  });
</script>

<div class='tinymce-loading-container' aria-busy={!ready}>
  {#if !ready}
    <div class='tinymce-loading' role='status'>
      <span class='spinner' aria-hidden='true'></span>
      Loading editor…
    </div>
  {/if}
  <div style:visibility={ready ? 'visible' : 'hidden'}>
    <textarea aria-labelledby={label} bind:this={textarea}></textarea>
  </div>
</div>

<style>
  .tinymce-loading-container { position: relative; min-height: 360px; background: #1b2636; }
  .tinymce-loading { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; gap: .5rem; color: #fff; }
  .spinner { width: 1rem; height: 1rem; border: 2px solid currentColor; border-right-color: transparent; border-radius: 50%; animation: spin .75s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
</style>
