import { XMarkIcon, TrashIcon } from "@heroicons/react/24/outline";
import type { Video } from "../types";

interface CollectionPanelProps {
  isOpen: boolean;
  onClose: () => void;
  videos: Video[];
  onRemoveVideo: (videoId: string) => void;
  onClearCollection: () => void;
  onDownloadAll: () => void;
}

export default function CollectionPanel({
  isOpen,
  onClose,
  videos,
  onRemoveVideo,
  onClearCollection,
  onDownloadAll,
}: CollectionPanelProps) {
  return (
    <>
      {/* Overlay */}
      <div
        className={`fixed inset-0 z-40 bg-black/50 transition-opacity duration-300 ${
          isOpen ? "opacity-100" : "pointer-events-none opacity-0"
        }`}
        onClick={onClose}
      />

      {/* Panel */}
      <div
        className={`fixed inset-y-0 right-0 z-50 w-full max-w-sm transform bg-white shadow-xl transition-transform duration-300 ease-in-out dark:bg-gray-900 ${
          isOpen ? "translate-x-0" : "translate-x-full"
        }`}
      >
        <div className="flex h-full flex-col">
          <div className="flex items-center justify-between border-b p-4 dark:border-white/10">
            <h2 className="text-lg font-semibold dark:text-white">
              Collection ({videos.length})
            </h2>
            <button
              onClick={onClose}
              className="rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-white/5 dark:hover:text-white"
            >
              <XMarkIcon className="h-6 w-6" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4">
            {videos.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center text-gray-500">
                <p>Your collection is empty.</p>
                <p className="text-sm">Add videos from search results.</p>
              </div>
            ) : (
              <ul className="space-y-3">
                {videos.map((video) => (
                  <li
                    key={video.id}
                    className="flex items-start justify-between gap-2 rounded-lg bg-gray-50 p-3 dark:bg-white/5"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium dark:text-white">
                        {video.title}
                      </p>
                      <p className="truncate text-xs text-gray-500">
                        {video.uploader}
                      </p>
                    </div>
                    <button
                      onClick={() => onRemoveVideo(video.id)}
                      className="rounded-md p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                    >
                      <TrashIcon className="h-4 w-4" />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {videos.length > 0 && (
            <div className="border-t p-4 dark:border-white/10">
              <button
                onClick={onClearCollection}
                className="w-full rounded-md border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 dark:border-white/10 dark:text-gray-300 dark:hover:bg-white/5"
              >
                Clear All
              </button>
              <button
                onClick={onDownloadAll}
                className="mt-2 w-full rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500"
              >
                Download All
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
