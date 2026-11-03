import { test } from "node:test";
import assert from "node:assert/strict";
import { serviceQuery } from "./service-navigation";
test("business navigation preserves service mode and shared scope without leaking object context", () => {
  const q = serviceQuery(
    new URLSearchParams(
      "lang=ar&service=local&review=1&overview-period=2026-10&ov-currency=USD&snapshot=secret&connection=old&ordinal=3&view=activation",
    ),
    "U01",
  );
  assert.equal(q.get("service"), "local");
  assert.equal(q.get("period"), "2026-10");
  assert.equal(q.get("ccy"), "USD");
  assert.equal(q.get("lang"), "ar");
  assert.equal(q.get("review"), "1");
  for (const k of ["snapshot", "connection", "ordinal", "view"])
    assert.equal(q.has(k), false);
});
