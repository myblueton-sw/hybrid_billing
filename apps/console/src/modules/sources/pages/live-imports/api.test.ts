import { test } from "node:test";
import assert from "node:assert/strict";
import { request, ServiceError } from "./api";
test("unchanged server evidence excludes transient request IDs", async () => {
  const original = globalThis.fetch;
  let count = 0;
  try {
    globalThis.fetch = async () =>
      new Response(
        JSON.stringify({ workspace: "a", request_id: String(++count) }),
      );
    const signal = new AbortController().signal;
    assert.deepEqual(
      await request("/profile", signal),
      await request("/profile", signal),
    );
  } finally {
    globalThis.fetch = original;
  }
});
test("server failure retains request identity and does not become empty success", async () => {
  const original = globalThis.fetch;
  try {
    globalThis.fetch = async () =>
      new Response(
        JSON.stringify({ error: "scope_denied", request_id: "request-7" }),
        { status: 403 },
      );
    await assert.rejects(
      () => request("", new AbortController().signal),
      (e: unknown) =>
        e instanceof ServiceError &&
        e.code === "scope_denied" &&
        e.requestId === "request-7",
    );
  } finally {
    globalThis.fetch = original;
  }
});
