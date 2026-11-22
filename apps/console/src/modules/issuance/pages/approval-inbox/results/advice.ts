import { digest, exact, fail, freeze, id, json, permission, readiness, revision, type Permission, type ReadCommand, type Readiness } from "./contract";
import { application, type ApplicationState } from "./history";
export type Action = "confirm" | "revoke" | "apply" | "recover";
export type Advice = Readonly<{ result: string; digest: string; revision: string; permissions: Readonly<Record<Exclude<Action, "recover">, Permission>>; readiness: Readonly<Record<Action, Readiness>>; applicationState: ApplicationState; applyingActor: string; applicationReceipt: string }>;
export async function loadAdvice(read: ReadCommand, result: string, hash: string): Promise<Advice> {
 if (!id(result) || !digest(hash)) return fail();
 const p = await read("result_evidence.advice", result, { digest: hash }), f = p.fields;
 exact(f, ["mode", "result", "digest", "revision", "permissions", "readiness", "application_state", "applying_actor", "application_receipt", "current_eligibility"]);
 if (!revision(p.revision) || f.revision !== p.revision || f.mode !== "synthetic" || f.result !== result || f.digest !== hash || f.current_eligibility !== "advisory_only" || !application(f.application_state, f.application_receipt) || (f.application_state === "not_started" ? f.applying_actor !== "" : !id(f.applying_actor))) return fail();
 const permissions = json(f.permissions), states = json(f.readiness);
 exact(permissions, ["confirm", "revoke", "apply"]); exact(states, ["confirm", "revoke", "apply", "recover"]);
 const parsedPermissions = { confirm: permission(permissions.confirm), revoke: permission(permissions.revoke), apply: permission(permissions.apply) };
 const parsedStates = { confirm: readiness(states.confirm), revoke: readiness(states.revoke), apply: readiness(states.apply), recover: readiness(states.recover) };
 for (const action of ["confirm", "revoke", "apply", "recover"] as const) {
  const access = parsedPermissions[action === "recover" ? "apply" : action], state = parsedStates[action];
  if (access.state !== "allowed" ? state.state !== "unchecked" || state.code !== access.code : state.state === "unchecked") return fail();
 }
 if (parsedStates.apply.state === "ready" && f.application_state !== "not_started" || parsedStates.recover.state === "ready" && f.application_state !== "unknown") return fail();
 return freeze({ result, digest: hash, revision: p.revision, permissions: parsedPermissions, readiness: parsedStates, applicationState: f.application_state, applyingActor: f.applying_actor, applicationReceipt: f.application_receipt });
}
