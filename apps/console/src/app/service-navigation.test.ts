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

import {
  businessSections,
  administrationSection,
  outsideBusinessNavigation,
  sectionFor,
} from "./service-ia";
import { planningScreens } from "../planning/registry";
test("approved IA covers all canonical screens once with separate accounting and administration", () => {
  const ids = [
    ...businessSections.flatMap((s) => s.screens),
    ...administrationSection.screens,
    ...outsideBusinessNavigation,
  ];
  assert.equal(businessSections.length, 8);
  assert.equal(new Set(ids).size, 49);
  assert.equal(ids.length, 49);
  assert.deepEqual([...ids].sort(), planningScreens.map((s) => s.id).sort());
  assert.equal(sectionFor("I02")?.key, "accounting");
  assert.equal(sectionFor("I01")?.key, "billing");
  assert.equal(sectionFor("B03")?.key, "policies");
  assert.equal(sectionFor("S04")?.key, "costs");
  for (const id of outsideBusinessNavigation)
    assert.equal(sectionFor(id), undefined);
});
