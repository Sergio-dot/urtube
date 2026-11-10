import { useState } from "react";
import {
  FORMATS,
  type DownloadOptions,
  type MediaType,
  type Tags,
  type Video,
} from "../api";
import { useTagSuggestion } from "../hooks/useTagSuggestion";
import { savedAs } from "../tags";
import Modal from "./Modal";
import Segmented from "./Segmented";
import TagFields from "./TagFields";

interface DownloadDialogProps {
  open: boolean;
  onClose: () => void;
  /** tags is set when downloading a single video. */
  onConfirm: (options: DownloadOptions, tags?: Tags) => void;
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

  // Music search results already know their tags, so they show at once.
  const known = suggestion?.tags ?? video?.tags;
  const tags: Tags = {
    artist: edits.artist ?? known?.artist ?? "",
    title: edits.title ?? known?.title ?? "",
    album: edits.album ?? known?.album ?? "",
    year: known?.year,
  };

  const handleTypeChange = (newType: MediaType) => {
    setType(newType);
    setFormat(FORMATS[newType][0]);
  };

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    onConfirm({ type, format }, video ? tags : undefined);
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
          <div>
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
          </div>
        ) : (
          <p className="help">
            Artist, title and album are detected for each video when its
            download starts.
          </p>
        )}

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" onClick={onClose} className="btn btn-secondary">
            Cancel
          </button>
          <button type="submit" className="btn btn-primary">
            Start download
          </button>
        </div>
      </form>
    </Modal>
  );
}
