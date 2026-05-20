import { useEffect } from "react";
import type { ProgressUpdate } from "../types";

export function useDownloadEvents(onUpdate: (update: ProgressUpdate) => void) {
  useEffect(() => {
    const eventSource = new EventSource("/api/v1/events");

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
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [onUpdate]);
}
