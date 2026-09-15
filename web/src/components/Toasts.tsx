import { useEffect } from "react";

export interface ToastItem {
  id: number;
  msg: string;
  kind: string;
}

export function ToastStack({ toasts, onDismiss }: { toasts: ToastItem[]; onDismiss: (id: number) => void }) {
  return (
    <div className="toasts" aria-live="assertive">
      {toasts.map((t) => (
        <Toast key={t.id} t={t} onDismiss={onDismiss} />
      ))}
    </div>
  );
}

function Toast({ t, onDismiss }: { t: ToastItem; onDismiss: (id: number) => void }) {
  useEffect(() => {
    const id = window.setTimeout(() => onDismiss(t.id), 4600);
    return () => window.clearTimeout(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [t.id]);

  return (
    <div className={`toast ${t.kind === "error" ? "err" : ""}`} onClick={() => onDismiss(t.id)}>
      <span className="toast-bar" />
      <span className="toast-kind">{t.kind === "error" ? "ERR" : t.kind === "live" ? "LIVE" : "PAPER"}</span>
      <span className="toast-msg">{t.msg}</span>
    </div>
  );
}