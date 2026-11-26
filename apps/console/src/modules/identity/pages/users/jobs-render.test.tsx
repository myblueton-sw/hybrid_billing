import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { BulkJob } from "../../../../service/identity-bulk";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { JobTable } = await import("./RevocationJobs");
const { Settings } = await import("./RolePeriod");

import { b } from "./bulk-copy";
const langs: Locale[] = ["en", "ko", "ar"];
const job: BulkJob = {
  id: "bulk-revocation-5", state: "executed", origin: "", effect: "suspend_membership", preparer: "sample-admin", prepared_at: "2026-11-19T09:00:00Z", expires_at: "2026-11-19T09:30:00Z",
  executor: "sample-admin", executed_at: "2026-11-19T09:05:00Z", deny_at: "", digest: "d", counts: { targets: 2, excluded: 0, duplicates: 0, succeeded: 2, held: 0, failed: 0, unprocessed: 0 },
  job_state: "succeeded", job_reason: "", ledger: { first_sequence: 1, last_sequence: 2, watermark: "w", reason: "" }, cleanup: { done: 2, total: 2 }, recorded_at: "", staged: 0,
  selection: "explicit", matched: 0, chosen: 0, fixed_at: "", fixed_revision: "", filter: null, search_masked: false,
};
const settings = [{ name: "bulk_revocation_max_targets", value: 500, unit: "", default: 500, min: 1, max: 5000, source: "default" }];
const count = (html: string, re: RegExp) => (html.match(re) ?? []).length;

function semantics(html: string, cols: number, rows: number, l: string) {
  assert.equal(count(html, /<table[^>]* role="table"/g), 1, l);
  assert.equal(count(html, /<(thead|tbody)[^>]* role="rowgroup"/g), 2, l);
  assert.equal(count(html, /<tr[^>]* role="row"/g), rows + 1, l);
  assert.equal(count(html, /<th[^>]* role="columnheader"/g), cols, l);
  assert.equal(count(html, /<td[^>]* role="cell"/g), cols * rows, l);
  assert.equal(count(html, /<t[rhd][ >]/g), count(html, /<t[rhd] [^>]*role="/g), l);
}

test("the jobs table keeps table roles so its stacked form is still read as a table", () => {
  for (const l of langs) semantics(renderToStaticMarkup(<JobTable l={l} jobs={[job]} open="" onOpen={() => {}} days={14} />), 4, 1, l);
});

test("the settings table keeps table roles so its stacked form is still read as a table", () => {
  for (const l of langs) semantics(renderToStaticMarkup(<Settings l={l} settings={settings} />), 5, 1, l);
});

test("a closed list that was never prepared is not shown with a count of people", () => {
  const zero = { targets: 0, excluded: 0, duplicates: 0, succeeded: 0, held: 0, failed: 0, unprocessed: 0 };
  const closed = (state: "discarded" | "expired", prepared_at: string, counts = zero): BulkJob => ({ ...job, id: "bulk-revocation-9", state, prepared_at, executor: "", executed_at: "", counts, job_state: "" });
  for (const l of langs) {
    for (const state of ["discarded", "expired"] as const) {
      const never = renderToStaticMarkup(<JobTable l={l} jobs={[closed(state, "")]} open="" onOpen={() => {}} days={14} />);
      assert.ok(never.includes(b.neverCounted(l)), l + state);
      assert.ok(!never.includes(b.people(l, 0)), l + state);
      const prepared = renderToStaticMarkup(<JobTable l={l} jobs={[closed(state, "2026-11-19T09:00:00Z", { ...zero, targets: 3 })]} open="" onOpen={() => {}} days={14} />);
      assert.ok(prepared.includes(b.people(l, 3)), l + state);
      assert.ok(!prepared.includes(b.neverCounted(l)), l + state);
    }
    const draft = renderToStaticMarkup(<JobTable l={l} jobs={[{ ...closed("discarded", ""), state: "draft" }]} open="" onOpen={() => {}} days={14} />);
    assert.ok(draft.includes(b.notCounted(l)) && !draft.includes(b.neverCounted(l)), l);
  }
});
