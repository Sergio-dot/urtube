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
export type DownloadStatus = "idle" | "loading" | "success" | "error";

export interface DownloadOptions {
  type: DownloadType;
  format: string;
}

export interface DownloadState {
  videoId: string;
  status: DownloadStatus;
  progress?: number;
  errorMessage?: string;
  options?: DownloadOptions;
}
