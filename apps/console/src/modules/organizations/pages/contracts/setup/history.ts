import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { bytes, digest, freeze, id, object, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { positiveVersion, setupDate, SetupError } from "./context";

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const required = ["configuration", "digest", "state", "module_observation", "revision", "contract_revision", "billing_profile", "billing_revision", "mode", "profile", "region", "tax", "zone", "languages", "proposer", "confirmer"];
const optional = ["setup_on", "scope_digest", "proposal_reason", "proposed_at", "approval_observed_at", "activated_at"];
const orderedTimestamp = (value: string) => value.slice(0, 19) + "." + value.slice(19, -1).replace(/^\./, "").padEnd(9, "0");
export type Configuration = Readonly<Record<typeof required[number], string> & Partial<Record<typeof optional[number], string>>>;
export type ConfigurationPage = Readonly<{ revision: string; at: string; contract: string; snapshot: string; next: string; count: number; rows: readonly Configuration[] }>;

export function configurationOf(value: unknown): Configuration {
  if (!object(value) || required.some(k => typeof value[k] !== "string") || Object.keys(value).some(k => !required.includes(k) && !optional.includes(k))
    || Object.values(value).some(v => typeof v !== "string" || bytes(v) > 4096) || bytes(JSON.stringify(value)) > 4096) return invalid();
  const f = value as Record<string, string>;
  if (![f.configuration, f.billing_profile, f.tax, f.proposer].every(id) || !digest(f.digest) || f.revision !== "1"
    || ![f.contract_revision, f.billing_revision].every(positiveVersion) || f.mode !== "erp_issuance" || f.profile !== "synthetic_erp_document_v1"
    || !/^[A-Z]{2}$/.test(f.region) || !f.zone || f.zone === "Local" || bytes(f.zone) > 128) return invalid();
  const languages = f.languages.split(",");
  if (!languages.length || languages.length > 3 || languages.some((v, i) => !["ar", "en", "ko"].includes(v) || i > 0 && v <= languages[i - 1])) return invalid();
  if (["setup_on", "scope_digest", "proposal_reason"].some(k => Object.hasOwn(f, k))
    && (!setupDate(f.setup_on) || !digest(f.scope_digest) || !f.proposal_reason?.trim())) return invalid();
  for (const k of ["proposed_at", "approval_observed_at", "activated_at"]) if (f[k] !== undefined && !timestamp(f[k])) return invalid();
  if (f.approval_observed_at && (!f.proposed_at || orderedTimestamp(f.approval_observed_at) < orderedTimestamp(f.proposed_at))) return invalid();
  const pending = f.state === "proposal_unknown" || f.state === "proposed";
  if (pending ? f.confirmer !== "" || f.approval_observed_at !== undefined || f.activated_at !== undefined : !id(f.confirmer) || f.confirmer === f.proposer) return invalid();
  switch (f.state) {
    case "proposal_unknown": if (f.module_observation !== (f.proposed_at ? "proposed" : "absent")) return invalid(); break;
    case "proposed": if (!f.proposed_at || f.module_observation !== "proposed") return invalid(); break;
    case "confirmation_unknown": if (!f.proposed_at || f.activated_at !== undefined || f.module_observation !== (f.approval_observed_at ? "active" : "proposed")) return invalid(); break;
    case "confirmed": if (!f.proposed_at || !f.approval_observed_at || f.activated_at !== f.approval_observed_at || f.module_observation !== "active") return invalid(); break;
    default: return invalid();
  }
  return freeze({ ...f }) as Configuration;
}

export function configurationsOf(e: ReadEnvelope, contract: string, after = "", snapshot = ""): ConfigurationPage {
  const f = e.fields, fixed = ["schema", "contract", "current_eligibility", "snapshot", "count", "next"];
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (fixed.some(k => !Object.hasOwn(f, k)) || keys.length > 10 || keys.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`)
    || f.schema !== "hb_issuance_configuration_page_v1" || f.contract !== contract || f.current_eligibility !== "unchecked"
    || !digest(f.snapshot) || snapshot && snapshot !== f.snapshot || !/^(?:0|[1-9][0-9]?)$|^100$/.test(f.count)) return invalid();
  const rows = keys.map(k => { try { return configurationOf(strictJSON(f[k], 4096)); } catch { return invalid(); } });
  const count = Number(f.count);
  if (count < rows.length || !after && rows.length !== Math.min(count, 10) || rows.some((r, i) => r.configuration <= (i ? rows[i - 1].configuration : after))
    || f.next !== "" && (rows.length !== 10 || f.next !== rows.at(-1)?.configuration)
    || !after && f.next !== (count > 10 ? rows.at(-1)?.configuration : "") || count === 0 && f.next !== "") return invalid();
  return freeze({ revision: e.revision, at: e.at, contract, snapshot: f.snapshot, next: f.next, count, rows });
}
