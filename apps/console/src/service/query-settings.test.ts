import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError, type AccessSetting } from "./admin";
import { pageLimitOf, sized, sizeFor, sizeRefused } from "./query-settings";
import { permissionFilters, withoutStaleCursor } from "../modules/identity/pages/access/Permissions";
import { qcopy } from "../modules/identity/pages/service-accounts/query-copy";
import { readFailure } from "../modules/identity/pages/access/Permissions";

const row = (name: string, value: number): AccessSetting => ({ name, value, unit: "rows", default: 50, min: 1, max: 100, source: "default" });
const rows = (value: number) => [row("service_query_page_size", value), row("service_query_scan", 1000), row("service_query_lifetime_seconds", 300)];
const client = (body: unknown, seen?: (path: string) => void) =>
  adminClient("csrf", (async (path) => { seen?.(String(path)); return new Response(JSON.stringify(body)); }) as typeof fetch);

test("page limit: read from the list settings, unknown when absent or out of range", () => {
  assert.equal(pageLimitOf(rows(10)), 10);
  assert.equal(pageLimitOf(rows(100)), 100);
  assert.equal(pageLimitOf([]), null);
  assert.equal(pageLimitOf([row("service_query_scan", 10)]), null);
  for (const bad of [0, 101, 2.5, Number.NaN]) assert.equal(pageLimitOf(rows(bad)), null, String(bad));
});

test("page limit: a list asks for the smaller of its size and the limit, and for no size while the limit is unknown", () => {
  assert.equal(sizeFor(50, 100), 50);
  assert.equal(sizeFor(50, 50), 50);
  assert.equal(sizeFor(50, 10), 10);
  assert.equal(sizeFor(10, 25), 10);
  assert.equal(sizeFor(50, null), undefined);
  assert.deepEqual([sized(50, 10), sized(25, 100), sized(25, null)], [{ limit: 10 }, { limit: 25 }, {}]);
  const none = permissionFilters(new URLSearchParams(), sizeFor(50, null));
  const ten = permissionFilters(new URLSearchParams("acc_state=active"), sizeFor(50, 10));
  const usual = permissionFilters(new URLSearchParams(), sizeFor(50, 100));
  assert.ok(none.valid && ten.valid && usual.valid);
  assert.equal("limit" in none.query, false);
  assert.deepEqual([ten.query.limit, ten.query.state, usual.query.limit], [10, "active", 50]);
});

test("page limit: a refused size is told apart from a refused query and worded for what it is", () => {
  const size = new AdminError("invalid", ["limit"], 0, "", "invalid_request");
  const query = new AdminError("invalid", ["after"], 0, "", "invalid_request");
  const moved = new AdminError("conflict", ["after"], 0, "", "revision_conflict");
  assert.deepEqual([sizeRefused(size), sizeRefused(query), sizeRefused(moved), sizeRefused(new AdminError("invalid", ["limit"])), sizeRefused(new Error("x"))], [true, false, false, false, false]);
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(readFailure(l, size), qcopy(l, "readSizeRefused"));
    assert.equal(readFailure(l, query), qcopy(l, "readQueryChanged"));
    assert.equal(readFailure(l, moved), qcopy(l, "readQueryChanged"));
    assert.notEqual(qcopy(l, "readSizeRefused"), qcopy(l, "readQueryChanged"));
  }
});

test("page limit: both entry reads carry the list settings, and an answer without them reads as none", async () => {
  const policy = await client({ query_settings: rows(10) }).policy();
  assert.equal(pageLimitOf(policy.query_settings), 10);
  assert.deepEqual((await client({}).policy()).query_settings, []);
  const base = { revision: "r1", catalog: [], catalog_incomplete: false, now: "2026-11-27T09:00:00Z", you: "me", started_at: "2026-11-27T08:00:00Z", settings: [], ledger: { state: "ok" } };
  const access = await client({ ...base, query_settings: rows(40) }).access();
  assert.equal(pageLimitOf(access.query_settings), 40);
  assert.deepEqual((await client(base).access()).query_settings, []);
  await assert.rejects(client({ ...base, query_settings: [{ name: "service_query_page_size" }] }).access(), (e: AdminError) => e.failure === "invalid");
});

test("page limit: a cursor issued under another size is dropped, any other cursor is kept", () => {
  const at = (size: number | undefined) => permissionFilters(new URLSearchParams("acc_state=active&acc_after=c1"), size);
  const moved = withoutStaleCursor(at(50), { after: "c1", size: undefined }, 50);
  assert.ok(moved.stale && moved.filters.valid);
  if (moved.filters.valid) assert.deepEqual(moved.filters.query, { limit: 50, state: "active" });
  const lowered = withoutStaleCursor(at(10), { after: "c1", size: 50 }, 10);
  assert.ok(lowered.stale && lowered.filters.valid && !("after" in lowered.filters.query));
  for (const [name, issued, size] of [["the same size", { after: "c1", size: 50 }, 50], ["a cursor from the address", null, 50], ["another cursor", { after: "c0", size: 25 }, 50],
    ["both unknown", { after: "c1", size: undefined }, undefined]] as const) {
    const kept = withoutStaleCursor(at(size), issued, size);
    assert.ok(!kept.stale && kept.filters.valid && kept.filters.query.after === "c1", name);
  }
  const invalid = permissionFilters(new URLSearchParams("acc_nonsense=1"), 50);
  assert.deepEqual(withoutStaleCursor(invalid, { after: "c1", size: 10 }, 50), { filters: invalid, stale: false });
});

