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

/* Formats a timestamp relative to now without a date library. */
export function toRelativeTime(iso: string, locale = 'en'): string {
  const seconds = (Date.parse(iso) - Date.now()) / 1000;
  if (!Number.isFinite(seconds)) return 'now';

  const formatter = new Intl.RelativeTimeFormat(locale, {numeric: 'auto'});
  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ['year', 31536000],
    ['month', 2592000],
    ['week', 604800],
    ['day', 86400],
    ['hour', 3600],
    ['minute', 60],
    ['second', 1],
  ];

  for (const [unit, duration] of units) {
    if (Math.abs(seconds) >= duration || unit === 'second') {
      return formatter.format(Math.round(seconds / duration), unit);
    }
  }
  return 'now';
}
