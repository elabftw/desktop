<script lang="ts">
  type Props = {
    id: string;
    value: string;
    placeholder?: string;
    required?: boolean;
    class?: string;
  };

  let {
    id,
    value = $bindable(),
    placeholder = '',
    required = false,
    class: className = '',
  }: Props = $props();

  let showPassword = $state(false);
</script>

<div class="password-input-wrapper">
  <input
    {id}
    bind:value
    type={showPassword ? 'text' : 'password'}
    placeholder={placeholder}
    {required}
    class={`input password-input ${className}`}
  />

  <button
    type="button"
    class="password-toggle"
    onclick={() => (showPassword = !showPassword)}
    aria-label={showPassword ? 'Hide password' : 'Show password'}
    aria-pressed={showPassword}
  >
    {#if showPassword}
      🙈
    {:else}
      👁️
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
