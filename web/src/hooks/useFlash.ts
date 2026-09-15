import { useEffect, useRef, useState } from "react";

export type FlashDir = null | "up" | "down";

export function useFlash(value: number): FlashDir {
  const ref = useRef(value);
  const [dir, setDir] = useState<FlashDir>(null);

  useEffect(() => {
    if (ref.current === value) return;
    setDir(value > ref.current ? "up" : "down");
    ref.current = value;
    const t = window.setTimeout(() => setDir(null), 900);
    return () => window.clearTimeout(t);
  }, [value]);

  return dir;
}