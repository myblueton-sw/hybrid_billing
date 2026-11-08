import { test } from "node:test";
import assert from "node:assert/strict";
import { readRecords, saveRecord, type ReviewRecord } from "./review-store";
const record: ReviewRecord = {
  screen: "index-intake",
  variants: {
    locale: "en",
    width: 1280,
    profile: "standard",
    mode: "synthetic",
    extensions: [],
    state: { phase: "ready" },
  },
  revision: "a".repeat(64),
  scope: "b".repeat(64),
  items: ["index-intake:table:records"],
  confirmedAt: "2026-11-01T00:00:00Z",
  evidence: "local-owner-acknowledgement",
};
function memory() {
  let value: string | null = null;
  return {
    getItem: () => value,
    setItem: (_k: string, v: string) => {
      value = v;
    },
  };
}
test("preserves prior revision and scope evidence", () => {
  const s = memory();
  saveRecord(s, record);
  saveRecord(s, { ...record, scope: "c".repeat(64) });
  saveRecord(s, { ...record, revision: "d".repeat(64) });
  assert.equal(readRecords(s).length, 3);
});
test("repeated confirmation replaces only exact revision/scope", () => {
  const s = memory();
  saveRecord(s, record);
  saveRecord(s, { ...record, confirmedAt: "2026-11-02T00:00:00Z" });
  assert.equal(readRecords(s).length, 1);
});
test("corrupt data is not overwritten", () => {
  let writes = 0;
  const s = {
    getItem: () => "{bad",
    setItem: () => {
      writes++;
    },
  };
  assert.throws(() => saveRecord(s, record));
  assert.equal(writes, 0);
});
test("quota failure remains a failure", () =>
  assert.throws(() =>
    saveRecord(
      {
        getItem: () => null,
        setItem: () => {
          throw Error("quota");
        },
      },
      record,
    ),
  ));
test("silent persistence loss is detected", () =>
  assert.throws(() =>
    saveRecord({ getItem: () => null, setItem: () => {} }, record),
  ));
