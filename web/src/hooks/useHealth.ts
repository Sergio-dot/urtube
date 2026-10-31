import { useEffect, useState } from "react";
import { getHealth, type Health } from "../api";

/** The server's dependency status, or null until it has answered. */
export function useHealth() {
  const [health, setHealth] = useState<Health | null>(null);

  useEffect(() => {
    let active = true;
    getHealth()
      .then((h) => active && setHealth(h))
      .catch((err) => console.error("Failed to check health:", err));
    return () => {
      active = false;
    };
  }, []);

  return health;
}
