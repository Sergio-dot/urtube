import { useState } from "react";
import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import {
  ExclamationTriangleIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import type { Settings, SettingsChange } from "../api";
import JellyfinFields from "./JellyfinFields";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  settings: Settings | null;
  /** Saves a change; rejects with the reason it can't be applied. */
  onSave: (change: SettingsChange) => Promise<void>;
}

export default function SettingsModal({
  isOpen,
  onClose,
  settings,
  onSave,
}: SettingsModalProps) {
  const [downloadDir, setDownloadDir] = useState(settings?.downloadDir ?? "");
  const [jellyfinUrl, setJellyfinUrl] = useState(settings?.jellyfin.url ?? "");
  const [jellyfinKey, setJellyfinKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const save = async (change: SettingsChange) => {
    setSaving(true);
    setError(null);
    try {
      await onSave(change);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save settings");
    } finally {
      setSaving(false);
    }
  };

  // Only what changed is sent, so a broken download folder doesn't stop
  // Jellyfin from being saved, and vice versa.
  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const change: SettingsChange = {};
    if (downloadDir.trim() !== settings?.downloadDir) {
      change.downloadDir = downloadDir.trim();
    }
    const urlChanged = jellyfinUrl.trim() !== (settings?.jellyfin.url ?? "");
    if (urlChanged || jellyfinKey.trim()) {
      change.jellyfin = { url: jellyfinUrl.trim() };
      if (jellyfinKey.trim()) change.jellyfin.apiKey = jellyfinKey.trim();
    }
    save(change);
  };

  const isDefault =
    settings !== null && settings.downloadDir === settings.defaultDownloadDir;
  // The saved folder's problem, until the user edits the field.
  const currentProblem =
    settings?.downloadDirError && downloadDir === settings.downloadDir
      ? settings.downloadDirError
      : null;

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

            <form onSubmit={handleSubmit}>
              <DialogTitle
                as="h3"
                className="text-lg font-semibold leading-6 dark:text-white"
              >
                Settings
              </DialogTitle>

              <div className="mt-6">
                <label
                  htmlFor="downloadDir"
                  className="text-sm font-medium dark:text-gray-300"
                >
                  Download folder
                </label>
                <input
                  id="downloadDir"
                  type="text"
                  autoComplete="off"
                  spellCheck={false}
                  value={downloadDir}
                  onChange={(e) => {
                    setDownloadDir(e.target.value);
                    setError(null);
                  }}
                  placeholder={settings?.defaultDownloadDir}
                  aria-describedby="downloadDir-help"
                  aria-invalid={error !== null || currentProblem !== null}
                  className="mt-2 block w-full rounded-xl border-0 px-3 py-2.5 font-mono text-sm text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 dark:bg-white/5 dark:text-white dark:ring-white/10 dark:focus:ring-indigo-500"
                />
                <p
                  id="downloadDir-help"
                  className="mt-2 text-xs text-gray-500 dark:text-gray-400"
                >
                  The full path of a folder on the server
                  {settings?.downloadRoot && (
                    <>
                      , inside{" "}
                      <code className="font-mono">{settings.downloadRoot}</code>
                    </>
                  )}
                  . Missing folders are created. Default:{" "}
                  <code className="font-mono">
                    {settings?.defaultDownloadDir}
                  </code>
                </p>

                {(error ?? currentProblem) && (
                  <div
                    role="alert"
                    className="mt-3 flex gap-2 rounded-lg bg-red-500/10 p-3 text-sm text-red-600 dark:text-red-400"
                  >
                    <ExclamationTriangleIcon
                      aria-hidden="true"
                      className="size-5 shrink-0"
                    />
                    <span>
                      {error ??
                        `Downloads can't be saved here: ${currentProblem}`}
                    </span>
                  </div>
                )}
              </div>

              <JellyfinFields
                saved={settings?.jellyfin}
                url={jellyfinUrl}
                onUrlChange={setJellyfinUrl}
                apiKey={jellyfinKey}
                onApiKeyChange={setJellyfinKey}
              />

              <div className="mt-8 flex flex-col gap-3 sm:flex-row-reverse">
                <button
                  type="submit"
                  disabled={saving || settings === null}
                  className="inline-flex w-full justify-center rounded-xl bg-indigo-600 px-3 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50 sm:w-auto transition-all"
                >
                  {saving ? "Saving…" : "Save"}
                </button>
                <button
                  type="button"
                  className="inline-flex w-full justify-center rounded-xl bg-white/5 px-3 py-2.5 text-sm font-semibold dark:text-white text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 dark:ring-white/10 hover:bg-gray-50 dark:hover:bg-white/10 sm:w-auto transition-all"
                  onClick={onClose}
                >
                  Cancel
                </button>
                {!isDefault && settings !== null && (
                  <button
                    type="button"
                    disabled={saving}
                    onClick={() => save({ downloadDir: "" })}
                    className="inline-flex w-full justify-center rounded-xl px-3 py-2.5 text-sm font-semibold text-indigo-600 hover:bg-indigo-50 disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-white/5 sm:mr-auto sm:w-auto transition-all"
                  >
                    Default folder
                  </button>
                )}
              </div>
            </form>
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}
