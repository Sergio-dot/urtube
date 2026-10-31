import {
  Disclosure,
  DisclosureButton,
  DisclosurePanel,
} from "@headlessui/react";
import {
  Bars3Icon,
  InboxIcon,
  XMarkIcon,
  SunIcon,
  MoonIcon,
} from "@heroicons/react/24/outline";
import logoDark from "../assets/logo-dark.png";
import logoLight from "../assets/logo-light.png";
import { useTheme } from "../hooks/useTheme";

const linkClass = (current: boolean) =>
  current
    ? "bg-gray-100 text-gray-900 dark:bg-gray-950/50 dark:text-white"
    : "text-gray-600 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-white/5 dark:hover:text-white";

export default function Navbar({
  onTogglePanel,
  queueSize,
}: {
  onTogglePanel: () => void;
  queueSize: number;
}) {
  const { theme, toggleTheme } = useTheme();
  return (
    <Disclosure
      as="nav"
      className="relative bg-white/50 after:pointer-events-none after:absolute after:inset-x-0 after:bottom-0 after:h-px after:bg-gray-200 dark:bg-gray-800/50 dark:after:bg-white/10"
    >
      <div className="mx-auto max-w-7xl px-2 sm:px-6 lg:px-8">
        <div className="relative flex h-16 items-center justify-between">
          <div className="absolute inset-y-0 left-0 flex items-center sm:hidden">
            {/* Mobile menu button*/}
            <DisclosureButton className="group relative inline-flex items-center justify-center rounded-md p-2 text-gray-400 hover:bg-white/5 hover:text-white focus:outline-2 focus:-outline-offset-1 focus:outline-indigo-500">
              <span className="absolute -inset-0.5" />
              <span className="sr-only">Open main menu</span>
              <Bars3Icon
                aria-hidden="true"
                className="block size-6 group-data-open:hidden"
              />
              <XMarkIcon
                aria-hidden="true"
                className="hidden size-6 group-data-open:block"
              />
            </DisclosureButton>
          </div>
          <div className="flex flex-1 items-center justify-center sm:items-stretch sm:justify-start">
            <div className="flex shrink-0 items-center">
              <img
                alt="urtube"
                src={logoLight}
                className="block h-4 w-auto dark:hidden"
              />
              <img
                alt="urtube"
                src={logoDark}
                className="hidden h-4 w-auto dark:block"
              />
            </div>
            <div className="hidden sm:ml-6 sm:block">
              <div className="flex space-x-4">
                <a
                  href="#"
                  aria-current="page"
                  className={`${linkClass(true)} rounded-md px-3 py-2 text-sm font-medium`}
                >
                  Dashboard
                </a>
                <button
                  type="button"
                  onClick={onTogglePanel}
                  className={`${linkClass(false)} rounded-md px-3 py-2 text-sm font-medium`}
                >
                  Queue
                </button>
              </div>
            </div>
          </div>
          <div className="absolute inset-y-0 right-0 flex items-center pr-2 sm:static sm:inset-auto sm:ml-6 sm:pr-0">
            <button
              type="button"
              onClick={onTogglePanel}
              aria-label={`Queue (${queueSize})`}
              title="Queue"
              className="relative rounded-full p-1 text-gray-500 hover:text-gray-900 focus:outline-2 dark:text-gray-400 dark:hover:text-white"
            >
              {queueSize > 0 && (
                <span className="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
                  {queueSize}
                </span>
              )}
              <InboxIcon aria-hidden="true" className="size-6" />
            </button>
            <button
              onClick={toggleTheme}
              type="button"
              className="relative rounded-full p-1 text-gray-500 hover:text-gray-900 focus:outline-2 dark:text-gray-400 dark:hover:text-white"
            >
              <span className="sr-only">Toggle theme</span>
              {theme === "dark" ? (
                <SunIcon className="size-6" />
              ) : (
                <MoonIcon className="size-6" />
              )}
            </button>
          </div>
        </div>
      </div>

      <DisclosurePanel className="sm:hidden">
        <div className="space-y-1 px-2 pt-2 pb-3">
          <DisclosureButton
            as="a"
            href="#"
            aria-current="page"
            className={`${linkClass(true)} block rounded-md px-3 py-2 text-base font-medium`}
          >
            Dashboard
          </DisclosureButton>
          <DisclosureButton
            as="button"
            type="button"
            onClick={onTogglePanel}
            className={`${linkClass(false)} block w-full text-left rounded-md px-3 py-2 text-base font-medium`}
          >
            Queue
          </DisclosureButton>
        </div>
      </DisclosurePanel>
    </Disclosure>
  );
}
