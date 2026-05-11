import "./App.css";
import Navbar from "./components/Navbar";
import Form from "./components/Form";
import SearchResults from "./components/SearchResults";
import ErrorModal from "./components/ErrorModal";
import { useEffect, useState } from "react";
import type { Video } from "./types";

function App() {
  const [showError, setShowError] = useState(false);
  const [errorTitle, setErrorTitle] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [results, setResults] = useState<Video[]>([]);
  const [loading, setLoading] = useState(false);

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
        }
      })
      .catch((err) => {
        console.error("Failed to check health:", err);
      });

    return () => clearTimeout(timer);
  }, []);

  return (
    <>
      <div className="navbar">
        <Navbar />
      </div>

      <div className="form">
        <Form onResults={setResults} onLoading={setLoading} />
      </div>

      {loading && (
        <div className="flex justify-center py-10">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500"></div>
        </div>
      )}

      <div className="results">
        <SearchResults results={results} />
      </div>

      <ErrorModal
        open={showError}
        onClose={setShowError}
        title={errorTitle}
        message={errorMessage}
      />
    </>
  );
}

export default App;
