import type { DownloadEvent, ServerStatus, Video } from "./api";

/**
 * A download's status in the UI: the server's, plus "starting" for a download
 * this client requested that the server hasn't reported on yet.
 */
export type DownloadStatus = "starting" | ServerStatus;

export interface Download {
  uuid: string;
  videoId: string;
  title: string;
  status: DownloadStatus;
  errorMessage?: string;
  percent: string;
  speed: string;
  eta: string;
  downloaded: string;
  total: string;
}

/** Downloads by ID, in the order they were first seen. */
export type Downloads = Record<string, Download>;

export type DownloadAction =
  /** This client started a download and got its ID back. */
  | { type: "started"; uuid: string; video: Video }
  /** The event stream reported a change. */
  | { type: "event"; event: DownloadEvent }
  /** The user asked to cancel; shown right away, before the server confirms. */
  | { type: "cancelRequested"; uuid: string }
  /** The user dismissed a download from the list. */
  | { type: "dismissed"; uuid: string };

/** The server is still working on it, so it can be cancelled. */
export function isActive(status: DownloadStatus): boolean {
  return (
    status === "starting" || status === "queued" || status === "downloading"
  );
}

export function isTerminal(status: DownloadStatus): boolean {
  return !isActive(status);
}

export function downloadsReducer(
  state: Downloads,
  action: DownloadAction,
): Downloads {
  switch (action.type) {
    case "started": {
      const { uuid, video } = action;
      return {
        ...state,
        // The event stream may have reported the download before the start
        // request returned; what it said wins.
        [uuid]: state[uuid] ?? {
          uuid,
          videoId: video.id,
          title: video.title,
          status: "starting",
          percent: "",
          speed: "",
          eta: "",
          downloaded: "",
          total: "",
        },
      };
    }
    case "event": {
      const { event } = action;
      const current = state[event.uuid];
      // Progress sent before the server handled a cancel would otherwise
      // briefly bring a cancelled download back.
      if (current?.status === "cancelled" && !isTerminal(event.status)) {
        return state;
      }
      return { ...state, [event.uuid]: { ...event } };
    }
    case "cancelRequested": {
      const current = state[action.uuid];
      if (!current || !isActive(current.status)) return state;
      return { ...state, [action.uuid]: { ...current, status: "cancelled" } };
    }
    case "dismissed": {
      if (!(action.uuid in state)) return state;
      const next = { ...state };
      delete next[action.uuid];
      return next;
    }
  }
}

// Which download a video's card shows when it has several: one in progress
// (so it can be cancelled), else a finished one, else the latest failure.
const priority: Record<DownloadStatus, number> = {
  starting: 3,
  queued: 3,
  downloading: 3,
  finished: 2,
  error: 1,
  cancelled: 0,
};

/**
 * Returns, for each video, the download its search card should show. Among
 * equally relevant downloads the most recent wins.
 */
export function downloadsByVideo(
  downloads: Downloads,
): Record<string, Download> {
  const byVideo: Record<string, Download> = {};
  for (const d of Object.values(downloads)) {
    const current = byVideo[d.videoId];
    if (!current || priority[d.status] >= priority[current.status]) {
      byVideo[d.videoId] = d;
    }
  }
  return byVideo;
}
