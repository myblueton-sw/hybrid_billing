import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient, attempt, failureOf, sequence } from "./admin";

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
  assert.deepEqual(out, { state: "user_created", fields: { link_ref: "abc", user_id: "u1" } });
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
  assert.deepEqual(empty, { revision: "", next: "", users: [] });
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

test("the policy read keeps the server times and defaults missing parts", async () => {
  const page = { revision: "r4", policy: null, proposals: [{ number: 2 }], now: "2026-11-16T09:00:00Z", signed_in_at: "2026-11-16T08:00:00Z", act_until: "", you: "u1" };
  const { f, calls } = fake(200, page);
  const got = await adminClient("tok", f).policy();
  assert.equal(calls[0].url, "/api/v1/identity/policy");
  assert.deepEqual(got, page);
  const empty = await adminClient("tok", fake(200, {}).f).policy();
  assert.deepEqual(empty, { revision: "", policy: null, proposals: [], now: "", signed_in_at: "", act_until: "", you: "" });
});

test("the access read and preview", async () => {
  const page = { revision: "r2", grants: [{ id: "g" }], delegations: [], catalog: [{ action: "view" }], subjects: [{ user_id: "u", login: "l" }], now: "n" };
  const { f, calls } = fake(200, page);
  assert.deepEqual(await adminClient("t", f).access(), page);
  assert.equal(calls[0].url, "/api/v1/identity/access");
  assert.deepEqual(await adminClient("t", fake(200, {}).f).access(), { revision: "", grants: [], delegations: [], catalog: [], subjects: [], now: "" });
  const p = fake(200, { decisions: [{ action: "view" }] });
  assert.equal((await adminClient("t", p.f).preview("u 1", "focus_data", "c1")).length, 1);
  assert.equal(p.calls[0].url, "/api/v1/identity/access/preview?subject=u+1&kind=focus_data&customer=c1");
  const q = fake(200, {});
  assert.deepEqual(await adminClient("t", q.f).preview("u", "k"), []);
  assert.equal(q.calls[0].url, "/api/v1/identity/access/preview?subject=u&kind=k");
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
