import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  cancelDownload,
  isValidOptions,
  searchVideos,
  startDownload,
  type Video,
} from "./api";

function mockFetch(status: number, body: unknown) {
  const fetchMock = vi
    .fn()
    .mockResolvedValue(
      new Response(body === undefined ? null : JSON.stringify(body), {
        status,
      }),
    );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

const video: Video = {
  id: "v1",
  title: "Song",
  uploader: "Artist",
  duration: 60,
  thumbnails: [],
  url: "https://example.com/v1",
  isLive: false,
};

describe("searchVideos", () => {
  it("encodes the query and parameters", async () => {
    const fetchMock = mockFetch(200, [video]);
    const results = await searchVideos({
      query: "a/b c",
      limit: 10,
      includeLive: true,
    });
    expect(results).toEqual([video]);
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/v1/search/a%2Fb%20c?limit=10&wantLiveStreams=true",
    );
  });

  it("resolves to no results on 404", async () => {
    mockFetch(404, { error: "no results found" });
    await expect(
      searchVideos({ query: "x", limit: 5, includeLive: false }),
    ).resolves.toEqual([]);
  });

  it("rejects with the server's message", async () => {
    mockFetch(500, { error: "internal server error" });
    await expect(
      searchVideos({ query: "x", limit: 5, includeLive: false }),
    ).rejects.toEqual(new ApiError(500, "internal server error"));
  });
});

describe("startDownload", () => {
  it("posts the video and options and returns the ID", async () => {
    const fetchMock = mockFetch(202, {
      uuid: "abc",
      message: "download started",
    });
    await expect(
      startDownload(video, { type: "audio", format: "mp3" }),
    ).resolves.toBe("abc");

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/download");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body)).toEqual({
      url: video.url,
      videoId: "v1",
      title: "Song",
      options: { type: "audio", format: "mp3" },
    });
  });

  it("rejects with the server's message", async () => {
    mockFetch(503, { error: "server is shutting down" });
    await expect(
      startDownload(video, { type: "audio", format: "mp3" }),
    ).rejects.toThrow("server is shutting down");
  });

  it("falls back to the status text for non-JSON errors", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          new Response("Bad Gateway", {
            status: 502,
            statusText: "Bad Gateway",
          }),
        ),
    );
    await expect(
      startDownload(video, { type: "audio", format: "mp3" }),
    ).rejects.toThrow("Bad Gateway");
  });
});

describe("cancelDownload", () => {
  it("sends DELETE for the download", async () => {
    const fetchMock = mockFetch(200, {
      uuid: "abc",
      message: "download cancelled",
    });
    await cancelDownload("abc");
    expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/download/abc");
    expect(fetchMock.mock.calls[0][1].method).toBe("DELETE");
  });
});

describe("isValidOptions", () => {
  it.each([
    [{ type: "video", format: "mp4" }, true],
    [{ type: "audio", format: "opus" }, true],
    [{ type: "audio", format: "mp4" }, false],
    [{ type: "playlist", format: "mp4" }, false],
    [{ type: "video" }, false],
    ["video", false],
    [null, false],
  ])("%j → %s", (value, want) => {
    expect(isValidOptions(value)).toBe(want);
  });
});
