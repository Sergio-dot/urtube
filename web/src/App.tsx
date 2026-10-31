import { useCallback, useEffect, useMemo, useState } from "react";
import Navbar from "./components/Navbar";
import Form from "./components/Form";
import SearchResults from "./components/SearchResults";
import ErrorModal from "./components/ErrorModal";
import Footer from "./components/Footer";
import QueuePanel from "./components/QueuePanel";
import DownloadModal from "./components/DownloadModal";
import DownloadStatusOverlay from "./components/DownloadStatusOverlay";
import Spinner from "./components/Spinner";
import SettingsModal from "./components/SettingsModal";
import {
  isValidOptions,
  type DownloadOptions,
  type Tags,
  type Video,
} from "./api";
import { downloadsByVideo } from "./downloads";
import { useDownloads } from "./hooks/useDownloads";
import { useHealth } from "./hooks/useHealth";
import { useLocalStorage } from "./hooks/useLocalStorage";
import { useQueue } from "./hooks/useQueue";
import { useSearch } from "./hooks/useSearch";
import { useSettings } from "./hooks/useSettings";

const DEFAULT_OPTIONS: DownloadOptions = { type: "video", format: "mp4" };

interface ErrorInfo {
  title: string;
  message: string;
}

/** Videos waiting for the user to pick download options. */
interface PendingDownload {
  videos: Video[];
  /** Whether they come from the queue, which is cleared once they start. */
  fromQueue: boolean;
}

function App() {
  const health = useHealth();
  const search = useSearch();
  const queue = useQueue();
  const { downloads, start, cancel, dismiss } = useDownloads();
  const { settings, refresh: refreshSettings, saveDownloadDir } = useSettings();
  const [preferences, setPreferences] = useLocalStorage(
    "urtube_download_preferences",
    DEFAULT_OPTIONS,
    isValidOptions,
  );

  // Kept after closing so the modal's text doesn't vanish as it fades out.
  const [error, setError] = useState<ErrorInfo>({ title: "", message: "" });
  const [showError, setShowError] = useState(false);
  const [showPanel, setShowPanel] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  // Remounts the settings dialog so it starts from the saved values.
  const [settingsKey, setSettingsKey] = useState(0);
  const [pending, setPending] = useState<PendingDownload | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  // Remounts the modal for each request so it starts from the preferences.
  const [modalKey, setModalKey] = useState(0);

  useEffect(() => {
    // Enable theme transitions only after the first paint.
    const timer = setTimeout(() => {
      document.body.style.setProperty("--transition-speed", "500ms");
    }, 100);
    return () => clearTimeout(timer);
  }, []);

  const requestDownload = useCallback((videos: Video[], fromQueue: boolean) => {
    setPending({ videos, fromQueue });
    setModalKey((k) => k + 1);
    setIsModalOpen(true);
  }, []);

  const openSettings = useCallback(() => {
    setSettingsKey((k) => k + 1);
    setShowSettings(true);
  }, []);

  const confirmDownload = async (options: DownloadOptions, tags?: Tags) => {
    if (!pending) return;
    setPreferences(options);
    if (pending.fromQueue) {
      queue.clear();
      setShowPanel(false);
    }

    // One at a time, so the server queues them in the order shown.
    const failures: string[] = [];
    for (const video of pending.videos) {
      try {
        await start(video, options, tags);
      } catch (err) {
        const reason = err instanceof Error ? err.message : "unknown error";
        failures.push(
          pending.videos.length > 1 ? `${video.title}: ${reason}` : reason,
        );
      }
    }
    if (failures.length > 0) {
      // A failure may be the download folder going bad: show it if so.
      refreshSettings();
      setError({ title: "Download failed", message: failures.join("\n") });
      setShowError(true);
    }
  };

  const byVideo = useMemo(() => downloadsByVideo(downloads), [downloads]);
  const newestFirst = useMemo(
    () => Object.values(downloads).reverse(),
    [downloads],
  );
  const isQueued = (videoId: string) =>
    queue.videos.some((v) => v.id === videoId);

  const modalTitle = !pending
    ? "Download"
    : pending.videos.length === 1
      ? `Download ${pending.videos[0].title}`
      : `Download Queue (${pending.videos.length} items)`;

  return (
    <div className="min-h-screen flex flex-col">
      <div className="navbar">
        <Navbar
          onTogglePanel={() => setShowPanel((open) => !open)}
          onOpenSettings={openSettings}
          queueSize={queue.videos.length}
          settingsAttention={Boolean(settings?.downloadDirError)}
        />
      </div>

      <main className="grow">
        <div className="form">
          <Form
            onSearch={search.search}
            health={health}
            downloadDirError={settings?.downloadDirError}
            onOpenSettings={openSettings}
          />
        </div>

        {search.loading ? (
          <div className="flex justify-center py-10">
            <Spinner className="size-12 text-indigo-500" />
          </div>
        ) : search.error ? (
          <p className="py-10 text-center text-sm text-red-500">
            Search failed: {search.error}
          </p>
        ) : search.results?.length === 0 ? (
          <p className="py-10 text-center text-sm text-gray-500 dark:text-gray-400">
            No videos found.
          </p>
        ) : search.results ? (
          <div className="results">
            <SearchResults
              results={search.results}
              downloads={byVideo}
              isQueued={isQueued}
              onQueue={queue.add}
              onUnqueue={queue.remove}
              onDownload={(video) => requestDownload([video], false)}
              onCancelDownload={cancel}
            />
          </div>
        ) : null}
      </main>

      <QueuePanel
        isOpen={showPanel}
        onClose={() => setShowPanel(false)}
        videos={queue.videos}
        onRemoveVideo={queue.remove}
        onClearQueue={queue.clear}
        onDownloadAll={() => requestDownload(queue.videos, true)}
      />

      <DownloadModal
        key={modalKey}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onConfirm={confirmDownload}
        title={modalTitle}
        initialOptions={preferences}
        video={pending?.videos.length === 1 ? pending.videos[0] : undefined}
      />

      <SettingsModal
        key={settingsKey}
        isOpen={showSettings}
        onClose={() => setShowSettings(false)}
        settings={settings}
        onSaveDownloadDir={saveDownloadDir}
      />

      <DownloadStatusOverlay
        downloads={newestFirst}
        onDismiss={dismiss}
        onCancel={cancel}
      />

      <ErrorModal
        open={showError}
        onClose={() => setShowError(false)}
        title={error.title}
        message={error.message}
      />

      <Footer version={health?.dependencies.version ?? null} />
    </div>
  );
}

export default App;
