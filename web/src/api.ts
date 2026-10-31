// The server's JSON API. The types mirror internal/httpapi/dto.go: keep the
// two in sync.

export interface Thumbnail {
  url: string;
  width?: number;
  height?: number;
}

export interface Video {
  id: string;
  title: string;
  uploader: string;
  /** Whole seconds; zero for live streams. */
  duration: number;
  thumbnails: Thumbnail[];
  url: string;
  isLive: boolean;
}

export type MediaType = "video" | "audio";

/** The formats the server accepts for each media type. */
export const FORMATS: Record<MediaType, readonly string[]> = {
  video: ["mp4", "mkv", "webm"],
  audio: ["mp3", "m4a", "wav", "opus"],
};

export interface DownloadOptions {
  type: MediaType;
  format: string;
}

export function isValidOptions(value: unknown): value is DownloadOptions {
  if (typeof value !== "object" || value === null) return false;
  const { type, format } = value as Record<string, unknown>;
  return (
    (type === "video" || type === "audio") &&
    typeof format === "string" &&
    FORMATS[type].includes(format)
  );
}

/** A download's status as reported by the server. */
export type ServerStatus =
  | "queued"
  | "downloading"
  | "finished"
  | "error"
  | "cancelled";

/**
 * One message on the event stream. Progress values are preformatted for
 * display and empty when unknown.
 */
export interface DownloadEvent {
  uuid: string;
  videoId: string;
  title: string;
  status: ServerStatus;
  errorMessage?: string;
  percent: string;
  speed: string;
  eta: string;
  downloaded: string;
  total: string;
}

export interface Health {
  status: "ok" | "degraded";
  dependencies: { ytdlp: boolean; version?: string };
}

export interface Settings {
  downloadDir: string;
  defaultDownloadDir: string;
  /** The folder a download folder must be inside, if any. */
  downloadRoot?: string;
  /** Why downloads can't be saved right now, if they can't. */
  downloadDirError?: string;
}

export const EVENTS_URL = "/api/v1/events";

/** An error response from the API, with the server's message. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/v1${path}`, init);
  const body = await response.json().catch(() => null);
  if (!response.ok) {
    const message =
      typeof body?.error === "string" ? body.error : response.statusText;
    throw new ApiError(response.status, message || "Request failed");
  }
  return body as T;
}

export interface SearchParams {
  query: string;
  limit: number;
  includeLive: boolean;
}

/** Searches YouTube. A search without matches resolves to an empty list. */
export async function searchVideos(
  { query, limit, includeLive }: SearchParams,
  signal?: AbortSignal,
): Promise<Video[]> {
  const params = new URLSearchParams({
    limit: String(limit),
    wantLiveStreams: String(includeLive),
  });
  try {
    return await request<Video[]>(
      `/search/${encodeURIComponent(query)}?${params}`,
      { signal },
    );
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) return [];
    throw err;
  }
}

/** Starts downloading video and resolves to the new download's ID. */
export async function startDownload(
  video: Video,
  options: DownloadOptions,
): Promise<string> {
  const { uuid } = await request<{ uuid: string }>("/download", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      url: video.url,
      videoId: video.id,
      title: video.title,
      options,
    }),
  });
  return uuid;
}

export async function cancelDownload(id: string): Promise<void> {
  await request(`/download/${encodeURIComponent(id)}`, { method: "DELETE" });
}

export function getHealth(): Promise<Health> {
  return request<Health>("/health");
}

export function getSettings(): Promise<Settings> {
  return request<Settings>("/settings");
}

/**
 * Changes the download folder; an empty string restores the default. Rejects
 * with the reason if the folder can't be used.
 */
export function updateSettings(downloadDir: string): Promise<Settings> {
  return request<Settings>("/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ downloadDir }),
  });
}
