import { useCallback, useEffect, useState } from "react";
import {
  getSettings,
  updateSettings,
  type Settings,
  type SettingsChange,
} from "../api";

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

  /** Saves a change; rejects with the server's reason. */
  const save = useCallback(async (change: SettingsChange) => {
    setSettings(await updateSettings(change));
  }, []);

  return { settings, refresh, save };
}
