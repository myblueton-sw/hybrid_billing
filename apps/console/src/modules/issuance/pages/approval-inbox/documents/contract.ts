import { strictJSON } from "./json";
import { DocumentError } from "./error";
export { DocumentError } from "./error";
export const fail = (): never => { throw new DocumentError("invalid_document_response"); };
export const object = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);
export function exact(v: unknown, keys: readonly string[]): asserts v is Record<string, unknown> {
  if (!object(v) || Object.keys(v).length !== keys.length || keys.some(k => !Object.hasOwn(v, k))) fail();
}
export const id = (v: unknown): v is string => typeof v === "string" && /^[A-Za-z0-9._:-]{1,128}$/.test(v);
export const digest = (v: unknown): v is string => typeof v === "string" && /^[0-9a-f]{64}$/.test(v);
export const revision = (v: unknown): v is string => typeof v === "string" && /^r(?:0|[1-9][0-9]*)$/.test(v);
export const bytes = (s: string) => new TextEncoder().encode(s).length;
export function timestamp(v: unknown): v is string {
  return typeof v === "string" && /^(?!0000)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/.test(v) && Number.isFinite(Date.parse(v)) && new Date(Date.parse(v)).toISOString().slice(0, 19) === v.slice(0, 19);
}
export function decimal(v: unknown, scale = 4, positive = false): v is string {
  return typeof v === "string" && v.length <= 64 && /^(?:0|[1-9][0-9]*)(?:\.[0-9]*[1-9])?$/.test(v) && (v.split(".")[1]?.length ?? 0) <= scale && (!positive || v !== "0");
}
export function json(raw: unknown): unknown {
  if (typeof raw !== "string" || bytes(raw) > 4096) return fail();
  return strictJSON(raw, 4096);
}
export function freeze<T>(v: T): Readonly<T> {
  if (v && typeof v === "object" && !Object.isFrozen(v)) { for (const child of Object.values(v)) freeze(child); Object.freeze(v); }
  return v;
}
export type Permission = Readonly<{ state: "allowed" | "denied" | "unavailable"; code: string }>;
export function permission(v: unknown): Permission {
  exact(v, ["state", "code"]);
  if (!(v.state === "allowed" && v.code === "" || v.state === "denied" && ["permission_denied", "unauthenticated"].includes(String(v.code)) || v.state === "unavailable" && ["authority_unavailable", "dependency_unavailable"].includes(String(v.code)))) return fail();
  return freeze({ state: v.state, code: v.code }) as Permission;
}
export type Organization = Readonly<{ WorkspaceID: string; IssuerLegalEntityID: string; BuyerLegalEntityID: string; PayerID: string; Currency: string; Path: "external_invoice" }>;
export type ContextHeader = Readonly<{ candidate: string; candidate_digest: string; profile: "synthetic_individual_net_v1"; currency: string; total: string; customer: string; period: string; organization: Organization; erp: Readonly<{ connection: string; product: string; tax_code: string; scale: number }> }>;
export type ChargeLine = Readonly<{ line: string; rule: string; kind: "rated"; amount: string }>;
export type FormContext = Readonly<{ request: string; digest: string; revision: string; header: ContextHeader; permission: Permission; lines: readonly ChargeLine[] }>;
export type DocumentRow = Readonly<{ document: string; digest: string; recorded_state: "pending" | "confirmed" | "revoked"; proposed_by: string; proposed_at: string }>;
export type PreparationRow = Readonly<{ preparation: string; document: string; document_digest: string; actor: string; recorded_receipt: string }>;
export type HistoryKind = "documents" | "preparations";
export type HistoryRow = DocumentRow | PreparationRow;
export type ReadEnvelope = Readonly<{ revision: string; at: string; fields: Readonly<Record<string, string>> }>;
export type RequestIdentity = Readonly<{ request: string; digest: string; workspace: string }>;
