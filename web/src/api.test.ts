import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  cancelDownload,
  getSettings,
  isValidOptions,
  searchVideos,
  startDownload,
  suggestTags,
  testJellyfin,
  updateSettings,
  type Video,
} from "./api";

function mockFetch(status: number, body: unknown) {
  const fetchMock = vi.fn().mockResolvedValue(
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
      source: "videos",
    });
    expect(results).toEqual([video]);
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/v1/search/a%2Fb%20c?limit=10&wantLiveStreams=true&source=videos",
    );
  });

  it("resolves to no results on 404", async () => {
    mockFetch(404, { error: "no results found" });
    await expect(
      searchVideos({
        query: "x",
        limit: 5,
        includeLive: false,
        source: "videos",
      }),
    ).resolves.toEqual([]);
  });

  it("rejects with the server's message", async () => {
    mockFetch(500, { error: "internal server error" });
    await expect(
      searchVideos({
        query: "x",
        limit: 5,
        includeLive: false,
        source: "videos",
      }),
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
      vi.fn().mockResolvedValue(
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

describe("settings", () => {
  it("loads the settings", async () => {
    const settings = {
      downloadDir: "/downloads",
      defaultDownloadDir: "/downloads",
      downloadDirError: "/downloads is not writable: permission denied",
    };
    const fetchMock = mockFetch(200, settings);
    await expect(getSettings()).resolves.toEqual(settings);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/settings");
  });

  it("saves the download folder", async () => {
    const fetchMock = mockFetch(200, {
      downloadDir: "/media/music",
      defaultDownloadDir: "/downloads",
    });
    await updateSettings({ downloadDir: "/media/music" });
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/settings");
    expect(init.method).toBe("PUT");
    expect(JSON.parse(init.body)).toEqual({ downloadDir: "/media/music" });
  });

  it("rejects with the reason a folder can't be used", async () => {
    mockFetch(400, { error: "/x is not writable: permission denied" });
    await expect(updateSettings({ downloadDir: "/x" })).rejects.toThrow(
      "/x is not writable: permission denied",
    );
  });
});

describe("tags", () => {
  it("startDownload sends tags", async () => {
    const fetchMock = mockFetch(202, { uuid: "abc" });
    await startDownload(
      video,
      { type: "audio", format: "mp3" },
      { artist: "Queen", title: "Song", album: "" },
    );
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).tags).toEqual({
      artist: "Queen",
      title: "Song",
      album: "",
    });
  });

  it("suggestTags encodes the URL", async () => {
    const fetchMock = mockFetch(200, {
      tags: { artist: "Queen", title: "Song", album: "" },
      knownArtists: [],
    });
    await suggestTags("https://www.youtube.com/watch?v=x&t=1");
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/v1/tags?url=https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3Dx%26t%3D1",
    );
  });
});

describe("music search", () => {
  it("sends the source", async () => {
    const fetchMock = mockFetch(200, []);
    await searchVideos({
      query: "queen",
      limit: 20,
      includeLive: false,
      source: "music",
    });
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/v1/search/queen?limit=20&wantLiveStreams=false&source=music",
    );
  });
});

describe("jellyfin", () => {
  it("sends only the Jellyfin change", async () => {
    const fetchMock = mockFetch(200, {});
    await updateSettings({
      jellyfin: { url: "http://jellyfin:8096", apiKey: "k" },
    });
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({
      jellyfin: { url: "http://jellyfin:8096", apiKey: "k" },
    });
  });

  it("tests a connection", async () => {
    const fetchMock = mockFetch(200, {
      message: "Connected to Home (Jellyfin 10.11.2)",
    });
    await expect(testJellyfin("http://j")).resolves.toBe(
      "Connected to Home (Jellyfin 10.11.2)",
    );
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/api/v1/settings/jellyfin/test");
    expect(JSON.parse(init.body)).toEqual({ url: "http://j" });
  });

  it("rejects with the reason", async () => {
    mockFetch(502, { error: "Jellyfin rejected the API key" });
    await expect(testJellyfin("http://j", "bad")).rejects.toThrow(
      "Jellyfin rejected the API key",
    );
  });
});
