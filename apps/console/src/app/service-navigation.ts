import { canonicalScope } from "../shared/scope-query";
export function serviceQuery(query: URLSearchParams, screen: string) {
  const source = canonicalScope(query, screen),
    next = new URLSearchParams();
  for (const key of [
    "lang",
    "mode",
    "scope",
    "period",
    "le",
    "ccy",
    "service",
    "review",
  ]) {
    const value = source.get(key);
    if (value !== null) next.set(key, value);
  }
  return next;
}
