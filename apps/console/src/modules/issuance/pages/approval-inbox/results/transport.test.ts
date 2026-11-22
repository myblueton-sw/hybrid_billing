import test from "node:test";
import assert from "node:assert/strict";
import { resultReadTransport } from "./transport";
import { ResultError } from "./contract";

const session = { user: "operator", workspace: "workspace", csrf: "synthetic-csrf", role: "operator", login: "synthetic" };
const sessionResponse = (override = {}) => Response.json({ ...session, csrf_token: session.csrf, ...override });
const response = (request: Record<string, unknown>, override = {}) => ({ request_id: request.request_id, status: "completed", target_id: request.target_id, result_revision: "r1", replay: false, as_of: "2026-11-22T00:00:00Z", fields: { mode: "synthetic" }, ...override });

test("result reads bind session, request and response", async () => {
 const calls: string[] = [];
 const read = resultReadTransport(session, async (input, init) => {
  calls.push(String(input));
  if (String(input).endsWith("/session")) return sessionResponse();
  const body = JSON.parse(String(init?.body));
  assert.equal(init?.credentials, "same-origin"); assert.equal(init?.cache, "no-store");
  assert.equal(body.target_id, "workspace"); assert.equal(body.idempotency_key, undefined);
  return Response.json(response(body));
 });
 const got = await read("result_evidence.preparations", "workspace", {});
 assert.ok(Object.isFrozen(got.fields));
 assert.deepEqual(calls, ["/api/v1/session", "/api/v1/commands/result_evidence.preparations"]);
});

test("changed actor workspace or CSRF prevents protected read", async () => {
 for (const override of [{ user: "other" }, { workspace: "other" }, { csrf_token: "other" }]) {
  let calls = 0;
  const read = resultReadTransport(session, async () => { calls++; return sessionResponse(override); });
  await assert.rejects(read("result_evidence.preparations", "workspace", {}), (e: unknown) => e instanceof ResultError && e.code === "session_changed");
  assert.equal(calls, 1);
 }
});

test("result reads reject unbound malformed oversized and duplicate-key responses", async () => {
 for (const variant of ["request", "target", "replay", "date", "fields", "bytes", "duplicate", "utf8"]) {
  const read = resultReadTransport(session, async (input, init) => {
   if (String(input).endsWith("/session")) return sessionResponse();
   const body = JSON.parse(String(init?.body));
   if (variant === "bytes") return new Response(" ".repeat(125001));
   if (variant === "duplicate") return new Response('{"fields":{},"fields":{}}');
   if (variant === "utf8") return new Response(new Uint8Array([0xff]));
   const overrides = { request: { request_id: "other" }, target: { target_id: "other" }, replay: { replay: true }, date: { as_of: "2026-02-30T00:00:00Z" }, fields: { fields: Object.fromEntries(Array.from({ length: 29 }, (_, i) => [`field_${i}`, "x"])) } };
   return Response.json(response(body, overrides[variant as keyof typeof overrides]));
  });
  await assert.rejects(read("result_evidence.preparations", "workspace", {}), ResultError, variant);
 }
});

test("result read transport refuses mutation commands without network calls", async () => {
 let calls = 0;
 const read = resultReadTransport(session, async () => { calls++; return sessionResponse(); });
 await assert.rejects(read("erp_result.apply", "result", {}), ResultError);
 assert.equal(calls, 0);
});

test("malformed session responses never dispatch a result command", async () => {
 for (const make of [
  () => new Response(" ".repeat(16385)),
  () => new Response('{"user":"operator","user":"operator"}'),
  () => sessionResponse({ user: ["operator"] }),
  () => sessionResponse({ workspace: ["workspace"] }),
  () => sessionResponse({ csrf_token: [session.csrf] }),
  () => new Response(new Uint8Array([0xff])),
 ]) {
  let calls = 0;
  const read = resultReadTransport(session, async () => { calls++; return make(); });
  await assert.rejects(read("result_evidence.preparations", "workspace", {}), (e: unknown) => e instanceof ResultError && e.code === "session_unavailable");
  assert.equal(calls, 1);
 }
});

test("stalled session aborts before command dispatch", async t => {
 t.mock.timers.enable({ apis: ["setTimeout"] });
 let calls = 0, aborted = false;
 const read = resultReadTransport(session, async (_, init) => {
  calls++; init?.signal?.addEventListener("abort", () => { aborted = true; });
  return new Promise<Response>(() => {});
 });
 const rejected = assert.rejects(read("result_evidence.preparations", "workspace", {}), (e: unknown) => e instanceof ResultError && e.code === "session_unavailable");
 t.mock.timers.tick(30000);
 await rejected; assert.equal(calls, 1); assert.equal(aborted, true);
});
