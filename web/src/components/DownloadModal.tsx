import { useState } from "react";
import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import {
  FORMATS,
  type DownloadOptions,
  type MediaType,
  type Tags,
  type Video,
} from "../api";
import { useTagSuggestion } from "../hooks/useTagSuggestion";
import TagFields from "./TagFields";
import { savedAs } from "../tags";

interface DownloadModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** tags is set when downloading a single video. */
  onConfirm: (options: DownloadOptions, tags?: Tags) => void;
  title: string;
  initialOptions: DownloadOptions;
  /** The video to download, when there is just one: its tags can be edited. */
  video?: Video;
}

export default function DownloadModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  initialOptions,
  video,
}: DownloadModalProps) {
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

  const handleConfirm = () => {
    onConfirm({ type, format }, video ? tags : undefined);
    onClose();
  };

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <DialogBackdrop
        transition
        className="fixed inset-0 bg-black/60 transition-opacity data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in"
      />

      <div className="fixed inset-0 z-10 overflow-y-auto">
        <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
          <DialogPanel
            transition
            className="relative transform overflow-hidden rounded-2xl bg-white dark:bg-gray-900 px-4 pb-4 pt-5 text-left shadow-xl transition-all data-closed:translate-y-4 data-closed:opacity-0 data-enter:duration-300 data-enter:ease-out data-leave:duration-200 data-leave:ease-in sm:my-8 sm:w-full sm:max-w-lg sm:p-6 sm:data-closed:translate-y-0 sm:data-closed:scale-95 ring-1 ring-black/5 dark:ring-white/10"
          >
            <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
              <button
                type="button"
                className="rounded-md bg-transparent text-gray-400 hover:text-gray-500 focus:outline-none"
                onClick={onClose}
              >
                <span className="sr-only">Close</span>
                <XMarkIcon className="h-6 w-6" aria-hidden="true" />
              </button>
            </div>
            <div className="sm:flex sm:items-start">
              <div className="mt-3 text-center sm:mt-0 sm:text-left w-full">
                <DialogTitle
                  as="h3"
                  className="text-lg font-semibold leading-6 dark:text-white"
                >
                  {title}
                </DialogTitle>
                <div className="mt-6 space-y-6">
                  {/* Type Selection */}
                  <div>
                    <label className="text-sm font-medium dark:text-gray-300">
                      Download Type
                    </label>
                    <div className="mt-2 grid grid-cols-2 gap-3">
                      <button
                        type="button"
                        onClick={() => handleTypeChange("video")}
                        className={`flex items-center justify-center rounded-xl py-2.5 text-sm font-semibold ring-1 ring-inset transition-all ${
                          type === "video"
                            ? "bg-indigo-600 text-white ring-indigo-600"
                            : "bg-white/5 text-gray-400 ring-white/10 hover:bg-white/10"
                        }`}
                      >
                        Video + Audio
                      </button>
                      <button
                        type="button"
                        onClick={() => handleTypeChange("audio")}
                        className={`flex items-center justify-center rounded-xl py-2.5 text-sm font-semibold ring-1 ring-inset transition-all ${
                          type === "audio"
                            ? "bg-indigo-600 text-white ring-indigo-600"
                            : "bg-white/5 text-gray-400 ring-white/10 hover:bg-white/10"
                        }`}
                      >
                        Audio Only
                      </button>
                    </div>
                  </div>

                  {/* Format Selection */}
                  <div>
                    <label className="text-sm font-medium dark:text-gray-300">
                      Format
                    </label>
                    <div className="mt-2 grid grid-cols-4 gap-2">
                      {FORMATS[type].map((f) => (
                        <button
                          type="button"
                          key={f}
                          onClick={() => setFormat(f)}
                          className={`flex items-center justify-center rounded-lg py-2 text-xs font-semibold ring-1 ring-inset transition-all ${
                            format === f
                              ? "bg-indigo-600/20 text-indigo-400 ring-indigo-500/50"
                              : "bg-white/5 text-gray-400 ring-white/10 hover:bg-white/10"
                          }`}
                        >
                          {f.toUpperCase()}
                        </button>
                      ))}
                    </div>
                    {format === "wav" && (
                      <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">
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
                      <p className="mt-3 text-xs text-gray-500 dark:text-gray-400">
                        Saved as{" "}
                        <span className="font-mono">
                          {loading && !known && !edits.title
                            ? "…"
                            : savedAs(tags, type === "audio", format)}
                        </span>
                      </p>
                    </div>
                  ) : (
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      Artist, title and album are detected for each video when
                      its download starts.
                    </p>
                  )}
                </div>
              </div>
            </div>
            <div className="mt-8 sm:flex sm:flex-row-reverse">
              <button
                type="button"
                className="inline-flex w-full justify-center rounded-xl bg-indigo-600 px-3 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 sm:ml-3 sm:w-auto transition-all"
                onClick={handleConfirm}
              >
                Start Download
              </button>
              <button
                type="button"
                className="mt-3 inline-flex w-full justify-center rounded-xl bg-white/5 px-3 py-2.5 text-sm font-semibold dark:text-white text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-white/10 hover:bg-gray-50 dark:hover:bg-white/10 sm:mt-0 sm:w-auto transition-all"
                onClick={onClose}
              >
                Cancel
              </button>
            </div>
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}
