import "./App.css";
import Navbar from "./components/Navbar";
import Form from "./components/Form";
import SearchResults from "./components/SearchResults";
import ErrorModal from "./components/ErrorModal";
import Footer from "./components/Footer";
import QueuePanel from "./components/QueuePanel";
import DownloadModal from "./components/DownloadModal";
import DownloadStatusOverlay from "./components/DownloadStatusOverlay";
import { useEffect, useState, useCallback } from "react";
import type {
  Video,
  DownloadState,
  DownloadOptions,
  ProgressUpdate,
  DownloadStatus,
} from "./types";
import { useDownloadEvents } from "./hooks/useDownloadEvents";

const PREFERENCES_KEY = "urtube_download_preferences";
const QUEUE_KEY = "urtube_video_queue";

function App() {
  const [showError, setShowError] = useState(false);
  const [errorTitle, setErrorTitle] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [results, setResults] = useState<Video[]>([]);
  const [loading, setLoading] = useState(false);
  const [ytdlpVersion, setYtdlpVersion] = useState<string | null>(null);

  const [queue, setQueue] = useState<Video[]>(() => {
    const saved = localStorage.getItem(QUEUE_KEY);
    return saved ? JSON.parse(saved) : [];
  });
  const [showPanel, setShowPanel] = useState(false);

  // downloadStates is now keyed by download ID (UUID)
  const [downloadStates, setDownloadStates] = useState<
    Record<string, DownloadState>
  >({});

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalKey, setModalKey] = useState(0);
  const [videoToDownload, setVideoToDownload] = useState<
    Video | Video[] | null
  >(null);
  const [preferences, setPreferences] = useState<DownloadOptions>(() => {
    const saved = localStorage.getItem(PREFERENCES_KEY);
    return saved ? JSON.parse(saved) : { type: "video", format: "mp4" };
  });

  const handleProgressUpdate = useCallback((update: ProgressUpdate) => {
    setDownloadStates((prev) => {
      const existing = prev[update.uuid];
      // Only update if we already have it OR if it's a new UUID we haven't seen.
      // We merge with 'existing' to preserve the 'title' and 'videoId' added by performDownload.
      return {
        ...prev,
        [update.uuid]: {
          ...(existing || {}),
          uuid: update.uuid,
          videoId: update.videoId,
          title: update.title,
          status: update.status as DownloadStatus,
          percent: update.percent,
          speed: update.speed,
          eta: update.eta,
          downloaded: update.downloaded,
          total: update.total,
          errorMessage: update.errorMessage,
        },
      };
    });
  }, []);

  useDownloadEvents(handleProgressUpdate);

  useEffect(() => {
    localStorage.setItem(QUEUE_KEY, JSON.stringify(queue));
  }, [queue]);

  useEffect(() => {
    localStorage.setItem(PREFERENCES_KEY, JSON.stringify(preferences));
  }, [preferences]);

  useEffect(() => {
    const timer = setTimeout(() => {
      document.body.style.setProperty("--transition-speed", "500ms");
    }, 100);

    // Check health
    fetch("/api/v1/health")
      .then((res) => res.json())
      .then((data) => {
        if (data.status === "degraded" || !data.dependencies.ytdlp) {
          setErrorTitle("Missing Dependency");
          setErrorMessage(
            "yt-dlp is not installed on the server. Please install it to use urtube: https://github.com/yt-dlp/yt-dlp",
          );
          setShowError(true);
        } else {
          setYtdlpVersion(data.dependencies.version);
        }
      })
      .catch((err) => {
        console.error("Failed to check health:", err);
      });

    return () => clearTimeout(timer);
  }, []);

  const addToQueue = (video: Video) => {
    if (!queue.some((v) => v.id === video.id)) {
      setQueue([...queue, video]);
    }
  };

  const removeFromQueue = (videoId: string) => {
    setQueue(queue.filter((v) => v.id !== videoId));
  };

  const clearQueue = () => {
    setQueue([]);
  };

  const handleDownloadRequest = (video: Video | Video[]) => {
    setVideoToDownload(video);
    setModalKey((prev) => prev + 1);
    setIsModalOpen(true);
  };

  const confirmDownload = (options: DownloadOptions) => {
    setPreferences(options);

    if (Array.isArray(videoToDownload)) {
      videoToDownload.forEach((v) => performDownload(v, options));
    } else if (videoToDownload) {
      performDownload(videoToDownload, options);
    }

    clearQueue();
    setShowPanel(false);
  };

  const performDownload = async (video: Video, options: DownloadOptions) => {
    try {
      const flags: {
        post_processing?: Record<string, string | boolean>;
        video_format?: Record<string, string>;
      } = {};
      if (options.type === "audio") {
        flags.post_processing = {
          extract_audio: true,
          audio_format: options.format,
        };
      } else {
        flags.video_format = {
          format: "bestvideo+bestaudio/best",
        };
        flags.post_processing = {
          recode_video: options.format,
        };
      }

      const response = await fetch("/api/v1/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: video.url,
          videoId: video.id,
          title: video.title,
          flags: flags,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || "Failed to start download");
      }

      const data = await response.json();
      const downloadUUID = data.uuid;

      setDownloadStates((prev) => {
        const existing = prev[downloadUUID];
        return {
          ...prev,
          [downloadUUID]: {
            ...existing,
            uuid: downloadUUID,
            videoId: video.id,
            title: video.title,
            status: existing?.status || "loading",
            options,
          },
        };
      });
    } catch (error) {
      console.error("Download error:", error);
      // For errors that happen before we get a download ID, we just show an error modal
      setErrorTitle("Download Failed");
      setErrorMessage(error instanceof Error ? error.message : "Unknown error");
      setShowError(true);
    }
  };

  const removeDownload = (id: string) => {
    setDownloadStates((prev) => {
      const newState = { ...prev };
      delete newState[id];
      return newState;
    });
  };

  const downloadsArray = Object.values(downloadStates).reverse();

  // Map videoId to status for SearchResults UI (green checkmark, etc.)
  const videoStatusMap = Object.values(downloadStates).reduce(
    (acc, dl) => {
      if (dl.videoId) {
        const current = acc[dl.videoId];
        // Priority: finished > downloading > loading > error
        if (current?.status === "finished" || current?.status === "success")
          return acc;
        if (
          current?.status === "downloading" &&
          (dl.status === "loading" || dl.status === "error")
        )
          return acc;

        acc[dl.videoId] = dl;
      }
      return acc;
    },
    {} as Record<string, DownloadState>,
  );

  return (
    <div className="min-h-screen flex flex-col">
      <div className="navbar">
        <Navbar
          onTogglePanel={() => setShowPanel(!showPanel)}
          queueSize={queue.length}
        />
      </div>

      <main className="grow">
        <div className="form">
          <Form onResults={setResults} onLoading={setLoading} />
        </div>

        {loading && (
          <div className="flex justify-center py-10">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
          </div>
        )}

        <div className="results">
          <SearchResults
            results={results}
            onAddVideo={addToQueue}
            onRemoveVideo={removeFromQueue}
            onDownloadVideo={handleDownloadRequest}
            onCancelDownload={removeDownload}
            queue={queue}
            downloadStates={videoStatusMap}
          />
        </div>
      </main>

      <QueuePanel
        isOpen={showPanel}
        onClose={() => setShowPanel(false)}
        videos={queue}
        onRemoveVideo={removeFromQueue}
        onClearQueue={clearQueue}
        onDownloadAll={() => handleDownloadRequest(queue)}
      />

      <DownloadModal
        key={modalKey}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onConfirm={confirmDownload}
        title={
          videoToDownload
            ? Array.isArray(videoToDownload)
              ? `Download Queue (${videoToDownload.length} items)`
              : `Download ${videoToDownload.title}`
            : "Download Preferences"
        }
        initialOptions={preferences}
      />

      <DownloadStatusOverlay
        downloads={downloadsArray}
        onRemove={removeDownload}
      />

      <ErrorModal
        open={showError}
        onClose={setShowError}
        title={errorTitle}
        message={errorMessage}
      />

      <Footer version={ytdlpVersion} />
    </div>
  );
}

export default App;
