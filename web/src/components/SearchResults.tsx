import type { Video, DownloadState } from "../types";
import DownloadButton from "./DownloadButton";

interface SearchResultsProps {
  results: Video[];
  onAddVideo: (video: Video) => void;
  onRemoveVideo: (id: string) => void;
  onDownloadVideo: (video: Video) => void;
  onCancelDownload: (id: string) => void;
  queue: Video[];
  downloadStates: Record<string, DownloadState>;
}

export default function SearchResults({
  results,
  onAddVideo,
  onRemoveVideo,
  onDownloadVideo,
  onCancelDownload,
  queue,
  downloadStates,
}: SearchResultsProps) {
  const formatDuration = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    const parts = [];
    if (hrs > 0) parts.push(hrs);
    parts.push(hrs > 0 ? mins.toString().padStart(2, "0") : mins);
    parts.push(secs.toString().padStart(2, "0"));
    return parts.join(":");
  };

  if (results.length === 0) return null;

  return (
    <div className="w-full max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {results.map((video) => {
          const duration = formatDuration(video.duration);
          const isLive = duration.includes("NaN");
          const isInQueue = queue.some((v) => v.id === video.id);
          const downloadState = downloadStates[video.id];

          return (
            <div
              key={video.id}
              className="group relative flex flex-col overflow-hidden rounded-2xl bg-white/5 ring-1 ring-inset dark:ring-white/10 ring-black/10 hover:bg-white/10 transition-all"
            >
              <div className="aspect-video w-full overflow-hidden bg-gray-800 border">
                <img
                  src={video.thumbnails[video.thumbnails.length - 1]?.url}
                  alt={video.title}
                  className="h-full w-full object-cover transition-transform group-hover:scale-105"
                />
                {isLive ? (
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
                    {duration}
                  </div>
                )}
              </div>
              <div className="flex flex-1 flex-col p-4">
                <h3 className="text-sm font-semibold dark:text-white line-clamp-2">
                  {video.title}
                </h3>
                <p className="mt-1 text-xs text-gray-400">{video.uploader}</p>
                <div className="flex mt-auto justify-end gap-1">
                  <DownloadButton
                    status={downloadState?.status || "idle"}
                    onClick={() => onDownloadVideo(video)}
                    onCancel={() => onCancelDownload(video.id)}
                  />
                  <button
                    onClick={
                      isInQueue
                        ? () => onRemoveVideo(video.id)
                        : () => onAddVideo(video)
                    }
                    className={`rounded-2xl px-1 py-1 text-xs font-semibold ring-1 ring-inset transition-all cursor-pointer ${
                      isInQueue
                        ? "text-green-400 ring-green-500/30 cursor-default"
                        : "text-indigo-400 ring-indigo-500/30 hover:text-white"
                    }`}
                  >
                    {isInQueue ? (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                        strokeWidth={1.5}
                        stroke="currentColor"
                        className="size-6"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="m4.5 12.75 6 6 9-13.5"
                        />
                      </svg>
                    ) : (
                      <svg
                        xmlns="http://www.w3.org/2000/svg"
                        fill="none"
                        viewBox="0 0 24 24"
                        strokeWidth={1.5}
                        stroke="currentColor"
                        className="size-6"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M12 4.5v15m7.5-7.5h-15"
                        />
                      </svg>
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
