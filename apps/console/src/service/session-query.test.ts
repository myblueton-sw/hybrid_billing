import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError } from "./admin";

const session = { handle: "0123456789abcdef", current: false, signed_in: "2026-11-20T00:00:00Z", ends: "2026-11-20T12:00:00Z", state: "active", can_end: true };
const query = { session: "other", workspace: "w", since: "", until: "", q: "", limit: 2 };
const page = { rows: [session], next: "cursor", query, reference_at: "2026-11-20T01:00:00Z", generation: "1", order: "current_then_signed_in_desc_handle_asc", count: { value: 1, certainty: "lower_bound" }, scan_stopped: true, settings: [] };
const selection = { kind: "others", selection: "selection-a", state: "building", reference_at: "now", expires_at: "later", generation: "1", observed: 2, count: null, digest: "", after: "resume", settings: [] };
const summary = { operation: "session-end-1", selection: "selection-a", kind: "others", state: "unconfirmed", recorded_count: 2, ended_count: null, gone_count: null,
  not_included_count: null, blocked_at: null, list_generation: "1", current_generation: null, generation_changed: null, digest: "digest", cleanup: null, replay: false };
function client(body: unknown, inspect?: (path: string, init?: RequestInit) => void) {
  return adminClient("csrf", (async (path, init) => { inspect?.(String(path), init); return new Response(JSON.stringify(body), { status: 200 }); }) as typeof fetch);
}
const invalid = (error: unknown) => error instanceof AdminError && error.failure === "invalid";

test("session query preserves lower-bound counts and sends only requested bounded filters", async () => {
  const got = await client(page, (path) => {
    const url = new URL(path, "http://localhost");
    assert.equal(url.pathname, "/api/v1/me/sessions");
    assert.deepEqual([...url.searchParams], [["session", "other"], ["limit", "2"]]);
  }).sessionQuery({ session: "other", limit: 2 });
  assert.deepEqual(got, page);
  for (const patch of [{ rows: null }, { rows: [session, {}] }, { scan_stopped: "true" }, { count: { value: 0, certainty: "unknown" } }, { rows: [{ ...session, can_end: "true" }] }]) {
    await assert.rejects(client({ ...page, ...patch }).sessionQuery({}), invalid);
  }
});

test("selection building count stays unknown and cannot masquerade as sealed", async () => {
  assert.equal((await client(selection).prepareSessionSelection()).count, null);
  for (const patch of [{ count: 0 }, { observed: "2" }, { state: "sealed" }, { after: "" }, { state: "future" }]) {
    await assert.rejects(client({ ...selection, ...patch }).prepareSessionSelection(), invalid);
  }
  await client(selection, (_, init) => assert.deepEqual(JSON.parse(String(init?.body)), { selection: "selection-a", after: "resume" })).prepareSessionSelection({ selection: "selection-a", after: "resume" });
});

test("unconfirmed operation never coerces null outcomes into zero", async () => {
  assert.deepEqual(await client(summary).sessionOperation("session-end-1"), summary);
  for (const patch of [{ ended_count: 0 }, { cleanup: {} }, { current_generation: "" }, { state: "complete" }]) {
    await assert.rejects(client({ ...summary, ...patch }).sessionOperation("session-end-1"), invalid);
  }
});

test("selected termination sends only immutable reference fingerprint and count", async () => {
  await client(summary, (path, init) => {
    assert.equal(path, "/api/v1/me/sessions/end");
    assert.deepEqual(JSON.parse(String(init?.body)), { selection: "selection-a", digest: "digest", count: 2 });
  }).endSelectedSessions({ selection: "selection-a", digest: "digest", count: 2 });
});

test("account read consumes bounded current-session metadata without population arrays", async () => {
  const { selfFixture } = await import("./read-fixtures");
  const bounded = { ...selfFixture, recorded_session_count: 151, accepted_operation_count: 5 };
  const got = await client(bounded).me();
  assert.deepEqual(got, bounded);
  assert.equal(Object.hasOwn(got, "sessions"), false);
  assert.equal(Object.hasOwn(got, "session_ends"), false);
  for (const patch of [{ current_session: null }, { current_session: { ...bounded.current_session, can_end: "false" } }, { recorded_session_count: "151" }, { accepted_operation_count: -1 }]) await assert.rejects(client({ ...bounded, ...patch }).me(), invalid);
});
