import {
  ChevronDownIcon,
  ChevronUpIcon,
  XMarkIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import { useState } from "react";
import type { DownloadState } from "../types";

interface DownloadStatusOverlayProps {
  downloads: DownloadState[];
  onRemove: (uuid: string) => void;
}

export default function DownloadStatusOverlay({
  downloads,
  onRemove,
}: DownloadStatusOverlayProps) {
  const [isMinimized, setIsMinimized] = useState(false);

  if (downloads.length === 0) return null;

  const activeCount = downloads.filter(
    (d) => d.status === "downloading" || d.status === "loading",
  ).length;

  const truncateSeconds = (durationStr: string | undefined) => {
    return (durationStr || "--").replace(/(\d+)\.(\d+)s$/, "$1s");
  };

  return (
    <div className="fixed bottom-4 left-4 z-100 w-80 overflow-hidden rounded-lg border bg-white shadow-2xl transition-all duration-300 dark:border-white/10 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between border-b bg-gray-50 p-3 dark:border-white/10 dark:bg-white/5">
        <h3 className="text-sm font-semibold dark:text-white">
          Downloads {activeCount > 0 && `(${activeCount} active)`}
        </h3>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setIsMinimized(!isMinimized)}
            className="rounded-md p-1 text-gray-500 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-white/10"
          >
            {isMinimized ? (
              <ChevronUpIcon className="h-4 w-4" />
            ) : (
              <ChevronDownIcon className="h-4 w-4" />
            )}
          </button>
        </div>
      </div>

      {/* List */}
      {!isMinimized && (
        <div className="max-h-96 overflow-y-auto p-2">
          {downloads.map((dl) => (
            <div
              key={dl.uuid}
              className="mb-2 last:mb-0 rounded-lg border border-gray-100 bg-white p-3 dark:border-white/5 dark:bg-white/5"
            >
              <div className="flex items-start justify-between gap-2 mb-2">
                <div className="min-w-0 flex-1">
                  <p className="truncate text-xs font-medium dark:text-white">
                    {dl.title || "Downloading..."}
                  </p>
                  <p className="truncate text-[10px] text-gray-500">
                    {dl.status === "finished"
                      ? "Completed"
                      : dl.status === "error"
                        ? dl.errorMessage
                        : dl.speed || "Starting..."}
                  </p>
                </div>
                <button
                  onClick={() => onRemove(dl.uuid)}
                  className="rounded-md p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-white/10"
                >
                  <XMarkIcon className="h-4 w-4" />
                </button>
              </div>

              {/* Progress and Icons */}
              <div className="flex items-center gap-3">
                <div className="flex-1">
                  <div className="h-1.5 w-full rounded-full bg-gray-100 dark:bg-gray-800">
                    <div
                      className={`h-full rounded-full transition-all duration-500 ${
                        dl.status === "finished"
                          ? "bg-green-500"
                          : dl.status === "error"
                            ? "bg-red-500"
                            : "bg-indigo-500"
                      }`}
                      style={{
                        width:
                          dl.status === "finished"
                            ? "100%"
                            : dl.percent || "0%",
                      }}
                    />
                  </div>
                </div>
                {dl.status === "finished" && (
                  <CheckCircleIcon className="h-4 w-4 text-green-500 shrink-0" />
                )}
                {dl.status === "error" && (
                  <ExclamationCircleIcon className="h-4 w-4 text-red-500 shrink-0" />
                )}
                {dl.status === "downloading" && (
                  <span className="text-[10px] font-medium tabular-nums text-gray-600 dark:text-gray-400">
                    {dl.percent || "0%"}
                  </span>
                )}
              </div>

              <div className="mt-1 flex justify-between text-[10px] text-gray-500">
                <span>
                  {dl.status === "downloading" && dl.downloaded && dl.total
                    ? `${dl.downloaded} / ${dl.total}`
                    : dl.speed || "--"}
                </span>
                <span>{truncateSeconds(dl.eta) || "--"}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
