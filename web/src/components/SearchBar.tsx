import { useState } from "react";
import { MagnifyingGlassIcon } from "@heroicons/react/20/solid";
import {
  MAX_MUSIC_RESULTS,
  type SearchParams,
  type SearchSource,
} from "../api";
import { useLocalStorage } from "../hooks/useLocalStorage";
import Segmented from "./Segmented";
import Spinner from "./Spinner";

interface SearchBarProps {
  onSearch: (params: SearchParams) => void;
  loading: boolean;
}

const limits = [5, 10, 15, 20, 50];

const sources: { value: SearchSource; label: string }[] = [
  { value: "videos", label: "Videos" },
  { value: "music", label: "Music" },
];

const isSearchSource = (v: unknown): v is SearchSource =>
  v === "videos" || v === "music";

export default function SearchBar({ onSearch, loading }: SearchBarProps) {
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(5);
  const [includeLive, setIncludeLive] = useState(false);
  const [source, setSource] = useLocalStorage<SearchSource>(
    "urtube_search_source",
    "videos",
    isSearchSource,
  );

  const music = source === "music";
  const options = music ? limits.filter((l) => l <= MAX_MUSIC_RESULTS) : limits;
  // A limit too high for music shows (and searches) as the music maximum.
  const effectiveLimit = Math.min(limit, options.at(-1)!);

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const trimmed = query.trim();
    if (trimmed) {
      onSearch({
        query: trimmed,
        limit: effectiveLimit,
        includeLive: !music && includeLive,
        source,
      });
    }
  };

  return (
    <form role="search" onSubmit={handleSubmit} className="space-y-3">
      <div className="flex gap-2">
        <div className="relative min-w-0 flex-1">
          <MagnifyingGlassIcon
            aria-hidden="true"
            className="pointer-events-none absolute top-1/2 left-3.5 size-5 -translate-y-1/2 text-muted"
          />
          <input
            type="search"
            name="search"
            aria-label={music ? "Search YouTube Music" : "Search YouTube"}
            autoComplete="off"
            enterKeyHint="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={
              music ? "Song, artist or album" : "Search YouTube videos"
            }
            className="input h-12 pl-11"
          />
        </div>
        <button
          type="submit"
          disabled={loading}
          className="btn btn-primary h-12 px-5"
        >
          {loading && <Spinner className="size-4" />}
          {loading ? "Searching…" : "Search"}
        </button>
      </div>

      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <Segmented
          legend="Search in"
          name="source"
          options={sources}
          value={source}
          onChange={setSource}
        />

        <label className="flex items-center gap-2 text-sm text-muted">
          Show
          <select
            value={effectiveLimit}
            onChange={(e) => setLimit(Number(e.target.value))}
            className="input h-11 w-auto cursor-pointer pr-8 sm:text-sm"
          >
            {options.map((l) => (
              <option key={l} value={l}>
                {l} results
              </option>
            ))}
          </select>
        </label>

        {music ? (
          <p className="text-sm text-muted">
            Songs come with artist and album.
          </p>
        ) : (
          <label className="flex min-h-11 cursor-pointer items-center gap-2 text-sm text-muted select-none">
            <input
              type="checkbox"
              checked={includeLive}
              onChange={(e) => setIncludeLive(e.target.checked)}
              className="size-4 cursor-pointer accent-(--accent)"
            />
            Include live streams
          </label>
        )}
      </div>
    </form>
  );
}
