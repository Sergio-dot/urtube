import { useState } from "react";
import { ExclamationTriangleIcon } from "@heroicons/react/20/solid";
import type { Settings, SettingsChange } from "../api";
import JellyfinFields from "./JellyfinFields";
import Modal from "./Modal";

interface SettingsDialogProps {
  open: boolean;
  onClose: () => void;
  settings: Settings | null;
  /** Saves a change; rejects with the reason it can't be applied. */
  onSave: (change: SettingsChange) => Promise<void>;
}

export default function SettingsDialog({
  open,
  onClose,
  settings,
  onSave,
}: SettingsDialogProps) {
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
    <Modal open={open} onClose={onClose} title="Settings">
      <form onSubmit={handleSubmit} className="space-y-5 px-5 pt-3 pb-5">
        <div>
          <div className="flex items-end justify-between gap-3">
            <label htmlFor="downloadDir" className="label">
              Download folder
            </label>
            {!isDefault && settings !== null && (
              <button
                type="button"
                disabled={saving}
                onClick={() => save({ downloadDir: "" })}
                className="cursor-pointer text-xs font-medium text-accent-text underline-offset-4 hover:underline disabled:opacity-50"
              >
                Restore default
              </button>
            )}
          </div>
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
            className="input mt-1.5 font-mono aria-invalid:border-danger"
          />
          <p id="downloadDir-help" className="help mt-1.5">
            The full path of a folder on the server
            {settings?.downloadRoot && (
              <>
                , inside{" "}
                <code className="font-mono break-all">
                  {settings.downloadRoot}
                </code>
              </>
            )}
            . Missing folders are created. Default:{" "}
            <code className="font-mono break-all">
              {settings?.defaultDownloadDir}
            </code>
          </p>

          {(error ?? currentProblem) && (
            <div
              role="alert"
              className="mt-3 flex gap-2 rounded-lg border border-danger/40 bg-danger/10 p-3 text-sm text-danger"
            >
              <ExclamationTriangleIcon
                aria-hidden="true"
                className="size-5 shrink-0"
              />
              <span className="min-w-0 break-words">
                {error ?? `Downloads can't be saved here: ${currentProblem}`}
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

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button type="button" onClick={onClose} className="btn btn-secondary">
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving || settings === null}
            className="btn btn-primary"
          >
            {saving ? "Saving…" : "Save"}
          </button>
        </div>
      </form>
    </Modal>
  );
}
