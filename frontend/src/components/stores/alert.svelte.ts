export type AlertState = {
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
};

export const alert = $state<{current: AlertState | null}>({
  current: null,
});

export function showAlert(nextAlert: AlertState | null): void {
  alert.current = nextAlert;
}

export function clearAlert(): void {
  alert.current = null;
}
