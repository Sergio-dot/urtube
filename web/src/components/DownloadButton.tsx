import { useState } from "react";
import type { DownloadStatus } from "../types";

interface DownloadButtonProps {
  status: DownloadStatus;
  onClick: () => void;
  onCancel?: () => void;
}

export default function DownloadButton({
  status,
  onClick,
  onCancel,
}: DownloadButtonProps) {
  const [isHovered, setIsHovered] = useState(false);

  const getIcon = () => {
    switch (status) {
      case "downloading":
      case "loading":
        if (isHovered && onCancel) {
          // X Icon for Cancel
          return (
            <svg
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
              strokeWidth={1.5}
              stroke="currentColor"
              className="size-6 text-red-400"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M6 18 18 6M6 6l12 12"
              />
            </svg>
          );
        }
        // Spinner
        return (
          <div className="relative flex items-center justify-center">
            <svg
              className="animate-spin size-6 text-indigo-500"
              xmlns="http://www.w3.org/2000/svg"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                className="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                strokeWidth="4"
              ></circle>
              <path
                className="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>
        );
      case "success":
        return (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
            className="size-6 text-green-400"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="m4.5 12.75 6 6 9-13.5"
            />
          </svg>
        );
      case "error":
        return (
          <svg
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            strokeWidth={1.5}
            stroke="currentColor"
            className="size-6 text-red-500"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z"
            />
          </svg>
        );
      default:
        return (
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
              d="M19.5 13.5 12 21m0 0-7.5-7.5M12 21V3"
            />
          </svg>
        );
    }
  };

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (status === "loading" && onCancel) {
      onCancel();
    } else if (status === "idle" || status === "error") {
      onClick();
    }
  };

  return (
    <button
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      onClick={handleClick}
      disabled={status === "success"}
      className={`rounded-2xl px-1 py-1 text-xs font-semibold ring-1 ring-inset transition-all cursor-pointer ${
        status === "success"
          ? "text-green-400 ring-green-500/30"
          : status === "error"
            ? "text-red-400 ring-red-500/30"
            : "text-indigo-400 ring-indigo-500/30 hover:text-white"
      } ${status === "loading" && isHovered ? "hover:ring-red-500/30" : ""}`}
      title={
        status === "loading"
          ? isHovered
            ? "Cancel Download"
            : "Downloading..."
          : status === "success"
            ? "Download Complete"
            : status === "error"
              ? "Download Failed - Retry"
              : "Download Video"
      }
    >
      {getIcon()}
    </button>
  );
}
