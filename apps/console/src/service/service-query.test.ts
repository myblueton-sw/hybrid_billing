import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError } from "./admin";
import { accountDetail, queryTime, serviceAccount, serviceAdmission, serviceAudience, serviceCredential, serviceDeclaration, serviceLimits, serviceOrganization, serviceOwner, servicePage, serviceScope, serviceRestriction } from "./service-query-fixtures";

function client(body: unknown, inspect?: (path: string, init: RequestInit) => void) {
  return adminClient("csrf", (async (path, init) => { inspect?.(String(path), init!); return new Response(JSON.stringify(body)); }) as typeof fetch);
}
const invalid = (e: unknown) => e instanceof AdminError && e.failure === "invalid";
const accounts = (body: unknown) => client(body).serviceAccounts({ workspace: "home", limit: 2 });

test("M04 preserves decimal precision and fetches only one bounded page", async () => {
  let calls = 0;
  const page = servicePage([serviceAccount]);
  const got = await client(page, (path, init) => {
    calls++;
    assert.equal(path, "/api/v1/identity/services?workspace=home&limit=2");
    assert.equal(init.method, "GET");
    assert.equal(init.cache, "no-store");
    assert.equal(init.credentials, "same-origin");
    assert.equal((init.headers as Record<string, string>)["X-CSRF-Token"], "csrf");
  }).serviceAccounts({ workspace: "home", limit: 2 });
  assert.equal(calls, 1);
  assert.deepEqual(got, page);
  assert.equal(got.rows[0].version, "9007199254740993");
  assert.equal(got.limits.credentials_per_service, "9007199254740993");
});

test("M04 rejects malformed rows, rounded totals, wrong scope and false completion", async () => {
  const page = servicePage([serviceAccount]);
  for (const patch of [{ rows: null }, { rows: [serviceAccount, {}] }, { rows: Array(101).fill(serviceAccount) }, { count: { value: 9007199254740993, certainty: "exact" } }, { count: { value: "01", certainty: "exact" } }, { count: { value: "-1", certainty: "lower_bound" } }, { count: { value: "1", certainty: "exact" } }, { next: "" }, { work_units: 10001 }, { reference_at: "now" }, { scan_stopped: "true" }, { query: { ...page.query, workspace: "foreign" } }, { query: { ...page.query, limit: 3 } }, { order: "expiry_desc" }, { admission: { ...serviceAdmission, certainty: "lower_bound" } }]) await assert.rejects(accounts({ ...page, ...patch }), invalid);
  for (const patch of [{ credential_summary: { ...serviceAccount.credential_summary, recorded: "2" } }, { version: 2 }, { declaration: { ...serviceDeclaration, scopes: null } }, { declaration: { ...serviceDeclaration, state: "missing" } }]) await assert.rejects(accounts(servicePage([{ ...serviceAccount, ...patch }])), invalid);
  const sparse = { ...page, rows: [], count: { value: "0", certainty: "lower_bound" }, scan_stopped: true };
  assert.deepEqual(await accounts(sparse), sparse);
  const terminal = { ...sparse, next: "", scan_stopped: false, count: { value: "0", certainty: "exact" } };
  assert.deepEqual(await accounts(terminal), terminal);
});

test("known service is canonical and does not invent unevaluated zero totals", async () => {
  const got = await client(accountDetail, (path) => assert.equal(path, "/api/v1/identity/services/svc-import?workspace=home")).serviceAccount("svc-import", "home");
  assert.deepEqual(got, accountDetail);
  for (const patch of [{ row: { ...accountDetail.row, id: "svc-other" } }, { workspace: "foreign" }, { eligibility: { ...accountDetail.eligibility, can_issue: "true" } }, { row: { ...accountDetail.row, credential_summary: { ...accountDetail.row.credential_summary, active: "0" } } }]) await assert.rejects(client({ ...accountDetail, ...patch }).serviceAccount("svc-import", "home"), invalid);
});

test("credential page separates frozen membership and current pickup without secrets", async () => {
  const page = servicePage([serviceCredential], "credentials");
  assert.deepEqual(await client(page).serviceCredentials({ workspace: "home", limit: 2 }), page);
  const expired = { ...serviceCredential, current_state: "expired", usable: false, secret_pickup: undefined, secret_pickup_until: undefined };
  assert.equal((await client(servicePage([expired], "credentials")).serviceCredentials({ workspace: "home", limit: 2 })).rows[0].state, "active");
  for (const patch of [{ secret_pickup_until: undefined }, { secret_pickup: "stored" }, { current_state: "expired" }, { usable: "false" }, { service_state: "retired" }]) await assert.rejects(client(servicePage([{ ...serviceCredential, ...patch }], "credentials")).serviceCredentials({ workspace: "home", limit: 2 }), invalid);
  await assert.rejects(client({ ...page, rows: [{ ...serviceCredential, secret: "unexpected", secret_ref: "unexpected" }] }).serviceCredentials({ workspace: "home", limit: 2 }), invalid);
});

test("all metadata paths and known IDs preserve requested targets", async () => {
  await client(servicePage([serviceOwner], "owners")).serviceOwners({ workspace: "home", limit: 2 });
  await client(servicePage([serviceOrganization], "organizations")).serviceOrganizations({ workspace: "home", limit: 2 });
  await client(servicePage([serviceAudience], "audiences")).serviceAudiences({ workspace: "home", limit: 2 });
  await client(servicePage([serviceScope], "scopes"), (p) => assert.equal(p, "/api/v1/identity/service-audiences/import/scopes?workspace=home&limit=2")).serviceScopes("import", { workspace: "home", limit: 2 });
  const detail = { workspace: "home", revision: "7", observed_at: queryTime };
  await client({ ...detail, row: serviceOwner }).serviceOwner("owner", "home");
  await client({ ...detail, row: serviceOrganization, eligibility: "eligible" }).serviceOrganization("unit", "home");
  await client({ ...detail, row: serviceScope, eligibility: "eligible", catalog_version: "a".repeat(64) }).serviceScope("import", "read", "home");
  await assert.rejects(client({ ...detail, row: { ...serviceScope, audience: "other" }, eligibility: "eligible", catalog_version: "a".repeat(64) }).serviceScope("import", "read", "home"), invalid);
  const credential = { ...detail, row: serviceCredential, eligibility: { can_revoke: true, step_up_state: "ready", execution_state: "ready" }, limits: serviceLimits };
  assert.deepEqual(await client(credential).serviceCredential(serviceCredential.id, "home"), credential);
  await assert.rejects(client({ ...credential, eligibility: {} }).serviceCredential(serviceCredential.id, "home"), invalid);
});

test("restrictions reject mixed unions and preserve literal named all", async () => {
  const read = (r: unknown) => client(servicePage([{ ...serviceScope, restrictions: r }], "scopes")).serviceScopes("import", { workspace: "home", limit: 2 });
  for (const patch of [{ customer: { any: false } }, { customer: { any: true, id: "x" } }, { customer: { id: "x", extra: true } }, { period: { from: "2026-13", until: "2027-01" } }, { period: { from: "2026-12", until: "2026-12" } }, { action: "approve" }, { fields: [] }, { fields: ["rows", "rows"] }, { fields: Array.from({ length: 1001 }, (_, i) => "field" + i) }, { action: "issue", fields: ["rows"] }, { extra: true }]) await assert.rejects(read({ ...serviceRestriction, ...patch }), invalid);
  const named = { ...serviceRestriction, customer: { id: "all" }, period: { from: "2026-12", until: "2027-01" } };
  assert.deepEqual((await read(named)).rows[0].restrictions, named);
});

test("history distinguishes absent and pruned declarations", async () => {
  const page = { ...servicePage([], "history"), next: "", count: { value: "0", certainty: "exact" }, availability: "not_recorded" };
  assert.equal((await client(page).serviceDeclarations("svc-import", { workspace: "home", limit: 2 })).availability, "not_recorded");
  const unavailable = { ...page, availability: "version_unavailable", query: { ...page.query, version: "1" } };
  assert.equal((await client(unavailable).serviceDeclarations("svc-import", { workspace: "home", limit: 2, version: "1" })).availability, "version_unavailable");
  await assert.rejects(client({ ...page, rows: [serviceDeclaration] }).serviceDeclarations("svc-import", { workspace: "home", limit: 2 }), invalid);
});

test("M04 response verification captures requested scope before a late reply", async () => {
  let finish!: (response: Response) => void;
  const query = { workspace: "home", limit: 2 };
  const api = adminClient("csrf", (() => new Promise<Response>((resolve) => { finish = resolve; })) as typeof fetch);
  const pending = api.serviceAccounts(query);
  query.workspace = "foreign";
  finish(new Response(JSON.stringify(servicePage([serviceAccount]))));
  assert.equal((await pending).query.workspace, "home");
  const wrong = { ...servicePage([serviceAccount]), query: { ...servicePage([]).query, q: "unrequested" } };
  await assert.rejects(accounts(wrong), invalid);
  await assert.rejects(accounts({ ...servicePage([serviceAccount]), search_input: "unrequested" }), invalid);
});

test("M04 preserves exact nanoseconds and rejects invalid chronology", async () => {
  const read = (row: unknown) => client(servicePage([row], "credentials")).serviceCredentials({ workspace: "home", limit: 2 });
  assert.equal((await read(serviceCredential)).rows[0].not_before, queryTime);
  for (const patch of [{ not_before: "2026-11-20T01:00:00.000000002Z", not_after: queryTime }, { not_before: "2026-02-31T00:00:00Z" }, { secret_pickup_until: queryTime }]) await assert.rejects(read({ ...serviceCredential, ...patch }), invalid);
});

test("M04 metadata keeps current unavailable facts and uniform denial", async () => {
  const detail = { workspace: "home", revision: "7", observed_at: queryTime };
  const unavailable = { ...detail, row: serviceOrganization, eligibility: "unavailable" };
  assert.equal((await client(unavailable).serviceOrganization("unit", "home")).eligibility, "unavailable");
  for (const [status, code, failure] of [[403, "permission_denied", "denied"], [409, "revision_conflict", "conflict"], [503, "dependency_unavailable", "busy"]] as const) {
    let calls = 0;
    const api = adminClient("csrf", (async () => { calls++; return new Response(JSON.stringify({ code, next_action: "none", request_id: "r" }), { status }); }) as typeof fetch);
    await assert.rejects(api.serviceOwner("owner", "home"), (e: unknown) => e instanceof AdminError && e.failure === failure);
    assert.equal(calls, 1);
  }
  await assert.rejects(client({ ...detail, row: serviceOwner, secret_ref: "unexpected" }).serviceOwner("owner", "home"), invalid);
});

test("current command flags cannot contradict observed prerequisites", async () => {
  for (const patch of [{ step_up_state: "reauthentication_required" }, { execution_state: "dependency_unavailable" }, { execution_state: "trust_hold" }, { state: "ineligible" }, { live_credentials: "9007199254740993" }]) await assert.rejects(client({ ...accountDetail, eligibility: { ...accountDetail.eligibility, ...patch } }).serviceAccount("svc-import", "home"), invalid);
  const suspended = { ...accountDetail, row: { ...accountDetail.row, state: "suspended" }, eligibility: { ...accountDetail.eligibility, can_issue: false, can_suspend: false, can_reinstate: true, operator_blocked: true } };
  await assert.rejects(client(suspended).serviceAccount("svc-import", "home"), invalid);
  const credential = { workspace: "home", revision: "7", observed_at: queryTime, row: serviceCredential, limits: serviceLimits, eligibility: { can_revoke: true, step_up_state: "ready", execution_state: "ready" } };
  await assert.rejects(client({ ...credential, eligibility: { ...credential.eligibility, step_up_state: "reauthentication_required" } }).serviceCredential(serviceCredential.id, "home"), invalid);
  await assert.rejects(client({ ...credential, row: { ...serviceCredential, state: "revoked", current_state: "revoked", usable: false, secret_pickup: undefined, secret_pickup_until: undefined } }).serviceCredential(serviceCredential.id, "home"), invalid);
  const reduction = { ...accountDetail, eligibility: { ...accountDetail.eligibility, state: "ineligible", can_issue: false, can_declare: false, execution_state: "trust_hold" } };
  assert.equal((await client(reduction).serviceAccount("svc-import", "home")).eligibility.can_suspend, true);
  const repair = { ...accountDetail, eligibility: { ...accountDetail.eligibility, state: "ineligible", can_issue: false } };
  assert.equal((await client(repair).serviceAccount("svc-import", "home")).eligibility.can_declare, true);
});
