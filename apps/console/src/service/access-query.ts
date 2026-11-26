import { array, boolean, choice, integer, record, shape, text, type Parser } from "./strict-read";
import { closed, decimal, identifier, requireRead, timestamp } from "./service-query-read";

export type PermissionSelector = { any: true } | { any: false; id: string };
export type PermissionScope = { customer: PermissionSelector; legal_entity: PermissionSelector; contract: PermissionSelector; period: string };
export type PermissionState = "active" | "pending" | "expired" | "revoked" | "revocation_unconfirmed" | "ended";
export type PermissionSource = "grant" | "delegation";
export type Permission = {
  source: PermissionSource; id: string; subject: string; client: string; delegator: string; parent: string; redelegate: boolean;
  action: string; kind: string; fields: string[]; scope: PermissionScope; not_before: string; not_after: string; state: PermissionState; effective: boolean; version: number;
};
export type PermissionQuery = { q?: string; subject?: string; action?: string; customer?: string; legal_entity?: string; source?: PermissionSource; state?: PermissionState; expires_before?: string; limit?: number; after?: string };
export type PermissionEcho = { q: string; subject: string; action: string; customer: string; legal_entity: string; source: "" | PermissionSource; state: "" | PermissionState; expires_before: string; limit: number };
export type PermissionPage = {
  rows: Permission[]; search_input: string; query: PermissionEcho; next: string; reference_at: string; observed_at: string; revision: string; order: "source_then_id";
  count: { value: string; certainty: "exact" | "lower_bound" }; work_units: number; scan_stopped: boolean;
};
export type PermissionDetail = { workspace: string; row: Permission; revision: string; observed_at: string };

const states = ["active", "pending", "expired", "revoked", "revocation_unconfirmed", "ended"] as const;
const emptyOr = <T>(parse: Parser<T>): Parser<T | ""> => (v) => (v === "" ? "" : parse(v));
const selector: Parser<PermissionSelector> = (v) => {
  const r = closed<{ any: boolean; id: string }>({ any: boolean, id: text })(v);
  if (r.any) { requireRead(r.id === ""); return { any: true }; }
  return { any: false, id: identifier(r.id) };
};
const period: Parser<string> = (v) => { const s = text(v); requireRead(s === "any" || /^[0-9]{1,7}-[0-9]{1,7}$/.test(s)); return s; };
const permission = closed<Permission>({
  source: choice("grant", "delegation"), id: identifier, subject: emptyOr(identifier), client: emptyOr(identifier), delegator: emptyOr(identifier), parent: emptyOr(identifier), redelegate: boolean,
  action: identifier, kind: identifier, fields: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(identifier)(v); },
  scope: closed<PermissionScope>({ customer: selector, legal_entity: selector, contract: selector, period }), not_before: timestamp, not_after: timestamp, state: choice(...states), effective: boolean, version: integer,
});
export const permissionOf: Parser<Permission> = (v) => {
  const r = permission(v);
  requireRead(Date.parse(r.not_before) <= Date.parse(r.not_after) && r.version >= 1);
  if (r.source === "grant") requireRead(r.delegator === "" && r.parent === "" && !r.redelegate && (r.subject === "") !== (r.client === "") && r.state !== "ended");
  else requireRead(r.client === "" && r.subject !== "" && r.delegator !== "" && r.delegator !== r.subject);
  requireRead(!r.effective || r.state === "active");
  return r;
};
const echo = closed<PermissionEcho>({ q: text, subject: emptyOr(identifier), action: emptyOr(identifier), customer: emptyOr(identifier), legal_entity: emptyOr(identifier),
  source: emptyOr(choice("grant", "delegation")), state: emptyOr(choice(...states)), expires_before: emptyOr(timestamp), limit: integer });
const pageShape = closed<PermissionPage>({ rows: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(permissionOf)(v); }, search_input: text, query: echo, next: text,
  reference_at: timestamp, observed_at: timestamp, revision: (v) => { const s = text(v); requireRead(s.length > 0); return s; }, order: choice("source_then_id"),
  count: closed({ value: decimal, certainty: choice("exact", "lower_bound") }), work_units: integer, scan_stopped: boolean });
export const permissionPageOf: Parser<PermissionPage> = (v) => {
  const r = pageShape(v);
  requireRead(r.query.limit >= 1 && r.query.limit <= 100 && r.rows.length <= r.query.limit && r.next.length <= 4096 && r.work_units >= 0 && r.work_units <= 10000);
  requireRead(!!r.next === (r.count.certainty === "lower_bound") && (!r.scan_stopped || !!r.next) && BigInt(r.count.value) >= BigInt(r.rows.length));
  requireRead(Date.parse(r.reference_at) <= Date.parse(r.observed_at));
  for (let i = 1; i < r.rows.length; i++) {
    const a = r.rows[i - 1], b = r.rows[i];
    requireRead(a.source === b.source ? a.id < b.id : a.source === "grant" && b.source === "delegation");
  }
  return r;
};
export const permissionDetailOf: Parser<PermissionDetail> = closed<PermissionDetail>({ workspace: identifier, row: permissionOf, revision: (v) => { const s = text(v); requireRead(s.length > 0); return s; }, observed_at: timestamp });

function applies(stored: PermissionSelector, wanted: string): boolean {
  return wanted === "" || stored.any || stored.id === wanted;
}

type Send = (path: string, init: RequestInit, request: string) => Promise<Record<string, unknown>>;
type Validate = <T>(parse: () => T) => T;
const keys = ["q", "subject", "action", "customer", "legal_entity", "source", "state", "expires_before", "limit", "after"] as const;

export function accessQueryClient(send: Send, request: () => string, validate: Validate) {
  return {
    async accessPermissions(query: PermissionQuery): Promise<PermissionPage> {
      const captured = validate(() => {
        requireRead(Object.keys(query).every((k) => (keys as readonly string[]).includes(k)));
        const out: Record<string, string> = {};
        for (const key of keys) {
          const value = query[key];
          if (value === undefined || value === "") continue;
          if (key === "limit") { requireRead(Number.isSafeInteger(value) && (value as number) >= 1 && (value as number) <= 100); out[key] = String(value); continue; }
          requireRead(typeof value === "string");
          if (key === "after") requireRead((value as string).length <= 4096);
          else if (key === "q") { const trimmed = (value as string).trim(); requireRead(new TextEncoder().encode(trimmed).length <= 64 && trimmed !== ""); out[key] = trimmed; continue; }
          else if (key === "expires_before") timestamp(value);
          else if (key !== "source" && key !== "state") identifier(value);
          out[key] = value as string;
        }
        return out;
      });
      const tail = new URLSearchParams(captured).toString();
      const body = await send("/identity/access/permissions" + (tail ? "?" + tail : ""), { method: "GET" }, request());
      return validate(() => {
        const page = permissionPageOf(body);
        const e = page.query;
        requireRead(page.search_input === (captured.q ?? "") && (e.q !== "") === (captured.q !== undefined));
        requireRead(e.subject === (captured.subject ?? "") && e.action === (captured.action ?? "") && e.customer === (captured.customer ?? "") && e.legal_entity === (captured.legal_entity ?? "") &&
          e.source === (captured.source ?? "") && e.state === (captured.state ?? "") && e.expires_before === (captured.expires_before ?? "") && (captured.limit === undefined || e.limit === Number(captured.limit)));
        requireRead(page.rows.every((r) => (!e.source || r.source === e.source) && (!e.subject || r.subject === e.subject) && (!e.action || r.action === e.action) && (!e.state || r.state === e.state) &&
          applies(r.scope.customer, e.customer) && applies(r.scope.legal_entity, e.legal_entity) && (!e.expires_before || Date.parse(r.not_after) < Date.parse(e.expires_before)) &&
          (!e.q || r.id.toLowerCase().includes(e.q) || r.subject.toLowerCase().includes(e.q))));
        if (!captured.after) requireRead(BigInt(page.count.value) === BigInt(page.rows.length));
        return page;
      });
    },
    async accessPermission(source: PermissionSource, id: string): Promise<PermissionDetail> {
      const target = validate(() => { requireRead(source === "grant" || source === "delegation"); return identifier(id); });
      const body = await send("/identity/access/permissions/" + source + "/" + encodeURIComponent(target), { method: "GET" }, request());
      return validate(() => { const detail = permissionDetailOf(body); requireRead(detail.row.id === target && detail.row.source === source); return detail; });
    },
  };
}
