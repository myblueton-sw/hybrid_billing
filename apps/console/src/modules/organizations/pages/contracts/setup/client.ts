import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { bytes, digest, freeze, id, object, revision, ResultError, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { bounded, checkSession, responseText } from "../../../../issuance/pages/approval-inbox/results/transport";
import { SetupError, setupContextOf, setupDate, type SetupContext } from "./context";
import { configurationsOf } from "./history";
import { actionRequest, setupActionOf, type ActionRequest } from "./action";
import { protectedChangesOf, protectedChannelsOf, protectedFields, protectedScope, protectedSummaryOf, type ProtectedField, type ProtectedScope } from "./protected";

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const envelopeKeys = ["request_id", "status", "target_id", "result_revision", "replay", "as_of", "fields"];
const optionKeys = ["schema", "contract", "on", "scope_digest", "current_eligibility", "option", "applicability", "mode", "profile", "region", "tax", "zone", "languages"];

export function setupFields(raw: unknown): Readonly<Record<string, string>> {
  if (!object(raw) || Object.keys(raw).length > 64
    || Object.entries(raw).some(([key, value]) => !/^[a-z][a-z0-9_]*$/.test(key) || typeof value !== "string" || bytes(value) > 4096)
    || bytes(JSON.stringify(raw)) > 65536) return invalid();
  return freeze({ ...raw }) as Readonly<Record<string, string>>;
}

export type SetupOptions = Readonly<{
  revision: string; at: string; contract: string; on: string; scopeDigest: string;
  option: "synthetic_erp_sample_v1"; applicability: "synthetic_only";
  mode: "erp_issuance"; profile: "synthetic_erp_document_v1";
  region: "US"; tax: "synthetic-tax"; zone: "UTC";
  languages: readonly ["ar", "en", "ko"]; currentEligibility: "unchecked";
}>;

function optionsOf(e: ReadEnvelope, scope: SetupContext): SetupOptions {
  const f = e.fields;
  if (Object.keys(f).length !== optionKeys.length || optionKeys.some(key => !Object.hasOwn(f, key))
    || f.schema !== "hb_issuance_setup_options_v1" || f.contract !== scope.contract || f.on !== scope.on || f.scope_digest !== scope.scopeDigest
    || f.current_eligibility !== "unchecked" || f.option !== "synthetic_erp_sample_v1" || f.applicability !== "synthetic_only"
    || f.mode !== "erp_issuance" || f.profile !== "synthetic_erp_document_v1" || f.region !== "US" || f.tax !== "synthetic-tax"
    || f.zone !== "UTC" || f.languages !== "ar,en,ko") return invalid();
  return freeze({ revision: e.revision, at: e.at, contract: f.contract, on: f.on, scopeDigest: f.scope_digest,
    option: f.option, applicability: f.applicability, mode: f.mode, profile: f.profile, region: f.region,
    tax: f.tax, zone: f.zone, languages: ["ar", "en", "ko"] as const, currentEligibility: f.current_eligibility });
}

export function setupReadClient(session: Session, fetcher: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  async function read(command: "context" | "options" | "configurations" | "action" | "protected" | "protected_channels" | "protected_changes", target: string, fields: Record<string, string>): Promise<ReadEnvelope> {
    const values = freeze({ ...fields });
    try { await checkSession(owner, fetcher); }
    catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "session_unavailable"); }
    const tried = attempt();
    let response: Response, raw: unknown;
    try {
      const result = await bounded(async signal => {
        const response = await fetcher(`/api/v1/commands/issuance_setup.${command}`, {
          signal, method: "POST", credentials: "same-origin", cache: "no-store",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": tried.request },
          body: JSON.stringify({ request_id: tried.request, target_id: target, body: values }),
        });
        return { response, raw: strictJSON(await responseText(response), 125000) };
      }, "none");
      response = result.response; raw = result.raw;
    } catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "service_unavailable"); }
    if (!object(raw)) return invalid();
    if (!response.ok) throw new SetupError(raw.request_id === tried.request && typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable");
    if (response.status !== 200 || Object.keys(raw).length !== envelopeKeys.length || envelopeKeys.some(key => !Object.hasOwn(raw, key))
      || raw.request_id !== tried.request || raw.target_id !== target || raw.status !== "completed" || raw.replay !== false
      || !revision(raw.result_revision) || !timestamp(raw.as_of)) return invalid();
    return freeze({ revision: raw.result_revision, at: raw.as_of, fields: setupFields(raw.fields) });
  }
  return {
    async action(input: ActionRequest) {
      const request = actionRequest(input);
      return setupActionOf(await read("action", request.contract, { ...request.fields, configuration: request.configuration, operation: request.operation }), request);
    },
    async configurations(contract: string, after = "", snapshot = "") {
      if (!id(contract) || (!!after !== !!snapshot) || after && (!id(after) || !digest(snapshot))) return invalid();
      return configurationsOf(await read("configurations", contract, after ? { after, snapshot } : {}), contract, after, snapshot);
    },
    async context(contract: string, on: string) {
      if (!id(contract) || !setupDate(on)) return invalid();
      return setupContextOf(await read("context", contract, { on }), contract, on);
    },
    async protectedSummary(input: ProtectedScope) {
      const scope = protectedScope(input);
      return protectedSummaryOf(await read("protected", scope.contract, { configuration: scope.configuration, digest: scope.digest }), scope);
    },
    async protectedChannels(input: ProtectedScope, after = "", snapshot = "") {
      const scope = protectedScope(input);
      if ((!!after !== !!snapshot) || after && (!id(after) || !digest(snapshot))) return invalid();
      return protectedChannelsOf(await read("protected_channels", scope.contract, { configuration: scope.configuration, digest: scope.digest, ...(after ? { after, snapshot } : {}) }), scope, after, snapshot);
    },
    async protectedChanges(input: ProtectedScope, field: ProtectedField, after = "", snapshot = "") {
      const scope = protectedScope(input);
      if (!protectedFields.includes(field) || (!!after !== !!snapshot) || after && (!id(after) || !digest(snapshot))) return invalid();
      return protectedChangesOf(await read("protected_changes", scope.contract, { configuration: scope.configuration, digest: scope.digest, field, ...(after ? { after, snapshot } : {}) }), scope, field, after, snapshot);
    },
    async options(context: SetupContext) {
      const scope = freeze({ ...context });
      if (!id(scope.contract) || !setupDate(scope.on) || !digest(scope.scopeDigest)) return invalid();
      return optionsOf(await read("options", scope.contract, { on: scope.on, scope_digest: scope.scopeDigest }), scope);
    },
  };
}
