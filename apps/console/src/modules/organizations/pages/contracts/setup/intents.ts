import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { rationaleValid } from "../../../../issuance/pages/approval-inbox/documents/packet";
import { digest, freeze, id, revision } from "../../../../issuance/pages/approval-inbox/results/contract";
import { positiveVersion, setupDate, SetupError, type SetupContext } from "./context";
import type { SetupOptions } from "./client";
import type { Configuration, ConfigurationPage } from "./history";
import type { SetupAction, SetupOperation } from "./action";

export type SetupIntent = Readonly<{ operation: SetupOperation; target: string; contract: string; actor: string; workspace: string; sessionCSRF: string; revision: string; args: Readonly<Record<string, string>>; rationale: string; attempt: Readonly<Attempt>; expected: Readonly<Record<string, string>> }>;
const immutable = ["configuration", "revision", "contract_revision", "billing_profile", "billing_revision", "mode", "profile", "region", "tax", "zone", "languages", "proposer", "setup_on", "scope_digest", "proposal_reason"];
const proposalKeys = ["contract", "on", "scope_digest", "mode", "profile", "region", "tax", "zone", "languages"];
const invalid = (): never => { throw new SetupError("invalid_setup_request"); };
export function setupIntent(input: SetupIntent): SetupIntent {
  if (!["propose_configuration", "confirm_configuration"].includes(input.operation) || ![input.target, input.contract, input.actor, input.workspace, input.attempt?.request, input.attempt?.key].every(id)
    || typeof input.sessionCSRF !== "string" || !input.sessionCSRF || input.sessionCSRF.length > 4096 || !revision(input.revision) || !rationaleValid(input.rationale)) return invalid();
  const expected = input.expected, args = input.args, keys = input.operation === "propose_configuration" ? proposalKeys : ["digest"];
  if (Object.keys(args).length !== keys.length || keys.some(k => typeof args[k] !== "string")
    || immutable.some(k => typeof expected[k] !== "string") || Object.keys(expected).some(k => !immutable.includes(k) && k !== "digest")) return invalid();
  if (expected.configuration !== input.target || expected.revision !== "1" || !positiveVersion(expected.contract_revision) || !positiveVersion(expected.billing_revision)
    || !id(expected.billing_profile) || !id(expected.proposer) || !setupDate(expected.setup_on) || !digest(expected.scope_digest) || !rationaleValid(expected.proposal_reason)
    || expected.mode !== "erp_issuance" || expected.profile !== "synthetic_erp_document_v1" || !/^[A-Z]{2}$/.test(expected.region) || !id(expected.tax) || !expected.zone || expected.zone === "Local"
    || expected.digest !== undefined && !digest(expected.digest)) return invalid();
  const languages = expected.languages.split(",");
  if (!languages.length || languages.length > 3 || languages.some((v, i) => !["ar", "en", "ko"].includes(v) || i > 0 && v <= languages[i - 1])) return invalid();
  if (input.operation === "propose_configuration") {
    if (expected.proposer !== input.actor || expected.proposal_reason !== input.rationale || args.contract !== input.contract || args.on !== expected.setup_on || args.scope_digest !== expected.scope_digest
      || ["mode", "profile", "region", "tax", "zone", "languages"].some(k => args[k] !== expected[k])) return invalid();
  } else if (!digest(args.digest) || expected.digest !== args.digest || expected.proposer === input.actor) return invalid();
  return freeze({ ...input, args: { ...args }, expected: { ...expected }, attempt: { ...input.attempt } });
}
function checked(guidance: SetupAction, operation: SetupOperation, contract: string, target: string) {
  if (guidance.permission !== "allowed" || guidance.stepUp !== "satisfied" || guidance.action !== "recheck_required" || guidance.code !== ""
    || guidance.request.operation !== operation || guidance.request.contract !== contract || guidance.request.configuration !== target) return invalid();
}
export function proposalIntent(session: Session, context: SetupContext, options: SetupOptions, languages: readonly string[], rationale: string, guidance: SetupAction, attempt: Attempt): SetupIntent {
  const target = guidance.request.configuration;
  checked(guidance, "propose_configuration", context.contract, target);
  if (options.contract !== context.contract || options.on !== context.on || options.scopeDigest !== context.scopeDigest || guidance.request.fields.on !== context.on || guidance.request.fields.scope_digest !== context.scopeDigest
    || languages.length === 0 || new Set(languages).size !== languages.length || languages.some(v => !options.languages.includes(v as "ar" | "en" | "ko"))) return invalid();
  const args = { contract: context.contract, on: context.on, scope_digest: context.scopeDigest, mode: options.mode, profile: options.profile, region: options.region, tax: options.tax, zone: options.zone, languages: [...languages].sort().join(",") };
  return setupIntent({ operation: "propose_configuration", target, contract: context.contract, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: guidance.revision, args, rationale, attempt,
    expected: { configuration: target, revision: "1", contract_revision: context.contractRevision, billing_profile: context.billingProfile, billing_revision: context.billingRevision, mode: args.mode, profile: args.profile, region: args.region, tax: args.tax, zone: args.zone, languages: args.languages, proposer: session.user, setup_on: context.on, scope_digest: context.scopeDigest, proposal_reason: rationale } });
}
export function retainedIntent(session: Session, page: ConfigurationPage, row: Configuration, guidance: SetupAction, rationale: string, attempt: Attempt): SetupIntent {
  const operation = guidance.request.operation;
  checked(guidance, operation, page.contract, row.configuration);
  if (page.revision !== guidance.revision || !page.rows.some(v => v.configuration === row.configuration && v.digest === row.digest) || !row.setup_on || !row.scope_digest || !row.proposal_reason) return invalid();
  if (operation === "propose_configuration" ? row.state !== "proposal_unknown" || row.proposer !== session.user || guidance.request.fields.on !== row.setup_on || guidance.request.fields.scope_digest !== row.scope_digest
    : !["proposed", "confirmation_unknown"].includes(row.state) || row.proposer === session.user || row.confirmer !== "" && row.confirmer !== session.user || guidance.request.fields.digest !== row.digest) return invalid();
  const expected = Object.fromEntries([...immutable, "digest"].map(k => [k, row[k]]));
  const args: Record<string, string> = operation === "propose_configuration" ? { contract: page.contract, on: row.setup_on, scope_digest: row.scope_digest, ...Object.fromEntries(["mode", "profile", "region", "tax", "zone", "languages"].map(k => [k, row[k]])) } : { digest: row.digest };
  return setupIntent({ operation, target: row.configuration, contract: page.contract, actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: guidance.revision, args, rationale: operation === "propose_configuration" ? row.proposal_reason : rationale, attempt, expected });
}
