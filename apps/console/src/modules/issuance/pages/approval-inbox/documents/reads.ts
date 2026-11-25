import { sum } from "./packet";
import { bytes, decimal, digest, exact, fail, freeze, id, json, permission, timestamp, type ChargeLine, type ContextHeader, type DocumentRow, type FormContext, type HistoryKind, type HistoryRow, type Permission, type PreparationRow, type ReadEnvelope, type RequestIdentity } from "./contract";

export function headerOf(raw: unknown, workspace: string): ContextHeader {
  const v = json(raw);
  exact(v, ["candidate", "candidate_digest", "profile", "currency", "total", "customer", "period", "organization", "erp"]);
  if (!id(v.candidate) || !digest(v.candidate_digest) || v.profile !== "synthetic_individual_net_v1" || typeof v.currency !== "string" || !/^[A-Z]{3}$/.test(v.currency) || !decimal(v.total, 4, true) || !id(v.customer) || typeof v.period !== "string" || !/^(?!0000)\d{4}-(?:0[1-9]|1[0-2])$/.test(v.period)) return fail();
  exact(v.organization, ["WorkspaceID", "IssuerLegalEntityID", "BuyerLegalEntityID", "PayerID", "Currency", "Path"]);
  const o = v.organization;
  if (o.WorkspaceID !== workspace || ![o.WorkspaceID, o.IssuerLegalEntityID, o.BuyerLegalEntityID, o.PayerID].every(id) || o.Currency !== v.currency || o.Path !== "external_invoice") return fail();
  exact(v.erp, ["connection", "product", "tax_code", "scale"]);
  if (![v.erp.connection, v.erp.product, v.erp.tax_code].every(id) || !Number.isInteger(v.erp.scale) || Number(v.erp.scale) < 0 || Number(v.erp.scale) > 4 || !decimal(v.total, Number(v.erp.scale), true)) return fail();
  return freeze(v) as ContextHeader;
}
export function lineOf(raw: unknown, scale: number): ChargeLine {
  const v = json(raw); exact(v, ["line", "rule", "kind", "amount"]);
  if (!id(v.line) || !id(v.rule) || v.kind !== "rated" || !decimal(v.amount, scale, true)) return fail();
  return freeze(v) as ChargeLine;
}
export function historyRowOf(raw: unknown, kind: HistoryKind): HistoryRow {
  const v = json(raw);
  if (kind === "documents") {
    exact(v, ["document", "digest", "recorded_state", "proposed_by", "proposed_at"]);
    if (!id(v.document) || !digest(v.digest) || !["pending", "confirmed", "revoked"].includes(String(v.recorded_state)) || !id(v.proposed_by) || !timestamp(v.proposed_at)) return fail();
    return freeze(v) as DocumentRow;
  }
  exact(v, ["preparation", "document", "document_digest", "actor", "recorded_receipt"]);
  if (!id(v.preparation) || !id(v.document) || !digest(v.document_digest) || !id(v.actor) || v.recorded_receipt !== "" && !digest(v.recorded_receipt)) return fail();
  return freeze(v) as PreparationRow;
}
export function pageRows(f: Readonly<Record<string, string>>, fixed: string[]): string[] {
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (keys.length > 10 || keys.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`) || fixed.some(k => !Object.hasOwn(f, k)) || Object.values(f).some(v => typeof v !== "string" || bytes(v) > 4096)) return fail();
  return keys.map(k => f[k]);
}
export type ReadCommand = (command: string, target: string, fields: Record<string, string>) => Promise<ReadEnvelope>;
function pageIdentity(p: ReadEnvelope, scope: RequestIdentity, snapshot: string) {
  const f = p.fields;
  if (f.mode !== "synthetic" || f.request !== scope.request || f.request_digest !== scope.digest || f.snapshot !== p.revision || snapshot && snapshot !== p.revision || f.next !== "" && !id(f.next)) fail();
}
export async function loadContext(read: ReadCommand, scope: RequestIdentity): Promise<FormContext> {
  if (!id(scope.request) || !digest(scope.digest) || !id(scope.workspace)) return fail();
  let after = "", snapshot = "", headerText = "", permissionText = "";
  let header: ContextHeader | undefined, proposal: Permission | undefined;
  const lines: ChargeLine[] = [], cursors = new Set<string>();
  for (let page = 0; page < 10; page++) {
    const p = await read("document_evidence.context", scope.request, { digest: scope.digest, ...(after ? { after, snapshot } : {}) });
    pageIdentity(p, scope, snapshot);
    const f = p.fields;
    if (snapshot && (headerText !== f.header || permissionText !== f.proposal_permission)) return fail();
    snapshot = p.revision; headerText = f.header; permissionText = f.proposal_permission;
    header = headerOf(f.header, scope.workspace); proposal = permission(json(f.proposal_permission));
    const rows = pageRows(f, ["mode", "request", "request_digest", "snapshot", "next", "header", "proposal_permission"]).map(raw => lineOf(raw, header!.erp.scale));
    for (const row of rows) { if (lines.length && row.line <= lines[lines.length - 1].line) return fail(); lines.push(row); }
    if (!f.next) {
      if (!lines.length || lines.length > 100 || sum(lines.map(l => l.amount)) !== header.total) return fail();
      return freeze({ request: scope.request, digest: scope.digest, revision: snapshot, header, permission: proposal, lines }) as FormContext;
    }
    if (rows.length !== 10 || f.next !== rows[rows.length - 1].line || cursors.has(f.next)) return fail();
    cursors.add(f.next); after = f.next;
  }
  return fail();
}
export async function loadHistory(read: ReadCommand, scope: RequestIdentity, kind: HistoryKind): Promise<Readonly<{ revision: string; rows: readonly HistoryRow[] }>> {
  if (!id(scope.request) || !digest(scope.digest) || !id(scope.workspace)) return fail();
  let after = "", snapshot = "";
  const rows: HistoryRow[] = [], cursors = new Set<string>();
  const key = (row: HistoryRow) => "preparation" in row ? row.preparation : row.document;
  for (let page = 0; page < 10; page++) {
    const p = await read("document_evidence.history", scope.request, { digest: scope.digest, kind, ...(after ? { after, snapshot } : {}) });
    pageIdentity(p, scope, snapshot);
    if (p.fields.kind !== kind || p.fields.current_eligibility !== "unchecked") return fail();
    snapshot = p.revision;
    const batch = pageRows(p.fields, ["mode", "request", "request_digest", "kind", "snapshot", "next", "current_eligibility"]).map(v => historyRowOf(v, kind));
    for (const row of batch) { if (rows.length && key(row) <= key(rows[rows.length - 1])) return fail(); rows.push(row); }
    if (!p.fields.next) return freeze({ revision: snapshot, rows });
    if (batch.length !== 10 || p.fields.next !== key(batch[batch.length - 1]) || cursors.has(p.fields.next)) return fail();
    cursors.add(p.fields.next); after = p.fields.next;
  }
  return fail();
}
