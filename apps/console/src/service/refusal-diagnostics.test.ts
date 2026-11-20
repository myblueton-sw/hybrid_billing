import { test } from "node:test";
import assert from "node:assert/strict";
import type { RowDiagnostic } from "./command-diagnostics";
import * as installation from "../modules/administration/pages/installation/live-model";
import * as integrations from "../modules/administration/pages/integrations/live-model";
import * as recovery from "../modules/administration/pages/recovery/live-model";
import * as retention from "../modules/administration/pages/retention/live-model";
import * as evidence from "../modules/administration/pages/release-evidence/live-model";

const variants = [
  { name: "P01", model: installation, command: "installation.approve", other: "installation.withdraw", refuse: (c: string, t: string, d: RowDiagnostic) => installation.settle(c, t, "hint", d) },
  { name: "P02", model: integrations, command: "integrations.plugin_stop", other: "integrations.plugin_disable", refuse: (c: string, t: string, d: RowDiagnostic) => integrations.settle(c, t, "hint", d) },
  { name: "P03", model: recovery, command: "recovery.approve", other: "recovery.withdraw", refuse: (c: string, t: string, d: RowDiagnostic) => recovery.settle(c, t, "hint", d) },
  { name: "P04", model: retention, command: "retention.approve", other: "retention.withdraw", refuse: (c: string, t: string, d: RowDiagnostic) => retention.settle(c, t, "hint", "refused", d) },
  { name: "P05", model: evidence, command: "release_evidence.approve", other: "release_evidence.withdraw", refuse: (c: string, t: string, d: RowDiagnostic) => evidence.settle(c, t, "hint", "refused", false, d) },
];
for (const v of variants) {
  test(`${v.name} references follow the exact retained command/target marker through remount, reset and scope`, () => {
    const target = "sample-target";
    const entry = (command: string, request: string): RowDiagnostic => ({ command, detail: { code: "prerequisite_failed", request, client: "client-330", target, cancelled: false } });
    v.model.setScope("workspace", "user", "session");
    v.refuse(v.command, target, entry(v.command, "first-330"));
    v.model.setScope("workspace", "user", "session");
    assert.deepEqual(v.model.refusalDiagnostics(target), [entry(v.command, "first-330")]);
    v.refuse(v.other, target, entry(v.other, "second-330"));
    assert.equal(v.model.refusalDiagnostics(target).length, 2);
    assert.deepEqual(v.model.refusalDiagnostics("foreign-target"), []);
    v.model.settle(v.command, target, "ok");
    assert.deepEqual(v.model.refusalDiagnostics(target), [entry(v.other, "second-330")]);
    v.model.setScope("other-workspace", "user", "session");
    assert.deepEqual(v.model.refusalDiagnostics(target), []);
    v.model.forget();
  });
}
