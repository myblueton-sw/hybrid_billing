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
let navigationGuard: (() => boolean) | null = null;
let acceptedHash = typeof window === "undefined" ? "" : window.location.hash;
export function setNavigationGuard(guard: (() => boolean) | null) {
  navigationGuard = guard;
}
if (typeof window !== "undefined") {
  const guardNavigation = () => {
    if (window.location.hash === acceptedHash) return;
    if (navigationGuard && !navigationGuard()) {
      window.history.replaceState(null, "", acceptedHash || "#");
      return;
    }
    acceptedHash = window.location.hash;
  };
  window.addEventListener("popstate", guardNavigation);
  window.addEventListener("hashchange", guardNavigation);
}
