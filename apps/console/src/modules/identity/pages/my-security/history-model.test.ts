import { test } from "node:test";
import assert from "node:assert/strict";
import type { AccessHistoryPage } from "../../../../service/access-history";
import { accessHistoryOf, HistoryParseError } from "../../../../service/access-history";
import { coverageGap, fromFor, historyState, noteLines, showSizeLimit } from "./history-model";

function page(extra: Partial<AccessHistoryPage> = {}): AccessHistoryPage {
  return {
    events: [], next: null, retention_days: 90, retention_source: "default", period_since: null,
    effective_days: 90, available_from: null, recording: { state: "recording", since: null, reason: null },
    notes: [], now: "2026-11-19T12:00:00Z", ...extra,
  };
}

test("the history period uses server time and does not invent a browser-time boundary", () => {
  assert.equal(fromFor("all", ""), undefined);
  assert.equal(fromFor("week", ""), undefined);
  assert.equal(fromFor("week", "not-a-time"), undefined);
  assert.equal(fromFor("week", "2026-11-19T12:00:00Z"), "2026-11-12T12:00:00Z");
  assert.equal(coverageGap("2026-11-01T00:00:00Z", "2026-11-10T00:00:00Z"), true);
  assert.equal(coverageGap(undefined, "2026-11-10T00:00:00Z"), false);
});

test("failed, unrecorded and empty periods stay distinct even when previous rows exist", () => {
  assert.equal(historyState(null, false), "loading");
  assert.equal(historyState(page(), true), "failed");
  assert.equal(historyState(page(), false), "none-kept");
  assert.equal(historyState(page({ available_from: "2026-11-01T00:00:00Z" }), false), "empty");
  assert.equal(showSizeLimit(page({ effective_days: 10 })), true);
  assert.equal(showSizeLimit(page()), false);
});

test("unknown loss counts stay unknown and stopped recording takes precedence in every locale", () => {
  const p = page({
    recording: { state: "stopped", since: "2026-11-19T10:00:00Z", reason: "write_failed" },
    notes: [
      { kind: "dropped", from: null, to: null, count: null },
      { kind: "dropped", from: null, to: null, count: 0 },
      { kind: "purge_deferred", from: null, to: null, count: null },
    ],
  });
  for (const l of ["en", "ko", "ar"] as const) {
    const lines = noteLines(l, p, (iso) => iso);
    assert.equal(lines.length, 3);
    assert.equal(lines[0].key, "recording");
    assert.equal(lines[0].strong, true);
    assert.equal(lines[1].key, "purge_deferred-2");
    assert.match(lines[2].key, /^dropped/);
    assert.ok(lines.every((line) => line.text.length > 0));
  }
});

test("the history parser refuses unknown event meanings and preserves unknown note counts", () => {
  const event = { at: "2026-11-19T11:00:00Z", kind: "signed_in", reason: "", workspace: "w", session: null, device: null, count: 0 };
  assert.equal(accessHistoryOf(page({ notes: [{ kind: "dropped", from: null, to: null, count: null }] })).notes[0].count, null);
  assert.throws(() => accessHistoryOf({ ...page(), events: [{ ...event, kind: "some-new-event" }] }), HistoryParseError);
  assert.throws(() => accessHistoryOf({ ...page(), events: [{ ...event, reason: "unknown-reason" }] }), HistoryParseError);
  assert.throws(() => accessHistoryOf({ ...page(), recording: { state: "ok" } }), HistoryParseError);
  assert.throws(() => accessHistoryOf({ ...page(), events: [{ ...event, count: -1 }] }), HistoryParseError);
});
