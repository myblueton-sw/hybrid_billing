import { test } from "node:test";
import assert from "node:assert/strict";
import {
  fixtureTargets,
  filterTargets,
  dateCounts,
  emptyFilter,
  isOverdue,
} from "./model";
test("closing targets preserve independent states, schedules and scope", () => {
  const rows = fixtureTargets("sample-a", "2026-10");
  assert.equal(rows.length, 1200);
  assert.equal(new Set(rows.map((r) => r.id)).size, 1200);
  for (const state of ["exempt", "complete", "held", "blocked"])
    assert.equal(rows.filter((r) => r.state === state).length, 240);
  assert(
    rows
      .filter((r) => r.state === "complete")
      .every((r) => r.erp !== "unknown"),
  );
  assert(
    rows.some(
      (r) => r.internal && r.erp === "unknown" && r.state === "blocked",
    ),
  );
  assert(rows.filter((r) => r.state === "exempt").every((r) => !isOverdue(r)));
  assert.equal(rows[0].original, "2026-11-01T09:00:00Z");
  assert.equal(rows[0].current, "2026-11-03T09:00:00Z");
  assert.notEqual(rows[0].id, fixtureTargets("sample-b", "2026-10")[0].id);
  assert.deepEqual(fixtureTargets("sample-a", "invalid"), []);
});
test("calendar totals cover the same complete filtered scope, not one page", () => {
  const rows = filterTargets(fixtureTargets("sample-a", "2026-10"), {
    ...emptyFilter,
    state: "blocked",
  });
  assert.equal(rows.length, 240);
  assert.equal(
    [...dateCounts(rows).values()].reduce((a, b) => a + b, 0),
    240,
  );
  assert.equal(
    filterTargets(rows, { ...emptyFilter, query: "missing" }).length,
    0,
  );
  assert.equal(
    filterTargets(fixtureTargets("sample-a", "2026-10"), {
      ...emptyFilter,
      query: "contract-001200",
    }).length,
    1,
  );
});

test("unknown contract path cannot broaden the selected scope", () => {
  assert.equal(
    filterTargets(fixtureTargets("sample-a", "2026-10"), {
      ...emptyFilter,
      path: "unsupported",
    }).length,
    0,
  );
});
