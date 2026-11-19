import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import { AdminError, adminClient, failureOf } from "./admin";
import { refusalDetail, Refusals } from "./refusal";
import { RefusalDetail } from "./RefusalDetail";

const tried = { request: "client-request", key: "client-key" };
const command = (f: typeof fetch, stepUp?: () => Promise<boolean>) => adminClient("csrf", f, stepUp).command("integrations.plugin_stop", "plugin:p", "r1", "reason", {}, tried);
const reply = (status: number, body: unknown, headers = {}) => new Response(JSON.stringify(body), { status, headers });

test("the server response identity survives throttling, independent of the attempted identity", async () => {
  const f = (async () => reply(429, { code: "rate_limited", request_id: "server-request" }, { "Retry-After": "9" })) as typeof fetch;
  await assert.rejects(command(f), (err: unknown) => {
    assert.ok(err instanceof AdminError);
    assert.equal(err.retryAfter, 9);
    assert.deepEqual(refusalDetail(err), { code: "rate_limited", request: "server-request" });
    return true;
  });
});

test("step-up uses the final command response and cancellation has no server diagnostics", async () => {
  let calls = 0;
  const f = (async () => ++calls === 1
    ? reply(422, { code: "reauthentication_required", request_id: "challenge" })
    : reply(409, { code: "revision_conflict", request_id: "final-response" })) as typeof fetch;
  await assert.rejects(command(f, async () => true), (err: unknown) => {
    assert.deepEqual(refusalDetail(err), { code: "revision_conflict", request: "final-response" });
    return true;
  });
  calls = 0;
  await assert.rejects(command(f, async () => false), (err: unknown) => {
    assert.equal(refusalDetail(err), undefined);
    return true;
  });
  assert.equal(calls, 1);
});

test("missing, malformed and network metadata cannot fabricate server details", async () => {
  for (const body of [{}, { code: {}, request_id: [] }, { code: "x\nsecret", request_id: "id\u202e" }, { code: "x".repeat(129), request_id: "x".repeat(129) }]) {
    assert.equal(refusalDetail(failureOf(503, body)), undefined);
  }
  assert.deepEqual(refusalDetail(failureOf(409, { code: "future_code", request_id: "server" })), { code: "future_code", request: "server" });
  assert.deepEqual(refusalDetail(failureOf(500, { code: "internal_error", request_id: "failed-server" })), { code: "internal_error", request: "failed-server" });
  await assert.rejects(command((async () => { throw new Error("offline"); }) as typeof fetch), (err: unknown) => {
    assert.equal(refusalDetail(err), undefined);
    return true;
  });
});

test("retained details cannot survive replacing or deleting their own refusal", () => {
  const store = new Refusals<string>();
  const detail = { code: "invalid_stage", request: "r1" };
  store.set("one", "stage", detail);
  detail.request = "changed";
  assert.equal(store.detail("one")?.request, "r1");
  const got = store.detail("one")!;
  got.request = "mutated";
  assert.equal(store.detail("one")?.request, "r1");
  store.set("one", "stage");
  assert.equal(store.detail("one"), undefined);
  store.set("two", "stage", detail);
  store.delete("two");
  assert.equal(store.detail("two"), undefined);
  store.set("three", "stage", detail);
  store.clear();
  assert.deepEqual([...store], []);
});

test("diagnostic values stay LTR in all three locales and missing details stay absent", () => {
  for (const locale of ["en", "ko", "ar"] as const) {
    const html = renderToStaticMarkup(<RefusalDetail locale={locale} detail={{ code: "invalid_stage", request: "server-id" }} />);
    assert.match(html, /<bdi dir="ltr">invalid_stage/);
    assert.match(html, /<bdi dir="ltr">server-id/);
    assert.ok(html.includes(locale === "ko" ? "요청 ID" : locale === "ar" ? "معرّف الطلب" : "Request ID"));
    assert.equal(renderToStaticMarkup(<RefusalDetail locale={locale} />), "");
  }
});
