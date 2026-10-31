import {
  ChevronDownIcon,
  ChevronUpIcon,
  XMarkIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
} from "@heroicons/react/24/outline";
import { useState } from "react";
import { isActive, type Download, type DownloadStatus } from "../downloads";

interface DownloadStatusOverlayProps {
  /** Newest first. */
  downloads: Download[];
  onDismiss: (uuid: string) => void;
  onCancel: (uuid: string) => void;
}

const barColor: Record<DownloadStatus, string> = {
  starting: "bg-indigo-500",
  queued: "bg-indigo-500",
  downloading: "bg-indigo-500",
  finished: "bg-green-500",
  error: "bg-red-500",
  cancelled: "bg-gray-400",
};

function statusLine(dl: Download): string {
  switch (dl.status) {
    case "finished":
      return "Completed";
    case "error":
      return dl.errorMessage || "Failed";
    case "cancelled":
      return "Cancelled";
    case "queued":
      return "Queued";
    default:
      return dl.speed || "Starting…";
  }
}

export default function DownloadStatusOverlay({
  downloads,
  onDismiss,
  onCancel,
}: DownloadStatusOverlayProps) {
  const [isMinimized, setIsMinimized] = useState(true);

  if (downloads.length === 0) return null;

  const activeCount = downloads.filter((d) => isActive(d.status)).length;

  return (
    <div className="fixed bottom-4 left-4 z-100 w-80 overflow-hidden rounded-lg border bg-white shadow-2xl transition-all duration-300 dark:border-white/10 dark:bg-gray-900">
      {/* Header */}
      <div className="flex items-center justify-between border-b bg-gray-50 p-3 dark:border-white/10 dark:bg-white/5">
        <h3 className="text-sm font-semibold dark:text-white">
          Downloads {activeCount > 0 && `(${activeCount} active)`}
        </h3>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={() => setIsMinimized(!isMinimized)}
            aria-label={isMinimized ? "Show downloads" : "Hide downloads"}
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
          {downloads.map((dl) => {
            const active = isActive(dl.status);
            const width =
              dl.status === "finished" || dl.status === "cancelled"
                ? "100%"
                : dl.percent || "0%";

            return (
              <div
                key={dl.uuid}
                className="mb-2 last:mb-0 rounded-lg border border-gray-100 bg-white p-3 dark:border-white/5 dark:bg-white/5"
              >
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium dark:text-white">
                      {dl.title || "Untitled"}
                    </p>
                    <p className="truncate text-[10px] text-gray-500">
                      {statusLine(dl)}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() =>
                      active ? onCancel(dl.uuid) : onDismiss(dl.uuid)
                    }
                    className="rounded-md p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-white/10"
                    aria-label={active ? "Cancel download" : "Dismiss"}
                    title={active ? "Cancel download" : "Dismiss"}
                  >
                    <XMarkIcon className="h-4 w-4" />
                  </button>
                </div>

                {/* Progress and Icons */}
                <div className="flex items-center gap-3">
                  <div className="flex-1">
                    <div className="h-1.5 w-full rounded-full bg-gray-100 dark:bg-gray-800">
                      <div
                        className={`h-full rounded-full transition-all duration-500 ${barColor[dl.status]}`}
                        style={{ width }}
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
                  <span>{dl.eta || "--"}</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
