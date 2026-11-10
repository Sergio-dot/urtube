import {
  ArrowDownTrayIcon,
  ArrowPathIcon,
  CheckIcon,
  PlusIcon,
  XMarkIcon,
} from "@heroicons/react/20/solid";
import type { Video } from "../api";
import { isActive, type Download } from "../downloads";
import { formatDuration, percentValue } from "../format";
import Spinner from "./Spinner";

/** Videos read best as a grid of thumbnails, songs as a compact list. */
export type ResultsLayout = "grid" | "list";

interface ResultsProps {
  results: Video[];
  layout: ResultsLayout;
  /** The download each video reflects, by video ID. */
  downloads: Record<string, Download>;
  isSelected: (videoId: string) => boolean;
  onSelect: (video: Video) => void;
  onDeselect: (videoId: string) => void;
  onDownload: (video: Video) => void;
  onCancelDownload: (downloadId: string) => void;
}

export default function Results({
  results,
  layout,
  downloads,
  isSelected,
  onSelect,
  onDeselect,
  onDownload,
  onCancelDownload,
}: ResultsProps) {
  const Item = layout === "grid" ? Card : Row;
  return (
    <ul
      className={
        layout === "grid"
          ? "grid grid-cols-[repeat(auto-fill,minmax(min(100%,15rem),1fr))] gap-4"
          : "divide-y divide-line overflow-hidden rounded-xl border border-line bg-surface"
      }
    >
      {results.map((video) => {
        const download = downloads[video.id];
        const selected = isSelected(video.id);
        return (
          <Item
            key={video.id}
            video={video}
            download={download}
            actions={
              <Actions
                video={video}
                download={download}
                selected={selected}
                compact={layout === "list"}
                onToggleSelect={() =>
                  selected ? onDeselect(video.id) : onSelect(video)
                }
                onDownload={() => onDownload(video)}
                onCancel={() => download && onCancelDownload(download.uuid)}
              />
            }
          />
        );
      })}
    </ul>
  );
}

interface ItemProps {
  video: Video;
  download: Download | undefined;
  actions: React.ReactNode;
}

const byline = (video: Video) =>
  video.album ? `${video.uploader} · ${video.album}` : video.uploader;

function Card({ video, download, actions }: ItemProps) {
  const downloading = download?.status === "downloading";
  return (
    <li className="flex flex-col overflow-hidden rounded-xl border border-line bg-surface">
      <div className="relative aspect-video bg-raised">
        {/* Decorative: the title is right below. */}
        <img
          src={video.thumbnails.at(-1)?.url}
          alt=""
          loading="lazy"
          className="size-full object-cover"
        />
        <DurationBadge video={video} className="absolute right-2 bottom-2" />
        {downloading && (
          <div
            aria-hidden="true"
            className="absolute inset-x-0 bottom-0 h-1 bg-black/50"
          >
            <div
              className="h-full bg-accent transition-[width] duration-500"
              style={{ width: `${percentValue(download.percent) ?? 0}%` }}
            />
          </div>
        )}
      </div>
      <div className="flex flex-1 flex-col gap-1 p-3">
        <h3
          className="line-clamp-2 text-sm leading-snug font-semibold"
          title={video.title}
        >
          {video.title}
        </h3>
        <p className="truncate text-xs text-muted">{byline(video)}</p>
        <div className="mt-auto flex items-center gap-2 pt-3">{actions}</div>
      </div>
    </li>
  );
}

function Row({ video, actions }: ItemProps) {
  return (
    <li className="flex items-center gap-3 p-2.5 sm:px-3">
      <img
        src={video.thumbnails[0]?.url}
        alt=""
        loading="lazy"
        className="size-12 shrink-0 rounded-md bg-raised object-cover"
      />
      <div className="min-w-0 flex-1">
        <h3 className="truncate text-sm font-semibold" title={video.title}>
          {video.title}
        </h3>
        <p className="truncate text-xs text-muted">{byline(video)}</p>
      </div>
      <DurationBadge video={video} className="max-sm:hidden" plain />
      <div className="flex shrink-0 items-center gap-1 sm:gap-2">{actions}</div>
    </li>
  );
}

function DurationBadge({
  video,
  className = "",
  plain = false,
}: {
  video: Video;
  className?: string;
  /** Text only, for use on a surface rather than over a thumbnail. */
  plain?: boolean;
}) {
  const box = plain
    ? "text-muted"
    : "rounded bg-black/80 px-1.5 py-0.5 text-white";
  return (
    <span
      className={`flex items-center gap-1.5 text-xs font-medium tabular-nums ${box} ${className}`}
    >
      {video.isLive ? (
        <>
          <span
            aria-hidden="true"
            className="size-1.5 rounded-full bg-red-500"
          />
          Live
        </>
      ) : (
        formatDuration(video.duration)
      )}
    </span>
  );
}

interface ActionsProps {
  video: Video;
  download: Download | undefined;
  selected: boolean;
  /** Icon-only on narrow screens, where a row has no room for labels. */
  compact: boolean;
  onToggleSelect: () => void;
  onDownload: () => void;
  onCancel: () => void;
}

function Actions({
  video,
  download,
  selected,
  compact,
  onToggleSelect,
  onDownload,
  onCancel,
}: ActionsProps) {
  const labelClass = compact ? "max-sm:sr-only" : "";

  // While the server works on it, the card shows progress and a way out.
  // Cancelling is always visible rather than revealed on hover, which touch
  // screens don't have.
  if (download && isActive(download.status)) {
    const progress =
      download.status === "queued" ? "Queued" : download.percent || "Starting…";
    return (
      <>
        <p
          className={`flex h-11 min-w-0 items-center gap-2 text-sm font-medium text-muted tabular-nums ${compact ? "" : "flex-1"}`}
        >
          <Spinner className="size-4 shrink-0 text-accent-text" />
          <span className="truncate">{progress}</span>
        </p>
        <button
          type="button"
          onClick={onCancel}
          aria-label={`Cancel download of ${video.title}`}
          title="Cancel download"
          className="icon-btn border border-line-strong"
        >
          <XMarkIcon aria-hidden="true" className="size-5" />
        </button>
      </>
    );
  }

  const [Icon, label, style] =
    download?.status === "finished"
      ? [CheckIcon, "Downloaded", "btn-secondary text-ok"]
      : download?.status === "error"
        ? [ArrowPathIcon, "Retry", "btn-secondary text-danger"]
        : [ArrowDownTrayIcon, "Download", "btn-primary"];
  const compactSize = compact ? "max-sm:w-11 max-sm:px-0" : "flex-1";

  return (
    <>
      <button
        type="button"
        onClick={onDownload}
        title={
          download?.status === "finished"
            ? "Downloaded. Download again, for example in another format"
            : download?.status === "error"
              ? download.errorMessage || "Download failed"
              : undefined
        }
        className={`btn ${style} ${compactSize}`}
      >
        <Icon aria-hidden="true" className="size-5" />
        <span className={labelClass}>{label}</span>
        <span className="sr-only">: {video.title}</span>
      </button>
      <button
        type="button"
        onClick={onToggleSelect}
        aria-pressed={selected}
        aria-label={`Select ${video.title} to download later`}
        title={
          selected ? "Selected: click to remove" : "Select to download later"
        }
        className={`icon-btn border ${
          selected
            ? "border-accent-text bg-accent/10 text-accent-text hover:bg-accent/20 hover:text-accent-text"
            : "border-line-strong"
        }`}
      >
        {selected ? (
          <CheckIcon aria-hidden="true" className="size-5" />
        ) : (
          <PlusIcon aria-hidden="true" className="size-5" />
        )}
      </button>
    </>
  );
}
