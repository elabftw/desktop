<script lang='ts'>
  type Props = {
    id: string;
    value: string;
    placeholder?: string;
    required?: boolean;
    class?: string;
    autofocus?: boolean;
  };

  let {
    id,
    value = $bindable(),
    placeholder = '',
    required = false,
    class: className = '',
    autofocus = false,
  }: Props = $props();

  let showPassword = $state(false);
</script>

<div class='password-input-wrapper'>
  <input
    {id}
    bind:value
    type={showPassword ? 'text' : 'password'}
    placeholder={placeholder}
    {required}
    {autofocus}
    class={`input password-input ${className}`}
  />

  <button
    type='button'
    class='password-toggle'
    onclick={() => (showPassword = !showPassword)}
    aria-label={showPassword ? 'Hide password' : 'Show password'}
    aria-pressed={showPassword}
  >
    {#if showPassword}
    <!-- Eye slash -->
    <svg xmlns='http://www.w3.org/2000/svg' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' aria-hidden='true'><path d='M17.94 17.94A10.94 10.94 0 0 1 12 20c-5 0-9.27-3.11-11-8a11.83 11.83 0 0 1 5.17-6.17'/><path d='M1 1l22 22'/><path d='M9.9 4.24A10.94 10.94 0 0 1 12 4c5 0 9.27 3.11 11 8a11.82 11.82 0 0 1-3.08 4.36'/><path d='M14.12 14.12a3 3 0 1 1-4.24-4.24'/></svg>
    {:else}
    <!-- Eye -->
    <svg xmlns='http://www.w3.org/2000/svg' width='20' height='20' viewBox='0 0 24 24' fill='none' stroke='white' stroke-width='2' stroke-linecap='round' stroke-linejoin='round' aria-hidden='true'><path d='M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z'/><circle cx='12' cy='12' r='3'/></svg>
    {/if}
  </button>
</div>

<style>
  .password-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .password-input {
    width: 100%;
    padding-right: 2.75rem;
  }

  .password-toggle {
    position: absolute;
    right: 0.75rem;
    border: none;
    background: transparent;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: inherit;
  }
</style>
