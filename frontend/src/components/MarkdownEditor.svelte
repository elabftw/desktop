<script lang="ts">
  import { onMount } from 'svelte';
  import EasyMDE from 'easymde';
  import 'easymde/dist/easymde.min.css';

  type Props = {
    value: string;
    onChange?: (value: string) => void;
  };

  let { value, onChange }: Props = $props();

  let textarea: HTMLTextAreaElement;
  let editor: EasyMDE;

  onMount(() => {
    editor = new EasyMDE({
      element: textarea,
      spellChecker: false,
      autofocus: false,
      status: false,
    });

    editor.value(value);

    editor.codemirror.on('change', () => {
      const next = editor.value();
      onChange?.(next);
    });

    return () => {
      editor.toTextArea();
    };
  });

  $effect(() => {
    if (editor && editor.value() !== value) {
      editor.value(value);
    }
  });
</script>

<textarea bind:this={textarea}></textarea>
