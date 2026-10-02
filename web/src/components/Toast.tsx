import { useEffect } from "react";
import "./Toast.css";

export type ToastMessage = { text: string; actionLabel?: string; onAction?: () => void };

// Toast shows a short message at the bottom with an optional action (e.g.
// "Deshacer"), and hides itself after a few seconds.
export function Toast({ toast, onClose }: { toast: ToastMessage | null; onClose: () => void }) {
  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(onClose, 6000);
    return () => clearTimeout(id);
  }, [toast, onClose]);

  if (!toast) return null;
  return (
    <div className="toast" role="status">
      <span>{toast.text}</span>
      {toast.onAction && (
        <button
          type="button"
          className="link"
          onClick={() => {
            toast.onAction?.();
            onClose();
          }}
        >
          {toast.actionLabel ?? "Deshacer"}
        </button>
      )}
    </div>
  );
}
