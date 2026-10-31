export interface Thumbnail {
  url: string;
  width?: number;
  height?: number;
}

export interface Video {
  id: string;
  title: string;
  uploader: string;
  duration: number;
  thumbnails: Thumbnail[];
  url: string;
  isLive: boolean;
}

export interface SelectOption {
  id: string | number;
  label: string;
  value: unknown;
}

export type DownloadType = "video" | "audio";
export type DownloadStatus =
  | "idle"
  | "loading"
  | "success"
  | "error"
  | "queued"
  | "downloading"
  | "finished"
  | "cancelled";

// A download the server is still working on, so it can be cancelled.
export function isActiveStatus(status: DownloadStatus): boolean {
  return status === "loading" || status === "queued" || status === "downloading";
}

// A download that completed successfully.
export function isDoneStatus(status: DownloadStatus): boolean {
  return status === "success" || status === "finished";
}

export interface DownloadOptions {
  type: DownloadType;
  format: string;
}

export interface YtdlpProgress {
  status: string;
  downloaded_bytes: number;
  total_bytes: number;
  total_bytes_estimate: number;
  percent_string: string;
  eta_string: string;
  speed_string: string;
  filename: string;
}

export interface ProgressUpdate {
  uuid: string;
  videoId: string;
  title: string;
  status: "queued" | "downloading" | "finished" | "error" | "cancelled";
  errorMessage?: string;
  percent: string;
  speed: string;
  eta: string;
  downloaded: string;
  total: string;
}

export interface DownloadState {
  uuid: string;
  videoId?: string;
  title?: string;
  status: DownloadStatus;
  percent?: string;
  speed?: string;
  eta?: string;
  downloaded?: string;
  total?: string;
  errorMessage?: string;
  options?: DownloadOptions;
}
