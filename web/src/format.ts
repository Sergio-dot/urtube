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
