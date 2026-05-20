<script lang='ts'>
  type AlertType = 'success' | 'error' | 'warning' | 'info';
  type AlertProps = {
    type?: AlertType;
    message?: string;
  };
  let { type = 'success', message = '' }: AlertProps = $props();
</script>

{#if message}
  <div class={`alert alert-${type}`}>
    <strong>{message}</strong>
  </div>
{/if}
