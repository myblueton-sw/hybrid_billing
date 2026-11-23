import { timestamp } from "../documents/contract";
import { digest, exact, fail, freeze, id, json, pageRows, preparationOf, revision, type Preparation, type ReadCommand } from "./contract";

export type ApplicationState = "not_started" | "unknown" | "recorded";
export type HistoryRow = Readonly<{ mode: "synthetic"; result: string; digest: string; recorded_state: "pending" | "confirmed" | "revoked"; receipt: "historical_decision"; application_state: ApplicationState; application_receipt: string; proposed_by: string; proposed_at: string }>;
export function application(state: unknown, receipt: unknown): state is ApplicationState {
 return (state === "not_started" || state === "unknown") && receipt === "" || state === "recorded" && digest(receipt);
}
function rowOf(raw: string): HistoryRow {
 const v = json(raw);
 exact(v, ["mode", "result", "digest", "recorded_state", "receipt", "application_state", "application_receipt", "proposed_by", "proposed_at"]);
 if (v.mode !== "synthetic" || !id(v.result) || !digest(v.digest) || typeof v.recorded_state !== "string" || !["pending", "confirmed", "revoked"].includes(v.recorded_state) || v.receipt !== "historical_decision" || !application(v.application_state, v.application_receipt) || !id(v.proposed_by) || !timestamp(v.proposed_at) || v.recorded_state === "pending" && v.application_state !== "not_started") return fail();
 return freeze(v) as HistoryRow;
}
export async function loadHistory(read: ReadCommand, selected: Preparation) {
 const scope = preparationOf(JSON.stringify(selected));
 let after = "", snapshot = "";
 const rows: HistoryRow[] = [];
 for (let index = 0; index < 10; index++) {
  const p = await read("result_evidence.history", scope.preparation, { digest: scope.preparation_digest, ...(after ? { after, snapshot } : {}) });
  const f = p.fields;
  if (!revision(p.revision) || f.snapshot !== p.revision || snapshot && snapshot !== p.revision || f.mode !== "synthetic" || f.preparation !== scope.preparation || f.digest !== scope.preparation_digest || f.current_eligibility !== "unchecked" || f.next !== "" && !id(f.next)) return fail();
  const batch = pageRows(f, ["mode", "preparation", "digest", "current_eligibility", "snapshot", "next"]).map(rowOf);
  for (const row of batch) { if (rows.length && row.result <= rows[rows.length - 1].result) return fail(); rows.push(row); }
  if (f.next && (batch.length !== 10 || f.next !== batch[batch.length - 1].result)) return fail();
  snapshot = p.revision; after = f.next;
  if (!after) return freeze({ preparation: scope, revision: snapshot, rows });
 }
 return fail();
}
