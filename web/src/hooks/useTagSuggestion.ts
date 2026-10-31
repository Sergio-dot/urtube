import { useEffect, useState } from "react";
import { suggestTags, type TagSuggestion } from "../api";

interface State {
  suggestion: TagSuggestion | null;
  error: string | null;
  loading: boolean;
}

/**
 * Fetches the tags urtube would use for the video at url, or does nothing
 * when url is null. A new url abandons the previous request.
 */
export function useTagSuggestion(url: string | null): State {
  const [state, setState] = useState<State>({
    suggestion: null,
    error: null,
    loading: url !== null,
  });

  useEffect(() => {
    if (url === null) return;
    const controller = new AbortController();
    suggestTags(url, controller.signal)
      .then((suggestion) =>
        setState({ suggestion, error: null, loading: false }),
      )
      .catch((err) => {
        if (controller.signal.aborted) return;
        setState({
          suggestion: null,
          error: err instanceof Error ? err.message : "Unknown error",
          loading: false,
        });
      });
    return () => controller.abort();
  }, [url]);

  return state;
}
