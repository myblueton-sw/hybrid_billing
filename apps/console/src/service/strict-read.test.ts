import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient, policyPageOf } from "./admin";
import { accessFixture, credentialFixture, grantFixture, orgFixture, requestFixture, revocationFixture, serviceFixture, servicesFixture, selfFixture } from "./read-fixtures";

function client(body: unknown) {
  return adminClient("t", (async () => new Response(JSON.stringify(body), { status: 200 })) as typeof fetch);
}
const invalid = (error: unknown) => error instanceof AdminError && error.failure === "invalid";
const access = (body: unknown) => client(body).access();
const services = (body: unknown) => client(body).services();
const me = (body: unknown) => client(body).me();
const organizations = (body: unknown) => client(body).organizations();

for (const [name, fixture, read, fields] of [
  ["access", accessFixture, access, ["grants", "delegations", "catalog", "subjects", "settings", "requests", "revocations"]],
  ["services", servicesFixture, services, ["services", "owners"]],
  ["self", selfFixture, me, ["identities"]],
  ["organizations", orgFixture, organizations, ["parties", "profiles", "members"]],
] as const) {
  test(`${name} requires arrays and rejects malformed members without dropping valid rows`, async () => {
    for (const field of fields) {
      for (const bad of [undefined, null, {}, "", [null], [false], [{}, null]]) {
        await assert.rejects(read({ ...fixture, [field]: bad }), invalid, `${field}: ${JSON.stringify(bad)}`);
      }
      await read({ ...fixture, [field]: [] });
    }
  });
}

test("access rejects malformed nested permission facts", async () => {
  for (const patch of [{ effective: "true" }, { effective: 1 }, { version: Number.MAX_SAFE_INTEGER + 1 }, { version: -1 }, { version: 1.5 },
    { state: "future" }, { scope: {} }, { scope: { ...grantFixture.scope, customer: false } }, { fields: ["rows", 1] }, { fields: null }]) {
    await assert.rejects(access({ ...accessFixture, grants: [grantFixture, { ...grantFixture, ...patch }] }), invalid);
  }
  for (const patch of [{ can_approve: "false" }, { can_reject: null }, { can_withdraw: 0 }, { current: "true" }, { eligible_approvers: 1.5 },
    { days: -1 }, { state: "future" }, { change: "update" }, { may_end: [1] }]) {
    await assert.rejects(access({ ...accessFixture, requests: [{ ...requestFixture, ...patch }] }), invalid);
  }
  const delegation = { ...grantFixture, delegator: "u", delegate: "v", parent: "g", redelegate: false, state: "ended" };
  assert.equal((await access({ ...accessFixture, delegations: [delegation] })).delegations[0].state, "ended");
  await assert.rejects(access({ ...accessFixture, delegations: [{ ...delegation, redelegate: "true" }] }), invalid);
  await assert.rejects(access({ ...accessFixture, catalog: [{ ...accessFixture.catalog[0], fields: [false] }] }), invalid);
  await assert.rejects(access({ ...accessFixture, subjects: [{ user_id: 1, login: "a" }] }), invalid);
  await assert.rejects(access({ ...accessFixture, settings: [{ ...accessFixture.settings[0], value: "24" }] }), invalid);
});

test("revocation receipts and cleanup rows retain validated facts only", async () => {
  const cleanup = { item: "C9", state: "done", reason: "chain_rules", refs: ["d1"] };
  const valid = { ...revocationFixture, cleanup: [cleanup] };
  assert.deepEqual((await access({ ...accessFixture, revocations: [valid] })).revocations[0], valid);
  for (const patch of [{ verified: "true" }, { target_present: 1 }, { receipt: null }, { receipt: { ...valid.receipt, sequence: Number.MAX_SAFE_INTEGER + 1 } },
    { history: [{ state: "received", at: 123 }] }, { ended_delegations: [1] }, { cleanup: [{ ...cleanup, refs: [false] }] },
    { cleanup: [{ ...cleanup, state: "future" }] }, { state: "future" }, { job_state: "future" }]) {
    await assert.rejects(access({ ...accessFixture, revocations: [{ ...valid, ...patch }] }), invalid);
  }
});

test("preview never infers an allowed decision", async () => {
  const decision = { action: "view", field: "rows", allowed: false, reason: "denied", delegations: [] };
  assert.deepEqual(await client({ decisions: [decision] }).preview("u", "k"), [decision]);
  for (const decisions of [undefined, null, [decision, {}], [{ ...decision, allowed: "false" }], [{ ...decision, delegations: [1] }]]) {
    await assert.rejects(client({ decisions }).preview("u", "k"), invalid);
  }
});

test("service credentials reject unsafe permissions, versions and supplied pickup metadata", async () => {
  for (const patch of [{ state: "future" }, { usable: "true" }, { secret_pickup: null }, { secret_pickup: "stored" },
    { secret_pickup: "open" }, { secret_pickup: "open", secret_pickup_until: "soon" }, { secret_pickup_until: "2026-11-21T00:00:00Z" },
    { state: "expired", secret_pickup: "collected" }]) {
    await assert.rejects(services({ ...servicesFixture, services: [{ ...serviceFixture, credentials: [{ ...credentialFixture, ...patch }] }] }), invalid);
  }
  for (const patch of [{ state: "future" }, { version: Number.MAX_SAFE_INTEGER + 1 }, { credentials: null }]) {
    await assert.rejects(services({ ...servicesFixture, services: [{ ...serviceFixture, ...patch }] }), invalid);
  }
  assert.equal((await services({ ...servicesFixture, limits: { ...servicesFixture.limits, credential_days: 0 } })).limits.credential_days, 0);
  for (const field of ["credentials_per_service", "services", "replay_seconds"]) {
    await assert.rejects(services({ ...servicesFixture, limits: { ...servicesFixture.limits, [field]: 0 } }), invalid);
  }
  for (const field of Object.keys(servicesFixture.limits)) {
    for (const bad of [undefined, null, -1, 1.5, "3600", Number.MAX_SAFE_INTEGER + 1]) {
      await assert.rejects(services({ ...servicesFixture, limits: { ...servicesFixture.limits, [field]: bad } }), invalid);
    }
  }
});

test("self identity and session facts reject coercion while empty identities remain empty", async () => {
  assert.deepEqual(await me(selfFixture), selfFixture);
  for (const patch of [{ identities: [{ method: 1, login: "l" }] }, { identities: [{ method: "local-password", login: null }] },
    { current_session: { ...selfFixture.current_session, current: "true" } }, { password_set: "false" }, { installation_administrator: "true" },
    { idle_minutes: -1 }, { max_age_seconds: Number.MAX_SAFE_INTEGER + 1 }]) {
    await assert.rejects(me({ ...selfFixture, ...patch }), invalid);
  }
  for (const patch of [{ current_session:null }, { current_session:{ ...selfFixture.current_session,state:"future" } }, { recorded_session_count:1.5 }, { accepted_operation_count:"0" }]) {
    await assert.rejects(me({ ...selfFixture,...patch }),invalid);
  }
});

test("organization profile and timeline strings and numeric revisions cannot be coerced", async () => {
  assert.deepEqual(await organizations(orgFixture), orgFixture);
  for (const patch of [{ parties: [{ ...orgFixture.parties[0], id: null }] }, { profiles: [{ ...orgFixture.profiles[0], payer: 1 }] },
    { profiles: [{ ...orgFixture.profiles[0], currency: {} }] }, { members: [{ ...orgFixture.members[0], targets: [false] }] },
    { members: [{ ...orgFixture.members[0], history: [{ ...orgFixture.members[0].history[0], until: null }] }] },
    { active_revision: Number.MAX_SAFE_INTEGER + 1 }, { scan_stopped: "false" }]) {
    await assert.rejects(organizations({ ...orgFixture, ...patch }), invalid);
  }
  const displayState = { ...orgFixture.parties[0], state: "future_display_state" };
  assert.equal((await organizations({ ...orgFixture, parties: [displayState] })).parties[0].state, displayState.state);
});

test("stricter fields distinguish a known empty list from unavailable evidence", () => {
  for (const value of [undefined, null, false, "max_age", ["max_age", false]]) assert.equal(policyPageOf({ trust: { stricter_fields: value } }).trust.stricter_fields, null);
  for (const value of [[], ["max_age", "step_up"], ["future_field"]]) assert.deepEqual(policyPageOf({ trust: { stricter_fields: value } }).trust.stricter_fields, value);
});
