import "./App.css";
import Navbar from "./components/Navbar";
import Form from "./components/Form";
import SearchResults from "./components/SearchResults";
import ErrorModal from "./components/ErrorModal";
import Footer from "./components/Footer";
import CollectionPanel from "./components/CollectionPanel";
import { useEffect, useState } from "react";
import type { Video } from "./types";

function App() {
  const [showError, setShowError] = useState(false);
  const [errorTitle, setErrorTitle] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [results, setResults] = useState<Video[]>([]);
  const [loading, setLoading] = useState(false);
  const [ytdlpVersion, setYtdlpVersion] = useState<string | null>(null);

  const [collection, setCollection] = useState<Video[]>([]);
  const [showPanel, setShowPanel] = useState(false);

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
            collection={collection}
          />
        </div>
      </main>

      <CollectionPanel
        isOpen={showPanel}
        onClose={() => setShowPanel(false)}
        videos={collection}
        onRemoveVideo={removeFromCollection}
        onClearCollection={clearCollection}
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
