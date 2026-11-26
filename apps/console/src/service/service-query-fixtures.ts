export const queryTime = "2026-11-20T01:00:00.000000001Z";
export const serviceQueryEcho = { workspace: "home", q: "", organization: "", owner: "", audience: "", scope: "", expiry: "all", state: "all", declaration: "all", service: "", collection: "", version: "", limit: 2 };
export const serviceLimits = { command_value_bytes: "8192", rationale_bytes: 256, body_bytes: 1048576, credential_days: 90, credentials_per_service: "9007199254740993", services: 200, replay_seconds: 300, credential_overlap_minutes: 1440, credential_overlap_max_minutes: 10080, secret_pickup_minutes: 10 };
export const serviceAdmission = { recorded: "8", limit: "200", certainty: "exact" };
export const serviceDeclaration = { state: "recorded", recorded_at: queryTime, version: "1", purpose: "Scheduled imports", organization: "unit", organization_name: "", organization_name_state: "unavailable", catalog_version: "a".repeat(64), scopes: [{ audience: "import", scope: "read" }] };
export const serviceAccount = { id: "svc-import", state: "active", version: "9007199254740993", owner: "owner", owner_login: "responsible", creator: "creator", creator_login: "creator", declaration: serviceDeclaration,
  credential_summary: { state: "exact_at_reference", recorded: "3", active: "1", expired: "1", revoked: "1", earliest_unexpired_expiry: "2026-12-20T00:00:00Z" }, recent_use_state: "unavailable" };
export const serviceCredential = { id: "cred-0000000000000001", service: "svc-import", service_state: "active", state: "active", current_state: "active", not_before: queryTime, not_after: "2026-12-20T00:00:00Z", usable: true, observed_at: queryTime, secret_pickup: "open", secret_pickup_until: "2026-11-20T01:05:00Z" };
export const serviceOwner = { id: "owner", login: "responsible", role: "member", from: queryTime, until: "", eligible_at_reference: true };
export const serviceOrganization = { id: "unit", name: "", name_state: "unavailable", kind: "internal_unit", parent: "entity" };
export const serviceAudience = { id: "import", resource: "urn:example:import", configured_scope_count: 2 };
export const serviceRestriction = { id: "read", action: "view", kind: "focus_data", fields: ["rows"], customer: { any: true }, legal_entity: { any: true }, contract: { any: true }, period: { any: true } };
export const serviceScope = { workspace: "home", id: "read", audience: "import", resource: "urn:example:import", restrictions: serviceRestriction };
export const serviceEligibility = { state: "eligible", can_issue: true, can_reinstate: false, can_declare: true, can_suspend: true, can_retire: true, step_up_state: "ready", execution_state: "ready", live_credentials: "1", credential_limit: "9007199254740993", operator_blocked: false };
export function servicePage(rows: unknown[], area = "accounts") {
  const query = { ...serviceQueryEcho };
  if (area !== "accounts") Object.assign(query, { expiry: "", state: "", declaration: "" });
  if (area === "credentials") Object.assign(query, { state: "all", collection: "all" });
  const order = ({ accounts: "service_id_asc", credentials: "service_id_asc_credential_id_asc", owners: "owner_id_asc", organizations: "organization_id_asc", audiences: "audience_id_asc", scopes: "scope_id_asc", history: "declaration_version_desc" } as Record<string, string>)[area];
  return { rows, search_input: "", query, next: "cursor", reference_at: queryTime, observed_at: queryTime, revision: "7", order, count: { value: String(rows.length), certainty: "lower_bound" }, work_units: 10, scan_stopped: false, settings: [], limits: serviceLimits, catalog_version: "a".repeat(64), ...(area === "accounts" ? { admission: serviceAdmission } : {}), ...(area === "history" ? { availability: "recorded" } : {}) };
}
export const accountDetail = { workspace: "home", row: { ...serviceAccount, credential_summary: { state: "not_evaluated", recorded: "", active: "", expired: "", revoked: "", earliest_unexpired_expiry: "" } }, eligibility: serviceEligibility, admission: serviceAdmission, revision: "7", observed_at: queryTime, settings: [], limits: serviceLimits, catalog_version: "a".repeat(64) };
