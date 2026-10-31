import { useEffect, useState } from "react";

function read<T>(
  key: string,
  initial: T | (() => T),
  isValid?: (value: unknown) => value is T,
): T {
  const fallback = () =>
    typeof initial === "function" ? (initial as () => T)() : initial;
  try {
    const raw = localStorage.getItem(key);
    if (raw === null) return fallback();
    const value: unknown = JSON.parse(raw);
    return !isValid || isValid(value) ? (value as T) : fallback();
  } catch {
    // Storage blocked (private mode) or corrupt JSON.
    return fallback();
  }
}

/**
 * State persisted in localStorage under key. Stored values that fail isValid,
 * such as ones written by an older version, are replaced by initial; storage
 * that is unavailable just leaves the state in memory.
 */
export function useLocalStorage<T>(
  key: string,
  initial: T | (() => T),
  isValid?: (value: unknown) => value is T,
) {
  const [value, setValue] = useState<T>(() => read(key, initial, isValid));

  useEffect(() => {
    try {
      localStorage.setItem(key, JSON.stringify(value));
    } catch {
      // Not persisted; the state still works for this session.
    }
  }, [key, value]);

  return [value, setValue] as const;
}
