import { test } from "node:test";
import assert from "node:assert/strict";
import { serviceClient, ServiceError, fragmentToken } from "./api";
type Call = { url: string; init: RequestInit };
function fake(responses: Array<[number, unknown, Record<string, string>?]>) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const [status, body, headers] = responses.shift() ?? [500, {}];
    return new Response(status === 204 ? null : JSON.stringify(body), {
      status,
      headers: headers ?? {},
    });
  }) as unknown as typeof fetch;
  return { f, calls };
}
test("login fetches a pre-session token once and sends it", async () => {
  const { f, calls } = fake([
    [200, { csrf_token: "pre" }],
    [200, { user: "u", workspace: "w", role: "administrator", login: "a@b.c", csrf_token: "s" }],
    [401, { code: "unauthenticated" }],
  ]);
  const c = serviceClient(f);
  const s = await c.login("w", "a@b.c", "pw");
  assert.deepEqual(s, { user: "u", workspace: "w", role: "administrator", login: "a@b.c", csrf: "s" });
  assert.equal(calls[0].url, "/api/v1/auth/csrf");
  assert.equal((calls[0].init.headers as Record<string, string>)["X-HB-Client"], "console");
  assert.equal((calls[1].init.headers as Record<string, string>)["X-CSRF-Token"], "pre");
  assert.equal(calls[1].init.body, JSON.stringify({ workspace: "w", login: "a@b.c", password: "pw" }));
  await assert.rejects(c.login("w", "x", "y"), (e: ServiceError) => e.failure === "unauthenticated");
  assert.equal(calls.length, 3);
});
test("failures map to stable kinds", async () => {
  const cases: Array<[number, string, string]> = [
    [429, "rate_limited", "limited"],
    [422, "link_invalid", "link"],
    [422, "password_form", "password"],
    [422, "invalid_request", "form"],
    [503, "dependency_unavailable", "unavailable"],
    [403, "origin_refused", "unavailable"],
  ];
  for (const [status, code, want] of cases) {
    const { f } = fake([[200, { csrf_token: "p" }], [status, { code }, { "Retry-After": "7" }]]);
    await assert.rejects(serviceClient(f).redeem("t", "p"), (e: ServiceError) => {
      assert.equal(e.failure, want);
      assert.equal(e.retryAfter, 7);
      return true;
    });
  }
});
test("current returns null when signed out and the session when signed in", async () => {
  const { f } = fake([
    [401, { code: "unauthenticated" }],
    [200, { user: "u", workspace: "w", role: "member", csrf_token: "c" }],
    [503, { code: "authority_unavailable" }],
  ]);
  const c = serviceClient(f);
  assert.equal(await c.current(), null);
  assert.equal((await c.current())?.role, "member");
  await assert.rejects(c.current(), (e: ServiceError) => e.failure === "unavailable");
});
test("peek and redeem bodies never put the token in a URL", async () => {
  const { f, calls } = fake([
    [200, { csrf_token: "p" }],
    [200, { purpose: "activate" }],
    [204, {}],
    [200, { purpose: "reset", login: "a@b.c" }],
  ]);
  const c = serviceClient(f);
  assert.deepEqual(await c.peek("tok"), { purpose: "activate" });
  await c.redeem("tok", "pw", { login: "a@b.c", email: "a@b.c" });
  assert.deepEqual(await c.peek("tok"), { purpose: "reset", login: "a@b.c" });
  for (const call of calls) assert.ok(!call.url.includes("tok"));
  assert.equal(calls[2].init.body, JSON.stringify({ token: "tok", password: "pw", login: "a@b.c", email: "a@b.c" }));
});
test("network failure is unavailable and resets the pre-session token", async () => {
  let n = 0;
  const f = (async (url: string) => {
    n++;
    if (n === 2) throw new TypeError("offline");
    return new Response(JSON.stringify(url.endsWith("csrf") ? { csrf_token: "p" + n } : { purpose: "set", login: "x" }), { status: 200 });
  }) as unknown as typeof fetch;
  const c = serviceClient(f);
  await assert.rejects(c.peek("t"), (e: ServiceError) => e.failure === "unavailable");
  assert.deepEqual(await c.peek("t"), { purpose: "set", login: "x" });
  assert.equal(n, 4);
});
test("logout sends the session token and errors without a body", async () => {
  const { f, calls } = fake([[204, {}]]);
  await serviceClient(f).logout({ user: "u", workspace: "w", role: "r", login: "", csrf: "s" });
  assert.equal((calls[0].init.headers as Record<string, string>)["X-CSRF-Token"], "s");
  const bad = (async () => new Response("not json", { status: 500 })) as unknown as typeof fetch;
  await assert.rejects(serviceClient(bad).current(), (e: ServiceError) => e.failure === "unavailable");
});
test("concurrent calls share one pre-session token", async () => {
  const { f, calls } = fake([
    [200, { csrf_token: "only" }],
    [200, { purpose: "set", login: "a" }],
    [200, { purpose: "set", login: "b" }],
  ]);
  const c = serviceClient(f);
  await Promise.all([c.peek("x"), c.peek("y")]);
  assert.equal(calls.filter((x) => x.url.endsWith("/auth/csrf")).length, 1);
  for (const call of calls.slice(1)) assert.equal((call.init.headers as Record<string, string>)["X-CSRF-Token"], "only");
});
test("a failed pre-session fetch is retried on the next call", async () => {
  const { f, calls } = fake([
    [503, { code: "dependency_unavailable" }],
    [200, { csrf_token: "p" }],
    [200, { purpose: "set", login: "a" }],
  ]);
  const c = serviceClient(f);
  await assert.rejects(c.peek("x"));
  assert.deepEqual(await c.peek("x"), { purpose: "set", login: "a" });
  assert.equal(calls.length, 3);
});
test("an expired pre-session is refreshed and retried once", async () => {
  const { f, calls } = fake([
    [200, { csrf_token: "old" }],
    [403, { code: "csrf_refused" }],
    [200, { csrf_token: "new" }],
    [200, { purpose: "set", login: "a" }],
    [403, { code: "csrf_refused" }],
    [200, { csrf_token: "newer" }],
    [403, { code: "csrf_refused" }],
  ]);
  const c = serviceClient(f);
  assert.deepEqual(await c.peek("x"), { purpose: "set", login: "a" });
  assert.equal((calls[3].init.headers as Record<string, string>)["X-CSRF-Token"], "new");
  await assert.rejects(c.peek("x"), (e: ServiceError) => e.failure === "expired");
});
test("a pre-session response without a token is unavailable", async () => {
  const { f } = fake([[200, {}]]);
  await assert.rejects(serviceClient(f).peek("x"), (e: ServiceError) => e.failure === "unavailable");
  const bad = (async () => new Response("not json", { status: 200 })) as unknown as typeof fetch;
  await assert.rejects(serviceClient(bad).peek("x"), (e: ServiceError) => e.failure === "unavailable");
});
test("fragment tokens are exactly 43 base64url characters", () => {
  const ok = "A".repeat(43);
  assert.equal(fragmentToken("#" + ok), ok);
  for (const bad of ["", "#", "#" + "A".repeat(42), "#" + "A".repeat(44), "#" + "!".repeat(43)]) assert.equal(fragmentToken(bad), "");
});
