import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import { detailOf, jobOf } from "../../../../service/identity-bulk";
import { b } from "./bulk-copy";
import { m } from "./match-copy";
import { fixedOfFields } from "./match-model";
import { noFilters } from "./filter-model";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { FixedBar, MatchAction, BasisLine, FixedFacts, BasisFacts, fillAll } = await import("./Matching");
const { PreviewBody } = await import("./BulkPrepare");
const { SelectionBar } = await import("./Selection");
const { Settings } = await import("./RolePeriod");

const api = {} as AdminClient;
const langs: Locale[] = ["en", "ko", "ar"];
const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/'/g, "&#x27;").replace(/"/g, "&quot;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
const fixed = fixedOfFields({ draft: "bulk-draft-3", matched: "12", fixed_at: "2026-11-19T10:00:00Z", fixed_revision: "r77", "filter.role": "member", "filter.q": "zq-secret" }, 11);

test("the fixed bar shows the fixed line, read-only chips and no checkbox", () => {
  for (const l of langs) {
    const html = renderToStaticMarkup(<FixedBar l={l} f={fixed} filters={{ ...noFilters, role: "member", q: "zq-secret" }} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />);
    assert.ok(!html.includes("type=\"checkbox\""), l);
    assert.ok(html.includes("m01-chip m01-chip-fixed") && !html.includes("<button type=\"button\" id=\"m01-chip"), l);
    assert.ok(html.includes(esc(m.fixedNoChecks(l))) && html.includes(esc(m.discardList(l))) && html.includes(esc(b.prepare(l))), l);
    assert.ok(html.includes(">12<") && html.includes("r77") && html.includes(esc(m.baseRevision(l))), l);
    assert.ok(!html.includes(esc(m.keepsOwn(l))), l);
    const changed = renderToStaticMarkup(<FixedBar l={l} f={fixed} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />);
    assert.ok(changed.includes(esc(m.keepsOwn(l))), l);
  }
  const counted = renderToStaticMarkup(<FixedBar l="en" f={{ ...fixed, counted: 12 }} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />);
  assert.ok(counted.includes("for these filters") && !counted.includes("when counted"));
  assert.ok(renderToStaticMarkup(<FixedBar l="en" f={fixed} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />).includes("(11 when counted)"));
  const everyone = renderToStaticMarkup(<FixedBar l="en" f={fixedOfFields({ matched: "4", fixed_at: "2026-11-19T10:00:00Z" }, null)} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />);
  assert.ok(everyone.includes("everyone in the list") && !everyone.includes("m01-fixedfilters"));
});

test("the select-matching trigger names the filtered and the unfiltered case", () => {
  for (const l of langs) {
    const filtered = renderToStaticMarkup(<MatchAction l={l} api={api} query={{ role: "member" }} filters={{ ...noFilters, role: "member" }} picked={0} pendingLimit={20} reload={async () => ""} onFixed={() => {}} onJobs={() => {}} say={() => {}} />);
    assert.ok(filtered.includes(esc(m.selectMatching(l))) && filtered.includes("aria-expanded=\"false\"") && !filtered.includes("m01-match\""), l);
    const all = renderToStaticMarkup(<MatchAction l={l} api={api} query={{}} filters={noFilters} picked={0} pendingLimit={20} reload={async () => ""} onFixed={() => {}} onJobs={() => {}} say={() => {}} />);
    assert.ok(all.includes(esc(m.selectAll(l))), l);
  }
});

test("the selection bar has no filter-wide note and offers the action after the count", () => {
  const plain = renderToStaticMarkup(<SelectionBar l="en" selected={new Set()} loaded={["u1"]} limit={500} atLimit={false} names={{}} onPrepare={() => {}} onClear={() => {}} onRemove={() => {}} onAll={() => {}} />);
  assert.ok(!plain.includes("not available yet") && plain.includes(esc(b.none("en"))));
  const withMatch = renderToStaticMarkup(<SelectionBar l="en" selected={new Set(["u1"])} loaded={["u1"]} limit={500} atLimit={false} names={{}} match={<button className="m01-match-open">x</button>}
    onPrepare={() => {}} onClear={() => {}} onRemove={() => {}} onAll={() => {}} />);
  assert.ok(withMatch.indexOf("m01-selactions") < withMatch.indexOf("m01-match-open"));
  const none = renderToStaticMarkup(<SelectionBar l="ko" selected={new Set()} loaded={["u1"]} limit={500} atLimit={false} names={{}} match={<span />} onPrepare={() => {}} onClear={() => {}} onRemove={() => {}} onAll={() => {}} />);
  assert.ok(none.includes(esc(m.none("ko"))));
});

const filterJob = (o: Record<string, unknown> = {}) => jobOf({ id: "bulk-revocation-5", state: "prepared", effect: "suspend_membership", preparer: "me", selection: "filter", matched: 10,
  fixed_at: "2026-11-19T10:00:00Z", fixed_revision: "r12", filter: { role: "member", q: "zq-secret" }, counts: { targets: 7, excluded: 3 }, ...o });

test("job rows state the basis, and the search text never reaches another administrator", () => {
  for (const l of langs) {
    const masked = renderToStaticMarkup(<BasisLine l={l} job={filterJob({ filter: { role: "member", q: "" }, search_masked: true })} days={14} />);
    assert.ok(masked.includes(esc(m.searchApplied(l))) && !masked.includes("zq-secret") && masked.includes(esc(m.basis(l, "filter"))), l);
    const own = renderToStaticMarkup(<BasisLine l={l} job={filterJob()} days={14} />);
    assert.ok(own.includes("zq-secret") && own.includes("dir=\"auto\""), l);
    assert.ok(renderToStaticMarkup(<BasisLine l={l} job={jobOf({ selection: "not_recorded" })} days={14} />).includes(esc(m.basis(l, "not_recorded"))), l);
    assert.ok(renderToStaticMarkup(<BasisLine l={l} job={jobOf({})} days={14} />).includes(esc(m.basis(l, "explicit"))), l);
    assert.ok(renderToStaticMarkup(<BasisLine l={l} job={filterJob({ filter: {} })} days={14} />).includes(esc(m.everyoneInList(l))), l);
    const detail = renderToStaticMarkup(<BasisFacts l={l} job={filterJob({ filter: { role: "member", q: "" }, search_masked: true })} days={14} />);
    assert.ok(detail.includes(esc(m.searchApplied(l))) && !detail.includes("zq-secret") && !detail.includes("title=") && detail.includes("m01-chip-fixed"), l);
    assert.ok(renderToStaticMarkup(<BasisFacts l={l} job={jobOf({})} days={14} />).includes(esc(m.basisL(l))), l);
  }
});

test("the filter-list preview states the fixed facts; the explicit preview keeps four reasons", () => {
  for (const l of langs) {
    const detail = detailOf({ job: { id: "bulk-revocation-5", state: "prepared", effect: "suspend_membership", preparer: "me", selection: "filter", matched: 10, fixed_at: "2026-11-19T10:00:00Z",
      fixed_revision: "r12", filter: { role: "member" }, counts: { targets: 7, excluded: 3 } }, impact: {}, exclusions: { self: 1, no_longer_matches: 2 }, outside_scope: [] });
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes("m01-fixedfacts") && html.includes(esc(m.arithmetic(l, 10, 7, 3))) && html.includes(esc(m.noLongerMatch(l, 2))) && html.includes(esc(b.reason(l, "no_longer_matches"))), l);
    assert.ok(html.indexOf(esc(b.reason(l, "not_found"))) < html.indexOf(esc(b.reason(l, "no_longer_matches"))) && html.indexOf(esc(b.reason(l, "no_longer_matches"))) < html.indexOf(esc(b.reason(l, "no_membership"))), l);
    const explicit = detailOf({ job: { id: "bulk-revocation-6", state: "prepared", effect: "suspend_membership", preparer: "me", counts: { targets: 2 } }, impact: {}, exclusions: {}, outside_scope: [] });
    const plain = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: explicit, targets: null }} onMore={() => {}} />);
    assert.ok(!plain.includes("m01-fixedfacts") && !plain.includes(esc(b.reason(l, "no_longer_matches"))), l);
  }
  const quiet = renderToStaticMarkup(<FixedFacts l="en" job={filterJob()} days={14} exclusions={{}} />);
  assert.ok(!quiet.includes("no longer match"));
});

test("settings show the new rows with hints and keep long numbers whole", () => {
  for (const l of langs) {
    const html = renderToStaticMarkup(<Settings l={l} settings={[{ name: "bulk_revocation_retained_people_max", value: 2000000, unit: "people", default: 2000000, min: 10000, max: 20000000, source: "default" },
      { name: "user_count_scan", value: 20000, unit: "people", default: 20000, min: 1000, max: 100000, source: "default" },
      { name: "user_filter_scan", value: 1000, unit: "people", default: 1000, min: 100, max: 10000, source: "default" },
      { name: "bulk_revocation_max_matching", value: 5000, unit: "people", default: 5000, min: 1, max: 100000, source: "start_flag" }]} />);
    for (const name of ["bulk_revocation_retained_people_max", "user_count_scan", "user_filter_scan", "bulk_revocation_max_matching"]) assert.ok(html.includes(esc(b.settingName(l, name))), l + name);
    for (const name of ["bulk_revocation_retained_people_max", "user_count_scan", "user_filter_scan"]) assert.ok(html.includes(esc(m.settingHint(l, name))), l + name);
    assert.ok(!/pending/i.test(m.settingHint(l, "bulk_revocation_retained_people_max")));
  }
});

test("tokens are filled in order", () => {
  assert.equal(renderToStaticMarkup(<p>{fillAll("{at} a {n} b {at}", { "{n}": "N", "{at}": "T" })}</p>), "<p>T<span>T</span> a <span>N</span> b <span>T</span></p>".replace("T<span>T</span>", "<span>T</span>"));
});

test("a fixed number below the counted lower bound says the count was a lower bound", () => {
  const below = { ...fixed, matched: 9, counted: 11, countedAtLeast: true };
  const html = renderToStaticMarkup(<FixedBar l="en" f={below} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />);
  assert.ok(html.includes("(at least 11 when counted)"), html);
  for (const l of ["ko", "ar"] as Locale[]) assert.ok(renderToStaticMarkup(<FixedBar l={l} f={below} filters={noFilters} days={14} number="3" error={null} onPrepare={() => {}} onDiscard={() => {}} say={() => {}} />).includes("11"), l);
});
