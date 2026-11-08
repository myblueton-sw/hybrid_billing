import { test } from "node:test";
import assert from "node:assert/strict";
import { assemble } from "./assemble";
import type { Module, Extension, Profile } from "../shared/types";
import { getProfile } from "../profiles";
const screen = () => null;
const base: Module = {
  id: "base",
  version: 1,
  requires: [],
  routes: [
    {
      id: "intake",
      title: { en: "Intake", ko: "수집", ar: "استلام" },
      component: screen,
    },
  ],
  slots: ["base/evidence/v1"],
};
const extension: Extension = {
  id: "guide",
  version: 1,
  module: "base",
  slot: "base/evidence/v1",
  slotVersion: 1,
  render: screen,
};
const profile: Profile = { id: "sample", modules: ["base"], extensions: [] };
test("standard and extended composition retain the same core route", () => {
  assert.equal(assemble([base], [extension], profile).routes[0].id, "intake");
  assert.deepEqual(
    assemble([base], [extension], { ...profile, extensions: ["guide"] })
      .extensions,
    [extension],
  );
});
test("unknown profile does not fall back", () =>
  assert.throws(() => getProfile("customer-private")));
for (const [name, mods, exts, p] of [
  ["duplicate module", [base, base], [extension], profile],
  ["unknown module", [base], [extension], { ...profile, modules: ["other"] }],
  [
    "missing dependency",
    [{ ...base, requires: ["other"] }],
    [extension],
    profile,
  ],
  ["cycle", [{ ...base, requires: ["base"] }], [extension], profile],
  [
    "route collision",
    [base, { ...base, id: "extra" }],
    [extension],
    { ...profile, modules: ["base", "extra"] },
  ],
  ["unknown extension", [base], [], { ...profile, extensions: ["guide"] }],
  [
    "incompatible slot",
    [base],
    [{ ...extension, slotVersion: 2 }],
    { ...profile, extensions: ["guide"] },
  ],
  [
    "unknown slot",
    [base],
    [{ ...extension, slot: "unknown" }],
    { ...profile, extensions: ["guide"] },
  ],
  [
    "duplicate contribution",
    [base],
    [extension],
    { ...profile, extensions: ["guide", "guide"] },
  ],
  [
    "unknown setting",
    [base],
    [extension],
    { ...profile, script: "https://invalid.test/code.js" },
  ],
] as [string, Module[], Extension[], Profile][]) {
  test(name, () => assert.throws(() => assemble(mods, exts, p)));
}

test("slot belongs to declared module", () =>
  assert.throws(() =>
    assemble(
      [base, { ...base, id: "other", routes: [], slots: [] }],
      [{ ...extension, module: "other" }],
      { ...profile, modules: ["base", "other"], extensions: ["guide"] },
    ),
  ));
