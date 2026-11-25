import { digest, freeze, id, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { SetupError, setupDate } from "./context";

export type SetupOperation = "propose_configuration" | "confirm_configuration";
export type ActionRequest = Readonly<{ contract: string; configuration: string; operation: SetupOperation; fields: Readonly<Record<string, string>> }>;
export type SetupAction = Readonly<{ request: ActionRequest; revision: string; at: string; permission: "allowed" | "denied" | "unavailable"; stepUp: "satisfied" | "unchecked" | "reauthentication_required" | "authentication_policy_unsatisfied"; action: "recheck_required" | "blocked" | "recorded"; code: string }>;
const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const keys = ["schema", "contract", "configuration", "operation", "permission", "step_up", "action", "code", "current_eligibility"];
const blocked = ["permission_denied", "self_approval_denied", "evidence_stale", "unsupported_capability", "configuration_exists", "configuration_pending", "dependency_unavailable"];
export function actionRequest(request: ActionRequest): ActionRequest {
  const { contract, configuration, operation, fields } = request;
  if (!id(contract) || !id(configuration) || !["propose_configuration", "confirm_configuration"].includes(operation)) return invalid();
  const expected = operation === "propose_configuration" ? ["on", "scope_digest"] : ["digest"];
  if (Object.keys(fields).length !== expected.length || expected.some(k => !Object.hasOwn(fields, k))) return invalid();
  if (operation === "propose_configuration" ? !setupDate(fields.on) || !digest(fields.scope_digest) : !digest(fields.digest)) return invalid();
  return freeze({ ...request, fields: { ...fields } });
}
export function setupActionOf(envelope: ReadEnvelope, input: ActionRequest): SetupAction {
  const request = actionRequest(input), f = envelope.fields;
  if (Object.keys(f).length !== keys.length || keys.some(k => !Object.hasOwn(f, k)) || f.schema !== "hb_issuance_setup_action_v1"
    || f.contract !== request.contract || f.configuration !== request.configuration || f.operation !== request.operation || f.current_eligibility !== "unchecked") return invalid();
  let valid = false;
  if (f.permission === "denied" || f.permission === "unavailable") valid = f.step_up === "unchecked" && f.action === "blocked" && f.code === (f.permission === "denied" ? "permission_denied" : "authority_unavailable");
  else if (f.permission === "allowed") {
    if (f.step_up === "unchecked") valid = f.action === "blocked" && ["installation_held", "authority_unavailable"].includes(f.code);
    else if (["reauthentication_required", "authentication_policy_unsatisfied"].includes(f.step_up)) valid = f.action === "blocked" && f.code === f.step_up;
    else if (f.step_up === "satisfied") valid = f.action === "blocked" ? blocked.includes(f.code) : ["recheck_required", "recorded"].includes(f.action) && f.code === "";
  }
  if (!valid) return invalid();
  return freeze({ request, revision: envelope.revision, at: envelope.at, permission: f.permission, stepUp: f.step_up, action: f.action, code: f.code }) as SetupAction;
}
