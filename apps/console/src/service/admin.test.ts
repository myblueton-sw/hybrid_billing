import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient, attempt, failureOf, ledgerOf, outcomeOf, reauthenticate, sequence } from "./admin";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown, headers: Record<string, string> = {}) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(typeof body === "string" ? body : JSON.stringify(body), { status, headers });
  }) as unknown as typeof fetch;
  return { f, calls };
}

test("failure codes map to a closed set", () => {
  const cases: [number, Record<string, unknown>, string][] = [
    [409, { code: "revision_conflict" }, "conflict"],
    [422, { code: "prerequisite_failed" }, "refused"],
    [422, { code: "invalid_stage" }, "stage"],
    [422, { code: "invalid_request", fields: ["login"] }, "invalid"],
    [403, { code: "permission_denied" }, "denied"],
    [403, { code: "csrf_refused" }, "unauthenticated"],
    [401, {}, "unauthenticated"],
    [429, {}, "limited"],
    [500, { code: "internal_error", operation_id: "op-1" }, "partial"],
    [500, { code: "internal_error" }, "unavailable"],
    [503, { code: "dependency_unavailable" }, "busy"],
    [409, { code: "idempotency_conflict" }, "stale"],
    [409, { code: "idempotency_key_expired" }, "stale"],
    [503, { code: "authority_unavailable" }, "unavailable"],
  ];
  for (const [status, body, want] of cases) assert.equal(failureOf(status, body).failure, want, JSON.stringify(body));
  assert.deepEqual(failureOf(422, { code: "invalid_request", fields: ["login"] }).fields, ["login"]);
  assert.equal(failureOf(500, { code: "internal_error", operation_id: "op-1" }).operation, "op-1");
});

test("commands send the envelope with csrf and the attempt identity", async () => {
  const { f, calls } = fake(200, { business_state: "user_created", fields: { link_ref: "abc", user_id: "u1" } });
  const tried = attempt();
  const out = await adminClient("tok", f).command("identity.create_user", "users", "r3", "why", { login: "a@b.c" }, tried);
  assert.deepEqual(out, { status: "", state: "user_created", job_id: "", fields: { link_ref: "abc", user_id: "u1" } });
  const sent = JSON.parse(String(calls[0].init.body));
  assert.deepEqual(sent, { request_id: tried.request, target_id: "users", idempotency_key: tried.key, expected_revision: "r3", rationale: "why", body: { login: "a@b.c" } });
  const h = calls[0].init.headers as Record<string, string>;
  assert.equal(h["X-CSRF-Token"], "tok");
  assert.equal(h["X-Request-Id"], tried.request);
  assert.equal(calls[0].url, "/api/v1/commands/identity.create_user");
  assert.equal(calls[0].init.cache, "no-store");
});

test("attempts are fresh per confirmation", () => {
  const a = attempt(), b = attempt();
  assert.notEqual(a.key, b.key);
  assert.notEqual(a.request, b.request);
  assert.match(a.key, /^m01-[0-9a-f]{24}$/);
});

test("reads page with a cursor", async () => {
  const { f, calls } = fake(200, { revision: "r9", next: "u2", users: [{ user_id: "u1" }] });
  const page = await adminClient("tok", f).users("u0", 2);
  assert.equal(calls[0].url, "/api/v1/identity/users?limit=2&after=u0");
  assert.equal(page.revision, "r9");
  assert.equal(page.next, "u2");
  assert.equal(page.users.length, 1);
  const empty = await adminClient("tok", fake(200, {}).f).users();
  assert.deepEqual({ revision: empty.revision, next: empty.next, users: empty.users, can: empty.can_bulk_revoke, settings: empty.settings },
    { revision: "", next: "", users: [], can: false, settings: [] });
  assert.equal(empty.clock.clock_base.choice, "default");
  assert.equal(empty.clock.wall_lead, null);
  const full = await adminClient("tok", fake(200, { users: [{ user_id: "u1", membership: { role: "administrator", state: "suspended", version: 3, role_period: "ended",
    suspended_for: "role_period_ended", ledger: "unconfirmed", role_until: "2026-11-18T10:00:00Z" }, revocation: { bulk: "bulk-revocation-1", state: "deny_confirmed" } },
    { user_id: "u2", membership: { role_period: "later", suspended_for: "x", ledger: "x" } }], can_bulk_revoke: true,
    wall_lead_beyond_tolerance: { wall_time: "a", server_time: "b" }, clock_base: { choice: "ledger", recorded: true }, recovery_state: { stranded: true, records: [3] } }).f).users();
  assert.deepEqual(full.users[0].membership, { role: "administrator", state: "suspended", version: 3, role_from: "", role_until: "2026-11-18T10:00:00Z", role_period: "ended",
    suspended_for: "role_period_ended", ledger: "unconfirmed" });
  assert.deepEqual(full.users[0].revocation, { bulk: "bulk-revocation-1", state: "deny_confirmed" });
  assert.deepEqual([full.users[1].membership?.role_period, full.users[1].membership?.suspended_for, full.users[1].membership?.ledger, full.users[1].revocation], ["open", "", "", null]);
  assert.deepEqual([full.can_bulk_revoke, full.clock.wall_lead, full.clock.clock_base.recorded, full.clock.recovery], [true, { wall_time: "a", server_time: "b" }, true, { stranded: true, records: [3] }]);
});

test("rate limits carry the wait", async () => {
  await assert.rejects(adminClient("t", fake(429, { code: "rate_limited" }, { "Retry-After": "3" }).f).users(), (e: AdminError) => e.failure === "limited" && e.retryAfter === 3);
  await assert.rejects(adminClient("t", fake(429, "x").f).users(), (e: AdminError) => e.failure === "limited" && e.retryAfter === 0);
});

test("network failure and unreadable bodies are unavailable", async () => {
  const broken = (async () => { throw new Error("down"); }) as unknown as typeof fetch;
  await assert.rejects(adminClient("t", broken).users(), (e: AdminError) => e.failure === "unavailable");
  await assert.rejects(adminClient("t", fake(502, "not json").f).users(), (e: AdminError) => e.failure === "unavailable");
});

test("collect returns only a well-formed activation link", async () => {
  const good = "http://127.0.0.1:5173/activate#" + "A".repeat(43);
  const { f, calls } = fake(200, { url: good });
  assert.equal(await adminClient("t", f).collect("a/b"), good);
  assert.equal(calls[0].url, "/api/v1/identity/links/a%2Fb");
  await assert.rejects(adminClient("t", fake(200, { url: "javascript:alert(1)" }).f).collect("x"), (e: AdminError) => e.failure === "unavailable");
  await assert.rejects(adminClient("t", fake(403, { code: "permission_denied" }).f).collect("x"), (e: AdminError) => e.failure === "denied");
});

test("only the latest request is accepted", () => {
  const s = sequence();
  const first = s.next();
  const second = s.next();
  assert.equal(s.latest(first), false);
  assert.equal(s.latest(second), true);
});

test("the policy read keeps the server times and the trust model and defaults missing parts", async () => {
  const change = {
    id: "trust-0a1b2c3d-1", number: 1, state: "independently_approved", change_revision: 2, change_digest: "c".repeat(64), old_epoch: { value: "4294967297", generation: 1, number: 1 },
    old_policy: 3, old_digest: "o".repeat(64), new_digest: "n".repeat(64), candidate: { methods: ["password"], mfa: "none", max_age_seconds: 3600, step_up: [{ action: "policy.change", max_age_seconds: 600, mfa: false }] },
    proposer: "a", validator: "b", approver: "b", activator: "", superseded_by: "", superseded_epoch: null, earlier_run: false, run_started_at: "",
    validation: { result: "validated", at: "t", evidence_digest: "e".repeat(64), evidence_current: true, items: [{ item: "test_login", result: "verified", reason: "", detail: "" }] },
    impact: { members: 3, sessions: 2, affected_ids: ["a", "b", "c"], weakens_mfa: false, signs_out_everyone: true }, activation: null, job: null, invalidation: null,
    can: { revise: false, validate: true, approve: false, activate: true, reject: true, cancel: false }, why_not: { approve: "not_validated", activate: "" }, act_until: "x",
    history: [{ state: "draft", by: "a", at: "t", rationale: "why" }],
  };
  const page = {
    revision: "r4", policy: { methods: ["password"], mfa: "none", max_age_seconds: 43200, step_up: [], number: 1, epoch: { value: "1", generation: 0, number: 1 }, proposer: "", validator: "", approver: "", activator: "", at: "t" },
    now: "2026-11-16T09:00:00Z", signed_in_at: "2026-11-16T08:00:00Z", authenticated_at: "2026-11-16T08:30:00Z", act_until: "", step_up_seconds: 900, you: "u1", started_at: "s",
    trust: { epoch: "1", epoch_generation: 0, epoch_number: 1, policy_number: 1, digest: "d", origin: "seed", carried_from: "", carried_line: 0, durable_unconfirmed: false, activated_at: "",
      reset_at: "", older_record_ignored: false, proposer: "", validator: "", approver: "", activator: "", change: "", hold: { reason: "commit_unknown", since: "t" } },
    ledger: { state: "ok", isolation_reason: "", problem_line: 0, records: 2, watermark: "2:x", generation: 0, generation_stated: false, not_durable: false },
    rights: { propose: true, validate: true, approve: true, activate: true }, holders: { qualifying_people: 3, members: 3, needed: 3, enrolment: "not_decided", ready: true, missing: [] }, changes: [change],
  };
  const { f, calls } = fake(200, page);
  const got = await adminClient("tok", f).policy();
  assert.equal(calls[0].url, "/api/v1/identity/policy");
  assert.deepEqual(got, page);
  const empty = await adminClient("tok", fake(200, { changes: [{ state: "nonsense" }], trust: { origin: "elsewhere" } }).f).policy();
  assert.equal(empty.policy, null);
  assert.equal(empty.trust.origin, "");
  assert.equal(empty.trust.hold, null);
  assert.equal(empty.ledger.state, "unknown");
  assert.equal(empty.holders.needed, 3);
  assert.equal(empty.changes[0].state, "draft");
  assert.equal(empty.changes[0].validation, null);
  assert.equal(empty.changes[0].impact.signs_out_everyone, true);
});

test("trust refusals map to their own failures", () => {
  const cases: [string, AdminError["failure"]][] = [["self_approval_denied", "self"], ["authority_epoch_conflict", "epoch"], ["approval_stale", "approval_stale"],
    ["evidence_stale", "evidence_stale"], ["independent_approval_required", "needs_approval"]];
  for (const [code, failure] of cases) assert.equal(failureOf(409, { code }).failure, failure, code);
});

test("the access read and preview", async () => {
  const page = { revision: "r2", grants: [{ id: "g" }], delegations: [], catalog: [{ action: "view" }], subjects: [{ user_id: "u", login: "l" }], now: "n" };
  const { f, calls } = fake(200, page);
  const got = await adminClient("t", f).access();
  assert.deepEqual({ revision: got.revision, grants: got.grants, delegations: got.delegations, catalog: got.catalog, subjects: got.subjects, now: got.now }, page);
  assert.equal(calls[0].url, "/api/v1/identity/access");
  const empty = await adminClient("t", fake(200, {}).f).access();
  assert.deepEqual(empty, { revision: "", grants: [], delegations: [], catalog: [], subjects: [], now: "", you: "", started_at: "", settings: [], requests: [], revocations: [], ledger: ledgerOf(null) });
  assert.equal(empty.ledger.state, "unknown");
  const full = await adminClient("t", fake(200, {
    you: "me", started_at: "s", settings: [{ name: "access_request_hours" }],
    requests: [{ id: "access-request-1" }, { id: "access-request-2", fields: ["rows"], may_end: ["d1"] }],
    revocations: [{ id: "revocation-1" }, { id: "revocation-2", history: [{ state: "received", at: "t" }], ended_delegations: ["d"], cleanup: [{ item: "C1" }, { item: "C9", refs: ["d"] }], receipt: { sequence: 2, watermark: "2:x", reason: "" } }],
    ledger: { state: "isolated", isolation_reason: "missing", records: 3, recovery: { at: "a", truncated_bytes: 5 }, not_durable: true },
  }).f).access();
  assert.equal(full.you, "me");
  assert.deepEqual(full.requests.map((q) => [q.fields, q.may_end]), [[[], []], [["rows"], ["d1"]]]);
  assert.deepEqual(full.revocations[0].receipt, { sequence: 0, watermark: "", reason: "" });
  assert.deepEqual(full.revocations[0].cleanup, []);
  assert.deepEqual(full.revocations[1].cleanup.map((c) => c.refs), [[], ["d"]]);
  assert.equal(full.ledger.state, "isolated");
  assert.equal(full.ledger.not_durable, true);
  assert.equal(ledgerOf({ state: "ok", not_durable: "yes" }).not_durable, false);
  assert.deepEqual(full.ledger.recovery, { at: "a", truncated_bytes: 5 });
  assert.equal(ledgerOf({ state: "strange" }).state, "unknown");
  const p = fake(200, { decisions: [{ action: "view" }] });
  assert.equal((await adminClient("t", p.f).preview("u 1", "focus_data", "c1")).length, 1);
  assert.equal(p.calls[0].url, "/api/v1/identity/access/preview?subject=u+1&kind=focus_data&customer=c1");
  const q = fake(200, {});
  assert.deepEqual(await adminClient("t", q.f).preview("u", "k"), []);
  assert.equal(q.calls[0].url, "/api/v1/identity/access/preview?subject=u&kind=k");
});

test("accepted results keep the status and the job", async () => {
  const { f } = fake(202, { status: "accepted", job_id: "revocation-3", fields: { request: "access-request-2", revocation: "revocation-3" } });
  const out = await adminClient("t", f).command("access.approve_request", "access-request-2", "r4", "why", { digest: "d" }, attempt());
  assert.deepEqual(out, { status: "accepted", state: "", job_id: "revocation-3", fields: { request: "access-request-2", revocation: "revocation-3" } });
  assert.deepEqual(outcomeOf({ status: "completed", business_state: "requested", fields: { request: "access-request-1", digest: "x" } }),
    { status: "completed", state: "requested", job_id: "", fields: { request: "access-request-1", digest: "x" } });
});

test("a replayed resend and a changed payload are told apart", async () => {
  const replay = await adminClient("t", fake(200, { status: "completed", business_state: "requested", replay: true, fields: { request: "access-request-1" } }).f)
    .command("access.request_grant", "grants", "r1", "why", {}, attempt());
  assert.equal(replay.fields.request, "access-request-1");
  await assert.rejects(adminClient("t", fake(409, { code: "idempotency_conflict" }).f).command("access.request_grant", "grants", "r1", "other", {}, attempt()),
    (e: unknown) => e instanceof AdminError && e.failure === "stale");
});

test("the services read defaults missing parts", async () => {
  const page = {
    revision: "r3",
    services: [{ id: "svc-a", state: "active", version: 1, creator: "c", creator_login: "cl", owner: "o", owner_login: "ol", credentials: [{ id: "cred-1" }] }, { id: "svc-b" }],
    owners: [{ user_id: "u", login: "l" }],
    limits: { credential_days: 30, credentials_per_service: 10, services: 200 },
    now: "n",
  };
  const { f, calls } = fake(200, page);
  const got = await adminClient("t", f).services();
  assert.equal(calls[0].url, "/api/v1/identity/services");
  assert.equal(got.services[0].credentials.length, 1);
  assert.deepEqual(got.services[1].credentials, []);
  assert.deepEqual(got.limits, { credential_days: 30, credentials_per_service: 10, services: 200 });
  assert.deepEqual(await adminClient("t", fake(200, {}).f).services(), {
    revision: "",
    services: [],
    owners: [],
    limits: { credential_days: 0, credentials_per_service: 0, services: 0 },
    now: "",
  });
});

test("secret returns only a well-formed secret and credential", async () => {
  const good = { secret: "hbsk_" + "A".repeat(43), credential: "cred-0123456789abcdef" };
  const { f, calls } = fake(200, good);
  assert.deepEqual(await adminClient("t", f).secret("r/1"), good);
  assert.equal(calls[0].url, "/api/v1/identity/links/r%2F1");
  for (const bad of [{ ...good, secret: "hbsk_short" }, { ...good, credential: "cred-XYZ" }, { url: "https://x/activate#" + "a".repeat(43) }, { ...good, secret: "<b>" + good.secret }]) {
    await assert.rejects(adminClient("t", fake(200, bad).f).secret("x"), (e: AdminError) => e.failure === "invalid");
  }
  await assert.rejects(adminClient("t", fake(403, { code: "permission_denied" }).f).secret("x"), (e: AdminError) => e.failure === "denied");
});

test("self reads and commands", async () => {
  const page = { user_id: "u", login: "l", email: "e", workspace: "w", role: "member", identities: [{ method: "local-password", login: "l" }], password_set: true,
    installation_administrator: false, sessions: [{ handle: "h", current: true, signed_in: "a", ends: "b" }], idle_minutes: 30, max_age_seconds: 3600, signed_in_at: "a", authenticated_at: "c", now: "n" };
  const r = fake(200, page);
  assert.deepEqual(await adminClient("t", r.f).me(), page);
  assert.equal(r.calls[0].url, "/api/v1/me");
  const empty = await adminClient("t", fake(200, {}).f).me();
  assert.deepEqual([empty.identities, empty.sessions, empty.password_set, empty.max_age_seconds], [[], [], false, 0]);
  const sent: Call[] = [];
  const noContent = (async (url: string, init: RequestInit) => {
    sent.push({ url, init });
    return new Response(null, { status: 204 });
  }) as unknown as typeof fetch;
  await adminClient("t", noContent).changePassword("a", "b");
  assert.equal(sent[0].url, "/api/v1/me/password");
  assert.equal(sent[0].init.body, JSON.stringify({ current: "a", new: "b" }));
  const e = fake(200, { ended: 2 });
  assert.equal(await adminClient("t", e.f).endSessions("others"), 2);
  assert.equal(e.calls[0].init.body, JSON.stringify({ session: "others" }));
  assert.equal(await adminClient("t", fake(200, {}).f).endSessions("x"), 0);
  for (const [code, failure] of [["current_password_wrong", "wrong"], ["password_locked", "locked"], ["password_form", "form"]]) {
    await assert.rejects(adminClient("t", fake(422, { code }).f).changePassword("a", "b"), (x: AdminError) => x.failure === failure);
  }
});

test("the organizations read builds its query and defaults missing parts", async () => {
  const r = fake(200, { revision: "r", relationship_revision: 2, active_revision: 3, on: "2026-11-16", workspace: { id: "w", profile: "msp" }, parties: [{ id: "p" }],
    profiles: [], kind: "payer", members: [{ member: "m", name: "", targets: [{ id: "t", name: "" }] }, { member: "n" }], next: "n" });
  const got = await adminClient("t", r.f).organizations({ on: "2026-11-16", kind: "payer", revision: 2, after: "a b", limit: 5 });
  assert.equal(r.calls[0].url, "/api/v1/organizations?on=2026-11-16&kind=payer&revision=2&after=a+b&limit=5");
  assert.equal(got.relationship_revision, 2);
  assert.deepEqual(got.members[1].targets, []);
  assert.deepEqual(got.members[1].history, []);
  assert.equal(got.members[0].history.length, 0);
  const e = fake(200, {});
  const empty = await adminClient("t", e.f).organizations();
  assert.equal(e.calls[0].url, "/api/v1/organizations");
  assert.deepEqual([empty.kind, empty.parties, empty.members, empty.workspace, empty.next, empty.active_revision], ["hierarchy", [], [], { id: "", profile: "" }, "", 0]);
  const z = fake(200, {});
  await adminClient("t", z.f).organizations({ revision: 0 });
  assert.equal(z.calls[0].url, "/api/v1/organizations?revision=0");
  assert.deepEqual([empty.filters, empty.provider_filter, empty.scan_stopped, empty.scan_budget], [{ subject: "any", entity: "", unattributed: false, q: "", member_count: 0, target: "" }, "", false, 0]);
});

test("the organizations read sends the server filters and reads their echo", async () => {
  const r = fake(200, { filters: { subject: "account", entity: "le-1", unattributed: true, q: "a b", member_count: 2, target: "g-1" }, provider_filter: "not_connected", scan_stopped: true, scan_budget: 1000 });
  const got = await adminClient("t", r.f).organizations({ kind: "view_group", subject: "account", entity: "le-1", unattributed: true, q: "a b", member: ["m-2", "m-1"], target: "g-1" });
  assert.equal(r.calls[0].url, "/api/v1/organizations?kind=view_group&subject=account&entity=le-1&unattributed=true&q=a+b&member=m-2&member=m-1&target=g-1");
  assert.deepEqual(got.filters, { subject: "account", entity: "le-1", unattributed: true, q: "a b", member_count: 2, target: "g-1" });
  assert.deepEqual([got.provider_filter, got.scan_stopped, got.scan_budget], ["not_connected", true, 1000]);
  const off = fake(200, { scan_stopped: "true", filters: { unattributed: "true" } });
  const plain = await adminClient("t", off.f).organizations({ unattributed: false, member: [] });
  assert.equal(off.calls[0].url, "/api/v1/organizations");
  assert.deepEqual([plain.scan_stopped, plain.filters.unattributed], [false, false]);
});

test("the targets read builds its query and defaults missing parts", async () => {
  const r = fake(200, { revision: "r", relationship_revision: 2, active_revision: 3, on: "2026-11-16", kind: "allocation_pool", next: "g-2", scan_stopped: true,
    targets: [{ id: "g-1", name: "One", members_on: 4 }, {}, null] });
  const got = await adminClient("t", r.f).targets({ kind: "view_group", on: "2026-11-16", revision: 0, after: "g 0", limit: 5, q: "north" });
  assert.equal(r.calls[0].url, "/api/v1/organizations/targets?kind=view_group&on=2026-11-16&revision=0&after=g+0&limit=5&q=north");
  assert.deepEqual(got.targets, [{ id: "g-1", name: "One", members_on: 4 }, { id: "", name: "", members_on: 0 }, { id: "", name: "", members_on: 0 }]);
  assert.deepEqual([got.kind, got.next, got.scan_stopped, got.relationship_revision, got.active_revision, got.on, got.revision], ["allocation_pool", "g-2", true, 2, 3, "2026-11-16", "r"]);
  const e = fake(200, {});
  const empty = await adminClient("t", e.f).targets({ kind: "payer" });
  assert.equal(e.calls[0].url, "/api/v1/organizations/targets?kind=payer");
  assert.deepEqual([empty.kind, empty.targets, empty.next, empty.scan_stopped, empty.relationship_revision], ["payer", [], "", false, 0]);
  await assert.rejects(adminClient("t", fake(422, { code: "invalid_request" }).f).targets({ kind: "payer" }), (x: AdminError) => x.failure === "invalid");
});

test("the plans read defaults missing parts and never invents a computed impact", async () => {
  const r = fake(200, { revision: "r", active_revision: 2, open_plans: 1, open_limit: 16, impact: { money: "computed", publication: "computed", erp: "computed" }, you: "u",
    plans: [{ id: "plan-1", state: "sealed", change: { kind: "hierarchy", member: "m", action: "set", target: "t", from: "2026-12-01" }, can_activate: true,
      impact: { money: "not_computed", publication: "", erp: 3 },
      members: [{ member: "m", name: "M", differs: [{ from: "2026-12-01", until: "" }] }, null, { member: "n" }] }, {},
      { id: "plan-3", impact: { money: "computed", publication: "Computed", erp: "computed" } }, { id: "plan-4", impact: "computed" },
      { id: "plan-5", impact: { money: "computed", publication: "computed", erp: "computed" } }] });
  const got = await adminClient("t", r.f).plans();
  assert.equal(r.calls[0].url, "/api/v1/organizations/plans");
  assert.equal("impact" in got, false);
  assert.deepEqual(got.plans[0].impact, { money: "not_computed", publication: "unknown", erp: "unknown" });
  assert.deepEqual(got.plans[1].impact, { money: "unknown", publication: "unknown", erp: "unknown" });
  assert.deepEqual(got.plans[2].impact, { money: "computed", publication: "unknown", erp: "computed" });
  assert.deepEqual(got.plans[3].impact, { money: "unknown", publication: "unknown", erp: "unknown" });
  assert.deepEqual(got.plans[4].impact, { money: "computed", publication: "computed", erp: "computed" });
  assert.equal(got.plans[0].can_activate, true);
  assert.deepEqual(got.plans[0].members[1].differs, []);
  assert.deepEqual(got.plans[0].members[1].before, []);
  assert.deepEqual(got.plans[1].change, { kind: "", member: "", action: "", target: "", from: "" });
  assert.deepEqual([got.plans[1].can_activate, got.plans[1].stale, got.plans[1].kinds, got.plans[1].violations], [false, false, [], []]);
  const e = await adminClient("t", fake(200, {}).f).plans();
  assert.deepEqual([e.plans, e.open_limit], [[], 0]);
  await assert.rejects(adminClient("t", fake(422, { code: "invalid_request", fields: ["from"] }).f).command("relationship.propose", "relationships", "r", "x", {}, attempt()),
    (x: AdminError) => x.failure === "invalid" && x.fields[0] === "from");
});

function scripted(replies: [number, unknown][]) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const [status, body] = replies[Math.min(calls.length - 1, replies.length - 1)];
    return new Response(status === 204 ? null : JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const refusal = { code: "reauthentication_required", retry: false, next_action: "reauthenticate", request_id: "x" };

test("a step-up refusal maps to reauth", () => {
  assert.equal(failureOf(403, refusal).failure, "reauth");
  assert.equal(failureOf(422, { code: "prerequisite_failed" }).failure, "refused");
});

test("a refused command is resent once, byte-identical, after reauthentication", async () => {
  const { f, calls } = scripted([[403, refusal], [200, { business_state: "done" }]]);
  let prompts = 0;
  const tried = attempt();
  const out = await adminClient("tok", f, async () => { prompts++; return true; }).command("identity.suspend_user", "u1", "r3", "why", { a: "1" }, tried);
  assert.equal(out.state, "done");
  assert.equal(prompts, 1);
  assert.equal(calls.length, 2);
  assert.equal(calls[1].url, calls[0].url);
  assert.equal(calls[1].init.body, calls[0].init.body);
  assert.deepEqual(calls[1].init.headers, calls[0].init.headers);
  assert.equal((calls[1].init.headers as Record<string, string>)["X-Request-Id"], tried.request);
  assert.equal(JSON.parse(String(calls[1].init.body)).idempotency_key, tried.key);
});

test("cancel does not resend and a second refusal does not loop", async () => {
  const cancelled = scripted([[403, refusal]]);
  await assert.rejects(adminClient("t", cancelled.f, async () => false).command("identity.suspend_user", "u1", "r3", "why", {}, attempt()),
    (e: AdminError) => e.failure === "reauth_cancelled");
  assert.equal(cancelled.calls.length, 1);
  const again = scripted([[403, refusal], [403, refusal], [200, {}]]);
  let prompts = 0;
  await assert.rejects(adminClient("t", again.f, async () => { prompts++; return true; }).command("identity.suspend_user", "u1", "r3", "why", {}, attempt()),
    (e: AdminError) => e.failure === "reauth");
  assert.equal(again.calls.length, 2);
  assert.equal(prompts, 1);
  const plain = scripted([[403, refusal]]);
  await assert.rejects(adminClient("t", plain.f).command("identity.suspend_user", "u1", "r3", "why", {}, attempt()), (e: AdminError) => e.failure === "reauth");
  const sessions = scripted([[403, refusal], [200, { ended: 2 }]]);
  assert.equal(await adminClient("t", sessions.f, async () => true).endSessions("others"), 2);
  assert.equal(sessions.calls.length, 2);
});

test("reads are never prompted and other refusals pass through", async () => {
  let prompts = 0;
  const stepUp = async () => { prompts++; return true; };
  const read = scripted([[403, refusal]]);
  await assert.rejects(adminClient("t", read.f, stepUp).users(), (e: AdminError) => e.failure === "reauth");
  const mfa = scripted([[422, { code: "prerequisite_failed" }]]);
  await assert.rejects(adminClient("t", mfa.f, stepUp).command("trust.propose", "trust", "r1", "x", {}, attempt()), (e: AdminError) => e.failure === "refused");
  assert.equal(prompts, 0);
  assert.equal(read.calls.length + mfa.calls.length, 2);
});

test("concurrent refusals share one prompt", async () => {
  const { f, calls } = scripted([[403, refusal], [403, refusal], [200, {}], [200, {}]]);
  let opened = 0;
  let release: (ok: boolean) => void = () => undefined;
  let shared: Promise<boolean> | null = null;
  const stepUp = () => {
    if (!shared) {
      opened++;
      shared = new Promise<boolean>((r) => { release = r; });
    }
    return shared;
  };
  const api = adminClient("t", f, stepUp);
  const a = api.command("identity.suspend_user", "u1", "r3", "x", {}, attempt());
  const b = api.command("identity.reinstate_user", "u2", "r3", "x", {}, attempt());
  await new Promise((r) => setTimeout(r, 0));
  release(true);
  await Promise.all([a, b]);
  assert.equal(opened, 1);
  assert.equal(calls.length, 4);
});

test("reauthenticate posts the password once and maps failures", async () => {
  const ok = scripted([[204, {}]]);
  await reauthenticate("tok", "secret value", ok.f);
  assert.equal(ok.calls[0].url, "/api/v1/me/reauthenticate");
  assert.equal(ok.calls[0].init.body, JSON.stringify({ password: "secret value" }));
  assert.equal((ok.calls[0].init.headers as Record<string, string>)["X-CSRF-Token"], "tok");
  assert.equal(ok.calls[0].init.method, "POST");
  for (const [status, body, want] of [[422, { code: "current_password_wrong" }, "wrong"], [401, { code: "unauthenticated" }, "unauthenticated"],
    [409, { code: "revision_conflict" }, "conflict"], [503, { code: "dependency_unavailable" }, "busy"]] as [number, unknown, string][]) {
    await assert.rejects(reauthenticate("t", "p", scripted([[status, body]]).f), (e: AdminError) => e.failure === want);
  }
  const broken = (async () => { throw new Error("down"); }) as unknown as typeof fetch;
  await assert.rejects(reauthenticate("t", "p", broken), (e: AdminError) => e.failure === "unavailable");
  const garbled = (async () => new Response("x", { status: 500 })) as unknown as typeof fetch;
  await assert.rejects(reauthenticate("t", "p", garbled), (e: AdminError) => e.failure === "unavailable");
});
