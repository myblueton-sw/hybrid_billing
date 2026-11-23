import { bytes, digest, freeze, id, object, revision, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";

export class SetupError extends Error {
  constructor(readonly code: string) { super(code); }
}

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const keys = ["schema", "contract", "contract_name", "on", "lifecycle_sequence", "contract_revision", "billing_profile", "billing_profile_name", "billing_revision", "buyer", "payer", "issuer", "currency", "scope_digest", "profile_basis", "current_eligibility"];

export function setupDate(value: unknown): value is string {
  return typeof value === "string" && /^(?!0000)\d{4}-\d{2}-\d{2}$/.test(value) && Number.isFinite(Date.parse(value + "T00:00:00Z")) && new Date(value + "T00:00:00Z").toISOString().slice(0, 10) === value;
}

function positiveVersion(value: string): boolean {
  return /^[1-9][0-9]{0,19}$/.test(value) && BigInt(value) <= 18446744073709551615n;
}

export type SetupContext = Readonly<{
  revision: string; at: string; contract: string; contractName: string; on: string;
  lifecycleSequence: string; contractRevision: string; billingProfile: string;
  billingProfileName: string; billingRevision: string; buyer: string; payer: string;
  issuer: string; currency: string; scopeDigest: string;
  profileBasis: "current"; currentEligibility: "unchecked";
}>;

export function setupContextOf(envelope: ReadEnvelope, contract: string, on: string): SetupContext {
  const f = envelope.fields;
  if (!id(contract) || !setupDate(on) || !revision(envelope.revision) || !timestamp(envelope.at)
    || !object(f) || Object.keys(f).length !== keys.length || keys.some(key => !Object.hasOwn(f, key))
    || Object.values(f).some(value => typeof value !== "string" || bytes(value) > 4096)
    || bytes(JSON.stringify(f)) > 65536) return invalid();
  if (f.schema !== "hb_issuance_setup_context_v1" || f.contract !== contract || f.on !== on
    || ![f.billing_profile, f.buyer, f.payer, f.issuer].every(id)
    || ![f.lifecycle_sequence, f.contract_revision, f.billing_revision].every(positiveVersion)
    || !/^[A-Z]{3}$/.test(f.currency) || !digest(f.scope_digest)
    || f.profile_basis !== "current" || f.current_eligibility !== "unchecked") return invalid();
  return freeze({ revision: envelope.revision, at: envelope.at, contract, on,
    contractName: f.contract_name, lifecycleSequence: f.lifecycle_sequence,
    contractRevision: f.contract_revision, billingProfile: f.billing_profile,
    billingProfileName: f.billing_profile_name, billingRevision: f.billing_revision,
    buyer: f.buyer, payer: f.payer, issuer: f.issuer, currency: f.currency,
    scopeDigest: f.scope_digest, profileBasis: "current", currentEligibility: "unchecked" });
}
