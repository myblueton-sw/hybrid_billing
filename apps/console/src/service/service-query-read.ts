import { array, boolean, choice, integer, optional, positive, record, ReadParseError, shape, text, type Parser } from "./strict-read";
import type { AccountDetail, AccountPage, AudienceDetail, CredentialDetail, CredentialSummary, DeclarationPage, OrganizationDetail, ScopeDetail, ServiceAccount, ServiceAdmission, ServiceAudience, ServiceCredential, ServiceDeclaration, ServiceDetail, ServiceEligibility, ServiceLimits, ServiceOrganization, ServiceOwner, ServicePage, ServicePeriod, ServiceRestriction, ServiceScope, ServiceSelector, QueryEcho } from "./service-query-types";

export function requireRead(condition: unknown): asserts condition {
  if (!condition) throw new ReadParseError();
}
export const decimal: Parser<string> = (v) => {
  const s = text(v);
  requireRead(/^(0|[1-9][0-9]{0,19})$/.test(s) && BigInt(s) <= 18446744073709551615n);
  return s;
};
export const identifier: Parser<string> = (v) => {
  const s = text(v);
  requireRead(/^[A-Za-z0-9._:-]{1,128}$/.test(s));
  return s;
};
export const timestamp: Parser<string> = (v) => {
  const s = text(v);
  requireRead(/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{1,9})?Z$/.test(s) && Number.isFinite(Date.parse(s)));
  requireRead(new Date(Date.parse(s)).toISOString().slice(0, 19) === s.slice(0, 19));
  return s;
};
function nanos(s: string): bigint {
  return BigInt(Date.parse(s.slice(0, 19) + "Z")) * 1000000n + BigInt((s.includes(".") ? s.slice(20, -1) : "").padEnd(9, "0"));
}
const emptyOr = <T>(parse: Parser<T>): Parser<T | ""> => (v) => v === "" ? "" : parse(v);
const nonempty: Parser<string> = (v) => { const s = text(v); requireRead(s.length > 0); return s; };
const digest: Parser<string> = (v) => { const s = text(v); requireRead(/^[a-f0-9]{64}$/.test(s)); return s; };
const bounded = <T>(parse: Parser<T>, max = 100): Parser<T[]> => (v) => { requireRead(Array.isArray(v) && v.length <= max); return array(parse)(v); };
export function closed<T>(fields: { [K in keyof T]-?: Parser<T[K]> }): Parser<T> {
  const parse = shape<T>(fields);
  return (v) => {
    const r = record(v);
    requireRead(Object.keys(r).every((k) => Object.hasOwn(fields, k)));
    return parse(r);
  };
}
const state = choice("active", "suspended", "retired");
const credentialState = choice("active", "expired", "revoked");
const nameState = choice("available", "unavailable");
const stepUp = choice("ready", "reauthentication_required", "prerequisite_failed", "unavailable");
const execution = choice("ready", "trust_hold", "dependency_unavailable");
const eligible = choice("eligible", "ineligible", "unavailable");
export const admissionOf = closed<ServiceAdmission>({ recorded: decimal, limit: decimal, certainty: choice("exact") });
export const limitsOf = closed<ServiceLimits>({ command_value_bytes: (v) => { const n = decimal(v); requireRead(BigInt(n) > 0n); return n; }, rationale_bytes: (v) => { const n = positive(v); requireRead(n <= 256); return n; }, body_bytes: (v) => { const n = positive(v); requireRead(n === 1048576); return n; }, credential_days: integer, credentials_per_service: decimal, services: positive, replay_seconds: positive, credential_overlap_minutes: positive, credential_overlap_max_minutes: positive, secret_pickup_minutes: positive });
const scopeRef = closed({ audience: identifier, scope: identifier });
const declaration = closed<ServiceDeclaration>({ state: choice("recorded", "missing", "unavailable"), recorded_at: emptyOr(timestamp), version: emptyOr(decimal), purpose: text, organization: emptyOr(identifier), organization_name: text, organization_name_state: nameState, catalog_version: emptyOr(digest), scopes: bounded(scopeRef, 64) });
export const declarationOf: Parser<ServiceDeclaration> = (v) => {
  const r = declaration(v);
  if (r.state === "recorded") requireRead(r.recorded_at && r.version && r.version !== "0" && r.purpose.trim() && r.organization && r.catalog_version && r.scopes.length);
  else requireRead(!r.recorded_at && !r.version && !r.purpose && !r.organization && !r.organization_name && r.organization_name_state === "unavailable" && !r.catalog_version && !r.scopes.length);
  requireRead(r.organization_name_state === "available" ? !!r.organization_name : !r.organization_name);
  const keys = r.scopes.map((x) => JSON.stringify([x.audience, x.scope]));
  requireRead(new Set(keys).size === keys.length);
  return r;
};
const summary = closed<CredentialSummary>({ state: choice("exact_at_reference", "not_evaluated"), recorded: text, active: text, expired: text, revoked: text, earliest_unexpired_expiry: emptyOr(timestamp) });
const summaryOf: Parser<CredentialSummary> = (v) => {
  const r = summary(v);
  const counts = [r.recorded, r.active, r.expired, r.revoked];
  if (r.state === "not_evaluated") requireRead(counts.every((x) => x === "") && !r.earliest_unexpired_expiry);
  else {
    counts.forEach(decimal);
    requireRead(BigInt(r.recorded) === BigInt(r.active) + BigInt(r.expired) + BigInt(r.revoked));
    requireRead(r.active === "0" ? !r.earliest_unexpired_expiry : !!r.earliest_unexpired_expiry);
  }
  return r;
};
export const accountOf = closed<ServiceAccount>({ id: identifier, state, version: decimal, owner: identifier, owner_login: text, creator: identifier, creator_login: text, declaration: declarationOf, credential_summary: summaryOf, recent_use_state: choice("unavailable") });
const credential = closed<ServiceCredential>({ id: identifier, service: identifier, service_state: state, not_before: timestamp, not_after: timestamp, state: credentialState, current_state: credentialState, usable: boolean, observed_at: timestamp, secret_pickup: optional(choice("open", "collected", "never_collected")), secret_pickup_until: optional(timestamp), replaces: optional(identifier), replaced_by: optional(identifier), rotation: optional(choice("replacing", "ending")) });
export const credentialOf: Parser<ServiceCredential> = (v) => {
  const r = credential(v);
  requireRead(nanos(r.not_before) < nanos(r.not_after));
  requireRead(!r.usable || r.current_state === "active" && r.service_state === "active");
  requireRead(r.secret_pickup === undefined || r.current_state === "active");
  requireRead(r.secret_pickup === "open" ? r.secret_pickup_until !== undefined : r.secret_pickup_until === undefined);
  if (r.secret_pickup_until !== undefined) requireRead(nanos(r.secret_pickup_until) > nanos(r.observed_at));
  requireRead(r.rotation === undefined || r.current_state === "active");
  requireRead(r.rotation !== "ending" || r.replaced_by !== undefined);
  requireRead(r.rotation !== "replacing" || r.replaces !== undefined);
  const { secret_pickup, secret_pickup_until, replaces, replaced_by, rotation, ...base } = r;
  return { ...base, ...(secret_pickup === undefined ? {} : { secret_pickup }), ...(secret_pickup_until === undefined ? {} : { secret_pickup_until }),
    ...(replaces === undefined ? {} : { replaces }), ...(replaced_by === undefined ? {} : { replaced_by }), ...(rotation === undefined ? {} : { rotation }) };
};
export const ownerOf = closed<ServiceOwner>({ id: identifier, login: text, role: choice("administrator", "member"), from: emptyOr(timestamp), until: emptyOr(timestamp), eligible_at_reference: boolean });
const organization = closed<ServiceOrganization>({ id: identifier, name: text, name_state: nameState, kind: choice("legal_entity", "internal_unit"), parent: emptyOr(identifier) });
export const organizationOf: Parser<ServiceOrganization> = (v) => {
  const r = organization(v);
  requireRead(r.name_state === "available" ? !!r.name : !r.name);
  return r;
};
export const audienceOf: Parser<ServiceAudience> = (v) => {
  const r = closed<ServiceAudience>({ id: identifier, resource: nonempty, configured_scope_count: integer })(v);
  requireRead(r.configured_scope_count <= 100000 && new TextEncoder().encode(r.resource).length <= 2048);
  return r;
};
const selectorOf: Parser<ServiceSelector> = (v) => {
  const r = record(v);
  return Object.hasOwn(r, "any") ? closed({ any: choiceTrue })(r) : closed({ id: identifier })(r);
};
const choiceTrue: Parser<true> = (v) => { requireRead(v === true); return true; };
const month: Parser<string> = (v) => { const s = text(v); requireRead(/^(19[0-9]{2}|[2-9][0-9]{3})-(0[1-9]|1[0-2])$/.test(s)); return s; };
const periodOf: Parser<ServicePeriod> = (v) => {
  const r = record(v);
  if (Object.hasOwn(r, "any")) return closed({ any: choiceTrue })(r);
  const p = closed({ from: month, until: month })(r);
  requireRead(p.from < p.until);
  return p;
};
const restriction = closed<ServiceRestriction>({ id: identifier, action: choice("view", "view_cost", "change_customer_policy", "change_selling_policy", "request_approval", "issue", "download", "deliver"), kind: identifier, fields: bounded(identifier, 1000), customer: selectorOf, legal_entity: selectorOf, contract: selectorOf, period: periodOf });
const restrictionOf: Parser<ServiceRestriction> = (v) => {
  const r = restriction(v);
  requireRead(new Set(r.fields).size === r.fields.length);
  requireRead(["view", "view_cost", "download"].includes(r.action) ? r.fields.length > 0 : r.fields.length === 0);
  return r;
};
export const scopeOf: Parser<ServiceScope> = (v) => {
  const r = closed<ServiceScope>({ workspace: identifier, id: identifier, audience: identifier, resource: nonempty, restrictions: restrictionOf })(v);
  requireRead(r.id === r.restrictions.id && new TextEncoder().encode(r.resource).length <= 2048);
  return r;
};
const setting = closed({ name: nonempty, value: integer, unit: text, default: integer, min: integer, max: integer, source: nonempty });
const settings = bounded(setting, 32);
const echo = closed<QueryEcho>({ workspace: identifier, q: text, organization: emptyOr(identifier), owner: emptyOr(identifier), audience: emptyOr(identifier), scope: emptyOr(identifier), expiry: text, state: text, declaration: text, service: emptyOr(identifier), collection: text, version: emptyOr(decimal), limit: positive });
export function pageOf<T>(row: Parser<T>, order: string): Parser<ServicePage<T>> {
  const parse = shape<ServicePage<T>>({ rows: bounded(row), search_input: text, query: echo, next: text, reference_at: timestamp, observed_at: timestamp, revision: nonempty, order: choice(order), count: closed({ value: decimal, certainty: choice("exact", "lower_bound") }), work_units: integer, scan_stopped: boolean, settings, limits: limitsOf, catalog_version: digest });
  return (v) => {
    const r = parse(v);
    requireRead(r.query.limit <= 100 && r.rows.length <= r.query.limit && r.work_units <= 10000 && r.next.length <= 4096);
    requireRead(new TextEncoder().encode(r.search_input).length <= 8192 && new TextEncoder().encode(r.query.q).length <= 256 && !/\p{Cc}/u.test(r.query.q));
    requireRead(!!r.next === (r.count.certainty === "lower_bound") && (!r.scan_stopped || !!r.next));
    requireRead(BigInt(r.count.value) >= BigInt(r.rows.length) && r.limits.credentials_per_service !== "0");
    const allowed = [...Object.keys(r), "admission", "availability"];
    requireRead(Object.keys(record(v)).every((k) => allowed.includes(k)));
    return r;
  };
}
export const accountPageOf: Parser<AccountPage> = (v) => {
  const page = pageOf(accountOf, "service_id_asc")(v);
  requireRead(page.rows.every((x) => x.credential_summary.state === "exact_at_reference"));
  return { ...page, admission: admissionOf(record(v).admission) };
};
export const declarationPageOf: Parser<DeclarationPage> = (v) => {
  const p = pageOf(declarationOf, "declaration_version_desc")(v);
  const availability = choice("recorded", "not_recorded", "version_unavailable")(record(v).availability);
  requireRead(p.rows.every((x) => x.state === "recorded"));
  if (availability !== "recorded") requireRead(p.rows.length === 0 && p.count.value === "0" && p.count.certainty === "exact");
  return { ...p, availability };
};
const eligibility = closed<ServiceEligibility>({ state: choice("eligible", "ineligible", "unavailable"), can_issue: boolean, can_reinstate: boolean, can_declare: boolean, can_suspend: boolean, can_retire: boolean, step_up_state: stepUp, execution_state: execution, live_credentials: emptyOr(decimal), credential_limit: emptyOr(decimal), operator_blocked: boolean });
export function detailOf<T>(row: Parser<T>): Parser<ServiceDetail<T>> {
  const parse = shape({ workspace: identifier, row, revision: nonempty, observed_at: timestamp });
  return (v) => {
    const allowed = ["workspace", "row", "revision", "observed_at", "eligibility", "admission", "settings", "limits", "catalog_version"];
    requireRead(Object.keys(record(v)).every((k) => allowed.includes(k)));
    return parse(v);
  };
}
export const accountDetailOf: Parser<AccountDetail> = (v) => {
  const r = record(v);
  const base = detailOf(accountOf)(r);
  requireRead(base.row.credential_summary.state === "not_evaluated");
  const controls = eligibility(r.eligibility);
  requireRead(controls.state === "unavailable" || controls.live_credentials !== "" && controls.credential_limit !== "");
  const limits = limitsOf(r.limits);
  const flags = [controls.can_issue, controls.can_reinstate, controls.can_declare, controls.can_suspend, controls.can_retire];
  if (flags.some(Boolean)) requireRead(controls.step_up_state === "ready" && controls.execution_state !== "dependency_unavailable");
  if (controls.can_issue || controls.can_reinstate || controls.can_declare) requireRead(controls.execution_state === "ready" && base.row.state !== "retired");
  if (controls.can_issue) requireRead(controls.state === "eligible" && base.row.state === "active" && !!controls.live_credentials && !!controls.credential_limit && BigInt(controls.live_credentials) < BigInt(controls.credential_limit) && limits.credential_days >= 1);
  if (controls.can_reinstate) requireRead(controls.state === "eligible" && base.row.state === "suspended" && !controls.operator_blocked && BigInt(base.row.version) < 18446744073709551615n);
  if (controls.can_suspend) requireRead(base.row.state === "active" && BigInt(base.row.version) < 18446744073709551615n);
  if (controls.can_retire) requireRead(base.row.state !== "retired" && BigInt(base.row.version) < 18446744073709551615n);
  return { ...base, eligibility: controls, admission: admissionOf(r.admission), settings: settings(r.settings), limits, catalog_version: digest(r.catalog_version) };
};
export const credentialDetailOf: Parser<CredentialDetail> = (v) => {
  const r = record(v);
  const base = detailOf(credentialOf)(r);
  const controls = closed({ can_revoke: boolean, can_rotate: boolean, step_up_state: stepUp, rotate_step_up_state: stepUp, execution_state: execution })(r.eligibility);
  if (controls.can_revoke) requireRead(base.row.current_state !== "revoked" && controls.step_up_state === "ready" && controls.execution_state === "ready");
  if (controls.can_rotate) requireRead(base.row.current_state === "active" && base.row.service_state === "active" && base.row.rotation !== "ending" && base.row.replaced_by === undefined && controls.rotate_step_up_state === "ready" && controls.execution_state === "ready");
  return { ...base, eligibility: controls, limits: limitsOf(r.limits) };
};
export const organizationDetailOf: Parser<OrganizationDetail> = (v) => ({ ...detailOf(organizationOf)(v), eligibility: eligible(record(v).eligibility) });
export const scopeDetailOf: Parser<ScopeDetail> = (v) => ({ ...detailOf(scopeOf)(v), eligibility: eligible(record(v).eligibility), catalog_version: digest(record(v).catalog_version) });

export const audienceDetailOf: Parser<AudienceDetail> = (v) => ({ ...detailOf(audienceOf)(v), catalog_version: digest(record(v).catalog_version) });
