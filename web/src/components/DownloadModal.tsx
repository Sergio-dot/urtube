import { useState } from "react";
import { Dialog, DialogBackdrop, DialogPanel, DialogTitle } from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";
import type { DownloadOptions, DownloadType } from "../types";

interface DownloadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: (options: DownloadOptions) => void;
  title: string;
  initialOptions?: DownloadOptions;
}

const VIDEO_FORMATS = ["mp4", "mkv", "webm"];
const AUDIO_FORMATS = ["mp3", "m4a", "wav", "opus"];

export default function DownloadModal({
  isOpen,
  onClose,
  onConfirm,
  title,
  initialOptions,
}: DownloadModalProps) {
  const [type, setType] = useState<DownloadType>(
    initialOptions?.type || "video",
  );
  const [format, setFormat] = useState(initialOptions?.format || "mp4");

  const handleTypeChange = (newType: DownloadType) => {
    setType(newType);
    setFormat(newType === "video" ? "mp4" : "mp3");
  };

  const handleConfirm = () => {
    onConfirm({ type, format });
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
                      {(type === "video" ? VIDEO_FORMATS : AUDIO_FORMATS).map(
                        (f) => (
                          <button
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
                        ),
                      )}
                    </div>
                  </div>
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
