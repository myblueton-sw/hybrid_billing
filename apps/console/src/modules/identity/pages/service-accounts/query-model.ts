import type {
  AccountQuery,
  CredentialQuery,
  PageQuery,
} from "../../../../service/service-query-types";
import { utf8Length } from "../../../../service/helpers";

export const accountKeys = [
  "q",
  "organization",
  "owner",
  "audience",
  "scope",
  "expiry",
  "state",
  "declaration",
  "limit",
  "after",
  "selected",
] as const;
export const credentialKeys = [
  "q",
  "service",
  "state",
  "collection",
  "limit",
  "after",
  "selected",
  "issue",
  "rotate",
] as const;
const id = /^[A-Za-z0-9._:-]{1,128}$/;
const enums: Record<string, string[]> = {
  expiry: ["all", "none", "expired", "expiring", "valid"],
  declaration: ["all", "recorded", "missing"],
  collection: ["all", "uncollected"],
};
export type Navigation<Q> =
  | { valid: true; query: Q; selected: string; issue: string; rotate: string }
  | { valid: false; field: string };

export function navigation(
  params: URLSearchParams,
  workspace: string,
  area: "accounts",
): Navigation<AccountQuery>;
export function navigation(
  params: URLSearchParams,
  workspace: string,
  area: "credentials",
): Navigation<CredentialQuery>;
export function navigation(
  params: URLSearchParams,
  workspace: string,
  area: "accounts" | "credentials",
): Navigation<AccountQuery | CredentialQuery> {
  const prefix = area === "accounts" ? "svc_a_" : "svc_c_";
  const keys: readonly string[] =
    area === "accounts" ? accountKeys : credentialKeys;
  const query: Record<string, string | number> = { workspace, limit: 25 };
  let selected = "",
    issue = "",
    rotate = "";
  if (utf8Length(params.toString()) > 16384)
    return { valid: false, field: "query" };
  for (const [name, value] of params) {
    if (!name.startsWith(prefix)) continue;
    const key = name.slice(prefix.length);
    if (!keys.includes(key) || params.getAll(name).length !== 1 || !value)
      return { valid: false, field: name };
    if (key === "q") {
      if (
        utf8Length(value.replace(/^\p{White_Space}+|\p{White_Space}+$/gu, "")) >
        64
      )
        return { valid: false, field: name };
    } else if (key === "limit") {
      if (!/^[1-9][0-9]{0,2}$/.test(value) || Number(value) > 100)
        return { valid: false, field: name };
    } else if (key === "after") {
      if (
        value.length > 4096 ||
        !/^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$/.test(value)
      )
        return { valid: false, field: name };
    } else if (key === "state") {
      if (
        !(
          area === "accounts"
            ? ["all", "active", "suspended", "retired"]
            : ["all", "active", "expired", "revoked"]
        ).includes(value)
      )
        return { valid: false, field: name };
    } else if (enums[key]) {
      if (!enums[key].includes(value)) return { valid: false, field: name };
    } else if (!id.test(value)) return { valid: false, field: name };
    if (key === "selected") selected = value;
    else if (key === "issue") issue = value;
    else if (key === "rotate") rotate = value;
    else query[key] = key === "limit" ? Number(value) : value;
  }
  if (query.scope && !query.audience)
    return { valid: false, field: prefix + "scope" };
  return {
    valid: true,
    query: query as AccountQuery | CredentialQuery,
    selected,
    issue,
    rotate: rotate && rotate === selected ? rotate : "",
  };
}

export function queryKey(query: PageQuery): string {
  const pairs = Object.entries(query)
    .filter(([key]) => key !== "after")
    .sort(([a], [b]) => a.localeCompare(b));
  return JSON.stringify(pairs);
}

export function cursorTrail() {
  const trails = new Map<string, string[]>();
  return {
    next(key: string, current: string, next: string) {
      if (!next || current === next) return false;
      const trail = trails.get(key) ?? [];
      trails.clear();
      trails.set(key, [...trail.slice(-19), current]);
      return true;
    },
    previous(key: string): string {
      const trail = trails.get(key);
      if (!trail?.length) return "";
      const last = trail.pop()!;
      return last;
    },
    clear() {
      trails.clear();
    },
  };
}

export function clearOwned(
  params: URLSearchParams,
  area: "accounts" | "credentials",
): Record<string, null> {
  const prefix = area === "accounts" ? "svc_a_" : "svc_c_";
  return Object.fromEntries(
    [...params.keys()]
      .filter((key) => key.startsWith(prefix))
      .map((key) => [key, null]),
  );
}
