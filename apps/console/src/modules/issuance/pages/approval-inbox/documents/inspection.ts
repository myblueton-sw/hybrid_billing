import { bytes, decimal, digest, exact, fail, freeze, id, json, permission, timestamp, type DocumentRow, type Organization, type Permission, type PreparationRow, type ReadEnvelope, type RequestIdentity } from "./contract";
import { rationaleValid, sum } from "./packet";
export type BasisLine = Readonly<{ Line: string; Rule: string; Kind: "rated"; Product: string; Quantity: string; Unit: string; TaxCode: string; Tax: string; Evidence: string }>;
export type Basis = Readonly<{ ID: string; Mode: "synthetic"; Profile: "synthetic_individual_net_v1"; Evidence: string; Workspace: string; Request: string; RequestDigest: string; Candidate: string; CandidateDigest: string; Path: "direct"; Document: Organization; Connection: Readonly<Record<string, unknown>>; Mappings: readonly Readonly<Record<string, unknown>>[]; Survey: Readonly<Record<string, unknown>>; ExpectedNet: string; ExpectedTax: string; Lines: readonly BasisLine[] }>;
export type DocumentInspection = Readonly<{ row: DocumentRow; revision: string; basis: Basis; basisDigest: string; rawBasis: string; reason: string; confirmation?: Readonly<{ actor: string; reason: string; at: string }>; revocation?: Readonly<{ actor: string; reason: string; at: string }> }>;
function ids(v: Record<string, unknown>, names: string[]) { if (names.some(k => !id(v[k]))) fail(); }
function record(v: unknown, keys: string[]): Record<string, unknown> { exact(v, keys); return v; }
function instant(v: string): bigint {
  const whole = v.slice(0, 19) + "Z", fraction = v.slice(19, -1).replace(/^\./, "");
  return BigInt(Date.parse(whole)) * 1000000n + BigInt(fraction.padEnd(9, "0"));
}
function time(v: Record<string, unknown>, key = "At") { if (!timestamp(v[key])) fail(); }
export function basisOf(raw: string, scope: RequestIdentity, document: string): Basis {
  const b = record(json(raw), ["ID", "Mode", "Profile", "Evidence", "Workspace", "Request", "RequestDigest", "Candidate", "CandidateDigest", "Path", "Document", "Connection", "Mappings", "Survey", "ExpectedNet", "ExpectedTax", "Lines"]);
  ids(b, ["ID", "Evidence", "Workspace", "Request", "Candidate"]);
  if (b.ID !== document || b.Workspace !== scope.workspace || b.Request !== scope.request || b.RequestDigest !== scope.digest || !digest(b.CandidateDigest) || b.Mode !== "synthetic" || b.Profile !== "synthetic_individual_net_v1" || b.Path !== "direct") return fail();
  const doc = record(b.Document, ["WorkspaceID", "IssuerLegalEntityID", "BuyerLegalEntityID", "PayerID", "Currency", "Path"]);
  ids(doc, ["WorkspaceID", "IssuerLegalEntityID", "BuyerLegalEntityID", "PayerID"]);
  if (doc.WorkspaceID !== scope.workspace || doc.Path !== "external_invoice" || typeof doc.Currency !== "string" || !/^[A-Z]{3}$/.test(doc.Currency)) return fail();
  const cv = record(b.Connection, ["Connection", "Author", "At", "Verified", "Evidence", "Verifier"]); ids(cv, ["Author"]); time(cv);
  if (typeof cv.Verified !== "boolean" || cv.Evidence !== "" && !id(cv.Evidence) || cv.Verifier !== "" && !id(cv.Verifier)) return fail();
  const c = record(cv.Connection, ["ID", "Entity", "System", "Transport", "Capabilities", "Currency", "Scale", "Interval", "Owner", "Template"]);
  ids(c, ["ID", "Entity", "System", "Owner"]);
  if (!["api", "file", "database"].includes(String(c.Transport)) || c.Currency !== doc.Currency || c.Entity !== doc.IssuerLegalEntityID || !Number.isInteger(c.Scale) || Number(c.Scale) < 0 || Number(c.Scale) > 4 || !Number.isSafeInteger(c.Interval) || Number(c.Interval) <= 0 || c.Template !== "" && !id(c.Template) || !Array.isArray(c.Capabilities) || c.Capabilities.length > 20 || !c.Capabilities.every(id) || new Set(c.Capabilities).size !== c.Capabilities.length) return fail();
  const scale = Number(c.Scale);
  if (!decimal(b.ExpectedNet, scale, true) || !decimal(b.ExpectedTax, scale) || !Array.isArray(b.Lines) || !b.Lines.length || b.Lines.length > 100) return fail();
  const seen = new Set<string>(), products = new Set<string>(), taxes = new Set<string>(); let unit = "";
  for (const rawLine of b.Lines) {
    const l = record(rawLine, ["Line", "Rule", "Kind", "Product", "Quantity", "Unit", "TaxCode", "Tax", "Evidence"]);
    ids(l, ["Line", "Rule", "Product", "Unit", "TaxCode", "Evidence"]);
    if (l.Kind !== "rated" || !decimal(l.Quantity, scale, true) || !decimal(l.Tax, scale) || seen.has(String(l.Line)) || unit && unit !== l.Unit) return fail();
    unit = String(l.Unit); seen.add(String(l.Line)); products.add(String(l.Product)); taxes.add(String(l.TaxCode));
  }
  if (sum(b.Lines.map(l => String(l.Tax))) !== b.ExpectedTax) return fail();
  const sv = record(b.Survey, ["Survey", "Author", "At"]); ids(sv, ["Author"]); time(sv);
  const survey = record(sv.Survey, ["Connection", "Customer", "Period", "Outcome", "Evidence"]); ids(survey, ["Connection", "Customer", "Evidence"]);
  if (survey.Connection !== c.ID || survey.Outcome !== "complete" || typeof survey.Period !== "string" || !/^(?!0000)\d{4}-(?:0[1-9]|1[0-2])$/.test(survey.Period)) return fail();
  const required = new Set([`customer:${survey.Customer}`, ...[...products].map(p => `product:${p}`), ...[...taxes].map(t => `tax:${t}`)]);
  if (!Array.isArray(b.Mappings) || b.Mappings.length !== required.size) return fail();
  for (const rawMapping of b.Mappings) {
    const mv = record(rawMapping, ["Mapping", "Revision", "Author", "At"]); ids(mv, ["Author"]); time(mv);
    const m = record(mv.Mapping, ["Connection", "Kind", "Platform", "External"]); ids(m, ["Connection", "Platform", "External"]);
    if (!["customer", "product", "tax"].includes(String(m.Kind)) || m.Connection !== c.ID || !Number.isSafeInteger(mv.Revision) || Number(mv.Revision) < 1 || !required.delete(`${m.Kind}:${m.Platform}`)) return fail();
  }
  if (required.size) return fail();
  return freeze(b) as Basis;
}
export function inspectionOf(p: ReadEnvelope, scope: RequestIdentity, row: DocumentRow): DocumentInspection {
  const f = p.fields, keys = ["mode", "document", "digest", "recorded_state", "receipt", "basis", "basis_digest", "current_eligibility", "proposed_by", "proposal_reason", "proposed_at"];
  for (const prefix of ["confirmation", "revocation"]) if (Object.keys(f).some(k => k.startsWith(prefix + "_"))) keys.push(prefix + "_by", prefix + "_reason", prefix + "_at");
  exact(f, keys);
  if (f.mode !== "synthetic" || f.document !== row.document || f.digest !== row.digest || !["pending", "confirmed", "revoked"].includes(f.recorded_state) || f.receipt !== "historical_evidence" || f.current_eligibility !== "unchecked" || !digest(f.basis_digest) || f.proposed_by !== row.proposed_by || f.proposed_at !== row.proposed_at || !rationaleValid(f.proposal_reason)) return fail();
  const event = (prefix: string) => {
    if (!Object.hasOwn(f, prefix + "_by")) return undefined;
    const actor = f[prefix + "_by"], reason = f[prefix + "_reason"], at = f[prefix + "_at"];
    if (!id(actor) || !rationaleValid(reason) || !timestamp(at) || instant(at) < instant(f.proposed_at)) return fail();
    return freeze({ actor, reason, at });
  };
  const confirmation = event("confirmation"), revocation = event("revocation");
  if (confirmation?.actor === f.proposed_by || confirmation && revocation && instant(revocation.at) < instant(confirmation.at) || f.recorded_state === "pending" && (confirmation || revocation) || f.recorded_state === "confirmed" && (!confirmation || revocation) || f.recorded_state === "revoked" && !revocation) return fail();
  return freeze({ row: { ...row, recorded_state: f.recorded_state as DocumentRow["recorded_state"] }, revision: p.revision, basis: basisOf(f.basis, scope, row.document), basisDigest: f.basis_digest, rawBasis: f.basis, reason: f.proposal_reason, confirmation, revocation });
}
export type Readiness = Readonly<{ state: "ready" | "unchecked" | "inapplicable" | "unavailable" | "blocked"; code: string }>;
export type Advice = Readonly<{ document: string; digest: string; revision: string; permissions: Readonly<Record<"propose" | "confirm" | "revoke" | "prepare", Permission>>; readiness: Readonly<Record<"confirm" | "revoke" | "prepare", Readiness>> }>;
export function adviceOf(p: ReadEnvelope, row: DocumentRow): Advice {
  const f = p.fields; exact(f, ["mode", "document", "digest", "current_eligibility", "permissions", "readiness", "preparation_scope"]);
  if (f.mode !== "synthetic" || f.document !== row.document || f.digest !== row.digest || f.current_eligibility !== "advisory" || f.preparation_scope !== "packet_only_key_capacity_and_source_coverage_unchecked") return fail();
  const rawPermissions = record(json(f.permissions), ["propose", "confirm", "revoke", "prepare"]), rawReady = record(json(f.readiness), ["confirm", "revoke", "prepare"]);
  const permissions = Object.fromEntries(Object.entries(rawPermissions).map(([k,v]) => [k, permission(v)])) as Advice["permissions"];
  const readiness: Record<string, Readiness> = {};
  for (const key of ["confirm", "revoke", "prepare"] as const) {
    const r = record(rawReady[key], ["state", "code"]), allowed = permissions[key].state === "allowed";
    const valid = !allowed ? r.state === "unchecked" && r.code === permissions[key].code : r.state === "ready" && r.code === "" || r.state === "inapplicable" && ["prerequisite_failed", "self_approval_denied"].includes(String(r.code)) || r.state === "unavailable" && ["authority_unavailable", "dependency_unavailable", "unsupported_capability"].includes(String(r.code)) || r.state === "blocked" && ["permission_denied", "unauthenticated", "prerequisite_failed", "evidence_stale", "self_approval_denied"].includes(String(r.code));
    if (!valid) return fail(); readiness[key] = r as Readiness;
  }
  return freeze({ document: row.document, digest: row.digest, revision: p.revision, permissions, readiness: readiness as Advice["readiness"] });
}
export type PreparationInspection = Readonly<{ row: PreparationRow; revision: string; observation: "matching_prepared" | "missing" | "unavailable" | "conflict"; observedDigest: string; submission: Readonly<Record<string, unknown>>; total: Readonly<Record<string, string>>; mappings: readonly string[] }>;
const amount = (v: unknown): v is string => typeof v === "string" && v.length <= 64 && /^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,4})?$/.test(v);
const canonical = (v: string) => v.includes(".") ? v.replace(/0+$/, "").replace(/\.$/, "") : v;
export function preparationInspectionOf(p: ReadEnvelope, row: PreparationRow): PreparationInspection {
  const f = p.fields; exact(f, ["mode", "preparation", "document_digest", "recorded_receipt", "observed_digest", "ledger_observation", "current_eligibility", "submission", "total", "mappings", "actor"]);
  if (f.mode !== "synthetic" || f.preparation !== row.preparation || f.document_digest !== row.document_digest || f.actor !== row.actor || f.current_eligibility !== "unchecked" || f.recorded_receipt !== "" && !digest(f.recorded_receipt) || row.recorded_receipt && f.recorded_receipt !== row.recorded_receipt || !["matching_prepared", "missing", "unavailable", "conflict"].includes(f.ledger_observation) || (f.ledger_observation === "matching_prepared" ? !digest(f.observed_digest) || f.recorded_receipt && f.recorded_receipt !== f.observed_digest : f.observed_digest !== "")) return fail();
  const sub = record(json(f.submission), ["Key", "Connection", "Purpose", "Customer", "Period", "Path", "Anchor", "Lines"]); ids(sub, ["Key", "Connection", "Customer"]);
  if (sub.Key !== row.preparation || sub.Purpose !== "customer_invoice" || sub.Path !== "direct" || sub.Anchor !== "" || typeof sub.Period !== "string" || !/^(?!0000)\d{4}-(?:0[1-9]|1[0-2])$/.test(sub.Period) || !Array.isArray(sub.Lines) || !sub.Lines.length || sub.Lines.length > 100) return fail();
  const seen = new Set<string>();
  for (const rawLine of sub.Lines) { const l = record(rawLine, ["ID", "Source", "Product", "TaxCode", "Quantity", "Net", "Tax"]); ids(l, ["ID", "Source", "Product", "TaxCode"]); if (!decimal(l.Quantity, 4, true) || !decimal(l.Net, 4, true) || !decimal(l.Tax) || seen.has(String(l.Source)) || l.ID !== `line-${String(l.Source).replace(/^source-/, "")}` || !/^source-[0-9a-f]{64}$/.test(String(l.Source))) return fail(); seen.add(String(l.Source)); }
  const total = record(json(f.total), ["Net", "Tax", "Quantity"]); if (!Object.values(total).every(amount)) return fail();
  for (const key of ["Net", "Tax", "Quantity"]) if (sum(sub.Lines.map(l => String(l[key]))) !== canonical(String(total[key]))) return fail();
  const mappings = json(f.mappings); if (!Array.isArray(mappings) || mappings.length > 201 || !mappings.length || mappings.some(m => typeof m !== "string" || !m || bytes(m) > 512) || new Set(mappings).size !== mappings.length) return fail();
  return freeze({ row: { ...row, recorded_receipt: f.recorded_receipt }, revision: p.revision, observation: f.ledger_observation as PreparationInspection["observation"], observedDigest: f.observed_digest, submission: sub, total: total as Record<string,string>, mappings });
}
