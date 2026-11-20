import { AdminError, type AdminClient } from "./admin";
import { array, boolean, choice, integer, ReadParseError, record, shape, text } from "./strict-read";

export const licenseStates = ["unselected", "noncommercial", "evaluation_pending", "evaluation_active", "evaluation_expired", "commercial_pending", "commercial_active", "commercial_expired", "unverified"] as const;
export const licenseOutcomes = ["initialized", "already_initialized", "mode_selected", "activation_recorded", "invalid", "unverifiable", "wrong_kind", "scope_mismatch", "time_unavailable", "clock_anomaly", "not_yet_valid", "expired", "superseded", "duplicate_right", "bad_supersedes", "accepted", "repeat", "key_compromised", "local_clock_behind", "terms_mismatch", "mode_mismatch", "stale_grant", "already_evaluated", "ai_not_allowed", "not_human"] as const;
export type LicenseMode = "noncommercial" | "evaluation" | "commercial";
const bounded = (value: unknown) => { const s = text(value); if (s.length > 4096) throw new ReadParseError(); return s; };
const instant = (value: unknown) => { const s = text(value); if (!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(s) || !Number.isFinite(Date.parse(s)) || new Date(s).toISOString() !== s.replace("Z", ".000Z")) throw new ReadParseError(); return s; };
const maybeInstant = (value: unknown) => value === "" ? "" : instant(value);
const id = (v: unknown) => { const s = text(v); if (!/^[A-Za-z0-9._:-]{1,128}$/.test(s)) throw new ReadParseError(); return s; };
const domainText = (v: unknown) => { const s = text(v); if (!s || new TextEncoder().encode(s).length > 200 || /[\u0000-\u001f\u007f-\u009f\ud800-\udfff]/u.test(s)) throw new ReadParseError(); return s; };
const scope = (value: unknown) => { const result = array(id)(value); if (!result.length || result.length > 32 || result.some((s,i) => i > 0 && result[i-1] >= s)) throw new ReadParseError(); return result; };
const unix = (v: unknown) => { const n = integer(v); if (n > 253402300799) throw new ReadParseError(); return n; };
const entitlementOf = shape({ id, subject_id: id, subject: domainText, agreement: id, scope, channel: choice("direct", "marketplace"), metric: choice("installations", "seats", "tenants", "documents"), start: unix, end: unix, perpetual: boolean });
const evaluationOf = shape({ grant: id, subject: id, terms: domainText, scope, start: instant, end: instant });
const statusOf = shape({
  mode: choice("", "noncommercial", "evaluation", "commercial"), state: choice(...licenseStates),
  reason: choice("", "time_unavailable", "clock_anomaly", "key_compromised", "local_clock_behind"), warning: boolean,
  actions: array(choice("register", "contact")), remaining_seconds: integer, last_outcome: choice("", ...licenseOutcomes),
  last_verified: choice("", ...licenseStates), last_verified_at: maybeInstant,
  entitlement: (v: unknown) => v === null ? null : entitlementOf(v), evaluation: (v: unknown) => v === null ? null : evaluationOf(v),
});
export type LicenseStatus = ReturnType<typeof statusOf>;
export type LicensePage = { revision: string; observed_at: string; state: "not_configured" } | {
  revision: string; observed_at: string; state: "uninitialized" | "configured"; installation: string; product: "hybrid-billing";
  time_authority: "not_connected" | "configured"; status: LicenseStatus | null;
};
export function licensePageOf(value: unknown): LicensePage {
  const r = record(value);
  const revision = text(r.revision);
  if (!/^r[0-9]+$/.test(revision)) throw new ReadParseError();
  const observed_at = instant(r.observed_at);
  const state = choice("not_configured", "uninitialized", "configured")(r.state);
  if (state === "not_configured") return { revision, observed_at, state };
  const installation = bounded(r.installation);
  if (!/^[A-Za-z0-9._:-]{1,128}$/.test(installation)) throw new ReadParseError();
  const product = choice("hybrid-billing")(r.product);
  const time_authority = choice("not_connected", "configured")(r.time_authority);
  const status = state === "configured" ? statusOf(r.status) : null;
  if (status) {
    const s = status;
    if (s.entitlement && (s.entitlement.start < 1 || (s.entitlement.perpetual ? s.entitlement.end !== 0 : s.entitlement.end <= s.entitlement.start))) throw new ReadParseError();
    if (s.evaluation && Date.parse(s.evaluation.end)-Date.parse(s.evaluation.start) !== 336*3600000) throw new ReadParseError();
    if (s.state === "unselected" && s.mode !== "" || s.state === "noncommercial" && s.mode !== "noncommercial" || s.state.startsWith("evaluation_") && s.mode !== "evaluation" || s.state.startsWith("commercial_") && s.mode !== "commercial") throw new ReadParseError();
    if ((s.state === "unverified") !== (s.reason !== "")) throw new ReadParseError();
    if (s.state === "evaluation_active" && (!s.evaluation) || s.state === "evaluation_expired" && !s.evaluation || s.state.startsWith("commercial_") && s.state !== "commercial_pending" && !s.entitlement) throw new ReadParseError();
    if (s.state !== "evaluation_active" && s.remaining_seconds !== 0 || !["evaluation_active", "evaluation_expired"].includes(s.state) && s.evaluation !== null || !["commercial_active", "commercial_expired"].includes(s.state) && s.entitlement !== null) throw new ReadParseError();
  }
  return { revision, observed_at, state, installation, product, time_authority, status };
}
export async function readLicense(api: Pick<AdminClient, "read">): Promise<LicensePage> {
  const value = await api.read("/licenses");
  try { return licensePageOf(value); } catch { throw new AdminError("invalid"); }
}
export type SignedEnvelope = { body: string; signature: string; terms?: string };
function canonicalBase64(value: unknown, max: number, exact?: number): string {
  const encoded = text(value);
  if (encoded.length > Math.ceil(max / 3) * 4) throw new ReadParseError();
  let bytes: string;
  try { bytes = atob(encoded); } catch { throw new ReadParseError(); }
  if (!bytes.length || bytes.length > max || exact !== undefined && bytes.length !== exact || btoa(bytes) !== encoded) throw new ReadParseError();
  return encoded;
}
export function signedEnvelopeOf(raw: string): SignedEnvelope {
  if (new TextEncoder().encode(raw).length > 12000) throw new ReadParseError();
  let r: Record<string, unknown>;
  try { r = record(JSON.parse(raw)); } catch { throw new ReadParseError(); }
  if (Object.keys(r).some((k) => !["body", "signature", "terms"].includes(k))) throw new ReadParseError();
  const terms = r.terms === undefined ? undefined : domainText(r.terms);
  return { body: canonicalBase64(r.body, 4096), signature: canonicalBase64(r.signature, 64, 64), ...(terms === undefined ? {} : { terms }) };
}
export function licenseSignedFields(envelope: SignedEnvelope, activation: boolean): Record<string, string> {
  if (activation && !envelope.terms) throw new ReadParseError();
  return { body: envelope.body.slice(0, 4096), ...(envelope.body.length > 4096 ? { body_tail: envelope.body.slice(4096) } : {}), signature: envelope.signature, ...(activation ? { terms: envelope.terms! } : {}) };
}
