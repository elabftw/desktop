import type { Attachment } from 'svelte/attachments';
import { BrowserOpenURL } from "../../wailsjs/runtime";

/*
 * Focuses the targeted element when it is mounted.
 * Usually used on inputs to replace the native `autofocus` attribute
 * and avoid Svelte a11y_autofocus warnings.
 */
export const autofocus: Attachment<HTMLElement> = (node) => {
  // queueMicrotask waits until element is ready in the page to run focus
  queueMicrotask(() => node.focus());
};

/* Converts an unknown caught error into a readable string. */
export function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/*
 * Creates a form submit handler that prevents the browser's default submit behavior.
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

/* opens a url in the preferred browser */
export function openExternalURL(url: string): void {
  if (!url) return;
  BrowserOpenURL(url);
}
