import type { LibraryMatch, Tags, Video } from "./api";

/** Formats a library path for display, like savedAs does. */
export function libraryPath(path: string): string {
  return path.split("/").join(" / ");
}

/** Whether the library holds anything a download would duplicate. */
export function hasMatch(match: LibraryMatch | undefined): boolean {
  return Boolean(match?.existing || match?.similar?.length);
}

/**
 * The videos of a batch that can be checked against the library before
 * downloading: those whose tags are already known (music results). The
 * others get their tags when the download runs, which checks then.
 */
export function checkable(videos: Video[]): { video: Video; tags: Tags }[] {
  return videos.flatMap((video) =>
    video.tags ? [{ video, tags: video.tags }] : [],
  );
}

export interface Flagged {
  video: Video;
  match: LibraryMatch;
}

/** Pairs checked videos with their matches, keeping those with any. */
export function flagged(checked: Video[], matches: LibraryMatch[]): Flagged[] {
  return checked.flatMap((video, i) =>
    hasMatch(matches[i]) ? [{ video, match: matches[i] }] : [],
  );
}

/**
 * Which flagged videos a batch leaves out unless the user says otherwise:
 * those already in the library, which the server would skip anyway. Similar
 * songs may be other versions the user wants, so they stay in.
 */
export function leftOutByDefault(items: Flagged[]): Set<string> {
  return new Set(
    items.filter(({ match }) => match.existing).map(({ video }) => video.id),
  );
}
