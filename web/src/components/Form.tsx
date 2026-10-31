import { useState } from "react";
import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
} from "@headlessui/react";
import { MagnifyingGlassIcon, CheckIcon } from "@heroicons/react/20/solid";
import { ChevronUpDownIcon } from "@heroicons/react/16/solid";
import {
  MAX_MUSIC_RESULTS,
  type Health,
  type SearchParams,
  type SearchSource,
} from "../api";
import { useLocalStorage } from "../hooks/useLocalStorage";

interface FormProps {
  onSearch: (params: SearchParams) => void;
  /** The server's health, or null while unknown. */
  health: Health | null;
  /** Why downloads can't be saved, if they can't. */
  downloadDirError?: string;
  onOpenSettings: () => void;
}

interface LimitOption {
  label: string;
  value: number;
}

const limitOptions: LimitOption[] = [5, 10, 15, 20, 50].map((value) => ({
  label: `${value} results`,
  value,
}));

const sources: { value: SearchSource; label: string }[] = [
  { value: "videos", label: "Videos" },
  { value: "music", label: "Music" },
];

const isSearchSource = (v: unknown): v is SearchSource =>
  v === "videos" || v === "music";

export default function Form({
  onSearch,
  health,
  downloadDirError,
  onOpenSettings,
}: FormProps) {
  const [query, setQuery] = useState("");
  const [limit, setLimit] = useState(limitOptions[0]);
  const [includeLive, setIncludeLive] = useState(false);
  const [source, setSource] = useLocalStorage<SearchSource>(
    "urtube_search_source",
    "videos",
    isSearchSource,
  );

  const music = source === "music";
  const options = music
    ? limitOptions.filter((o) => o.value <= MAX_MUSIC_RESULTS)
    : limitOptions;
  // A limit too high for music shows (and searches) as the music maximum.
  const effectiveLimit = options.includes(limit) ? limit : options.at(-1)!;

  const handleSubmit = (e: React.SubmitEvent) => {
    e.preventDefault();
    const trimmed = query.trim();
    if (trimmed) {
      onSearch({
        query: trimmed,
        limit: effectiveLimit.value,
        includeLive: !music && includeLive,
        source,
      });
    }
  };

  const [statusColor, statusText] =
    health === null
      ? ["bg-gray-400", "Connecting…"]
      : !health.dependencies.ytdlp
        ? ["bg-red-500", "yt-dlp unavailable on the server"]
        : downloadDirError
          ? ["bg-amber-500", "Download folder unavailable"]
          : ["bg-green-500", "Ready to download"];

  return (
    <div className="flex flex-col items-center justify-center py-20 px-4 sm:px-6 lg:px-8">
      <div className="w-full max-w-2xl space-y-10">
        <div className="text-center">
          <h2 className="text-4xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-6xl">
            Dashboard
          </h2>
          <p className="mt-6 text-lg leading-8 text-gray-600 dark:text-gray-400">
            Search for videos to start downloading your favorite content.
          </p>
        </div>

        <form className="mt-10 space-y-3" onSubmit={handleSubmit}>
          {/* Row 1: search input + limit select */}
          <div className="flex items-center gap-x-3">
            <div className="relative grow">
              <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                <MagnifyingGlassIcon
                  className="h-5 w-5 text-gray-400"
                  aria-hidden="true"
                />
              </div>
              <input
                type="text"
                name="search"
                id="search"
                autoComplete="off"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                className="block w-full rounded-2xl border-0 py-4 pl-12 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6 dark:bg-white/5 dark:text-white dark:ring-white/10 dark:focus:ring-indigo-500 transition-all"
                placeholder="e.g. Never gonna give you up"
              />
            </div>

            <div className="w-40 shrink-0">
              <SelectMenu
                options={options}
                selected={effectiveLimit}
                onChange={setLimit}
              />
            </div>
          </div>

          {/* Row 2: source, live streams, submit */}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
              <div
                role="radiogroup"
                aria-label="Search in"
                className="inline-flex rounded-xl bg-gray-100 p-1 dark:bg-white/5"
              >
                {sources.map(({ value, label }) => (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={source === value}
                    onClick={() => setSource(value)}
                    className={`rounded-lg px-3 py-1.5 text-sm font-medium transition-all ${
                      source === value
                        ? "bg-white text-gray-900 shadow-sm dark:bg-indigo-500 dark:text-white"
                        : "text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white"
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
              {music ? (
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  Songs with artist and album
                </span>
              ) : (
                <label className="flex cursor-pointer items-center gap-x-3 select-none">
                  <div className="group grid size-4 grid-cols-1 shrink-0">
                    <input
                      id="includeLive"
                      name="includeLive"
                      type="checkbox"
                      checked={includeLive}
                      onChange={(e) => setIncludeLive(e.target.checked)}
                      className="col-start-1 row-start-1 appearance-none rounded-sm border border-gray-300 bg-white checked:border-indigo-500 checked:bg-indigo-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 dark:border-white/10 dark:bg-white/5 dark:checked:border-indigo-500 dark:checked:bg-indigo-500"
                    />
                    <svg
                      fill="none"
                      viewBox="0 0 14 14"
                      className="pointer-events-none col-start-1 row-start-1 size-3.5 self-center justify-self-center stroke-white group-has-disabled:stroke-white/25"
                    >
                      <path
                        d="M3 8L6 11L11 3.5"
                        strokeWidth={2}
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        className="opacity-0 group-has-checked:opacity-100"
                      />
                    </svg>
                  </div>
                  <span className="text-sm font-medium text-gray-700 dark:text-white">
                    Include live streams
                  </span>
                </label>
              )}
            </div>

            <button
              type="submit"
              className="rounded-2xl bg-indigo-600 px-8 py-4 text-sm font-semibold text-white shadow-xs hover:bg-indigo-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 dark:bg-indigo-500 dark:hover:bg-indigo-400 transition-all"
            >
              Search
            </button>
          </div>
        </form>

        <div className="mt-10 flex items-center justify-center gap-x-6">
          {downloadDirError ? (
            <button
              type="button"
              onClick={onOpenSettings}
              title={downloadDirError}
              className="flex items-center gap-x-2 text-sm text-amber-600 underline-offset-4 hover:underline dark:text-amber-400"
            >
              <span
                className={`flex h-2 w-2 rounded-full ${statusColor}`}
              ></span>
              {statusText}: fix in Settings
            </button>
          ) : (
            <div className="flex items-center gap-x-2 text-sm text-gray-500 dark:text-gray-400 select-none">
              <span
                className={`flex h-2 w-2 rounded-full ${statusColor}`}
              ></span>
              {statusText}
            </div>
          )}
          <div className="h-4 w-px bg-gray-200 dark:bg-white/10"></div>
          <p className="text-sm text-gray-500 dark:text-gray-400 select-none">
            Powered by{" "}
            <code className="font-mono text-indigo-600 dark:text-indigo-400">
              yt-dlp
            </code>
          </p>
        </div>
      </div>
    </div>
  );
}

interface SelectMenuProps {
  options: LimitOption[];
  selected: LimitOption;
  onChange: (option: LimitOption) => void;
}

function SelectMenu({ options, selected, onChange }: SelectMenuProps) {
  return (
    <Listbox value={selected} onChange={onChange}>
      <div className="relative">
        <ListboxButton className="grid w-full cursor-default grid-cols-1 rounded-2xl bg-white py-4 pr-2 pl-4 text-left text-gray-900 ring-gray-300 ring-1 ring-inset focus:ring-2 focus:ring-inset focus:ring-indigo-500 sm:text-sm/6  dark:bg-white/5 dark:text-white dark:ring-white/10 dark:focus:ring-indigo-500 transition-all">
          <span className="col-start-1 row-start-1 flex items-center gap-3 pr-6">
            <span className="block truncate">{selected.label}</span>
          </span>
          <ChevronUpDownIcon
            aria-hidden="true"
            className="col-start-1 row-start-1 size-5 self-center justify-self-end text-gray-400 sm:size-4"
          />
        </ListboxButton>

        <ListboxOptions
          transition
          className="absolute z-10 mt-2 max-h-56 w-full overflow-auto rounded-xl bg-gray-200 py-1 text-base ring-1 ring-white/10 data-leave:transition data-leave:duration-100 data-leave:ease-in data-closed:data-leave:opacity-0 sm:text-sm dark:bg-gray-900"
        >
          {options.map((option) => (
            <ListboxOption
              key={option.value}
              value={option}
              className="group relative cursor-default py-2 pr-9 pl-3 text-gray-900 dark:text-white select-none data-focus:bg-indigo-500 data-focus:outline-hidden"
            >
              <div className="flex items-center">
                <span className="block truncate font-normal group-data-selected:font-semibold">
                  {option.label}
                </span>
              </div>

              <span className="absolute inset-y-0 right-0 flex items-center pr-4 text-indigo-400 group-not-data-selected:hidden group-data-focus:text-white">
                <CheckIcon aria-hidden="true" className="size-5" />
              </span>
            </ListboxOption>
          ))}
        </ListboxOptions>
      </div>
    </Listbox>
  );
}
