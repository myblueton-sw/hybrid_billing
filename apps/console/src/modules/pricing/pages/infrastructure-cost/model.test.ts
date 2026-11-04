import { test } from "node:test";
import assert from "node:assert/strict";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { issues, preview, removeEntry, type Draft } from "./model";
import { context, save, restore, clear, storageKey } from "./store";
const id = (n: number) => n.toString(16).padStart(64, "0");
function row(n: number, value: string | null): FocusRow {
  return {
    id: id(n),
    currency: "USD",
    billed_cost: value,
    effective_cost: value,
    charge_month: "2024-09",
    charge_start: "2024-09-01",
    charge_end: "2024-10-01",
  } as FocusRow;
}
function source(
  rows = [row(1, "1000000000000000001"), row(2, "-1"), row(3, null)],
): OverviewData {
  return {
    company: "sample",
    snapshot: id(10),
    source_snapshot: id(11),
    selection: {
      period: "2024-09",
      provider: "VMware",
      account: "",
      q: "",
      basis: "billed_cost",
    },
    rows,
    after: 0,
    account_after: 0,
  } as OverviewData;
}
function draft(): Draft {
  return {
    name: "Review",
    entries: [1, 2, 3].map((n) => ({
      row: id(n),
      category: "unclassified",
      parent: null,
      reference: "",
    })),
  };
}
function memory() {
  const map = new Map<string, string>();
  return {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => {
      map.set(k, v);
    },
    removeItem: (k: string) => {
      map.delete(k);
    },
  };
}
test("exact signed subtotal retains partial, null and cancellation", () => {
  assert.deepEqual(preview(draft(), source()).gross, {
    coefficient: "1000000000000000000",
    present: 2,
    missing: 1,
  });
  assert.deepEqual(
    preview(draft(), source([row(1, null), row(2, null), row(3, null)])).gross,
    { coefficient: null, present: 0, missing: 3 },
  );
  const d = draft();
  d.entries.pop();
  assert.equal(
    preview(d, source([row(1, "1"), row(2, "-1")])).gross?.coefficient,
    "0",
  );
});
test("whole-row chains exclude each descendant once and preserve credits", () => {
  const d = draft();
  d.entries[1].parent = id(1);
  d.entries[1].reference = "contract";
  d.entries[2].parent = id(2);
  d.entries[2].reference = "contract";
  const p = preview(d, source());
  assert.equal(p.review?.coefficient, "1000000000000000001");
  assert.deepEqual(p.excluded, { coefficient: "-1", present: 1, missing: 1 });
  const removed = removeEntry(d, id(2));
  assert.equal(removed.entries[1].parent, null);
  assert.equal(removed.entries[1].reference, "");
});
test("invalid graph and duplicate selection cannot produce review total", () => {
  for (const kind of ["duplicate", "self", "foreign", "cycle", "reference"]) {
    const d = draft();
    if (kind === "duplicate") d.entries.push({ ...d.entries[0] });
    if (kind === "self") d.entries[0].parent = id(1);
    if (kind === "foreign") d.entries[0].parent = id(99);
    if (kind === "reference") d.entries[0].parent = id(2);
    if (kind === "cycle") {
      d.entries[0].parent = id(2);
      d.entries[1].parent = id(1);
      d.entries.forEach((e) => (e.reference = "ref"));
    }
    assert.ok(issues(d, source()).length, kind);
    assert.equal(preview(d, source()).review, null, kind);
    if (kind === "duplicate")
      assert.equal(
        preview(d, source()).gross?.coefficient,
        "1000000000000000000",
      );
  }
});
test("currency and exact intervals cannot be silently reconciled", () => {
  const d = draft();
  d.entries[1].parent = id(1);
  d.entries[1].reference = "ref";
  for (const change of [
    { currency: "CNY" },
    { charge_month: "2024-10" },
    { charge_start: "2024-09-02" },
    { charge_end: "2024-09-30" },
  ]) {
    const s = source();
    Object.assign(s.rows[1], change);
    assert.ok(issues(d, s).length);
    assert.equal(preview(d, s).review, null);
  }
});
test("save requires bounded named nonempty specific-month selection", () => {
  const d = draft();
  assert.deepEqual(issues(d, source(), true), []);
  for (const bad of [
    { ...d, name: " " },
    { ...d, name: "x".repeat(121) },
    { ...d, entries: [] },
    { ...d, entries: Array.from({ length: 26 }, () => d.entries[0]) },
  ])
    assert.ok(issues(bad, source(), true).length);
  const s = source();
  s.selection.period = "all";
  assert.ok(issues(d, s, true).includes("month_required"));
});
test("metadata only restore and scoped query fingerprint", async () => {
  const s = source(),
    c = await context(s),
    m = memory(),
    d = draft();
  save(m, c, d, s);
  assert.deepEqual(restore(m, c, s), d);
  const raw = m.getItem(storageKey(c))!;
  assert.ok(!raw.includes("1000000000000000001"));
  assert.ok(!raw.includes("billed_cost"));
  const q = source();
  q.selection.q = "private-account-search";
  const qc = await context(q);
  save(m, qc, d, q);
  assert.ok(!m.getItem(storageKey(qc))!.includes(q.selection.q));
  clear(m, c);
  assert.equal(restore(m, c, s), null);
});
test("stale, foreign and malformed saved data fail closed", async () => {
  const s = source(),
    c = await context(s),
    d = draft();
  const changes: ((b: any) => void)[] = [
    (b) => (b.context.source = id(55)),
    (b) => (b.context.selection = id(56)),
    (b) => (b.context.company = "foreign"),
    (b) => (b.context.after = 25),
    (b) => (b.context.accountAfter = 25),
    (b) => (b.context.query = id(99)),
    (b) => (b.version = 2),
    (b) => (b.draft.extra = "x"),
    (b) => (b.draft.entries[0].row = id(99)),
    (b) => (b.draft.entries[0].parent = [id(2), id(3)]),
    (b) => (b.draft.entries[0].category = "invented"),
    (b) => (b.draft.entries[0].reference = "x".repeat(301)),
  ];
  for (const mutate of changes) {
    const m = memory();
    save(m, c, d, s);
    const b = JSON.parse(m.getItem(storageKey(c))!);
    mutate(b);
    m.setItem(storageKey(c), JSON.stringify(b));
    assert.throws(() => restore(m, c, s));
  }
  for (const text of ["{", "x".repeat(32769)]) {
    const m = memory();
    m.setItem(storageKey(c), text);
    assert.throws(() => restore(m, c, s));
  }
  const changed = source();
  changed.selection.basis = "effective_cost";
  assert.notEqual((await context(changed)).query, c.query);
});
test("write and deletion readback failures remain visible failures", async () => {
  const s = source(),
    c = await context(s),
    d = draft();
  assert.throws(() =>
    save(
      { getItem: () => null, setItem: () => {}, removeItem: () => {} },
      c,
      d,
      s,
    ),
  );
  assert.throws(() =>
    save(
      {
        getItem: () => null,
        setItem: () => {
          throw Error("quota");
        },
        removeItem: () => {},
      },
      c,
      d,
      s,
    ),
  );
  assert.throws(() =>
    clear(
      { getItem: () => "not removed", setItem: () => {}, removeItem: () => {} },
      c,
    ),
  );
});

test("mixed currencies never emit a single currency or gross sum", () => {
  const s = source();
  s.rows[1].currency = "CNY";
  const p = preview(draft(), s);
  assert.equal(p.gross, null);
  assert.equal(p.currency, null);
});
test("save projects only allowed fields and rejects context-data mismatch", async () => {
  const s = source(),
    c = await context(s),
    m = memory();
  const d = { ...draft(), amount: "private-amount" };
  save(m, { ...c, secret: "private-secret" } as typeof c, d, s);
  const raw = m.getItem(storageKey(c))!;
  assert.ok(!raw.includes("private-"));
  for (const field of ["source_snapshot", "snapshot", "company"] as const) {
    const changed = { ...s, [field]: "changed" };
    assert.throws(() => save(m, c, d, changed));
    assert.throws(() => restore(m, c, changed));
  }
});
