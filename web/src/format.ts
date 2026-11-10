/** Formats a duration in seconds as m:ss or h:mm:ss. */
export function formatDuration(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds));
  const hrs = Math.floor(s / 3600);
  const mins = Math.floor((s % 3600) / 60);
  const secs = String(s % 60).padStart(2, "0");
  return hrs > 0
    ? `${hrs}:${String(mins).padStart(2, "0")}:${secs}`
    : `${mins}:${secs}`;
}

/**
 * The number in a progress string such as "42.5%", limited to 0–100, or
 * undefined when there is none (the server sends "" while it is unknown).
 */
export function percentValue(percent: string): number | undefined {
  const value = Number.parseFloat(percent);
  return Number.isNaN(value) ? undefined : Math.min(100, Math.max(0, value));
}
