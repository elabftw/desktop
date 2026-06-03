<script lang='ts'>
  type Props = {
    title: string;
    onClose: () => void;
  };

  let {title, onClose}: Props = $props();

  function stopClick(event: MouseEvent): void {
    event.stopPropagation();
  }
</script>

<div class='modal-backdrop' role='presentation' onclick={onClose}>
  <dialog
    class='modal panel'
    aria-labelledby='modal-title'
    open
    onclick={stopClick}
  >
    <header class='flex justify-between items-center border-bottom mb-2'>
      <h1 id='modal-title'>{title}</h1>

      <button class='btn btn-secondary' type='button' onclick={onClose} aria-label='Close modal'>
        ×
      </button>
    </header>

    <div>
      <slot/>
    </div>

    <footer class='flex justify-end gap-1 mt-2'>
      <slot name='actions'/>
    </footer>
  </dialog>
</div>
