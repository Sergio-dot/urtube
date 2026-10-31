import { useCallback } from "react";
import type { Video } from "../api";
import { useLocalStorage } from "./useLocalStorage";

const isVideoList = (value: unknown): value is Video[] =>
  Array.isArray(value) &&
  value.every((v) => typeof v?.id === "string" && typeof v?.url === "string");

/** The videos set aside to download together, kept across reloads. */
export function useQueue() {
  const [videos, setVideos] = useLocalStorage<Video[]>(
    "urtube_video_queue",
    [],
    isVideoList,
  );

  const add = useCallback(
    (video: Video) =>
      setVideos((q) => (q.some((v) => v.id === video.id) ? q : [...q, video])),
    [setVideos],
  );
  const remove = useCallback(
    (videoId: string) => setVideos((q) => q.filter((v) => v.id !== videoId)),
    [setVideos],
  );
  const clear = useCallback(() => setVideos([]), [setVideos]);

  return { videos, add, remove, clear };
}
