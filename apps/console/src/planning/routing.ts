import {
  planningScreens,
  legacyRedirects,
  type PlanningScreen,
} from "./registry";
export type Audience = "admin" | "customer" | "unauthenticated";
export type RouteContext = {
  audience: Audience;
  viewable: ReadonlySet<string>;
  assistant: boolean;
  designTools: boolean;
};
export type ResolvedRoute = {
  screen: PlanningScreen | null;
  projection: Audience;
  path: string;
  query: URLSearchParams;
  objectId?: string;
  tab?: string;
  step?: string;
  tool?: string;
  legacyId?: string;
  denied: boolean;
};
const interimSteps = new Set([
  "start",
  "profile",
  "attribution",
  "connection",
  "diagnostics",
  "schedule",
  "first-receipt",
  "readiness",
]);
export function visibleTo(screen: PlanningScreen, context: RouteContext) {
  if (context.audience === "unauthenticated")
    return screen.audience === "prelogin";
  const audienceAllowed =
    context.audience === "customer"
      ? screen.audience === "portal" || screen.audience === "admin+portal"
      : screen.audience === "admin" || screen.audience === "admin+portal";
  return (
    audienceAllowed &&
    context.viewable.has(screen.id) &&
    (screen.id !== "A01" || context.assistant)
  );
}
export function resolveRoute(
  hash: string,
  context: RouteContext,
): ResolvedRoute {
  const raw = hash.replace(/^#/, ""),
    at = raw.indexOf("?");
  let path = at < 0 ? raw : raw.slice(0, at);
  const query = new URLSearchParams(at < 0 ? "" : raw.slice(at + 1));
  const base = { projection: context.audience, query, denied: false };
  const fallback = (): ResolvedRoute => {
    const id =
      context.audience === "admin"
        ? "U01"
        : context.audience === "customer"
          ? "C01"
          : "L01";
    const home = planningScreens.find((s) => s.id === id)!;
    return {
      ...base,
      screen: visibleTo(home, context) ? home : null,
      path: home.path,
      denied: !visibleTo(home, context),
    };
  };
  const redirect = Object.hasOwn(legacyRedirects, path)
    ? legacyRedirects[path]
    : undefined;
  const legacyId = redirect ? path : undefined;
  if (redirect) path = redirect.target;
  if (
    ["/design/screens", "/design/overview", "/help/contribute"].includes(path)
  ) {
    return context.audience === "admin" && context.designTools
      ? { ...base, screen: null, path, tool: path, legacyId }
      : fallback();
  }
  if (!path.startsWith("/") || /%2f|%5c|\\|\/\//i.test(path)) return fallback();
  let parts: string[];
  try {
    parts = path.slice(1).split("/").map(decodeURIComponent);
  } catch {
    return fallback();
  }
  if (
    parts.some(
      (p) => !p || p === "." || p === ".." || /[/?#\\\u0000-\u001f]/.test(p),
    )
  )
    return fallback();
  const screen = [...planningScreens]
    .sort((a, b) => b.path.length - a.path.length)
    .find((s) => path === s.path || path.startsWith(s.path + "/"));
  if (!screen || !visibleTo(screen, context)) return fallback();
  const suffix = parts.slice(screen.path.slice(1).split("/").length);
  if (screen.id === "M05" && suffix.length) return fallback();
  const isWizard = ["wizard", "prelogin-wizard"].includes(screen.archetype);
  let objectId: string | undefined,
    tab: string | undefined,
    step: string | undefined;
  if (isWizard) {
    if (
      suffix.length > 1 ||
      (suffix[0] &&
        !screen.tabs.some((t) => t.key === suffix[0]) &&
        !(screen.id === "S02" && interimSteps.has(suffix[0])))
    )
      return fallback();
    step = suffix[0];
  } else {
    if (suffix.length > 2) return fallback();
    objectId = suffix[0];
    tab = suffix[1];
    if (tab && !screen.tabs.some((t) => t.key === tab)) return fallback();
  }
  return { ...base, screen, path, objectId, tab, step, legacyId };
}
export function planningUrl(
  path: string,
  current: URLSearchParams,
  changes: Record<string, string> = {},
) {
  const query = new URLSearchParams(current);
  for (const [key, value] of Object.entries(changes))
    value ? query.set(key, value) : query.delete(key);
  return `#${path}${query.size ? "?" + query.toString() : ""}`;
}
