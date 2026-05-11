import type { Video } from "../types";

interface SearchResultsProps {
  results: Video[];
}

export default function SearchResults({ results }: SearchResultsProps) {
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
        {results.map((video) => (
          <div
            key={video.id}
            className="group relative flex flex-col overflow-hidden rounded-2xl bg-white/5 ring-1 ring-inset ring-white/10 hover:bg-white/10 transition-all cursor-pointer"
          >
            <div className="aspect-video w-full overflow-hidden bg-gray-800">
              <img
                src={video.thumbnails[video.thumbnails.length - 1]?.url}
                alt={video.title}
                className="h-full w-full object-cover transition-transform group-hover:scale-105"
              />
              <div className="absolute top-2 right-2 rounded bg-black/80 px-2 py-1 text-xs font-medium text-white">
                {formatDuration(video.duration)}
              </div>
            </div>
            <div className="flex flex-1 flex-col p-4">
              <h3 className="text-sm font-semibold text-white line-clamp-2">
                {video.title}
              </h3>
              <p className="mt-1 text-xs text-gray-400">{video.uploader}</p>

              <div className="mt-4 flex items-center justify-between">
                <button className="rounded-lg bg-indigo-600/20 px-3 py-1.5 text-xs font-semibold text-indigo-400 ring-1 ring-inset ring-indigo-500/30 hover:bg-indigo-600 hover:text-white transition-all">
                  Download
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
