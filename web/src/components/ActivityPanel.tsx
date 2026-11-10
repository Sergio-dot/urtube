import {
  CheckCircleIcon,
  DocumentDuplicateIcon,
  ExclamationCircleIcon,
  MinusCircleIcon,
  XMarkIcon,
} from "@heroicons/react/20/solid";
import type { Video } from "../api";
import { isActive, isTerminal, type Download } from "../downloads";
import { percentValue } from "../format";
import { libraryPath } from "../library";

interface ActivityPanelProps {
  /** Newest first. */
  downloads: Download[];
  onCancel: (uuid: string) => void;
  onDismiss: (uuid: string) => void;
  /** Videos set aside to download together. */
  selected: Video[];
  onDeselect: (videoId: string) => void;
  onClearSelected: () => void;
  onDownloadSelected: () => void;
}

/**
 * Everything in motion: the server's downloads with live progress, and the
 * videos picked to download together.
 */
export default function ActivityPanel({
  downloads,
  onCancel,
  onDismiss,
  selected,
  onDeselect,
  onClearSelected,
  onDownloadSelected,
}: ActivityPanelProps) {
  const active = downloads.filter((d) => isActive(d.status));
  const ended = downloads.filter((d) => isTerminal(d.status));
  const activeCount = active.length;
  // What is still running stays on top, where its progress is in view.
  const ordered = [...active, ...ended];

  return (
    <div className="space-y-8">
      <section aria-labelledby="downloads-heading">
        <div className="flex min-h-11 items-center justify-between gap-2">
          <h2 id="downloads-heading" className="text-sm font-semibold">
            Downloads
            {activeCount > 0 && (
              <span className="ml-2 font-normal text-muted">
                {activeCount} active
              </span>
            )}
          </h2>
          {ended.length > 0 && (
            <button
              type="button"
              onClick={() => ended.forEach((d) => onDismiss(d.uuid))}
              className="btn btn-ghost -mr-3 h-9 px-3 text-xs"
            >
              Clear finished
            </button>
          )}
        </div>
        {downloads.length === 0 ? (
          <p className="help rounded-xl border border-dashed border-line-strong p-4">
            Nothing downloading. Downloads appear here with live progress,
            including ones started from another device.
          </p>
        ) : (
          <ul className="space-y-2">
            {ordered.map((download) => (
              <DownloadItem
                key={download.uuid}
                download={download}
                onCancel={onCancel}
                onDismiss={onDismiss}
              />
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="selected-heading">
        <div className="flex min-h-11 items-center justify-between gap-2">
          <h2 id="selected-heading" className="text-sm font-semibold">
            Selected
            {selected.length > 0 && (
              <span className="ml-2 font-normal text-muted">
                {selected.length}
              </span>
            )}
          </h2>
          {selected.length > 0 && (
            <button
              type="button"
              onClick={onClearSelected}
              className="btn btn-ghost -mr-3 h-9 px-3 text-xs"
            >
              Clear
            </button>
          )}
        </div>
        {selected.length === 0 ? (
          <p className="help rounded-xl border border-dashed border-line-strong p-4">
            Use the + button on a result to collect videos across searches, then
            download them all with one set of options.
          </p>
        ) : (
          <>
            <ul className="divide-y divide-line rounded-xl border border-line bg-surface">
              {selected.map((video) => (
                <li
                  key={video.id}
                  className="flex items-center gap-2 py-1 pr-1 pl-3"
                >
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">
                      {video.title}
                    </p>
                    <p className="truncate text-xs text-muted">
                      {video.uploader}
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={() => onDeselect(video.id)}
                    aria-label={`Remove ${video.title} from the selection`}
                    title="Remove"
                    className="icon-btn"
                  >
                    <XMarkIcon aria-hidden="true" className="size-5" />
                  </button>
                </li>
              ))}
            </ul>
            <button
              type="button"
              onClick={onDownloadSelected}
              className="btn btn-primary mt-3 w-full"
            >
              Download{" "}
              {selected.length === 1 ? "1 video" : `all ${selected.length}`}
            </button>
          </>
        )}
      </section>
    </div>
  );
}

function statusLine(d: Download): string {
  switch (d.status) {
    case "finished":
      return "Saved to the library";
    case "error":
      return d.errorMessage || "Failed";
    case "cancelled":
      return "Cancelled";
    case "skipped":
      return d.errorMessage || "Skipped: already in the library";
    case "queued":
      return "Waiting for a free slot";
    case "starting":
      return "Starting…";
    case "downloading": {
      const size =
        d.downloaded && d.total ? `${d.downloaded} / ${d.total}` : "";
      const eta = d.eta ? `${d.eta} left` : "";
      return [size, d.speed, eta].filter(Boolean).join(" · ") || "Starting…";
    }
  }
}

function DownloadItem({
  download,
  onCancel,
  onDismiss,
}: {
  download: Download;
  onCancel: (uuid: string) => void;
  onDismiss: (uuid: string) => void;
}) {
  const { status } = download;
  const active = isActive(status);
  const title = download.title || "Untitled";
  const percent =
    status === "downloading" ? percentValue(download.percent) : undefined;

  return (
    <li className="rounded-xl border border-line bg-surface py-1 pr-1 pl-3">
      <div className="flex items-start gap-2">
        {/* State is carried by the icon's shape and the text, not just colour. */}
        {status === "finished" && (
          <CheckCircleIcon
            aria-hidden="true"
            className="mt-3 size-5 shrink-0 text-ok"
          />
        )}
        {status === "error" && (
          <ExclamationCircleIcon
            aria-hidden="true"
            className="mt-3 size-5 shrink-0 text-danger"
          />
        )}
        {status === "cancelled" && (
          <MinusCircleIcon
            aria-hidden="true"
            className="mt-3 size-5 shrink-0 text-muted"
          />
        )}
        {status === "skipped" && (
          <DocumentDuplicateIcon
            aria-hidden="true"
            className="mt-3 size-5 shrink-0 text-muted"
          />
        )}
        <div className="min-w-0 flex-1 py-2">
          <p className="truncate text-sm font-medium" title={title}>
            {title}
          </p>
          <p
            className={`text-xs tabular-nums ${
              status === "error"
                ? "break-words text-danger"
                : status === "skipped"
                  ? "break-words text-muted"
                  : "truncate text-muted"
            }`}
          >
            {statusLine(download)}
          </p>
          {/* Only a hint: the download went ahead, and the user decides. */}
          {status !== "skipped" && download.similar?.length ? (
            <p className="mt-1 text-xs break-words text-muted">
              <span className="font-medium text-warn">
                You may already have this song:
              </span>{" "}
              {download.similar.map(libraryPath).join(", ")}
            </p>
          ) : null}
        </div>
        {percent !== undefined && (
          <span className="py-2 text-sm font-semibold tabular-nums">
            {Math.round(percent)}%
          </span>
        )}
        <button
          type="button"
          onClick={() => (active ? onCancel : onDismiss)(download.uuid)}
          aria-label={
            active ? `Cancel download of ${title}` : `Dismiss ${title}`
          }
          title={active ? "Cancel download" : "Dismiss"}
          className="icon-btn"
        >
          <XMarkIcon aria-hidden="true" className="size-5" />
        </button>
      </div>
      {active && (
        <div
          role="progressbar"
          aria-label={`Download progress of ${title}`}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={percent}
          className="mr-2 mb-2 h-1.5 overflow-hidden rounded-full bg-raised"
        >
          {percent === undefined ? (
            <div className="h-full w-2/5 animate-indeterminate rounded-full bg-muted/60" />
          ) : (
            <div
              className="h-full rounded-full bg-accent transition-[width] duration-500"
              style={{ width: `${percent}%` }}
            />
          )}
        </div>
      )}
    </li>
  );
}
