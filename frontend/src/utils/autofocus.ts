/*
 * Focuses the targeted element when it is mounted.
 * Usually used on inputs to replace the native `autofocus` attribute
 * and avoid Svelte a11y_autofocus warnings.
 */
export function autofocus(node: HTMLElement) {
  queueMicrotask(() => node.focus());
  return {};
}
