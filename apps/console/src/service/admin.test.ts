import { credentialRowOf } from "./admin";
import { accessFixture, requestFixture, revocationFixture, credentialFixture, orgFixture } from "./read-fixtures";
import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient, attempt, failureOf, pickupGone, ledgerOf, outcomeOf, planOf, reauthenticate, sequence, usersQueryString } from "./admin";
import { selfEndOf, selfSessionOf } from "./self-end";

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
  const page = await adminClient("tok", f).users({ after: "a:u0", limit: 2 });
  assert.equal(calls[0].url, "/api/v1/identity/users?after=a%3Au0&limit=2");
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
  assert.equal(full.users[1].membership?.state, "unknown");
  assert.equal(full.users[1].membership?.role, "unknown");
  assert.deepEqual([full.can_bulk_revoke, full.clock.wall_lead, full.clock.clock_base.recorded, full.clock.recovery], [true, { wall_time: "a", server_time: "b" }, true, { stranded: true, records: [3] }]);
});

test("users query sends only the set filters and reads the page facts", async () => {
  assert.equal(usersQueryString({}), "limit=50");
  assert.equal(usersQueryString({ q: "김 a", status: "active", membership: "", role: "member", period: "ends_soon", signin: "set_link", invitation: "pending", holder: "true", after: "d:u9", limit: 7 }),
    "q=%EA%B9%80+a&status=active&role=member&period=ends_soon&signin=set_link&invitation=pending&holder=true&after=d%3Au9&limit=7");
  const { f, calls } = fake(200, { next_login: "ann@x", filters: { q: "ann", status: "active", extra: "x" }, scan_stopped: true, scan_budget: 1000, order: "login_then_departed_user_id",
    list_version: "7.3", auth_methods: { connected: ["password"], not_connected: ["saml", "oidc", "ldap"] }, warning_days: 14 });
  const page = await adminClient("tok", f).users({ membership: "none" });
  assert.equal(calls[0].url, "/api/v1/identity/users?membership=none&limit=50");
  assert.deepEqual(page.filters, { q: "ann", status: "active", membership: "", role: "", period: "", signin: "", invitation: "", holder: "" });
  assert.deepEqual([page.scan_stopped, page.scan_budget, page.order, page.list_version, page.warning_days, page.next_login], [true, 1000, "login_then_departed_user_id", "7.3", 14, "ann@x"]);
  assert.deepEqual(page.auth_methods, { connected: ["password"], not_connected: ["saml", "oidc", "ldap"] });
  const bare = await adminClient("tok", fake(200, { auth_methods: { connected: "x" } }).f).users();
  assert.deepEqual([bare.scan_stopped, bare.scan_budget, bare.list_version, bare.auth_methods, bare.warning_days, bare.filters.q, bare.next_login], [false, 0, "", { connected: [], not_connected: [] }, 0, "", ""]);
  await assert.rejects(adminClient("t", fake(409, { code: "revision_conflict", fields: ["after"] }).f).users({ after: "a:u1" }),
    (e: AdminError) => e.failure === "conflict" && e.fields[0] === "after");
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
    trust: { stricter_fields: null, epoch: "1", epoch_generation: 0, epoch_number: 1, policy_number: 1, digest: "d", origin: "seed", carried_from: "", carried_line: 0, durable_unconfirmed: false, activated_at: "",
      reset_at: "", older_record_ignored: false, proposer: "", validator: "", approver: "", activator: "", change: "", hold: { reason: "commit_unknown", since: "t" } },
    ledger: { state: "ok", isolation_reason: "", problem_line: 0, records: 2, watermark: "2:x", generation: 0, generation_stated: false, not_durable: false },
    rights: { propose: true, validate: true, approve: true, activate: true }, holders: { qualifying_people: 3, members: 3, needed: 3, enrolment: "not_decided", ready: true, missing: [] }, changes: [change], waiting_activation: "", earlier_run_kept: false,
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
  assert.equal(empty.changes[0].impact, null);
  const computed = await adminClient("tok", fake(200, { changes: [{ state: "proposed", impact: { members: 0, sessions: 0, affected_ids: [], signs_out_everyone: true } }, { state: "proposed", impact: null }, { state: "proposed", impact: [] }] }).f).policy();
  assert.deepEqual(computed.changes[0].impact, { members: 0, sessions: 0, affected_ids: [], weakens_mfa: false, signs_out_everyone: true });
  assert.equal(computed.changes[1].impact, null);
  assert.equal(computed.changes[2].impact, null);
});

test("trust refusals map to their own failures", () => {
  const cases: [string, AdminError["failure"]][] = [["self_approval_denied", "self"], ["authority_epoch_conflict", "epoch"], ["approval_stale", "approval_stale"],
    ["evidence_stale", "evidence_stale"], ["independent_approval_required", "needs_approval"]];
  for (const [code, failure] of cases) assert.equal(failureOf(409, { code }).failure, failure, code);
});

test("the access read and preview", async () => {
  const page = accessFixture;
  const { f, calls } = fake(200, page);
  const got = await adminClient("t", f).access();
  const { grants: _g, delegations: _d, subjects: _s, requests: _q, revocations: _r, ...kept } = page;
  assert.deepEqual(got, { ...kept, ledger: ledgerOf(page.ledger) });
  for (const gone of ["grants", "delegations", "subjects", "requests", "revocations"]) assert.equal(gone in got, false, gone);
  assert.equal(calls[0].url, "/api/v1/identity/access");
  await assert.rejects(adminClient("t", fake(200, {}).f).access(), (e: AdminError) => e.failure === "invalid");
  const full = await adminClient("t", fake(200, {
    ...accessFixture, you: "me", started_at: "s",
    requests: [requestFixture, { ...requestFixture, id: "access-request-2", fields: ["rows"], may_end: ["d1"] }],
    revocations: [revocationFixture, { ...revocationFixture, id: "revocation-2", history: [{ state: "received", at: "t" }], ended_delegations: ["d"], cleanup: [{ item: "C1", state: "done", reason: "", refs: [] }, { item: "C9", state: "done", reason: "", refs: ["d"] }], receipt: { sequence: 2, watermark: "2:x", reason: "" } }],
    ledger: { state: "isolated", isolation_reason: "missing", records: 3, recovery: { at: "a", truncated_bytes: 5 }, not_durable: true },
  }).f).access();
  assert.equal(full.you, "me");
  assert.equal(full.ledger.state, "isolated");
  assert.equal(full.ledger.not_durable, true);
  assert.equal(ledgerOf({ state: "ok", not_durable: "yes" }).not_durable, false);
  assert.deepEqual(full.ledger.recovery, { at: "a", truncated_bytes: 5 });
  assert.equal(ledgerOf({ state: "strange" }).state, "unknown");
  const p = fake(200, { decisions: [{ action: "view", field: "rows", allowed: true, reason: "grant", delegations: [] }] });
  assert.equal((await adminClient("t", p.f).preview("u 1", "focus_data", "c1")).length, 1);
  assert.equal(p.calls[0].url, "/api/v1/identity/access/preview?subject=u+1&kind=focus_data&customer=c1");
  const q = fake(200, {});
  await assert.rejects(adminClient("t", q.f).preview("u", "k"), (e: AdminError) => e.failure === "invalid");
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

test("P02 credential metadata keeps optional pickup states without copying secrets", () => {
  const at = "2026-11-21T00:00:00Z";
  const rows = [
    { ...credentialFixture, id: "c-open", secret_pickup: "open", secret_pickup_until: at, secret: "hbsk_x", secret_ref: "ref" },
    { ...credentialFixture, id: "c-never", secret_pickup: "never_collected" },
    { ...credentialFixture, id: "c-got", secret_pickup: "collected" }, credentialFixture,
  ];
  const got = rows.map(credentialRowOf);
  const [open, never, collected, absent] = got;
  assert.deepEqual(open, { ...credentialFixture, id: "c-open", secret_pickup: "open", secret_pickup_until: at });
  assert.equal(never.secret_pickup, "never_collected");
  assert.equal(collected.secret_pickup, "collected");
  assert.equal(Object.hasOwn(absent, "secret_pickup"), false);
  assert.equal(JSON.stringify(got).includes("hbsk_"), false);
  assert.equal(JSON.stringify(got).includes("secret_ref"), false);
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
    installation_administrator: false, current_session: { handle: "h", current: true, signed_in: "a", ends: "b", state: "active", can_end: false, not_included_operation: "" }, recorded_session_count: 1, accepted_operation_count: 0, sessions_generation: "g", idle_minutes: 30, absolute_minutes: 720,
    max_age_seconds: 3600, signed_in_at: "a", authenticated_at: "c", now: "n" };
  const r = fake(200, page);
  assert.deepEqual(await adminClient("t", r.f).me(), page);
  assert.equal(r.calls[0].url, "/api/v1/me");
  await assert.rejects(adminClient("t", fake(200, {}).f).me(), (e: AdminError) => e.failure === "invalid");
  const sent: Call[] = [];
  const noContent = (async (url: string, init: RequestInit) => {
    sent.push({ url, init });
    return new Response(null, { status: 204 });
  }) as unknown as typeof fetch;
  await adminClient("t", noContent).changePassword("a", "b");
  assert.equal(sent[0].url, "/api/v1/me/password");
  assert.equal(sent[0].init.body, JSON.stringify({ current: "a", new: "b" }));
  const e = fake(202, { operation: "session-end-1", kind: "others", count: 2, state: "cleanup_running" });
  const op = await adminClient("t", e.f).endOthers("g", ["a", "b"]);
  assert.deepEqual([op.operation, op.count, op.state], ["session-end-1", 2, "cleanup_running"]);
  assert.equal(e.calls[0].init.body, JSON.stringify({ session: "others", generation: "g", handles: ["a", "b"] }));
  const one = fake(202, { kind: "one", not_included: [{ handle: "x" }] });
  assert.deepEqual((await adminClient("t", one.f).endOne("h1")).not_included, []);
  assert.equal(one.calls[0].init.body, JSON.stringify({ session: "h1" }));
  for (const [code, failure] of [["current_password_wrong", "wrong"], ["password_locked", "locked"], ["password_form", "form"]]) {
    await assert.rejects(adminClient("t", fake(422, { code }).f).changePassword("a", "b"), (x: AdminError) => x.failure === failure);
  }
});

test("the organizations read builds its query and rejects missing parts", async () => {
  const r = fake(200, orgFixture);
  assert.deepEqual(await adminClient("t", r.f).organizations({ on: "2026-11-16", kind: "payer", revision: 2, after: "a b", limit: 5 }), orgFixture);
  assert.equal(r.calls[0].url, "/api/v1/organizations?on=2026-11-16&kind=payer&revision=2&after=a+b&limit=5");
  await assert.rejects(adminClient("t", fake(200, {}).f).organizations(), (e: AdminError) => e.failure === "invalid");
  const z = fake(200, orgFixture);
  await adminClient("t", z.f).organizations({ revision: 0 });
  assert.equal(z.calls[0].url, "/api/v1/organizations?revision=0");
});

test("the organizations read sends the server filters and reads their echo", async () => {
  const r = fake(200, { ...orgFixture, filters: { subject: "account", entity: "le-1", unattributed: true, q: "a b", member_count: 2, target: "g-1" }, provider_filter: "not_connected", scan_stopped: true, scan_budget: 1000 });
  const got = await adminClient("t", r.f).organizations({ kind: "view_group", subject: "account", entity: "le-1", unattributed: true, q: "a b", member: ["m-2", "m-1"], target: "g-1" });
  assert.equal(r.calls[0].url, "/api/v1/organizations?kind=view_group&subject=account&entity=le-1&unattributed=true&q=a+b&member=m-2&member=m-1&target=g-1");
  assert.deepEqual(got.filters, { subject: "account", entity: "le-1", unattributed: true, q: "a b", member_count: 2, target: "g-1" });
  assert.deepEqual([got.provider_filter, got.scan_stopped, got.scan_budget], ["not_connected", true, 1000]);
  const off = fake(200, { ...orgFixture, scan_stopped: "true", filters: { ...orgFixture.filters, unattributed: "true" } });
  await assert.rejects(adminClient("t", off.f).organizations({ unattributed: false, member: [] }), (e: AdminError) => e.failure === "invalid");
  assert.equal(off.calls[0].url, "/api/v1/organizations");
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
  const sessions = scripted([[403, refusal], [202, { count: 2 }]]);
  assert.equal((await adminClient("t", sessions.f, async () => true).endOthers("g", ["a"])).count, 2);
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

test("the view group drafts read defaults missing parts and never invents a state", async () => {
  const r = fake(200, { revision: "r4", active_revision: 2, open: 1, open_limit: 16, next: "vgd-2",
    settings: [{ name: "view_group_drafts_open", value: 16, unit: "drafts", default: 16, min: 1, max: 200, source: "default" }],
    drafts: [{ id: "vgd-1", name: "Ops", group: "group-1", group_label: "group-1", new_group: true, member_count: 2, members: [{ id: "a", name: "A" }, null, {}],
      author: "u", author_login: "root", created: "c", updated: "u2", state: "proposed", activated_revision: 0, plan: { id: "plan-1", state: "sealed", blocked: "impact_not_computed" } },
    {}, null, { state: "Weird", plan: "x", members: "y" }] });
  const got = await adminClient("t", r.f).viewGroupDrafts("vgd-0", 10);
  assert.equal(r.calls[0].url, "/api/v1/organizations/view-group-drafts?after=vgd-0&limit=10");
  assert.deepEqual([got.revision, got.active_revision, got.open, got.open_limit, got.next, got.settings.length], ["r4", 2, 1, 16, "vgd-2", 1]);
  const d = got.drafts[0];
  assert.deepEqual(d.members, [{ id: "a", name: "A" }, { id: "", name: "" }]);
  assert.deepEqual(d.plan, { id: "plan-1", state: "sealed", closed: "", blocked: "impact_not_computed", rule: "" });
  assert.deepEqual([d.state, d.new_group, d.member_count, d.author_login, d.pruned], ["proposed", true, 2, "root", false]);
  assert.deepEqual([got.drafts[1].state, got.drafts[1].plan, got.drafts[1].members, got.drafts[2].id], ["unknown", null, [], ""]);
  assert.deepEqual([got.drafts[3].state, got.drafts[3].plan, got.drafts[3].members], ["unknown", null, []]);
  const e = fake(200, {});
  const empty = await adminClient("t", e.f).viewGroupDrafts();
  assert.equal(e.calls[0].url, "/api/v1/organizations/view-group-drafts");
  assert.deepEqual([empty.drafts, empty.settings, empty.next, empty.open_limit], [[], [], "", 0]);
});

test("the draft members read pages, reports a missing draft and keeps other failures", async () => {
  const r = fake(200, { revision: "r", draft: "vgd-1", member_count: 3, pruned: true, next: "b", members: [{ id: "a", name: "A" }, { id: "b" }] });
  const got = await adminClient("t", r.f).draftMembers("vgd 1", "a", 2);
  assert.equal(r.calls[0].url, "/api/v1/organizations/view-group-drafts/vgd%201/members?after=a&limit=2");
  assert.deepEqual(got, { revision: "r", draft: "vgd-1", member_count: 3, pruned: true, next: "b", members: [{ id: "a", name: "A" }, { id: "b", name: "" }] });
  const e = fake(200, {});
  assert.deepEqual(await adminClient("t", e.f).draftMembers("vgd-1"), { revision: "", draft: "", member_count: 0, pruned: false, next: "", members: [] });
  assert.equal(e.calls[0].url, "/api/v1/organizations/view-group-drafts/vgd-1/members");
  assert.equal(await adminClient("t", fake(404, { code: "unsupported_command" }).f).draftMembers("vgd-x"), "missing");
  await assert.rejects(adminClient("t", fake(503, { code: "dependency_unavailable" }).f).draftMembers("vgd-x"), (x: AdminError) => x.failure === "busy");
});

test("a plan row carries its draft block or null", () => {
  assert.deepEqual(planOf({ draft: { id: "vgd-1", name: "Ops", group: "group-1", new_group: true, member_count: 3 } }).draft, { id: "vgd-1", name: "Ops", group: "group-1", new_group: true, member_count: 3 });
  assert.deepEqual(planOf({ draft: {} }).draft, { id: "", name: "", group: "", new_group: false, member_count: 0 });
  assert.equal(planOf({ draft: null }).draft, null);
  assert.equal(planOf({}).draft, null);
});

test("session end answers are parsed field by field into a closed shape", () => {
  const op = selfEndOf({
    operation: "session-end-3", kind: "others", count: 1, state: "cleanup_incomplete", blocked_at: "t", list_generation: "a", current_generation: "b", generation_changed: true,
    digest: "d", cleanup: { done: 0, already_ended: 0, failed: 1, pending: 0 }, replay: true,
    sessions: [{ handle: "h", signed_in: "s", result: "ended", items: [{ item: "C3", state: "failed", reason: "retry_budget_spent" }, { item: "C9", state: "weird" }] },
      { handle: "g", result: "already_gone" }, { handle: "q", result: "nonsense" }],
    not_included: [{ handle: "n", signed_in: "s2" }, "junk"],
  });
  assert.deepEqual(op.sessions.map((s) => s.result), ["ended", "already_gone", "already_gone"]);
  assert.deepEqual(op.sessions[0].items, [{ item: "C3", state: "failed", reason: "retry_budget_spent" }, { item: "C9", state: "pending", reason: "" }]);
  assert.deepEqual(op.not_included, [{ handle: "n", signed_in: "s2" }, { handle: "", signed_in: "" }]);
  assert.deepEqual([op.state, op.generation_changed, op.replay, op.cleanup.failed], ["cleanup_incomplete", true, true, 1]);
  const empty = selfEndOf(null);
  assert.deepEqual([empty.kind, empty.state, empty.count, empty.sessions, empty.not_included], ["others", "cleanup_running", 0, [], []]);
  assert.deepEqual(selfSessionOf({ handle: 5, current: "yes" }), { handle: "", current: false, signed_in: "", ends: "" });
});

const historyBody = (change: Record<string, unknown> = {}) => ({
  events: [
    { at: "2026-11-19T10:00:00Z", kind: "signed_in", reason: "", workspace: "w", session: { handle: "0123456789abcdef", current: true, active: true }, device: { browser: "chrome", os: "macos" }, count: 0 },
    { at: "2026-11-19T09:00:00Z", kind: "sign_in_refused", reason: "wrong_password", workspace: null, session: null, device: { browser: "unknown", os: "unknown" }, count: 0 },
  ],
  next: "MS0w",
  retention_days: 90,
  retention_source: "default",
  period_since: "2026-11-01T00:00:00Z",
  effective_days: 90,
  available_from: "2026-11-02T00:00:00Z",
  recording: { state: "recording", since: null, reason: null },
  notes: [{ kind: "dropped", from: "2026-11-18T00:00:00Z", to: "2026-11-18T01:00:00Z", count: null }, { kind: "held", from: null, to: null, count: 0 }],
  now: "2026-11-19T10:00:01Z",
  ...change,
});

test("access history read parses the page, sends only from and before, and drops count 0 notes", async () => {
  const { f, calls } = fake(200, historyBody());
  const got = await adminClient("t", f).accessHistory({ from: "2026-11-12T10:00:01Z", before: "MS0w" }, "req-1");
  assert.equal(calls[0].url, "/api/v1/me/access-history?from=2026-11-12T10%3A00%3A01Z&before=MS0w");
  assert.equal((calls[0].init.headers as Record<string, string>)["X-Request-Id"], "req-1");
  assert.equal(got.events.length, 2);
  assert.equal(got.events[1].session, null);
  assert.equal(got.events[0].device?.browser, "chrome");
  assert.equal(got.next, "MS0w");
  assert.deepEqual(got.notes.map((n) => [n.kind, n.count]), [["dropped", null]]);
  const plain = fake(200, historyBody());
  await adminClient("t", plain.f).accessHistory({}, "req-2");
  assert.equal(plain.calls[0].url, "/api/v1/me/access-history");
});

test("access history read fails on an unknown kind, reason, note or device instead of dropping it", async () => {
  const bad = [
    historyBody({ events: [{ at: "x", kind: "teleported", reason: "", workspace: null, session: null, device: null, count: 0 }] }),
    historyBody({ events: [{ at: "x", kind: "session_ended", reason: "magic", workspace: "w", session: null, device: null, count: 0 }] }),
    historyBody({ notes: [{ kind: "rumour", from: null, to: null, count: null }] }),
    historyBody({ events: [{ at: "x", kind: "signed_in", reason: "", workspace: "w", session: null, device: { browser: "mosaic", os: "macos" }, count: 0 }] }),
    historyBody({ retention_days: "90" }),
  ];
  for (const body of bad) {
    await assert.rejects(adminClient("t", fake(200, body).f).accessHistory({}, "r"), (err: unknown) => err instanceof AdminError && err.failure === "invalid");
  }
  await assert.rejects(adminClient("t", fake(401, { code: "unauthenticated" }).f).accessHistory({}, "r"), (err: unknown) => err instanceof AdminError && err.failure === "unauthenticated");
});

test("diagnostic metadata retains only confirmed response tokens", () => {
  const error = failureOf(422, { code: "prerequisite_failed", request_id: "server-330" });
  assert.equal((error as AdminError & { request: string }).request, "server-330");
  for (const value of [null, 7, {}, "line\nbreak", "x".repeat(257)]) {
    assert.equal((failureOf(422, { code: value, request_id: value }) as AdminError & { request: string }).request, "");
    assert.equal(failureOf(422, { code: value }).code, "");
  }
});

test("rate limit and cancelled step-up preserve the response diagnostic identity", async () => {
  await assert.rejects(adminClient("t", fake(429, { code: "rate_limited", request_id: "limited-330" }, { "Retry-After": "3" }).f).integrations(),
    (e: AdminError & { request: string }) => e.code === "rate_limited" && e.request === "limited-330" && e.retryAfter === 3);
  await assert.rejects(adminClient("t", fake(403, { code: "reauthentication_required", request_id: "step-330" }).f, async () => false)
    .command("integrations.plugin_stop", "plugin:sample", "r1", "synthetic reason", {}, { request: "client-330", key: "key-330" }),
    (e: AdminError & { request: string }) => e.failure === "reauth_cancelled" && e.code === "reauthentication_required" && e.request === "step-330");
});


test("raw401 retires authentication once and never starts step-up, even with a misleading body", async () => {
  const net=fake(401,{code:"reauthentication_required"});let asked=0,notified=0;
  const api=adminClient("synthetic",net.f,async()=>{asked++;return true;});
  const unsubscribe=api.subscribeAuthenticationLoss(()=>{notified++;});
  await assert.rejects(api.endSelectedSessions({selection:"s",digest:"d",count:1}),AdminError);
  assert.equal(asked,0);assert.equal(net.calls.length,1);assert.equal(notified,1);
  await assert.rejects(api.me(),AdminError);assert.equal(notified,1);assert.equal(net.calls.length,1);unsubscribe();
  let late=0;api.subscribeAuthenticationLoss(()=>{late++;})();assert.equal(late,1);
});

test("raw401 is observed before its response body resolves", {timeout:1000}, async () => {
  let release!: (value:unknown)=>void;let observed=false;
  const body=new Promise<unknown>(done=>{release=done;});
  const net=(async()=>({status:401,ok:false,headers:new Headers(),json:()=>body})) as unknown as typeof fetch;
  const api=adminClient("synthetic",net);api.subscribeAuthenticationLoss(()=>{observed=true;});
  const pending=api.me();await new Promise<void>(done=>setImmediate(done));const beforeBody=observed;
  release({code:"unauthenticated"});await assert.rejects(pending,AdminError);assert.equal(beforeBody,true);
});

test("authentication observers can detach without changing transport failure", async () => {
  const net=fake(401,{});const api=adminClient("synthetic",net.f);let notified=0;
  const unsubscribe=api.subscribeAuthenticationLoss(()=>{notified++;});unsubscribe();
  await assert.rejects(api.me(),AdminError);assert.equal(notified,0);
});

test("authentication retirement before a step-up retry prevents another mutation POST", async () => {
  let posts=0;
  const net=(async(path:string)=>{
    if(path.endsWith("/me/sessions/end")){posts++;return new Response(JSON.stringify({code:"reauthentication_required"}),{status:403});}
    return new Response(JSON.stringify({code:"unauthenticated"}),{status:401});
  }) as unknown as typeof fetch;
  let api:ReturnType<typeof adminClient>;
  api=adminClient("synthetic",net,async()=>{await assert.rejects(api.me(),AdminError);return true;});
  let failure:unknown;
  try{await api.endSelectedSessions({selection:"s",digest:"d",count:1});}catch(error){failure=error;}
  assert.equal(posts,1);
  assert.ok(failure instanceof AdminError&&failure.failure==="unauthenticated"&&failure.code===""&&failure.request==="");
  await assert.rejects(api.me(),error=>error instanceof AdminError&&error.failure==="unauthenticated"&&error.code==="");
});

test("a refused CSRF context retires authentication without treating permission denial as sign-out", async () => {
  for(const [code,expected] of [["csrf_refused",1],["permission_denied",0]] as const){
    const net=fake(403,{code});const api=adminClient("synthetic",net.f);let observed=0;
    api.subscribeAuthenticationLoss(()=>{observed++;});await assert.rejects(api.me(),AdminError);assert.equal(observed,expected);
  }
});

test("a pickup is gone only when the service refuses it", () => {
  assert.equal(pickupGone(failureOf(403, { code: "permission_denied" })), true);
  for (const e of [failureOf(429, {}), failureOf(503, {}), failureOf(401, {}), failureOf(403, { code: "reauthentication_required" }), new AdminError("unavailable"), new Error("x")]) assert.equal(pickupGone(e), false);
});

test("the plans read carries the closed history facts and the closed plans read builds its query", async () => {
  const facts = { total: 120, next: "71", kept: 200, dropped: 3 };
  const first = await adminClient("t", fake(200, { revision: "r", plans: [], closed: facts }).f).plans();
  assert.deepEqual(first.closed, facts);
  const bare = await adminClient("t", fake(200, { revision: "r", plans: [] }).f).plans();
  assert.deepEqual(bare.closed, { total: 0, next: "", kept: 0, dropped: 0 });
  const odd = await adminClient("t", fake(200, { revision: "r", plans: [], closed: { total: "5", next: 7, kept: -1, dropped: 1.5 } }).f).plans();
  assert.deepEqual(odd.closed, { total: 0, next: "", kept: 0, dropped: 0 });
  const r = fake(200, { plans: [{ id: "p-1", state: "discarded" }], next: "12", matching: 40, closed: facts, q: "ann" });
  const got = await adminClient("t", r.f).closedPlans({ after: "71", limit: 50, q: "a b" });
  assert.equal(r.calls[0].url, "/api/v1/organizations/plans/closed?after=71&limit=50&q=a+b");
  assert.deepEqual([got.plans.length, got.plans[0].id, got.next, got.matching, got.q, got.closed.total], [1, "p-1", "12", 40, "ann", 120]);
  const none = fake(200, { plans: [], next: "", matching: 0, closed: facts, q: "" });
  await adminClient("t", none.f).closedPlans();
  assert.equal(none.calls[0].url, "/api/v1/organizations/plans/closed");
  for (const bad of [{}, { plans: [] }, { plans: "x", next: "" }, { plans: [], next: 5 }]) {
    await assert.rejects(adminClient("t", fake(200, bad).f).closedPlans(), (e: AdminError) => e.failure === "invalid", JSON.stringify(bad));
  }
});
