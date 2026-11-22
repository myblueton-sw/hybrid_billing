import type { Attempt } from "../../../../../service/admin";
import { strictJSON } from "../documents/json";
import { timestamp } from "../documents/contract";
import { rationaleValid } from "../documents/packet";
import { bytes, digest, exact, freeze, id, revision, ResultError } from "../results/contract";
export const profile = "synthetic_reported_net_plus_tax_v1" as const;
export const statuses = ["evidence_ready", "target_unavailable", "capacity_reached", "dependency_unavailable", "hold_active"] as const;
export type Scope = Readonly<{ result: string; digest: string; author: string; currency: string }>;
export type Preview = Readonly<{ scope: Scope; target: string; requester: string; contributors: readonly string[]; amount: string; net: string; tax: string; currency: string; profile: typeof profile; policy: string; candidate: string; at: string; revision: string; status: typeof statuses[number] }>;
export type Intent = Readonly<{ family: "erp_final"; preview: Preview; rationale: string; attempt: Readonly<Attempt> }>;
export type Receipt = Readonly<{ target: string; digest: string; state: "pending"; revision: string; operation: string; replay: boolean }>;
const amount = (v: unknown): v is string => typeof v === "string" && /^(?:0|[1-9][0-9]{0,13})(?:\.[0-9]{1,4})?$/.test(v);
const gross = (v: unknown): v is string => typeof v === "string" && /^(?:0|[1-9][0-9]{0,13})(?:\.[0-9]{0,3}[1-9])?$/.test(v);
export function validScope(scope: Scope): boolean { return id(scope.result) && digest(scope.digest) && id(scope.author) && /^[A-Z]{3}$/.test(scope.currency); }
export function previewOf(raw: unknown, scope: Scope, target: string, requester: string): Preview {
 try {
  exact(raw,["mode","receipt","request","result","result_digest","candidate_digest","author","requester","contributors","amount","currency","net","tax","amount_basis","policy_revision","checked_at","revision","request_status"]);
  if (!validScope(scope) || !id(target) || !id(requester) || !Object.values(raw).every(v=>typeof v === "string" && bytes(v)<=1024) || raw.mode!=="synthetic" || raw.receipt!=="current_request_preview" || raw.request!==target || raw.result!==scope.result || raw.result_digest!==scope.digest || raw.author!==scope.author || raw.requester!==requester || raw.currency!==scope.currency || raw.amount_basis!==profile || !digest(raw.candidate_digest) || !gross(raw.amount) || !amount(raw.net) || !amount(raw.tax) || !timestamp(raw.checked_at) || !revision(raw.revision) || typeof raw.policy_revision!=="string" || !/^[1-9][0-9]{0,8}$/.test(raw.policy_revision) || !statuses.includes(raw.request_status as typeof statuses[number])) throw new Error();
  const contributors = strictJSON(String(raw.contributors),1024);
  if (!Array.isArray(contributors) || contributors.length<1 || contributors.length>2 || !contributors.every(id) || new Set(contributors).size!==contributors.length || !contributors.includes(scope.author) || contributors.join("\0")!==[...contributors].sort().join("\0")) throw new Error();
  return freeze({scope:structuredClone(scope),target,requester,contributors,amount:raw.amount,net:raw.net,tax:raw.tax,currency:scope.currency,profile,policy:raw.policy_revision,candidate:raw.candidate_digest,at:raw.checked_at,revision:raw.revision,status:raw.request_status}) as Preview;
 } catch { throw new ResultError("invalid_final_preview"); }
}
export function freezeIntent(preview: Preview, rationale: string, tried: Attempt): Intent {
 if (preview.status!=="evidence_ready" || preview.profile!==profile || !validScope(preview.scope) || !id(preview.target) || !digest(preview.candidate) || !revision(preview.revision) || !id(tried.request) || !id(tried.key) || !rationaleValid(rationale)) throw new ResultError("invalid_final_request");
 return freeze({family:"erp_final",preview:structuredClone(preview),rationale,attempt:{...tried}});
}
