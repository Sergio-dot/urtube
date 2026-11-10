import {
  ArrowDownTrayIcon,
  Cog6ToothIcon,
  MoonIcon,
  SunIcon,
} from "@heroicons/react/24/outline";
import type { Health } from "../api";
import logoDark from "../assets/logo-dark.png";
import logoLight from "../assets/logo-light.png";
import { useTheme } from "../hooks/useTheme";

interface HeaderProps {
  /** The server's health, or null while unknown. */
  health: Health | null;
  /** Why downloads can't be saved, if they can't. */
  downloadDirError?: string;
  /** Downloads the server is still working on. */
  activeCount: number;
  /** Videos set aside to download together. */
  selectedCount: number;
  onOpenActivity: () => void;
  onOpenSettings: () => void;
}

export default function Header({
  health,
  downloadDirError,
  activeCount,
  selectedCount,
  onOpenActivity,
  onOpenSettings,
}: HeaderProps) {
  const { theme, toggleTheme } = useTheme();
  const badge = activeCount + selectedCount;

  return (
    <header className="sticky top-0 z-30 border-b border-line bg-bg/85 backdrop-blur-md">
      <div className="mx-auto flex h-14 max-w-[1600px] items-center gap-4 px-4 sm:px-6">
        <h1>
          <img
            alt="urtube"
            src={logoLight}
            className="h-4 w-auto dark:hidden"
          />
          <img
            alt="urtube"
            src={logoDark}
            className="hidden h-4 w-auto dark:block"
          />
        </h1>

        <ServerStatus
          health={health}
          downloadDirError={downloadDirError}
          onOpenSettings={onOpenSettings}
        />

        <div className="ml-auto flex items-center">
          {/* From xl up the activity panel is always on screen instead. */}
          <button
            type="button"
            onClick={onOpenActivity}
            aria-label={`Downloads: ${activeCount} active, ${selectedCount} selected`}
            title="Downloads"
            className="icon-btn relative xl:hidden"
          >
            <ArrowDownTrayIcon aria-hidden="true" className="size-5" />
            {badge > 0 && (
              <span
                aria-hidden="true"
                className="absolute top-1 right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-accent px-1 text-[10px] font-bold text-on-accent"
              >
                {badge}
              </span>
            )}
          </button>
          <button
            type="button"
            onClick={onOpenSettings}
            aria-label="Settings"
            title="Settings"
            className="icon-btn"
          >
            <Cog6ToothIcon aria-hidden="true" className="size-5" />
          </button>
          <button
            type="button"
            onClick={toggleTheme}
            aria-label={
              theme === "dark"
                ? "Switch to light theme"
                : "Switch to dark theme"
            }
            title="Toggle theme"
            className="icon-btn -mr-2"
          >
            {theme === "dark" ? (
              <SunIcon aria-hidden="true" className="size-5" />
            ) : (
              <MoonIcon aria-hidden="true" className="size-5" />
            )}
          </button>
        </div>
      </div>
    </header>
  );
}

function ServerStatus({
  health,
  downloadDirError,
  onOpenSettings,
}: Pick<HeaderProps, "health" | "downloadDirError" | "onOpenSettings">) {
  const [dot, text] =
    health === null
      ? ["bg-muted", "Connecting…"]
      : !health.dependencies.ytdlp
        ? ["bg-danger", "yt-dlp unavailable on the server"]
        : downloadDirError
          ? ["bg-warn", "Download folder unavailable"]
          : ["bg-ok", "Ready"];

  const content = (
    <>
      <span aria-hidden="true" className={`size-2 rounded-full ${dot}`} />
      {text}
    </>
  );

  // The one problem the user can fix from here leads straight to the fix.
  if (health?.dependencies.ytdlp && downloadDirError) {
    return (
      <button
        type="button"
        onClick={onOpenSettings}
        title={downloadDirError}
        className="flex min-h-11 min-w-0 cursor-pointer items-center gap-2 text-left text-xs font-medium text-warn underline-offset-4 hover:underline"
      >
        {content}
        <span className="max-sm:hidden">· Fix in Settings</span>
      </button>
    );
  }
  return (
    <p
      role="status"
      className="flex min-w-0 items-center gap-2 text-xs font-medium text-muted"
    >
      {content}
    </p>
  );
}
