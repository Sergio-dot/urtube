import "./App.css";
import Navbar from "./components/Navbar";
import Form from "./components/Form";
import SearchResults from "./components/SearchResults";
import ErrorModal from "./components/ErrorModal";
import Footer from "./components/Footer";
import CollectionPanel from "./components/CollectionPanel";
import DownloadModal from "./components/DownloadModal";
import { useEffect, useState } from "react";
import type { Video, DownloadState, DownloadOptions } from "./types";

const PREFERENCES_KEY = "urtube_download_preferences";

function App() {
  const [showError, setShowError] = useState(false);
  const [errorTitle, setErrorTitle] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [results, setResults] = useState<Video[]>([]);
  const [loading, setLoading] = useState(false);
  const [ytdlpVersion, setYtdlpVersion] = useState<string | null>(null);

  const [collection, setCollection] = useState<Video[]>([]);
  const [showPanel, setShowPanel] = useState(false);

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

  const addToCollection = (video: Video) => {
    if (!collection.some((v) => v.id === video.id)) {
      setCollection([...collection, video]);
    }
  };

  const removeFromCollection = (videoId: string) => {
    setCollection(collection.filter((v) => v.id !== videoId));
  };

  const clearCollection = () => {
    setCollection([]);
  };

  const handleDownloadRequest = (video: Video | Video[]) => {
    setVideoToDownload(video);
    setModalKey((prev) => prev + 1);
    setIsModalOpen(true);
  };

  const confirmDownload = (options: DownloadOptions) => {
    setPreferences(options);
    localStorage.setItem(PREFERENCES_KEY, JSON.stringify(options));

    if (Array.isArray(videoToDownload)) {
      videoToDownload.forEach((v) => performDownload(v, options));
    } else if (videoToDownload) {
      performDownload(videoToDownload, options);
    }
  };

  const performDownload = async (video: Video, options: DownloadOptions) => {
    // Prevent multiple downloads of the same video if already loading or success
    if (
      downloadStates[video.id]?.status === "loading" ||
      downloadStates[video.id]?.status === "success"
    ) {
      return;
    }

    setDownloadStates((prev) => ({
      ...prev,
      [video.id]: { videoId: video.id, status: "loading", options },
    }));

    try {
      const flags: any = {};
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
          flags: flags,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || "Failed to start download");
      }

      setDownloadStates((prev) => ({
        ...prev,
        [video.id]: { videoId: video.id, status: "success", options },
      }));
    } catch (error) {
      console.error("Download error:", error);
      setDownloadStates((prev) => ({
        ...prev,
        [video.id]: {
          videoId: video.id,
          status: "error",
          errorMessage:
            error instanceof Error ? error.message : "Unknown error",
          options,
        },
      }));
    }
  };

  const cancelDownload = (videoId: string) => {
    // TODO: BE endpoint implementation is missing, for now just reset the UI state.
    // After implementation, send a DELETE/POST to endpoint e.g. /api/v1/download/cancel/:id
    setDownloadStates((prev) => {
      const newState = { ...prev };
      delete newState[videoId];
      return newState;
    });
  };

  return (
    <div className="min-h-screen flex flex-col">
      <div className="navbar">
        <Navbar
          onTogglePanel={() => setShowPanel(!showPanel)}
          collectionSize={collection.length}
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
            onAddVideo={addToCollection}
            onRemoveVideo={removeFromCollection}
            onDownloadVideo={handleDownloadRequest}
            onCancelDownload={cancelDownload}
            collection={collection}
            downloadStates={downloadStates}
          />
        </div>
      </main>

      <CollectionPanel
        isOpen={showPanel}
        onClose={() => setShowPanel(false)}
        videos={collection}
        onRemoveVideo={removeFromCollection}
        onClearCollection={clearCollection}
        onDownloadAll={() => handleDownloadRequest(collection)}
      />

      <DownloadModal
        key={modalKey}
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onConfirm={confirmDownload}
        title={
          videoToDownload
            ? Array.isArray(videoToDownload)
              ? `Download Collection (${videoToDownload.length} items)`
              : `Download ${videoToDownload.title}`
            : "Download Preferences"
        }
        initialOptions={preferences}
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
