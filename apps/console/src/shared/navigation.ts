import { useSyncExternalStore } from "react";
export function useLocation() {
  return useSyncExternalStore(
    (callback) => {
      window.addEventListener("hashchange", callback);
      return () => window.removeEventListener("hashchange", callback);
    },
    () => location.hash,
    () => "",
  );
}
export function routeUrl(id: string, values: Record<string, string> = {}) {
  const current = new URLSearchParams(location.hash.split("?")[1]);
  for (const [key, value] of Object.entries(values)) current.set(key, value);
  return `#${id}?${current.toString()}`;
}
