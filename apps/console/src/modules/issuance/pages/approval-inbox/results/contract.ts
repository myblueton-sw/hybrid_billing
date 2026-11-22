import { strictJSON } from "../documents/json";
export class ResultError extends Error {
 constructor(readonly code: string, readonly effect: "none" | "unknown" = "none", readonly operation = "") { super(code); }
}
export const fail = (): never => { throw new ResultError("invalid_result_response"); };
export const bytes = (value: string) => new TextEncoder().encode(value).length;
export const object = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);
export function exact(v: unknown, keys: readonly string[]): asserts v is Record<string, unknown> {
 if (!object(v) || Object.keys(v).length !== keys.length || keys.some(k => !Object.hasOwn(v, k))) fail();
}
export const id = (v: unknown): v is string => typeof v === "string" && /^[A-Za-z0-9._:-]{1,128}$/.test(v);
export const digest = (v: unknown): v is string => typeof v === "string" && /^[0-9a-f]{64}$/.test(v);
export const revision = (v: unknown): v is string => typeof v === "string" && /^r(?:0|[1-9][0-9]*)$/.test(v);
export function freeze<T>(v: T): Readonly<T> {
 if (v && typeof v === "object" && !Object.isFrozen(v)) { for (const child of Object.values(v)) freeze(child); Object.freeze(v); }
 return v;
}
export function json(v: unknown): unknown {
 if (typeof v !== "string" || bytes(v) > 4096) return fail();
 try { return strictJSON(v, 4096); } catch { return fail(); }
}
export type ReadEnvelope = Readonly<{ revision: string; at: string; fields: Readonly<Record<string, string>> }>;
export type ReadCommand = (command: string, target: string, fields: Record<string, string>) => Promise<ReadEnvelope>;
export type Preparation = Readonly<{ preparation: string; preparation_digest: string; document: string; document_digest: string; connection: string; customer: string; period: string; currency: string; preparation_actor: string }>;
export type ReferenceLine = Readonly<{ line: string; expected_product: string; expected_unit: string }>;
export type Permission = Readonly<{ state: "allowed" | "denied" | "unavailable"; code: string }>;
export type Readiness = Readonly<{ state: "ready" | "blocked" | "inapplicable" | "unchecked" | "unavailable"; code: string }>;
export type Context = Readonly<{ preparation: Preparation; revision: string; permission: Permission; readiness: Readiness; references: readonly ReferenceLine[] }>;
export function permission(v: unknown): Permission {
 exact(v, ["state", "code"]);
 if (typeof v.code !== "string" || !(v.state === "allowed" && v.code === "" || v.state === "denied" && ["permission_denied", "unauthenticated"].includes(String(v.code)) || v.state === "unavailable" && ["authority_unavailable", "dependency_unavailable"].includes(String(v.code)))) return fail();
 return freeze(v) as Permission;
}
export function readiness(v: unknown): Readiness {
 exact(v, ["state", "code"]);
 const codes: Record<string, readonly string[]> = { ready: [""], blocked: ["permission_denied", "unauthenticated", "prerequisite_failed", "evidence_stale", "self_approval_denied"], inapplicable: ["prerequisite_failed", "self_approval_denied"], unchecked: ["permission_denied", "unauthenticated", "authority_unavailable", "dependency_unavailable"], unavailable: ["authority_unavailable", "dependency_unavailable", "unsupported_capability"] };
 if (typeof v.state !== "string" || typeof v.code !== "string" || !codes[v.state]?.includes(v.code)) return fail();
 return freeze(v) as Readiness;
}
export function preparationOf(raw: string): Preparation {
 const v = json(raw); exact(v, ["preparation", "preparation_digest", "document", "document_digest", "connection", "customer", "period", "currency", "preparation_actor"]);
 if (![v.preparation, v.document, v.connection, v.customer, v.preparation_actor].every(id) || !digest(v.preparation_digest) || !digest(v.document_digest) || typeof v.period !== "string" || !/^(?!0000)\d{4}-(?:0[1-9]|1[0-2])$/.test(v.period) || typeof v.currency !== "string" || !/^[A-Z]{3}$/.test(v.currency)) return fail();
 return freeze(v) as Preparation;
}
export function pageRows(f: Readonly<Record<string, string>>, fixed: readonly string[]): string[] {
 const rows = Object.keys(f).filter(k => !fixed.includes(k)).sort();
 if (rows.length > 10 || rows.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`) || fixed.some(k => !Object.hasOwn(f, k)) || Object.values(f).some(v => typeof v !== "string" || bytes(v) > 4096)) return fail();
 return rows.map(k => f[k]);
}
