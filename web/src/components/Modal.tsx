import {
  Dialog,
  DialogBackdrop,
  DialogPanel,
  DialogTitle,
} from "@headlessui/react";
import { XMarkIcon } from "@heroicons/react/24/outline";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}

/**
 * The shell shared by the dialogs: a sheet rising from the bottom edge on
 * phones, where it is within thumb reach, and a centred panel elsewhere.
 */
export default function Modal({ open, onClose, title, children }: ModalProps) {
  return (
    <Dialog open={open} onClose={onClose} className="relative z-50">
      <DialogBackdrop
        transition
        className="fixed inset-0 bg-black/60 transition-opacity duration-200 ease-out data-closed:opacity-0"
      />
      <div className="fixed inset-0 overflow-y-auto">
        <div className="flex min-h-full items-end justify-center sm:items-center sm:p-4">
          <DialogPanel
            transition
            className="w-full rounded-t-2xl border border-line bg-surface pb-[env(safe-area-inset-bottom)] shadow-2xl transition duration-200 ease-out data-closed:translate-y-4 data-closed:opacity-0 sm:max-w-lg sm:rounded-2xl"
          >
            <div className="flex items-start gap-3 px-5 pt-4">
              <DialogTitle className="min-w-0 flex-1 pt-2 text-lg font-semibold">
                {title}
              </DialogTitle>
              <button
                type="button"
                onClick={onClose}
                aria-label="Close"
                className="icon-btn -mr-2"
              >
                <XMarkIcon aria-hidden="true" className="size-5" />
              </button>
            </div>
            {children}
          </DialogPanel>
        </div>
      </div>
    </Dialog>
  );
}
