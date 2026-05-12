<script lang='ts'>
  let {type = 'success', message = ''}: {
    type?: 'success' | 'error' | 'warning' | 'info';
    message?: string;
  } = $props();
</script>

{#if message}
  <div class={`alert alert-${type}`}>
    <strong>{message}</strong>
  </div>
{/if}
