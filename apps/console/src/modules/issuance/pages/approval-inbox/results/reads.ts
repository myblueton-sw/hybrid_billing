import { digest, exact, fail, freeze, id, json, pageRows, permission, preparationOf, readiness, revision, type Context, type Preparation, type ReadCommand, type ReadEnvelope, type ReferenceLine } from "./contract";

function page(p: ReadEnvelope, snapshot: string) {
 if (!revision(p.revision) || p.fields.snapshot !== p.revision || snapshot && snapshot !== p.revision || p.fields.mode !== "synthetic" || p.fields.current_eligibility !== "unchecked" || p.fields.next !== "" && !id(p.fields.next)) fail();
}
function continuation<T>(rows: readonly T[], batch: readonly T[], next: string, key: (v: T) => string) {
 if (rows.length > 100 || next && (batch.length !== 10 || next !== key(batch[batch.length - 1]))) fail();
}
export async function loadPreparations(read: ReadCommand, workspace: string): Promise<Readonly<{ revision: string; rows: readonly Preparation[] }>> {
 if (!id(workspace)) return fail();
 let after = "", snapshot = "";
 const rows: Preparation[] = [];
 for (let index = 0; index < 10; index++) {
  const p = await read("result_evidence.preparations", workspace, after ? { after, snapshot } : {});
  page(p, snapshot); if (p.fields.workspace !== workspace) return fail();
  const batch = pageRows(p.fields, ["mode", "workspace", "current_eligibility", "snapshot", "next"]).map(preparationOf);
  for (const row of batch) { if (rows.length && row.preparation <= rows[rows.length - 1].preparation) return fail(); rows.push(row); }
  continuation(rows, batch, p.fields.next, r => r.preparation);
  snapshot = p.revision; after = p.fields.next;
  if (!after) return freeze({ revision: snapshot, rows });
 }
 return fail();
}
export async function loadContext(read: ReadCommand, selected: Preparation): Promise<Context> {
 const scope = preparationOf(JSON.stringify(selected));
 if (!digest(scope.preparation_digest)) return fail();
 let after = "", snapshot = "", fixedText = "";
 let parsedPermission, parsedReadiness;
 const references: ReferenceLine[] = [];
 const fixed = ["mode", "preparation", "preparation_digest", "document", "document_digest", "connection", "customer", "period", "currency", "profile", "schema", "permissions", "proposal_readiness", "current_eligibility", "snapshot", "next"];
 for (let index = 0; index < 10; index++) {
  const p = await read("result_evidence.context", scope.preparation, { digest: scope.preparation_digest, ...(after ? { after, snapshot } : {}) });
  page(p, snapshot); const f = p.fields;
  for (const name of ["preparation", "preparation_digest", "document", "document_digest", "connection", "customer", "period", "currency"] as const) if (f[name] !== scope[name]) return fail();
  if (f.profile !== "synthetic_result_one_to_one_v1" || f.schema !== "hb_erp_result_packet_v1") return fail();
  const header = JSON.stringify(fixed.filter(k => !["snapshot", "next"].includes(k)).map(k => f[k]));
  if (snapshot && header !== fixedText) return fail(); fixedText = header;
  const perms = json(f.permissions); exact(perms, ["propose"]); parsedPermission = permission(perms.propose); parsedReadiness = readiness(json(f.proposal_readiness));
  if (parsedPermission.state !== "allowed" && (parsedReadiness.state !== "unchecked" || parsedReadiness.code !== parsedPermission.code) || parsedPermission.state === "allowed" && ["inapplicable", "unchecked"].includes(parsedReadiness.state)) return fail();
  const batch = pageRows(f, fixed).map(raw => { const row = json(raw); exact(row, ["line", "expected_product", "expected_unit"]); if (![row.line, row.expected_product, row.expected_unit].every(id)) return fail(); return freeze(row) as ReferenceLine; });
  for (const row of batch) { if (references.length && row.line <= references[references.length - 1].line) return fail(); references.push(row); }
  continuation(references, batch, f.next, r => r.line); snapshot = p.revision; after = f.next;
  if (!after) { if (!references.length) return fail(); return freeze({ preparation: scope, revision: snapshot, permission: parsedPermission, readiness: parsedReadiness, references }); }
 }
 return fail();
}
