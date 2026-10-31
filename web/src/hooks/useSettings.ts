import { useCallback, useEffect, useState } from "react";
import { getSettings, updateSettings, type Settings } from "../api";

/** The server's settings, or null until loaded. */
export function useSettings() {
  const [settings, setSettings] = useState<Settings | null>(null);

  const refresh = useCallback(async () => {
    try {
      setSettings(await getSettings());
    } catch (err) {
      console.error("Failed to load settings:", err);
    }
  }, []);

  useEffect(() => {
    let active = true;
    getSettings()
      .then((s) => active && setSettings(s))
      .catch((err) => console.error("Failed to load settings:", err));
    return () => {
      active = false;
    };
  }, []);

  /** Saves a new download folder; rejects with the server's reason. */
  const saveDownloadDir = useCallback(async (dir: string) => {
    setSettings(await updateSettings(dir));
  }, []);

  return { settings, refresh, saveDownloadDir };
}
