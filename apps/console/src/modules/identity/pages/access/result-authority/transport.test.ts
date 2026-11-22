import test from "node:test";
import assert from "node:assert/strict";
import { readTransport } from "./transport";
import { AuthorityError } from "./contract";

const session = { user: "operator", workspace: "workspace", csrf: "synthetic-csrf", role: "operator", login: "synthetic" };
const sessionResponse = (override = {}) => Response.json({ ...session, csrf_token: session.csrf, ...override });
const response = (request: Record<string, unknown>, override = {}) => ({ request_id: request.request_id, status: "completed", target_id: request.target_id, result_revision: "r1", replay: false, as_of: "2026-11-22T00:00:00Z", fields: { mode: "synthetic" }, ...override });

test("authority reads bind session, request and response", async () => {
 const calls: string[] = [];
 const read = readTransport(session, async (input, init) => {
  calls.push(String(input));
  if (String(input).endsWith("/session")) return sessionResponse();
  const body = JSON.parse(String(init?.body));
  assert.equal(init?.credentials, "same-origin"); assert.equal(init?.cache, "no-store");
  assert.equal(body.target_id, "workspace"); assert.equal(body.idempotency_key, undefined);
  return Response.json(response(body));
 });
 const got = await read("result_authority.history", "workspace", {});
 assert.ok(Object.isFrozen(got.fields));
 assert.deepEqual(calls, ["/api/v1/session", "/api/v1/commands/result_authority.history"]);
});

test("changed actor workspace or CSRF prevents protected read", async () => {
 for (const override of [{ user: "other" }, { workspace: "other" }, { csrf_token: "other" }]) {
  let calls = 0;
  const read = readTransport(session, async () => { calls++; return sessionResponse(override); });
  await assert.rejects(read("result_authority.history", "workspace", {}), (e: unknown) => e instanceof AuthorityError && e.code === "session_changed");
  assert.equal(calls, 1);
 }
});

test("authority reads reject unbound malformed oversized and duplicate-key responses", async () => {
 for (const variant of ["request", "target", "replay", "date", "fields", "bytes", "duplicate", "utf8"]) {
  const read = readTransport(session, async (input, init) => {
   if (String(input).endsWith("/session")) return sessionResponse();
   const body = JSON.parse(String(init?.body));
   if (variant === "bytes") return new Response(" ".repeat(125001));
   if (variant === "duplicate") return new Response('{"fields":{},"fields":{}}');
   if (variant === "utf8") return new Response(new Uint8Array([0xff]));
   const overrides = { request: { request_id: "other" }, target: { target_id: "other" }, replay: { replay: true }, date: { as_of: "2026-02-30T00:00:00Z" }, fields: { fields: Object.fromEntries(Array.from({ length: 29 }, (_, i) => [`field_${i}`, "x"])) } };
   return Response.json(response(body, overrides[variant as keyof typeof overrides]));
  });
  await assert.rejects(read("result_authority.history", "workspace", {}), AuthorityError, variant);
 }
});

test("authority read transport refuses mutation commands without network calls", async () => {
 let calls = 0;
 const read = readTransport(session, async () => { calls++; return sessionResponse(); });
 await assert.rejects(read("result_authority.approve", "result", {}), AuthorityError);
 assert.equal(calls, 0);
});

test("malformed session responses never dispatch a authority command", async () => {
 for (const make of [
  () => new Response(" ".repeat(16385)),
  () => new Response('{"user":"operator","user":"operator"}'),
  () => sessionResponse({ user: ["operator"] }),
  () => sessionResponse({ workspace: ["workspace"] }),
  () => sessionResponse({ csrf_token: [session.csrf] }),
  () => new Response(new Uint8Array([0xff])),
 ]) {
  let calls = 0;
  const read = readTransport(session, async () => { calls++; return make(); });
  await assert.rejects(read("result_authority.history", "workspace", {}), (e: unknown) => e instanceof AuthorityError && e.code === "session_unavailable");
  assert.equal(calls, 1);
 }
});

test("stalled session aborts before command dispatch", async t => {
 t.mock.timers.enable({ apis: ["setTimeout"] });
 let calls = 0, aborted = false;
 const read = readTransport(session, async (_, init) => {
  calls++; init?.signal?.addEventListener("abort", () => { aborted = true; });
  return new Promise<Response>(() => {});
 });
 const rejected = assert.rejects(read("result_authority.history", "workspace", {}), (e: unknown) => e instanceof AuthorityError && e.code === "session_unavailable");
 t.mock.timers.tick(30000);
 await rejected; assert.equal(calls, 1); assert.equal(aborted, true);
});

test("authority read freezes owner and fields before asynchronous session validation", async () => {
 const owner={...session}, fields={collection:"ceilings",capability:"apply"};
 let release!:(r:Response)=>void;
 const read=readTransport(owner,async(input,init)=>{
  if(String(input).endsWith("/session"))return new Promise<Response>(resolve=>{release=resolve;});
  const body=JSON.parse(String(init?.body));
  assert.deepEqual(body.body,{collection:"ceilings",capability:"apply"});
  assert.equal((init?.headers as Record<string,string>)["X-CSRF-Token"],session.csrf);
  return Response.json(response(body));
 });
 const pending=read("result_authority.context","workspace",fields);
 owner.user="changed";owner.csrf="changed";fields.capability="revoke";
 release(sessionResponse());await pending;
});

test("stalled authority response body aborts with no effect", async t => {
 t.mock.timers.enable({apis:["setTimeout"]});
 let began!:()=>void,aborted=false;
 const started=new Promise<void>(resolve=>{began=resolve;});
 const read=readTransport(session,async(input,init)=>{
  if(String(input).endsWith("/session"))return sessionResponse();
  init?.signal?.addEventListener("abort",()=>{aborted=true;});
  began();return new Response(new ReadableStream({start(){}}));
 });
 const rejected=assert.rejects(read("result_authority.history","workspace",{}),e=>e instanceof AuthorityError&&e.code==="request_timeout"&&e.effect==="none");
 await started;t.mock.timers.tick(30000);await rejected;assert.equal(aborted,true);
});

test("oversized streamed response is cancelled without publishing partial fields", async()=>{
 let cancelled=false;
 const read=readTransport(session,async input=>{
  if(String(input).endsWith("/session"))return sessionResponse();
  return new Response(new ReadableStream({pull(controller){controller.enqueue(new Uint8Array(50001).fill(32));},cancel(){cancelled=true;}}));
 });
 await assert.rejects(read("result_authority.history","workspace",{}),AuthorityError);
 assert.equal(cancelled,true);
});
