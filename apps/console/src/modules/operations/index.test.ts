import { test } from "node:test";
import assert from "node:assert/strict";
import { workRecords, findWorkRecord } from "./index";
test("synthetic work records retain scoped canonical references and no amounts", () => {
  const a = workRecords("sample-a"),
    b = workRecords("sample-b");
  assert.equal(a.length, 57);
  assert.equal(new Set([...a, ...b].map((r) => r.id)).size, 114);
  assert.equal(findWorkRecord("sample-b", a[0].id), undefined);
  assert.equal(findWorkRecord("sample-a", null), undefined);
  assert.equal(findWorkRecord("sample-a", a[0].id, "2026-09"), undefined);
  assert.equal(a[0].source, "sample-a-REC-0000001");
  assert(a.every((r) => !("net" in r) && !("amount" in r)));
  assert.equal(a.filter((r) => r.owner === "DEMO-PERSON-A").length, 29);
});
