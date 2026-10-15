<script lang='ts'>
  type Props = { type?: string | null };
  let {type = null}: Props = $props();
</script>

{#if type === 'experiment'}
  <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 448 512' fill='currentColor'><path d='M288 0H128c-17.7 0-32 14.3-32 32s14.3 32 32 32v151.5L7.5 426.3C2.6 435 0 444.7 0 454.7 0 486.4 25.6 512 57.3 512h333.4c31.6 0 57.3-25.6 57.3-57.3 0-10-2.6-19.8-7.5-28.4L320 215.5V64c17.7 0 32-14.3 32-32S337.7 0 320 0h-32zM192 215.5V64h64v151.5c0 11.1 2.9 22.1 8.4 31.8L306 320H142l41.6-72.7c5.5-9.7 8.4-20.6 8.4-31.8z'/></svg>
{:else if type === 'resource'}
  <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 512 512' fill='currentColor'><path d='M192 32c0-17.7 14.3-32 32-32h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32V32zm32 352h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zm192 0h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zM320 192h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32h-64c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32zm-182.6-3.9c12.5-12.5 32.8-12.5 45.3 0l45.3 45.3c12.5 12.5 12.5 32.8 0 45.3l-45.3 45.3c-12.5 12.5-32.8 12.5-45.3 0l-45.3-45.4c-12.5-12.5-12.5-32.8 0-45.3l45.3-45.3zM32 384h64c17.7 0 32 14.3 32 32v64c0 17.7-14.3 32-32 32H32c-17.7 0-32-14.3-32-32v-64c0-17.7 14.3-32 32-32z'/></svg>
{:else}
  <svg aria-hidden='true' width='1em' height='1em' viewBox='0 0 24 24' fill='none' stroke='currentColor' stroke-width='1.75'><rect x='3' y='3' width='5' height='5'/><rect x='9.5' y='3' width='5' height='5'/><rect x='16' y='3' width='5' height='5'/><rect x='3' y='9.5' width='5' height='5'/><rect x='9.5' y='9.5' width='5' height='5'/><rect x='16' y='9.5' width='5' height='5'/><rect x='3' y='16' width='5' height='5'/><rect x='9.5' y='16' width='5' height='5'/><rect x='16' y='16' width='5' height='5'/></svg>
{/if}
