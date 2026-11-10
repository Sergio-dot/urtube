import {
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from "@heroicons/react/20/solid";
import { useState } from "react";
import {
  FORMATS,
  type DownloadOptions,
  type LibraryMatch,
  type MediaType,
  type Tags,
  type Video,
} from "../api";
import { useLibraryMatches } from "../hooks/useLibraryMatches";
import { useTagSuggestion } from "../hooks/useTagSuggestion";
import {
  checkable,
  flagged,
  leftOutByDefault,
  libraryPath,
  type Flagged,
} from "../library";
import { savedAs } from "../tags";
import Modal from "./Modal";
import Segmented from "./Segmented";
import Spinner from "./Spinner";
import TagFields from "./TagFields";

interface DownloadDialogProps {
  open: boolean;
  onClose: () => void;
  /**
   * videos are the ones the user kept; tags is set when downloading a
   * single video.
   */
  onConfirm: (options: DownloadOptions, videos: Video[], tags?: Tags) => void;
  /** What is about to be downloaded; one video's tags can be edited. */
  videos: Video[];
  initialOptions: DownloadOptions;
}

const types: { value: MediaType; label: string }[] = [
  { value: "video", label: "Video" },
  { value: "audio", label: "Audio only" },
];

export default function DownloadDialog({
  open,
  onClose,
  onConfirm,
  videos,
  initialOptions,
}: DownloadDialogProps) {
  const video = videos.length === 1 ? videos[0] : undefined;
  const [type, setType] = useState(initialOptions.type);
  const [format, setFormat] = useState(initialOptions.format);
  const { suggestion, loading, error } = useTagSuggestion(video?.url ?? null);
  // What the user typed; untouched fields show the suggestion.
  const [edits, setEdits] = useState<Partial<Record<keyof Tags, string>>>({});
  // Duplicates the user ticked or unticked; the rest follow leftOutByDefault.
  const [kept, setKept] = useState<Record<string, boolean>>({});

  // Music search results already know their tags, so they show at once.
  const known = suggestion?.tags ?? video?.tags;
  const tags: Tags = {
    artist: edits.artist ?? known?.artist ?? "",
    title: edits.title ?? known?.title ?? "",
    album: edits.album ?? known?.album ?? "",
    year: known?.year,
  };
  // The album lookup can still change the tags, and with them where the
  // file goes: downloading before it answers would save what the user
  // never saw. A failed lookup ends the wait.
  const lookingUp = video !== undefined && loading;

  // One video is checked as its tags stand, once they are settled; a batch
  // checks the videos whose tags are already known.
  const checked = video ? [] : checkable(videos);
  const matches = useLibraryMatches(
    type,
    video
      ? !lookingUp && tags.title.trim()
        ? [tags]
        : null
      : checked.map((c) => c.tags),
  );
  const duplicates = video
    ? []
    : flagged(
        checked.map((c) => c.video),
        matches,
      );
  const leftOut = leftOutByDefault(duplicates);
  const isKept = (id: string) => kept[id] ?? !leftOut.has(id);
  const toDownload = videos.filter((v) => isKept(v.id));

  const handleTypeChange = (newType: MediaType) => {
    setType(newType);
    setFormat(FORMATS[newType][0]);
  };

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    if (lookingUp || toDownload.length === 0) return;
    onConfirm({ type, format }, toDownload, video ? tags : undefined);
    onClose();
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={video ? "Download" : `Download ${videos.length} videos`}
    >
      <form onSubmit={handleSubmit} className="space-y-5 px-5 pt-3 pb-5">
        {video && (
          <div className="flex items-center gap-3 rounded-xl bg-raised p-2.5">
            <img
              src={video.thumbnails[0]?.url}
              alt=""
              className="h-12 w-20 shrink-0 rounded-md bg-surface object-cover"
            />
            <div className="min-w-0">
              <p className="line-clamp-2 text-sm leading-snug font-medium">
                {video.title}
              </p>
              <p className="truncate text-xs text-muted">{video.uploader}</p>
            </div>
          </div>
        )}

        <div className="space-y-2">
          <Segmented
            legend="What to download"
            name="type"
            options={types}
            value={type}
            onChange={handleTypeChange}
            fill
          />
          <Segmented
            legend="Format"
            name="format"
            options={FORMATS[type].map((f) => ({
              value: f,
              label: f.toUpperCase(),
            }))}
            value={format}
            onChange={setFormat}
            fill
          />
          {format === "wav" && (
            <p className="help">
              WAV files can't hold cover art and only basic tags.
            </p>
          )}
        </div>

        {video ? (
          <div aria-busy={lookingUp}>
            <TagFields
              values={tags}
              onChange={(field, value) =>
                setEdits((e) => ({ ...e, [field]: value }))
              }
              knownArtists={suggestion?.knownArtists ?? []}
              loading={loading && !known}
              error={error}
            />
            <p className="help mt-3">
              Saved as{" "}
              <span className="font-mono break-all text-fg">
                {loading && !known && !edits.title
                  ? "…"
                  : savedAs(tags, type === "audio", format)}
              </span>
            </p>
            <LibraryNotice match={matches[0]} />
          </div>
        ) : (
          <div className="space-y-3">
            <p className="help">
              Artist, title and album are detected for each video when its
              download starts
              {checked.length < videos.length &&
                ", and videos already in your library are skipped then"}
              .
            </p>
            <DuplicateList
              items={duplicates}
              total={videos.length}
              isKept={isKept}
              onToggle={(id, keep) => setKept((k) => ({ ...k, [id]: keep }))}
            />
          </div>
        )}

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" onClick={onClose} className="btn btn-secondary">
            Cancel
          </button>
          <button
            type="submit"
            disabled={lookingUp || toDownload.length === 0}
            className="btn btn-primary"
          >
            {lookingUp ? (
              <>
                <Spinner className="size-4" />
                Looking up tags…
              </>
            ) : toDownload.length < videos.length ? (
              `Download ${toDownload.length} of ${videos.length}`
            ) : (
              "Start download"
            )}
          </button>
        </div>
      </form>
    </Modal>
  );
}

/**
 * What the library already holds for a single download. The live region is
 * always rendered so screen readers announce the notice when it appears.
 */
function LibraryNotice({ match }: { match?: LibraryMatch }) {
  const similar = match?.similar ?? [];
  return (
    <div role="status" className="empty:hidden mt-3">
      {match?.existing ? (
        <div className="flex gap-2 rounded-lg border border-warn/40 bg-warn/10 p-3 text-sm">
          <ExclamationTriangleIcon
            aria-hidden="true"
            className="size-5 shrink-0 text-warn"
          />
          <p className="min-w-0 break-words">
            Already in your library as{" "}
            <span className="font-mono">{libraryPath(match.existing)}</span>, so
            this download will be skipped. Change the title or album to keep
            both, or delete the file first to replace it.
          </p>
        </div>
      ) : similar.length > 0 ? (
        <div className="flex gap-2 rounded-lg border border-line bg-raised p-3 text-sm">
          <InformationCircleIcon
            aria-hidden="true"
            className="size-5 shrink-0 text-muted"
          />
          <div className="min-w-0">
            <p>You may already have this song:</p>
            <ul className="mt-1 space-y-0.5">
              {similar.map((p) => (
                <li
                  key={p}
                  className="font-mono text-xs break-words text-muted"
                >
                  {libraryPath(p)}
                </li>
              ))}
            </ul>
          </div>
        </div>
      ) : null}
    </div>
  );
}

/** The videos of a batch the library may already have, to tick or untick. */
function DuplicateList({
  items,
  total,
  isKept,
  onToggle,
}: {
  items: Flagged[];
  total: number;
  isKept: (id: string) => boolean;
  onToggle: (id: string, keep: boolean) => void;
}) {
  return (
    <div>
      {/* Always rendered so the count is announced when the check returns. */}
      <p role="status" className="text-sm font-medium empty:hidden">
        {items.length > 0 &&
          `${items.length} of ${total} may already be in your library`}
      </p>
      {items.length > 0 && (
        <fieldset className="mt-2">
          <legend className="help">Unticked videos won't be downloaded.</legend>
          <ul className="mt-2 max-h-60 divide-y divide-line overflow-y-auto rounded-xl border border-line">
            {items.map(({ video, match }) => (
              <li key={video.id}>
                <label className="flex min-h-11 cursor-pointer items-start gap-3 px-3 py-2">
                  <input
                    type="checkbox"
                    checked={isKept(video.id)}
                    onChange={(e) => onToggle(video.id, e.target.checked)}
                    className="mt-0.5 size-4 shrink-0 cursor-pointer accent-(--accent)"
                  />
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-medium">
                      {video.tags?.title || video.title}
                    </span>
                    <span className="help block break-words">
                      {match.existing
                        ? `Already in library: ${libraryPath(match.existing)}`
                        : `Similar: ${(match.similar ?? []).map(libraryPath).join(", ")}`}
                    </span>
                  </span>
                </label>
              </li>
            ))}
          </ul>
        </fieldset>
      )}
    </div>
  );
}
