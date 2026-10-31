import { useEffect } from "react";
import { EVENTS_URL, type DownloadEvent } from "../api";

const INITIAL_RETRY_MS = 1000;
const MAX_RETRY_MS = 30000;

/**
 * Calls onEvent for every message on the server's download event stream. On
 * connect the server first sends every active download, so reconnecting
 * catches up on anything missed.
 */
export function useDownloadEvents(onEvent: (event: DownloadEvent) => void) {
  useEffect(() => {
    let eventSource: EventSource;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let retryDelay = INITIAL_RETRY_MS;

    const connect = () => {
      eventSource = new EventSource(EVENTS_URL);

      eventSource.onopen = () => {
        retryDelay = INITIAL_RETRY_MS;
      };

      eventSource.onmessage = (message) => {
        try {
          onEvent(JSON.parse(message.data) as DownloadEvent);
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
  }, [onEvent]);
}
