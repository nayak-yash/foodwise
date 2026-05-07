import { LoaderCircle, X } from "lucide-react";

export function Loading({ children }: { children: React.ReactNode }) {
  return (
    <div className="loading">
      <LoaderCircle className="spin" size={20} />
      {children}
    </div>
  );
}

export function Notice({
  message,
  error = false,
  onDismiss,
}: {
  message: string;
  error?: boolean;
  onDismiss: () => void;
}) {
  if (!message) return null;
  return (
    <div role={error ? "alert" : "status"} className={`message ${error ? "error" : "success"}`}>
      {message}
      <button aria-label="Dismiss message" onClick={onDismiss}>
        <X size={16} />
      </button>
    </div>
  );
}
