import { test } from "node:test";
import assert from "node:assert/strict";
import { readSyntheticEvidence, latestRequest } from "./api";
const query = {
  scope: "sample-b",
  filter: "all",
  search: "",
  page: 1,
  scenario: "normal",
};
test("pages remain bounded and have disjoint IDs", async () => {
  const signal = new AbortController().signal;
  const a = await readSyntheticEvidence(query, signal),
    b = await readSyntheticEvidence({ ...query, page: 2 }, signal);
  assert.equal(a.total, 57);
  assert.equal(a.rows.length, 10);
  assert.equal(b.rows.length, 10);
  assert.equal(new Set([...a.rows, ...b.rows].map((r) => r.id)).size, 20);
});
test("missing filter has independent expected count", async () => {
  const r = await readSyntheticEvidence(
    { ...query, filter: "missing" },
    new AbortController().signal,
  );
  assert.equal(r.total, 15);
});
test("empty and error are distinct", async () => {
  assert.equal(
    (
      await readSyntheticEvidence(
        { ...query, scenario: "empty" },
        new AbortController().signal,
      )
    ).total,
    0,
  );
  await assert.rejects(
    readSyntheticEvidence(
      { ...query, scenario: "error" },
      new AbortController().signal,
    ),
  );
});
test("aborted requests reject and late response epochs are ineligible", async () => {
  const gate = latestRequest(),
    old = gate.next(),
    fresh = gate.next();
  assert.equal(gate.isCurrent(old), false);
  assert.equal(gate.isCurrent(fresh), true);
  gate.invalidate();
  assert.equal(gate.isCurrent(fresh), false);
  const c = new AbortController();
  c.abort();
  await assert.rejects(readSyntheticEvidence(query, c.signal), {
    name: "AbortError",
  });
});

test("provider, sort and 25-row pages preserve independent fixture counts", async () => {
  const signal = new AbortController().signal;
  const filtered = await readSyntheticEvidence(
    { ...query, provider: "AWS", size: 25 },
    signal,
  );
  assert.equal(filtered.total, 19);
  assert(filtered.rows.every((row) => row.provider === "AWS"));
  const descending = await readSyntheticEvidence(
    { ...query, sort: "desc", size: 25 },
    signal,
  );
  assert.equal(descending.rows.length, 25);
  assert.equal(descending.rows[0].id, "sample-b-REC-0000057");
  const last = await readSyntheticEvidence(
    { ...query, size: 25, page: 3 },
    signal,
  );
  assert.equal(last.rows.length, 7);
});
