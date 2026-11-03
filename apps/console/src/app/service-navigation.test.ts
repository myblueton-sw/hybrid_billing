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
  dataSection,
  contextualGroups,
  navigationScreenIds,
  relatedScreens,
  outsideBusinessNavigation,
  sectionFor,
} from "./service-ia";
import { planningScreens } from "../planning/registry";
test("draft keeps all canonical identities while nesting contextual work", () => {
  const ids = [
    ...businessSections.flatMap((s) => s.screens),
    ...dataSection.screens,
    ...administrationSection.screens,
    ...contextualGroups.flatMap((g) => g.screens),
    ...outsideBusinessNavigation,
  ];
  assert.equal(new Set(ids).size, 49);
  assert.equal(ids.length, 49);
  assert.deepEqual([...ids].sort(), planningScreens.map((s) => s.id).sort());
  assert.equal(sectionFor("I02")?.key, "settlement");
  assert.equal(sectionFor("I03")?.key, "settlement");
  assert.equal(sectionFor("S02")?.key, "data");
  assert.equal(sectionFor("B03")?.key, "costs");
  assert.equal(sectionFor("B04")?.key, "organizations");
  assert.ok(relatedScreens("I01").includes("I03"));
});
test("allowed contextual children retain entry without a permitted parent", () => {
  for (const child of ["B02", "I03", "O02", "R02", "M03"]) {
    const section = sectionFor(child)!;
    assert.deepEqual(navigationScreenIds(section, new Set([child])), [child]);
  }
  assert.deepEqual(navigationScreenIds(dataSection, new Set()), []);
});
