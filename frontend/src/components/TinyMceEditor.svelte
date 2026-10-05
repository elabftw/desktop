<script lang="ts">
  import { onMount } from 'svelte';
  import tinymce from 'tinymce/tinymce';
  import type { Editor } from 'tinymce/tinymce';
  import 'tinymce/models/dom';
  import 'tinymce/icons/default';
  import 'tinymce/themes/silver';
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
  let syncingFromProps = false;

  onMount(() => {
    let destroyed = false;

    void tinymce.init({
      target: textarea,
      menubar: false,
      plugins: 'lists link table code',
      toolbar: 'undo redo | blocks | bold italic | bullist numlist | link table | code',
      height: 360,
      branding: false,
      promotion: false,
      skin: false,
      content_css: false,
      content_style: 'body { background: #1b2636; color: #fff; font-family: Nunito, sans-serif; } a { color: #9ed9f4; }',
      setup: (instance) => {
        editor = instance;

        instance.on('init', () => {
          instance.setContent(value);
          ready = true;
        });

        instance.on('input change undo redo', () => {
          if (!syncingFromProps) {
            onChange?.(instance.getContent());
          }
        });
      },
    }).then((editors) => {
      if (destroyed) {
        editors.forEach((instance) => instance.remove());
      }
    });

    return () => {
      destroyed = true;
      ready = false;
      editor?.remove();
      editor = null;
    };
  });

  $effect(() => {
    if (!editor || !ready || editor.getContent() === value) return;

    syncingFromProps = true;
    editor.setContent(value);
    syncingFromProps = false;
  });
</script>

<textarea aria-labelledby={label} bind:this={textarea}></textarea>
