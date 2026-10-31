import { useCallback, useEffect, useRef, useState } from "react";
import { searchVideos, type SearchParams, type Video } from "../api";

/** Runs searches; a new search abandons the one still in flight. */
export function useSearch() {
  // null until the first search completes.
  const [results, setResults] = useState<Video[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const inFlight = useRef<AbortController | null>(null);

  useEffect(() => () => inFlight.current?.abort(), []);

  const search = useCallback(async (params: SearchParams) => {
    inFlight.current?.abort();
    const controller = new AbortController();
    inFlight.current = controller;

    setLoading(true);
    setError(null);
    try {
      setResults(await searchVideos(params, controller.signal));
    } catch (err) {
      if (controller.signal.aborted) return;
      setResults([]);
      setError(err instanceof Error ? err.message : "Search failed");
    } finally {
      if (inFlight.current === controller) {
        inFlight.current = null;
        setLoading(false);
      }
    }
  }, []);

  return { results, loading, error, search };
}
