export const viewFields = {
  L01: { method: ["workspace", "loginId"], mfa: ["issue"] },
  L02: { invitation: ["email"], identity: ["displayName"], result: [] },
  L03: {
    route: ["workspace", "loginId", "routeKind"],
    proof: ["issue"],
    replacement: [],
    result: [],
  },
  M01: {
    users: ["search"],
    invitations: ["email", "workspace", "roleRef", "expires"],
    membership: ["userRef", "workspace", "roleRef", "starts", "expires"],
    revocation: ["targetRef", "scope", "reason"],
  },
  M02: {
    providers: [
      "method",
      "workspace",
      "issuer",
      "subjectRef",
      "redirect",
      "audience",
      "endpoint",
      "caRef",
      "searchScope",
    ],
    policies: [
      "workspace",
      "method",
      "sso",
      "mfaRequirement",
      "mappingRef",
      "roleCeiling",
      "reason",
    ],
    test: ["connectionRef", "principalRef"],
    keys: ["connectionRef", "credentialRef", "version", "reason"],
    recovery: ["connectionRef", "evidenceRef", "reason"],
    sync: ["connectionRef", "verification", "reason"],
  },
  M03: {
    roles: ["roleRef", "action", "entityRef", "customerRef", "contractRef"],
    grants: [
      "subjectRef",
      "subjectKind",
      "action",
      "entityRef",
      "customerRef",
      "contractRef",
      "parentRef",
      "redelegation",
      "starts",
      "expires",
    ],
    effects: [
      "subjectRef",
      "action",
      "entityRef",
      "customerRef",
      "contractRef",
    ],
    support: ["supportRef", "scope", "reason", "expires"],
  },
  M04: {
    accounts: ["purpose", "organizationRef", "ownerRef", "audience", "scope"],
    credentials: ["accountRef", "credentialRef", "version"],
    usage: ["accountRef", "starts", "expires"],
    transfer: ["accountRef", "ownerRef", "reason"],
  },
  M05: {
    account: ["displayName", "email"],
    methods: [],
    recovery: [],
    sessions: ["sessionKind"],
    notifications: [
      "timeZone",
      "quietStart",
      "quietEnd",
      "digest",
      "severity",
      "channel",
    ],
  },
} as const;
export type ScreenId = keyof typeof viewFields;
export function fieldsFor(id: string, view: string): readonly string[] {
  if (!Object.hasOwn(viewFields, id)) return [];
  const s = viewFields[id as ScreenId];
  return s && Object.hasOwn(s, view)
    ? (s[view as keyof typeof s] as readonly string[])
    : [];
}
export function allowedView(id: string, view: string) {
  if (!Object.hasOwn(viewFields, id)) return false;
  const s = viewFields[id as ScreenId];
  return !!s && Object.hasOwn(s, view);
}
export function contextKey(id: string, view: string, params: URLSearchParams) {
  return JSON.stringify([
    id,
    view,
    ...[
      "scope",
      "le",
      "customer",
      "contract",
      "period",
      "asof",
      "ccy",
      "permission_revision",
    ].map((k) => params.get(k)),
  ]);
}
export type Rule = {
  required?: boolean;
  kind?:
    | "email"
    | "reference"
    | "url"
    | "ldaps"
    | "date"
    | "time"
    | "zone"
    | "service-scope";
  max?: number;
};
export function validateDraft(
  data: FormData,
  rules: Record<string, Rule>,
): string[] {
  const errors = Object.entries(rules)
    .filter(([key, r]) => {
      const raw = data.get(key);
      if (typeof raw !== "string") return r.required;
      const v = raw.trim();
      if (!v) return !!r.required;
      if (v.length > (r.max ?? 256) || /[\u0000-\u001f\u007f]/.test(v))
        return true;
      switch (r.kind) {
        case "email":
          return !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v);
        case "service-scope":
          return /\b(?:approve|approval)\b/i.test(v);
        case "reference":
          return !/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(v);
        case "url":
        case "ldaps":
          try {
            const u = new URL(v);
            return (
              !u.hostname ||
              u.protocol !== (r.kind === "ldaps" ? "ldaps:" : "https:") ||
              !!u.username ||
              !!u.password ||
              !!u.hash ||
              !!u.search
            );
          } catch {
            return true;
          }
        case "date": {
          const n = Date.parse(v + "T00:00:00Z");
          return (
            !/^\d{4}-\d{2}-\d{2}$/.test(v) ||
            v.startsWith("0000") ||
            !Number.isFinite(n) ||
            new Date(n).toISOString().slice(0, 10) !== v
          );
        }
        case "time":
          return !/^([01]\d|2[0-3]):[0-5]\d$/.test(v);
        case "zone":
          try {
            new Intl.DateTimeFormat("en", { timeZone: v });
            return false;
          } catch {
            return true;
          }
        default:
          return false;
      }
    })
    .map(([key]) => key);
  const starts = data.get("starts"),
    expires = data.get("expires");
  if (
    rules.starts?.kind === "date" &&
    rules.expires?.kind === "date" &&
    typeof starts === "string" &&
    typeof expires === "string" &&
    starts &&
    expires &&
    !errors.includes("starts") &&
    !errors.includes("expires") &&
    starts >= expires
  )
    errors.push("expires");
  const quietStart = data.get("quietStart"),
    quietEnd = data.get("quietEnd");
  if (
    rules.quietStart &&
    rules.quietEnd &&
    typeof quietStart === "string" &&
    quietStart &&
    quietStart === quietEnd &&
    !errors.includes("quietEnd")
  )
    errors.push("quietEnd");
  return errors;
}
export function reviewState(
  view: string,
  revision: number,
  dirty: boolean,
  stage: string,
  errors: string[],
  instance = "",
  variant = "",
) {
  return JSON.stringify({
    phase: "unconnected",
    view,
    revision,
    dirty,
    stage,
    errors,
    instance,
    variant,
  });
}
export const reference: Rule = { required: true, kind: "reference" };
export const required: Rule = { required: true };
