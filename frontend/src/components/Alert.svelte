<script lang='ts'>
  type AlertType = 'success' | 'error' | 'warning' | 'info';

  type AlertProps = {
    type?: AlertType;
    message?: string;
  };

  let { type = 'success', message = '' }: AlertProps = $props();

  let visible = $state(true);
</script>

{#if message && visible}
  <div class={`alert alert-${type} flex justify-between items-center`}>
    <strong>{message}</strong>

    <button
      class='alert-close'
      type='button'
      aria-label='Close alert'
      onclick={() => visible = false}
    >
      ×
    </button>
  </div>
{/if}
