import { useEffect } from "react";
import type { ProgressUpdate } from "../types";

const INITIAL_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;

export function useDownloadEvents(onUpdate: (update: ProgressUpdate) => void) {
  useEffect(() => {
    let eventSource: EventSource;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let retryDelay = INITIAL_RETRY_MS;

    const connect = () => {
      eventSource = new EventSource("/api/v1/events");

      eventSource.onopen = () => {
        retryDelay = INITIAL_RETRY_MS;
      };

      eventSource.onmessage = (event) => {
        try {
          const update: ProgressUpdate = JSON.parse(event.data);
          onUpdate(update);
        } catch (err) {
          console.error("Failed to parse download event", err);
        }
      };

      eventSource.onerror = (err) => {
        console.error("EventSource failed:", err);
        // The browser retries dropped connections by itself, but gives up for
        // good on a failed response (e.g. a proxy 502 while the server
        // restarts). Only then do we reconnect manually, with backoff.
        if (eventSource.readyState !== EventSource.CLOSED) return;
        retryTimer = setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, MAX_RETRY_MS);
      };
    };

    connect();

    return () => {
      clearTimeout(retryTimer);
      eventSource.close();
    };
  }, [onUpdate]);
}
