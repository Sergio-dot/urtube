import type { Tags } from "./api";

/**
 * Where a download will be saved, for display: Artist / Album / Title for
 * audio (without the album when there is none), just the title for video.
 * The server may reuse an existing folder spelled differently.
 */
export function savedAs(tags: Tags, audio: boolean, format: string): string {
  const title = tags.title.trim() || "Untitled";
  const parts = audio
    ? [tags.artist.trim() || "Unknown Artist", tags.album.trim(), title]
    : [title];
  return `${parts.filter(Boolean).join(" / ")}.${format}`;
}
