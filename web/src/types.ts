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
  value: any;
}
