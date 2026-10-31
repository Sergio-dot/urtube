import { describe, expect, it } from "vitest";
import type { DownloadEvent, Video } from "./api";
import {
  downloadsByVideo,
  downloadsReducer,
  type Download,
  type DownloadStatus,
  type Downloads,
} from "./downloads";

const video: Video = {
  id: "v1",
  title: "Song",
  uploader: "Artist",
  duration: 60,
  thumbnails: [],
  url: "https://example.com/v1",
  isLive: false,
};

function event(
  uuid: string,
  status: DownloadEvent["status"],
  extra: Partial<DownloadEvent> = {},
): DownloadEvent {
  return {
    uuid,
    videoId: "v1",
    title: "Song",
    status,
    percent: "",
    speed: "",
    eta: "",
    downloaded: "",
    total: "",
    ...extra,
  };
}

function download(
  uuid: string,
  status: DownloadStatus,
  videoId = "v1",
): Download {
  return { ...event(uuid, "queued"), videoId, status };
}

describe("downloadsReducer", () => {
  it("adds a started download as starting", () => {
    const state = downloadsReducer({}, { type: "started", uuid: "a", video });
    expect(state.a).toMatchObject({
      uuid: "a",
      videoId: "v1",
      title: "Song",
      status: "starting",
    });
  });

  it("keeps what the event stream reported before the start request returned", () => {
    let state = downloadsReducer(
      {},
      {
        type: "event",
        event: event("a", "downloading", { percent: "10.00%" }),
      },
    );
    state = downloadsReducer(state, { type: "started", uuid: "a", video });
    expect(state.a).toMatchObject({ status: "downloading", percent: "10.00%" });
  });

  it("applies events, including for downloads started elsewhere", () => {
    const state = downloadsReducer(
      {},
      { type: "event", event: event("b", "queued") },
    );
    expect(state.b.status).toBe("queued");
  });

  it("marks active downloads cancelled right away", () => {
    const state = downloadsReducer(
      { a: download("a", "downloading") },
      { type: "cancelRequested", uuid: "a" },
    );
    expect(state.a.status).toBe("cancelled");
  });

  it("does not mark finished downloads cancelled", () => {
    const before: Downloads = { a: download("a", "finished") };
    expect(
      downloadsReducer(before, { type: "cancelRequested", uuid: "a" }),
    ).toBe(before);
  });

  it("ignores progress that arrives after a cancel request", () => {
    let state = downloadsReducer(
      { a: download("a", "downloading") },
      { type: "cancelRequested", uuid: "a" },
    );
    state = downloadsReducer(state, {
      type: "event",
      event: event("a", "downloading"),
    });
    expect(state.a.status).toBe("cancelled");
  });

  it("accepts the final status after a cancel request", () => {
    let state = downloadsReducer(
      { a: download("a", "downloading") },
      { type: "cancelRequested", uuid: "a" },
    );
    state = downloadsReducer(state, {
      type: "event",
      event: event("a", "finished"),
    });
    expect(state.a.status).toBe("finished");
  });

  it("removes dismissed downloads", () => {
    const state = downloadsReducer(
      { a: download("a", "error") },
      { type: "dismissed", uuid: "a" },
    );
    expect(state).toEqual({});
  });
});

describe("downloadsByVideo", () => {
  const pick = (...ds: Download[]) =>
    downloadsByVideo(Object.fromEntries(ds.map((d) => [d.uuid, d])));

  it("prefers a download in progress over a finished one", () => {
    expect(
      pick(download("a", "finished"), download("b", "queued")).v1.uuid,
    ).toBe("b");
    expect(
      pick(download("a", "downloading"), download("b", "finished")).v1.uuid,
    ).toBe("a");
  });

  it("prefers finished over failed, and failed over cancelled", () => {
    expect(
      pick(download("a", "finished"), download("b", "error")).v1.uuid,
    ).toBe("a");
    expect(
      pick(download("a", "error"), download("b", "cancelled")).v1.uuid,
    ).toBe("a");
  });

  it("shows the newest of equally relevant downloads", () => {
    expect(pick(download("a", "error"), download("b", "error")).v1.uuid).toBe(
      "b",
    );
  });

  it("keeps videos apart", () => {
    const byVideo = pick(
      download("a", "finished", "v1"),
      download("b", "error", "v2"),
    );
    expect(byVideo.v1.uuid).toBe("a");
    expect(byVideo.v2.uuid).toBe("b");
  });
});
