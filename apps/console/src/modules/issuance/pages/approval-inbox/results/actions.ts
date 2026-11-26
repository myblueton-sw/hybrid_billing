import { sameApplicationContext } from "./relation";
import type { Attempt } from "../../../../../service/admin";
import { freeze, ResultError, type Preparation } from "./contract";
import { freezeIntent } from "./mutations";
import type { Action } from "./advice";
import type { inspectWithAdvice } from "./inspection";
export type ResultDetail = Awaited<ReturnType<typeof inspectWithAdvice>>;
export function reviewAction(scope: Preparation, detail: ResultDetail, actor: string, action: Action, rationale: string, tried: Attempt) {
 const { inspection: v, advice: a } = detail, permission = a.permissions[action === "recover" ? "apply" : action];
 if (!sameApplicationContext(v.applicationContext,a.applicationContext) || !actor || v.packet.Preparation !== scope.preparation || v.packet.PreparationDigest !== scope.preparation_digest || v.packet.Connection !== scope.connection || v.packet.Currency !== scope.currency || v.packet.Customer !== scope.customer || v.packet.Period !== scope.period || v.result !== a.result || v.digest !== a.digest || v.revision !== a.revision || v.applicationState !== a.applicationState || v.applicationReceipt !== a.applicationReceipt) throw new ResultError("evidence_stale");
 if (permission.state !== "allowed" || a.readiness[action].state !== "ready") throw new ResultError("prerequisite_failed");
 if (action === "confirm" && (v.state !== "pending" || v.proposedBy === actor) || action === "revoke" && v.state === "revoked") throw new ResultError("prerequisite_failed");
 if ((action === "apply" || action === "recover") && (v.state !== "confirmed" || v.comparison.length || !v.confirmation || v.applicationReceipt)) throw new ResultError("prerequisite_failed");
 if (action === "apply" && v.applicationState !== "not_started" || action === "recover" && (v.applicationState !== "unknown" || a.applyingActor !== actor)) throw new ResultError("prerequisite_failed");
 return freeze({ scope, action, detail, intent: freezeIntent({ action: action === "recover" ? "apply" : action, target: v.result, fields: { digest: v.digest }, revision: v.revision, rationale, attempt: tried }) });
}
export type ActionReview = ReturnType<typeof reviewAction>;
