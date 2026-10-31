import { useState } from "react";
import {
  ArrowDownIcon,
  CheckIcon,
  ExclamationCircleIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import { isActive, type DownloadStatus } from "../downloads";
import Spinner from "./Spinner";

/** "idle": the video has no download yet. */
export type ButtonStatus = DownloadStatus | "idle";

interface DownloadButtonProps {
  status: ButtonStatus;
  onStart: () => void;
  onCancel?: () => void;
}

export default function DownloadButton({
  status,
  onStart,
  onCancel,
}: DownloadButtonProps) {
  const [isHovered, setIsHovered] = useState(false);

  const active = status !== "idle" && isActive(status);
  const done = status === "finished";
  const showCancel = active && isHovered && onCancel !== undefined;

  const icon = showCancel ? (
    <XMarkIcon className="size-6 text-red-400" />
  ) : active ? (
    <Spinner className="size-6 text-indigo-500" />
  ) : done ? (
    <CheckIcon className="size-6 text-green-400" />
  ) : status === "error" ? (
    <ExclamationCircleIcon className="size-6 text-red-500" />
  ) : (
    <ArrowDownIcon className="size-6" />
  );

  const label = showCancel
    ? "Cancel download"
    : status === "queued"
      ? "Queued"
      : active
        ? "Downloading…"
        : done
          ? "Download complete"
          : status === "error"
            ? "Download failed, retry"
            : "Download";

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (active) {
      onCancel?.();
    } else if (!done) {
      onStart();
    }
  };

  return (
    <button
      type="button"
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      onClick={handleClick}
      disabled={done}
      title={label}
      aria-label={label}
      className={`rounded-2xl px-1 py-1 text-xs font-semibold ring-1 ring-inset transition-all cursor-pointer ${
        done
          ? "text-green-400 ring-green-500/30"
          : status === "error"
            ? "text-red-400 ring-red-500/30"
            : "text-indigo-400 ring-indigo-500/30 hover:text-white"
      } ${showCancel ? "hover:ring-red-500/30" : ""}`}
    >
      {icon}
    </button>
  );
}
