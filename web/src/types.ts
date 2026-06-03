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
  | "downloading"
  | "finished"
  | "cancelled";

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
  status: "downloading" | "finished" | "error" | "cancelled";
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
