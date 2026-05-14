import githubIcon from "../assets/github-svgrepo-com.svg";
export default function Footer({ version }: { version: string | null }) {
  return (
    <>
      <footer className="bg-indigo-600 shadow-xs border-t border-default sticky bottom-0 w-full">
        <div className="w-full mx-auto max-w-7xl p-4 md:flex md:items-center md:justify-between">
          <div className="flex flex-col sm:flex-row sm:items-center gap-2">
            {version && (
              <span className="text-white text-xs text-body-muted sm:pl-2">
                yt-dlp version: {version}
              </span>
            )}
          </div>
          <ul className="flex flex-wrap items-center mt-3 text-sm font-medium text-gray-500 dark:text-gray-400 sm:mt-0">
            <li>
              <a
                href="https://github.com/Sergio-dot/urtube"
                className="hover:text-gray-900 dark:hover:text-white transition-colors"
                target="_blank"
                rel="noopener noreferrer"
              >
                <img
                  src={githubIcon}
                  alt="Github Repository Link"
                  className="w-5 h-5 invert"
                />
              </a>
            </li>
          </ul>
        </div>
      </footer>
    </>
  );
}
