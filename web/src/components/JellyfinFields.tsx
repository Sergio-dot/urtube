import { useState } from "react";
import { testJellyfin, type JellyfinSettings } from "../api";

interface JellyfinFieldsProps {
  saved: JellyfinSettings | undefined;
  url: string;
  onUrlChange: (url: string) => void;
  /** What the user typed; empty keeps the saved key. */
  apiKey: string;
  onApiKeyChange: (key: string) => void;
}

const inputClass =
  "mt-2 block w-full rounded-xl border-0 px-3 py-2.5 font-mono text-sm text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 dark:bg-white/5 dark:text-white dark:ring-white/10 dark:focus:ring-indigo-500";

/** Jellyfin URL and API key, with a connection test. */
export default function JellyfinFields({
  saved,
  url,
  onUrlChange,
  apiKey,
  onApiKeyChange,
}: JellyfinFieldsProps) {
  const [testing, setTesting] = useState(false);
  const [result, setResult] = useState<{ ok: boolean; text: string } | null>(
    null,
  );

  const test = async () => {
    setTesting(true);
    setResult(null);
    try {
      const message = await testJellyfin(
        url.trim(),
        apiKey.trim() || undefined,
      );
      setResult({ ok: true, text: message });
    } catch (err) {
      setResult({
        ok: false,
        text: err instanceof Error ? err.message : "Test failed",
      });
    } finally {
      setTesting(false);
    }
  };

  return (
    <fieldset className="mt-8 border-t border-gray-200 pt-6 dark:border-white/10">
      <legend className="sr-only">Jellyfin</legend>
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold dark:text-white">Jellyfin</h4>
        <span
          className={`text-xs font-medium ${
            saved?.enabled
              ? "text-green-600 dark:text-green-400"
              : "text-gray-500 dark:text-gray-400"
          }`}
        >
          {saved?.enabled ? "On" : "Off"}
        </span>
      </div>
      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
        After downloads finish, urtube asks Jellyfin to scan its libraries so
        new songs show up right away. Create an API key in Jellyfin under
        Dashboard → API Keys.
      </p>

      <label
        htmlFor="jellyfinUrl"
        className="mt-4 block text-sm font-medium dark:text-gray-300"
      >
        Server URL
      </label>
      <input
        id="jellyfinUrl"
        type="url"
        autoComplete="off"
        spellCheck={false}
        value={url}
        onChange={(e) => {
          onUrlChange(e.target.value);
          setResult(null);
        }}
        placeholder="http://192.168.1.10:8096"
        className={inputClass}
      />

      <label
        htmlFor="jellyfinKey"
        className="mt-4 block text-sm font-medium dark:text-gray-300"
      >
        API key
      </label>
      <input
        id="jellyfinKey"
        type="password"
        autoComplete="off"
        spellCheck={false}
        value={apiKey}
        onChange={(e) => {
          onApiKeyChange(e.target.value);
          setResult(null);
        }}
        placeholder={
          saved?.apiKeySet ? "Saved: leave empty to keep it" : "Paste the key"
        }
        className={inputClass}
      />

      <div className="mt-3 flex items-center gap-3">
        <button
          type="button"
          onClick={test}
          disabled={
            testing || !url.trim() || !(apiKey.trim() || saved?.apiKeySet)
          }
          className="rounded-lg px-3 py-1.5 text-sm font-semibold text-indigo-600 ring-1 ring-inset ring-indigo-500/30 hover:bg-indigo-50 disabled:opacity-50 dark:text-indigo-400 dark:hover:bg-white/5"
        >
          {testing ? "Testing…" : "Test connection"}
        </button>
        {result && (
          <span
            role="status"
            className={`text-xs ${
              result.ok
                ? "text-green-600 dark:text-green-400"
                : "text-red-600 dark:text-red-400"
            }`}
          >
            {result.text}
          </span>
        )}
      </div>
    </fieldset>
  );
}
