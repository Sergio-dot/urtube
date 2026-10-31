import { useCallback, useReducer } from "react";
import * as api from "../api";
import { downloadsReducer } from "../downloads";
import { useDownloadEvents } from "./useDownloadEvents";

/**
 * Tracks every download the server reports, including ones started by other
 * clients, and starts, cancels and dismisses them.
 */
export function useDownloads() {
  const [downloads, dispatch] = useReducer(downloadsReducer, {});

  const onEvent = useCallback(
    (event: api.DownloadEvent) => dispatch({ type: "event", event }),
    [],
  );
  useDownloadEvents(onEvent);

  /** Starts a download; rejects with the server's message if it refuses. */
  const start = useCallback(
    async (video: api.Video, options: api.DownloadOptions) => {
      const uuid = await api.startDownload(video, options);
      dispatch({ type: "started", uuid, video });
    },
    [],
  );

  const cancel = useCallback(async (uuid: string) => {
    dispatch({ type: "cancelRequested", uuid });
    try {
      await api.cancelDownload(uuid);
    } catch (err) {
      // Most likely it ended first; the event stream reports how.
      console.error("Failed to cancel download:", err);
    }
  }, []);

  const dismiss = useCallback(
    (uuid: string) => dispatch({ type: "dismissed", uuid }),
    [],
  );

  return { downloads, start, cancel, dismiss };
}
