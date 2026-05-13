import type { Action } from 'svelte/action';

/*
 * Focuses the targeted element when it is mounted.
 * Usually used on inputs to replace the native `autofocus` attribute
 * and avoid Svelte a11y_autofocus warnings.
 */
export const autofocus: Action<HTMLElement> = (node) => {
  queueMicrotask(() => node.focus());
  return {};
};

/* Converts an unknown caught error into a readable string. */
export function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/*
 * Creates a form submit handler that prevents the browser'''s default submit behavior.
 * Useful in Svelte 5 as a replacement for the old `on:submit|preventDefault`
 * event modifier pattern.
 */
export function preventDefaultSubmit(
  fn: () => void | Promise<void>
): (e: SubmitEvent) => void {
  return (e) => {
    e.preventDefault();
    void fn();
  };
}
