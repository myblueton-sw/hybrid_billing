import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import { ledgerOf } from "../../../../service/admin";
import type { ListedRevocation } from "../../../../service/access-lists";
import { LedgerLine, RevocationRecords } from "./Revocations";
import { SettingsDisclosure } from "./Settings";

type Revocation = ListedRevocation;
const rev = (change: Partial<Revocation>): Revocation => ({
  id: "revocation-1", origin: "this_process", recorded_at: "2026-11-18T09:00:00Z", target_kind: "grant", target: "g1", target_present: true, replay: "", state: "revocation_complete",
  history: [], ended_delegations: [], deny_generation: 3, job_state: "succeeded", job_reason: "", requested_by: "p", approved_by: "a", request: "access-request-1",
  received_at: "2026-11-18T09:00:00Z", deny_at: "2026-11-18T09:00:00Z", receipt: { sequence: 2, watermark: "2:abcdef", reason: "" }, verified: true,
  cleanup: [{ item: "C1", state: "done", reason: "deny_generation", refs: [] }, { item: "C5", state: "outside_scope", reason: "cannot_recall", refs: [] }], completed_at: "",
  target_version: 1, target_subject: "sample-member", target_subject_login: "member@example.invalid", target_action: "view", target_resource_kind: "focus_data", target_fields: ["rows"], ...change,
});
const settings = [{ name: "access_request_hours", value: 24, unit: "hours", default: 168, min: 1, max: 720, source: "start_flag" }];
const page = (revocations: Revocation[], ledger = ledgerOf({ state: "ok", records: 2, watermark: "2:abcdef" })) => ({ revocations, ledger, settings });
const html = (p: ReturnType<typeof page>) => renderToStaticMarkup(<><LedgerLine l="en" ledger={p.ledger} /><RevocationRecords l="en" rows={p.revocations} /></>);

test("each revocation state renders its copy and job state", () => {
  const cases: [Revocation["state"], Revocation["job_state"], RegExp][] = [
    ["received", "running", /The security block is not confirmed yet\..*|Job: running/],
    ["deny_confirmed", "running", /access block are confirmed/],
    ["cleanup_running", "running", /stays in effect while sessions, keys, caches and jobs are cleaned up/],
    ["revocation_complete", "succeeded", /cleanup of the designated scope are complete/],
    ["decision_unconfirmed", "waiting", /could not be confirmed\. This is not shown as a successful revocation/],
  ];
  for (const [state, job, copy] of cases) {
    const out = html(page([rev({ state, job_state: job })]));
    assert.match(out, copy, state);
    assert.match(out, new RegExp(`data-state="${state}"`));
    assert.match(out, job === "succeeded" ? /Job: succeeded/ : job === "waiting" ? /Job: waiting/ : /Job: running/);
  }
});

test("an unconfirmed revocation says this process only and starts no cleanup", () => {
  const out = html(page([rev({ state: "decision_unconfirmed", job_state: "waiting", receipt: { sequence: 0, watermark: "", reason: "ledger_isolated" },
    cleanup: [{ item: "C1", state: "not_started", reason: "deny_generation", refs: [] }] })], ledgerOf({ state: "isolated", isolation_reason: "integrity_failed", problem_line: 12 })));
  assert.match(out, /this server process only/);
  assert.match(out, /Not started: waits for a confirmed decision/);
  assert.match(out, /integrity check at line 12/);
  assert.match(out, /The ledger is isolated/);
  assert.doesNotMatch(out, /cleanup items settled/);
});

test("a replayed record shows its origin and the outside scope apart", () => {
  const out = html(page([rev({ origin: "ledger_replay", replay: "absent", target_present: false })]));
  assert.match(out, /From an earlier server run/);
  assert.match(out, /Target not present after restart/);
  assert.match(out, /Outside the controlled scope \(not counted as cleaned\)/);
  assert.match(out, /1 of 2 cleanup items settled/);
  assert.match(out, /2:abcdef/);
  assert.match(out, /detects damage to the file/);
});

test("the settings disclosure reads the source in words", () => {
  const out = renderToStaticMarkup(<SettingsDisclosure l="ko" settings={page([]).settings} />);
  assert.match(out, /승인 설정/);
  assert.match(out, /서버 시작 시 지정/);
  assert.equal(renderToStaticMarkup(<SettingsDisclosure l="ko" settings={[]} />), "");
});

test("a record's heading names the holder and the access from its own facts", () => {
  const grant = html(page([rev({})]));
  assert.match(grant, /<h3>Grant: View · Rows for <bdi dir="ltr">member<wbr\/>@example\.invalid<\/bdi><\/h3>/);
  assert.match(grant, /class="live-meta access-target">[^<]* · <bdi dir="ltr" class="live-mono" translate="no">g1<\/bdi>/);
  const delegation = html(page([rev({ target_kind: "delegation", target: "d1" })]));
  assert.match(delegation, /<h3>Delegation to <bdi dir="ltr">member<wbr\/>@example\.invalid<\/bdi>: View · Rows<\/h3>/);
  const unnamed = html(page([rev({ target_subject_login: "" })]));
  assert.match(unnamed, /for <bdi dir="ltr">sample-member<\/bdi>/);
  const gone = { target_subject: "", target_subject_login: "", target_action: "", target_resource_kind: "", target_fields: [] };
  assert.match(html(page([rev(gone)])), /<h3>Grant g1<\/h3>/);
  assert.match(html(page([rev({ ...gone, target_kind: "delegation", target: "d1" })])), /<h3>Delegation d1<\/h3>/);
  assert.doesNotMatch(html(page([rev(gone)])), /access-target/);
});
