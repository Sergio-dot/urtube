import { CheckIcon, PlusIcon } from "@heroicons/react/24/outline";
import type { Video } from "../api";
import type { Download } from "../downloads";
import { formatDuration } from "../format";
import DownloadButton from "./DownloadButton";

interface SearchResultsProps {
  results: Video[];
  /** The download each video's card reflects, by video ID. */
  downloads: Record<string, Download>;
  isQueued: (videoId: string) => boolean;
  onQueue: (video: Video) => void;
  onUnqueue: (videoId: string) => void;
  onDownload: (video: Video) => void;
  onCancelDownload: (downloadId: string) => void;
}

export default function SearchResults({
  results,
  downloads,
  isQueued,
  onQueue,
  onUnqueue,
  onDownload,
  onCancelDownload,
}: SearchResultsProps) {
  return (
    <div className="w-full max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {results.map((video) => {
          const queued = isQueued(video.id);
          const download = downloads[video.id];

          return (
            <div
              key={video.id}
              className="group relative flex flex-col overflow-hidden rounded-2xl bg-white/5 ring-1 ring-inset dark:ring-white/10 ring-black/10 hover:bg-white/10 transition-all"
            >
              <div className="aspect-video w-full overflow-hidden bg-gray-800 border">
                <img
                  src={video.thumbnails.at(-1)?.url}
                  alt={video.title}
                  loading="lazy"
                  className="h-full w-full object-cover transition-transform group-hover:scale-105"
                />
                {video.isLive ? (
                  <div className="absolute top-2 right-2 flex items-center gap-1.5 rounded bg-black/80 px-2 py-1">
                    <span className="relative flex h-2 w-2">
                      <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-500 opacity-75" />
                      <span className="relative inline-flex h-2 w-2 rounded-full bg-red-500" />
                    </span>
                    <span className="text-xs font-bold text-white tracking-wide">
                      Live
                    </span>
                  </div>
                ) : (
                  <div className="absolute top-2 right-2 rounded bg-black/80 px-2 py-1 text-xs font-medium text-white">
                    {formatDuration(video.duration)}
                  </div>
                )}
              </div>
              <div className="flex flex-1 flex-col p-4">
                <h3 className="text-sm font-semibold dark:text-white line-clamp-2">
                  {video.title}
                </h3>
                <p className="mt-1 text-xs text-gray-400">
                  {video.album
                    ? `${video.uploader} · ${video.album}`
                    : video.uploader}
                </p>
                <div className="flex mt-auto justify-end gap-1">
                  <DownloadButton
                    status={download?.status ?? "idle"}
                    onStart={() => onDownload(video)}
                    onCancel={
                      download
                        ? () => onCancelDownload(download.uuid)
                        : undefined
                    }
                  />
                  <button
                    type="button"
                    onClick={() =>
                      queued ? onUnqueue(video.id) : onQueue(video)
                    }
                    title={queued ? "Remove from queue" : "Add to queue"}
                    aria-label={queued ? "Remove from queue" : "Add to queue"}
                    className={`rounded-2xl px-1 py-1 text-xs font-semibold ring-1 ring-inset transition-all cursor-pointer ${
                      queued
                        ? "text-green-400 ring-green-500/30"
                        : "text-indigo-400 ring-indigo-500/30 hover:text-white"
                    }`}
                  >
                    {queued ? (
                      <CheckIcon className="size-6" />
                    ) : (
                      <PlusIcon className="size-6" />
                    )}
                  </button>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
