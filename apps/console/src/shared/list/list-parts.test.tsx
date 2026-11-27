import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import type { AccessSetting } from "../../service/admin";
import { qcopy } from "../../modules/identity/pages/service-accounts/query-copy";
import { chipsOf, permissionFilters, requestedSize } from "../../modules/identity/pages/access/Permissions";
import { lc } from "./list-copy";
import { lowered, nextFocusAfterRemoval, pageSizeOptions, sizeSelectShown } from "./list-model";
import { AppliedFilters, ListPager, ListSettings, ListSummary } from "./ListParts";

const noop = () => undefined;
const langs = ["en", "ko", "ar"] as const;

test("list model: page size options are bounded by the limit and keep the size in use", () => {
  assert.deepEqual(pageSizeOptions(50), [10, 25, 50]);
  assert.deepEqual(pageSizeOptions(100), [10, 25, 50, 100]);
  assert.deepEqual(pageSizeOptions(40), [10, 25, 40]);
  assert.deepEqual(pageSizeOptions(5), [5]);
  assert.deepEqual(pageSizeOptions(null), []);
  assert.deepEqual(pageSizeOptions(50, 30), [10, 25, 30, 50]);
  assert.deepEqual(pageSizeOptions(50, 70), [10, 25, 50], "a size above the limit is not offered");
  assert.deepEqual([sizeSelectShown(5), sizeSelectShown(null), sizeSelectShown(10), sizeSelectShown(25)], [false, false, false, true]);
  assert.deepEqual([lowered(50, 10), lowered(10, 10), lowered(50, null)], [true, false, false]);
  assert.deepEqual([nextFocusAfterRemoval(["a", "b", "c"], "b"), nextFocusAfterRemoval(["a", "b", "c"], "c"), nextFocusAfterRemoval(["a"], "a"), nextFocusAfterRemoval(["a"], "x")], ["c", "b", null, null]);
});

test("list copy: the pager and summary words match the screens' existing query wording", () => {
  for (const l of langs) for (const k of ["exact", "lower", "reference", "size", "first", "previous", "next"] as const) assert.equal(lc[k](l), qcopy(l, k), `${l} ${k}`);
});

test("list parts: the summary states the count and its certainty and announces it", () => {
  for (const l of langs) {
    const exact = renderToStaticMarkup(<ListSummary l={l} count="240" exact reference="t" />);
    const lower = renderToStaticMarkup(<ListSummary l={l} count="25" exact={false} reference="t" />);
    assert.ok(exact.includes(`${qcopy(l, "exact")}: 240`) && exact.includes('data-certainty="exact"') && exact.includes('role="status"') && exact.includes(`${qcopy(l, "reference")}: t`));
    assert.ok(lower.includes(`${qcopy(l, "lower")}: 25`) && lower.includes('data-certainty="lower_bound"'));
  }
});

test("list parts: the pager shows paging only beyond one page and the size choice only with a choice", () => {
  const one = renderToStaticMarkup(<ListPager l="en" title="Grants" first hasNext={false} rows={3} onFirst={noop} onPrevious={noop} onNext={noop} />);
  assert.equal(one, "", "a single page with no size choice shows no pager");
  const paged = renderToStaticMarkup(<ListPager l="en" title="Grants" first hasNext rows={25} onFirst={noop} onPrevious={noop} onNext={noop} />);
  assert.ok(paged.includes('aria-label="Grants, pages"') && paged.split("disabled").length === 3 && paged.includes(lc.onPage("en", 25)) && !paged.includes("<select"));
  const size = { id: "s", requested: 50, current: 50, limit: 100, onChange: noop };
  const chosen = renderToStaticMarkup(<ListPager l="ko" title="권한" first hasNext={false} rows={3} size={size} onFirst={noop} onPrevious={noop} onNext={noop} />);
  assert.ok(chosen.includes('<select id="s"') && chosen.includes(`>${qcopy("ko", "size")}</label>`) && !chosen.includes("<nav") && (chosen.match(/<option /g) ?? []).length === 4);
  const capped = renderToStaticMarkup(<ListPager l="ar" title="x" first hasNext={false} rows={5} size={{ ...size, current: 5, limit: 5 }} onFirst={noop} onPrevious={noop} onNext={noop} />);
  assert.ok(capped.includes(lc.lowered("ar", 5)) && !capped.includes("<select"), "a lowered size is stated even without a choice");
  const unknown = renderToStaticMarkup(<ListPager l="en" title="x" first={false} hasNext rows={25} size={{ ...size, limit: null }} onFirst={noop} onPrevious={noop} onNext={noop} />);
  assert.ok(!unknown.includes("<select") && !unknown.includes(lc.lowered("en", 50)) && unknown.includes("<nav"));
});

test("list parts: chips state each applied filter with a named remove control and a clear-all", () => {
  for (const l of langs) {
    const out = renderToStaticMarkup(<AppliedFilters l={l} id="f" chips={[{ key: "state", label: "State", value: "Active" }, { key: "subject", label: "Holder", value: "sample-member", mono: true }]}
      onRemove={noop} onClear={noop} fallback={() => null} />);
    assert.ok(out.includes('aria-labelledby="f"') && out.includes(`>${lc.applied(l)}</p>`) && out.includes(`aria-label="${lc.remove(l)} State: Active"`) && out.includes(`>${lc.clear(l)}</button>`));
    assert.ok(out.includes('<bdi dir="ltr" class="live-mono">sample-member</bdi>') && (out.match(/list-chip-remove/g) ?? []).length === 2);
  }
  assert.equal(renderToStaticMarkup(<AppliedFilters l="en" id="f" chips={[]} onRemove={noop} onClear={noop} fallback={() => null} />), "");
});

test("list parts: the list settings name the three settings with their units and source", () => {
  const row = (name: string, value: number, unit: string, source = "default"): AccessSetting => ({ name, value, unit, default: value, min: 1, max: 100, source });
  const rows = [row("service_query_page_size", 10, "rows", "start_flag"), row("service_query_scan", 1000, "work units"), row("service_query_lifetime_seconds", 300, "seconds")];
  for (const l of langs) {
    const out = renderToStaticMarkup(<ListSettings l={l} rows={rows} />);
    assert.ok(out.includes(lc.settingsTitle(l)) && out.includes(lc.name.service_query_page_size(l)) && out.includes(lc.name.service_query_scan(l)) && out.includes(lc.name.service_query_lifetime_seconds(l)));
    assert.ok(out.includes(lc.source(l, "start_flag")) && out.includes(lc.source(l, "default")) && out.includes('class="live-strong"'));
  }
  const en = renderToStaticMarkup(<ListSettings l="en" rows={rows} />);
  assert.ok(en.includes(">10 rows<") && en.includes(">1000 records<") && en.includes(">300 seconds<"));
  assert.equal(renderToStaticMarkup(<ListSettings l="en" rows={[]} />), "");
});

test("permissions: the rows per page live in the address and the chips come from the applied filters", () => {
  assert.equal(requestedSize(new URLSearchParams()), 50);
  assert.equal(requestedSize(new URLSearchParams("acc_limit=25")), 25);
  for (const bad of ["0", "101", "07", "2.5", "x"]) assert.equal(requestedSize(new URLSearchParams("acc_limit=" + bad)), null, bad);
  const sized = permissionFilters(new URLSearchParams("acc_limit=25&acc_state=active"), 25);
  assert.ok(sized.valid && sized.applied === 1 && sized.query.limit === 25);
  assert.equal(permissionFilters(new URLSearchParams("acc_limit=101"), 50).valid, false);
  const applied = permissionFilters(new URLSearchParams("acc_state=active&acc_subject=sample-member&acc_source=delegation&acc_before=2026-12-31"), 50);
  assert.ok(applied.valid);
  if (applied.valid) {
    const chips = chipsOf("en", applied.draft, []);
    assert.deepEqual(chips.map((c) => c.key), ["subject", "source", "state", "before"]);
    assert.deepEqual(chips.map((c) => !!c.mono), [true, false, false, true]);
    assert.ok(chips.every((c) => c.value !== "" && c.label !== ""));
  }
});
