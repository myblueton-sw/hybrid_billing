export const queryKeys = [
  "period",
  "provider",
  "account",
  "basis",
  "q",
  "after",
  "account_after",
  "snapshot",
] as const;
export function focusQuery(params: URLSearchParams) {
  const q = new URLSearchParams();
  for (const k of queryKeys)
    if (params.has(`focus-${k}`))
      q.set(
        k,
        k === "q"
          ? params.get(`focus-${k}`)!.trim()
          : params.get(`focus-${k}`)!,
      );
  return q;
}
export function focusHref(
  path: "/home" | "/source-data",
  params: URLSearchParams,
  snapshot?: string,
) {
  const next = new URLSearchParams({ lang: params.get("lang") ?? "en" });
  for (const [k, v] of focusQuery(params))
    if (!["after", "account_after"].includes(k)) next.set(`focus-${k}`, v);
  if (snapshot) next.set("focus-snapshot", snapshot);
  return `#${path}?${next}`;
}
