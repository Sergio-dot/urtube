import { useCallback, useMemo, useState } from "react";
import { Dialog, DialogBackdrop, DialogPanel } from "@headlessui/react";
import { ExclamationTriangleIcon, XMarkIcon } from "@heroicons/react/20/solid";
import ActivityPanel from "./components/ActivityPanel";
import DownloadDialog from "./components/DownloadDialog";
import Header from "./components/Header";
import Results, { type ResultsLayout } from "./components/Results";
import SearchBar from "./components/SearchBar";
import SettingsDialog from "./components/SettingsDialog";
import {
  isValidOptions,
  type DownloadOptions,
  type SearchParams,
  type Settings,
  type Tags,
  type Video,
} from "./api";
import { downloadsByVideo, isActive } from "./downloads";
import { useDownloads } from "./hooks/useDownloads";
import { useHealth } from "./hooks/useHealth";
import { useLocalStorage } from "./hooks/useLocalStorage";
import { useQueue } from "./hooks/useQueue";
import { useSearch } from "./hooks/useSearch";
import { useSettings } from "./hooks/useSettings";

const DEFAULT_OPTIONS: DownloadOptions = { type: "video", format: "mp4" };

/** Videos waiting for the user to pick download options. */
interface PendingDownload {
  videos: Video[];
  /** Whether they are the selection, which is cleared once they start. */
  fromSelection: boolean;
}

function App() {
  const health = useHealth();
  const search = useSearch();
  const selection = useQueue();
  const { downloads, announcement, start, cancel, dismiss } = useDownloads();
  const {
    settings,
    refresh: refreshSettings,
    save: saveSettings,
  } = useSettings();
  const [preferences, setPreferences] = useLocalStorage(
    "urtube_download_preferences",
    DEFAULT_OPTIONS,
    isValidOptions,
  );

  const [layout, setLayout] = useState<ResultsLayout>("grid");
  const [failure, setFailure] = useState<string | null>(null);
  // The activity panel as a drawer, for screens too narrow to keep it open.
  const [showActivity, setShowActivity] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  // Remounts the settings dialog so it starts from the saved values.
  const [settingsKey, setSettingsKey] = useState(0);
  // Kept after closing so the dialog's content doesn't vanish as it fades.
  const [pending, setPending] = useState<PendingDownload | null>(null);
  const [showDownload, setShowDownload] = useState(false);
  // Remounts the dialog for each request so it starts from the preferences.
  const [downloadKey, setDownloadKey] = useState(0);

  const runSearch = (params: SearchParams) => {
    setLayout(params.source === "music" ? "list" : "grid");
    search.search(params);
  };

  const requestDownload = useCallback(
    (videos: Video[], fromSelection: boolean) => {
      setPending({ videos, fromSelection });
      setDownloadKey((k) => k + 1);
      setShowDownload(true);
    },
    [],
  );

  const openSettings = useCallback(() => {
    setSettingsKey((k) => k + 1);
    setShowSettings(true);
  }, []);

  const confirmDownload = async (options: DownloadOptions, tags?: Tags) => {
    if (!pending) return;
    setPreferences(options);
    setFailure(null);
    if (pending.fromSelection) selection.clear();

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
      setFailure(failures.join("\n"));
    }
  };

  const byVideo = useMemo(() => downloadsByVideo(downloads), [downloads]);
  const newestFirst = useMemo(
    () => Object.values(downloads).reverse(),
    [downloads],
  );
  const activeCount = newestFirst.filter((d) => isActive(d.status)).length;
  const selectedCount = selection.videos.length;
  const isSelected = (videoId: string) =>
    selection.videos.some((v) => v.id === videoId);

  const activity = (
    <ActivityPanel
      downloads={newestFirst}
      onCancel={cancel}
      onDismiss={dismiss}
      selected={selection.videos}
      onDeselect={selection.remove}
      onClearSelected={selection.clear}
      onDownloadSelected={() => {
        setShowActivity(false);
        requestDownload(selection.videos, true);
      }}
    />
  );

  return (
    <div className="flex min-h-svh flex-col">
      <Header
        health={health}
        downloadDirError={settings?.downloadDirError}
        activeCount={activeCount}
        selectedCount={selectedCount}
        onOpenActivity={() => setShowActivity(true)}
        onOpenSettings={openSettings}
      />

      <div className="mx-auto flex w-full max-w-[1600px] flex-1">
        <main className="min-w-0 flex-1 px-4 pt-6 pb-28 sm:px-6 xl:pb-10">
          <div className="mx-auto max-w-3xl">
            <SearchBar onSearch={runSearch} loading={search.loading} />
            {failure && (
              <div
                role="alert"
                className="mt-4 flex items-start gap-2 rounded-xl border border-danger/40 bg-danger/10 py-1 pr-1 pl-3 text-danger"
              >
                <ExclamationTriangleIcon
                  aria-hidden="true"
                  className="mt-3 size-5 shrink-0"
                />
                <div className="min-w-0 flex-1 py-2.5 text-sm">
                  <p className="font-semibold">Download failed</p>
                  <p className="break-words whitespace-pre-line">{failure}</p>
                </div>
                <button
                  type="button"
                  onClick={() => setFailure(null)}
                  aria-label="Dismiss"
                  className="icon-btn text-danger hover:text-danger"
                >
                  <XMarkIcon aria-hidden="true" className="size-5" />
                </button>
              </div>
            )}
          </div>

          <div className="mt-8" aria-busy={search.loading}>
            {search.loading ? (
              <ResultsSkeleton layout={layout} />
            ) : search.error ? (
              <Message title="Search failed" tone="danger">
                {search.error}
              </Message>
            ) : search.results?.length === 0 ? (
              <Message title="No results">
                Nothing matched that search. Try different words
                {layout === "list" && ", or search Videos instead of Music"}.
              </Message>
            ) : search.results ? (
              <div className={layout === "list" ? "mx-auto max-w-3xl" : ""}>
                <Results
                  results={search.results}
                  layout={layout}
                  downloads={byVideo}
                  isSelected={isSelected}
                  onSelect={selection.add}
                  onDeselect={selection.remove}
                  onDownload={(video) => requestDownload([video], false)}
                  onCancelDownload={cancel}
                />
              </div>
            ) : (
              <Welcome settings={settings} />
            )}
          </div>

          <footer className="mt-12 flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-muted">
            {health?.dependencies.version && (
              <span>yt-dlp {health.dependencies.version}</span>
            )}
            <a
              href="https://github.com/Sergio-dot/urtube"
              target="_blank"
              rel="noopener noreferrer"
              className="underline-offset-4 hover:text-fg hover:underline"
            >
              GitHub
            </a>
          </footer>
        </main>

        <aside
          aria-label="Downloads and selection"
          className="sticky top-14 hidden h-[calc(100svh-3.5rem)] w-96 shrink-0 overflow-y-auto border-l border-line p-6 xl:block"
        >
          {activity}
        </aside>
      </div>

      {/* Below xl the panel is off screen, so what it holds stays one tap away. */}
      {(activeCount > 0 || selectedCount > 0) && (
        <div className="pointer-events-none fixed inset-x-0 bottom-0 z-20 flex justify-center p-4 pb-[calc(1rem+env(safe-area-inset-bottom))] xl:hidden">
          <button
            type="button"
            onClick={() => setShowActivity(true)}
            className="btn pointer-events-auto h-12 rounded-full border border-line-strong bg-surface px-5 text-fg shadow-xl hover:bg-raised"
          >
            {activeCount > 0 && (
              <span className="flex items-center gap-2">
                <span
                  aria-hidden="true"
                  className="size-2 animate-pulse rounded-full bg-accent"
                />
                {activeCount} downloading
              </span>
            )}
            {activeCount > 0 && selectedCount > 0 && (
              <span aria-hidden="true" className="text-muted">
                ·
              </span>
            )}
            {selectedCount > 0 && <span>{selectedCount} selected</span>}
          </button>
        </div>
      )}

      <Dialog
        open={showActivity}
        onClose={() => setShowActivity(false)}
        className="relative z-40 xl:hidden"
      >
        <DialogBackdrop
          transition
          className="fixed inset-0 bg-black/60 transition-opacity duration-200 ease-out data-closed:opacity-0"
        />
        <DialogPanel
          transition
          aria-label="Downloads and selection"
          className="fixed inset-y-0 right-0 flex w-full max-w-sm flex-col border-l border-line bg-bg shadow-2xl transition duration-200 ease-out data-closed:translate-x-full"
        >
          <div className="flex justify-end px-2 pt-2">
            <button
              type="button"
              onClick={() => setShowActivity(false)}
              aria-label="Close"
              className="icon-btn"
            >
              <XMarkIcon aria-hidden="true" className="size-5" />
            </button>
          </div>
          <div className="flex-1 overflow-y-auto px-5 pb-[calc(1.25rem+env(safe-area-inset-bottom))]">
            {activity}
          </div>
        </DialogPanel>
      </Dialog>

      <DownloadDialog
        key={`download-${downloadKey}`}
        open={showDownload}
        onClose={() => setShowDownload(false)}
        onConfirm={confirmDownload}
        videos={pending?.videos ?? []}
        initialOptions={preferences}
      />

      <SettingsDialog
        key={`settings-${settingsKey}`}
        open={showSettings}
        onClose={() => setShowSettings(false)}
        settings={settings}
        onSave={saveSettings}
      />

      {/* One live region for the whole app: finished and failed downloads are
          announced without every progress tick being read out. */}
      <p role="status" className="sr-only">
        {announcement}
      </p>
    </div>
  );
}

function Message({
  title,
  tone = "muted",
  children,
}: {
  title: string;
  tone?: "muted" | "danger";
  children: React.ReactNode;
}) {
  return (
    <div
      role={tone === "danger" ? "alert" : "status"}
      className="mx-auto max-w-md py-12 text-center"
    >
      <p className={`font-semibold ${tone === "danger" ? "text-danger" : ""}`}>
        {title}
      </p>
      <p className="mt-1 text-sm break-words text-muted">{children}</p>
    </div>
  );
}

/** Shown before the first search: what happens next, and where files go. */
function Welcome({ settings }: { settings: Settings | null }) {
  const steps = [
    [
      "Search",
      "Find a video, or switch to Music for songs with artist and album.",
    ],
    ["Choose", "Pick video or audio and a format, and check the tags."],
    ["Done", "The file lands in your library folder, named and tagged."],
  ];
  return (
    <div className="mx-auto max-w-3xl">
      <ol className="grid gap-3 sm:grid-cols-3">
        {steps.map(([title, text], i) => (
          <li
            key={title}
            className="rounded-xl border border-line bg-surface p-4"
          >
            <p className="text-xs font-semibold text-accent-text tabular-nums">
              {i + 1}
            </p>
            <p className="mt-1 text-sm font-semibold">{title}</p>
            <p className="help mt-1">{text}</p>
          </li>
        ))}
      </ol>
      {settings && (
        <p className="help mt-4 text-center">
          Saving to{" "}
          <code className="font-mono break-all text-fg">
            {settings.downloadDir}
          </code>
          {settings.jellyfin.enabled && " · Jellyfin refreshes after downloads"}
        </p>
      )}
    </div>
  );
}

/** Placeholders shaped like the results, so the page doesn't jump on load. */
function ResultsSkeleton({ layout }: { layout: ResultsLayout }) {
  const label = <p className="sr-only">Searching…</p>;
  if (layout === "list") {
    return (
      <div role="status" className="mx-auto max-w-3xl animate-pulse space-y-2">
        {label}
        {Array.from({ length: 5 }, (_, i) => (
          <div key={i} className="h-17 rounded-xl bg-raised" />
        ))}
      </div>
    );
  }
  return (
    <div
      role="status"
      className="grid animate-pulse grid-cols-[repeat(auto-fill,minmax(min(100%,15rem),1fr))] gap-4"
    >
      {label}
      {Array.from({ length: 8 }, (_, i) => (
        <div key={i} className="rounded-xl bg-raised">
          <div className="aspect-video" />
          <div className="h-28" />
        </div>
      ))}
    </div>
  );
}

export default App;
