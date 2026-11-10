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

  // The server only uses a saved key with the server it was saved for.
  const canUseSavedKey =
    saved?.apiKeySet &&
    url.trim().replace(/\/+$/, "") === saved.url.replace(/\/+$/, "");

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
    <fieldset className="border-t border-line pt-5">
      <legend className="sr-only">Jellyfin</legend>
      <div className="flex items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">Jellyfin library refresh</h3>
        <span
          className={`rounded-full border px-2 py-0.5 text-xs font-medium ${
            saved?.enabled
              ? "border-ok/40 text-ok"
              : "border-line-strong text-muted"
          }`}
        >
          {saved?.enabled ? "On" : "Off"}
        </span>
      </div>
      <p className="help mt-1">
        After downloads finish, urtube asks Jellyfin to scan its libraries so
        new songs show up right away. Create an API key in Jellyfin under
        Dashboard → API Keys.
      </p>

      <label htmlFor="jellyfinUrl" className="label mt-4">
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
        className="input mt-1.5 font-mono"
      />

      <label htmlFor="jellyfinKey" className="label mt-4">
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
          canUseSavedKey ? "Saved: leave empty to keep it" : "Paste the key"
        }
        className="input mt-1.5 font-mono"
      />

      <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-1">
        <button
          type="button"
          onClick={test}
          disabled={
            testing || !url.trim() || !(apiKey.trim() || canUseSavedKey)
          }
          className="btn btn-secondary"
        >
          {testing ? "Testing…" : "Test connection"}
        </button>
        {result && (
          <span
            role="status"
            className={`text-xs ${result.ok ? "text-ok" : "text-danger"}`}
          >
            {result.text}
          </span>
        )}
      </div>
    </fieldset>
  );
}
