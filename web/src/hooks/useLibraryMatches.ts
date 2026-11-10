import { useEffect, useState } from "react";
import {
  checkLibrary,
  type LibraryMatch,
  type MediaType,
  type Tags,
} from "../api";

/** Waits for typing to pause before asking the server again. */
const debounceMs = 300;

/**
 * Checks what the library already holds that downloading each of items as
 * type would duplicate, or nothing when items is null or empty. It re-checks
 * when the items change, after a pause, abandoning the previous request.
 * Failures leave it empty: the warning is a courtesy, and downloads check
 * again when they run.
 */
export function useLibraryMatches(
  type: MediaType,
  items: Tags[] | null,
): LibraryMatch[] {
  const [result, setResult] = useState<{
    key: string;
    matches: LibraryMatch[];
  } | null>(null);
  // Callers build items on every render; compare them by value.
  const key = items?.length ? JSON.stringify([type, items]) : null;

  useEffect(() => {
    if (key === null) return;
    const [checkType, checkItems] = JSON.parse(key) as [MediaType, Tags[]];
    const controller = new AbortController();
    const timer = setTimeout(() => {
      checkLibrary(checkType, checkItems, controller.signal)
        .then((matches) => setResult({ key, matches }))
        .catch(() => {
          if (!controller.signal.aborted) setResult({ key, matches: [] });
        });
    }, debounceMs);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [key]);

  // Until the new answer arrives the last one stands, so a warning doesn't
  // flicker away and back on every keystroke.
  if (key === null || result === null) return [];
  return result.matches;
}
