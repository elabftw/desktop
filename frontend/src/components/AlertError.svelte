<script lang="ts">
  export let message: string | null | undefined = null;
  export let title = 'Error:';
</script>

{#if message}
  <div class="alert alert-error">
    <strong>{title}</strong> {message}
  </div>
{/if}
