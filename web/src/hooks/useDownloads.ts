import { useCallback, useReducer, useState } from "react";
import * as api from "../api";
import { downloadsReducer } from "../downloads";
import { useDownloadEvents } from "./useDownloadEvents";

/**
 * Tracks every download the server reports, including ones started by other
 * clients, and starts, cancels and dismisses them.
 */
export function useDownloads() {
  const [downloads, dispatch] = useReducer(downloadsReducer, {});

  // The latest outcome worth telling a screen reader user about.
  const [announcement, setAnnouncement] = useState("");

  const onEvent = useCallback((event: api.DownloadEvent) => {
    dispatch({ type: "event", event });
    const title = event.title || "Untitled";
    if (event.status === "finished") setAnnouncement(`Downloaded ${title}`);
    if (event.status === "error") setAnnouncement(`Download failed: ${title}`);
  }, []);
  useDownloadEvents(onEvent);

  /** Starts a download; rejects with the server's message if it refuses. */
  const start = useCallback(
    async (video: api.Video, options: api.DownloadOptions, tags?: api.Tags) => {
      const uuid = await api.startDownload(video, options, tags);
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

  return { downloads, announcement, start, cancel, dismiss };
}
