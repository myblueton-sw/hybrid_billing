import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { PolicyPage, TrustChange } from "../../../../service/admin";
import { ChangesHead, Conditions, CurrentRules, Detail, Revoked, ShortConditions } from "./TrustChanges";
import { AdminError } from "../../../../service/admin";
import { trustFailure } from "./trust-model";
import { ConfirmBody } from "./TrustConfirmBody";
import { AfterActivation } from "./TrustAfter";
import { ty } from "./trust-copy";
import type { Confirming } from "./trust-model";
import { changeFixture, pageFixture } from "./trust-fixtures";

const noop = () => undefined;
const detail = (page: PolicyPage, c: TrustChange, l: "en" | "ko" | "ar" = "en") =>
  renderToStaticMarkup(<Detail l={l} page={page} c={c} split onClose={noop} onAction={noop} onOpen={noop} />);
const whole = (page: PolicyPage, l: "en" | "ko" | "ar" = "en") =>
  renderToStaticMarkup(<><Conditions l={l} page={page} onOpen={noop} /><CurrentRules l={l} page={page} loading={false} onReload={noop} /><ChangesHead l={l} page={page} onPropose={noop} /></>);
const items = [
  { item: "test_login", result: "verified", reason: "", detail: "" }, { item: "recovery_path", result: "verified", reason: "", detail: "" },
  { item: "issuer_subject", result: "not_applicable", reason: "local_password", detail: "" }, { item: "provider_diagnostics", result: "not_applicable", reason: "no_provider_connection", detail: "" },
  { item: "external_status", result: "not_applicable", reason: "no_external_identity", detail: "" }, { item: "mfa_weakening", result: "information", reason: "unchanged", detail: "" },
  { item: "affected_subjects", result: "information", reason: "", detail: "3" }, { item: "session_impact", result: "information", reason: "", detail: "2" },
  { item: "record_size", result: "verified", reason: "", detail: "" },
];
const glyphs = /[✓✗→←]/;

test("an activated change shows the new version and the invalidation items", () => {
  const c = changeFixture({ state: "activated", activator: "sample-admin-3", activation: { epoch: { value: "2", generation: 0, number: 2 }, ledger_sequence: 5, watermark: "5:abc", at: "2026-11-18T09:00:00Z" },
    invalidation: { state: "complete", items: [{ item: "console_sessions", result: "done", ended: 4, remaining: 0, reason: "" }, { item: "admitted_reads", result: "unknown", reason: "admitted_before_activation" }] },
    validation: { result: "validated", evidence_current: true, items } });
  const out = detail(pageFixture(), c);
  assert.match(out, /The new epoch and invalidation intent are committed/);
  assert.match(out, /<bdi dir="ltr" class="live-mono">0\.2<\/bdi>/);
  assert.match(out, /Invalidation progress/);
  assert.match(out, /Sessions ended: 4/);
  assert.match(out, /<td role="cell" data-label="Result" class="m02-word live-strong">Unknown<\/td>/);
  assert.doesNotMatch(out, glyphs);
  assert.doesNotMatch(out, /data-action=/);
});

test("an unconfirmed activation waits, never says activated and offers no retry", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const c = changeFixture({ state: "activation_unknown", job: { id: "trust-activation-trust-0a1b2c3d-1", state: "waiting", reason: "commit_unknown" } });
    const page = pageFixture({ trust: { hold: { reason: "commit_unknown", since: "2026-11-18T09:00:00Z" } }, ledger: { state: "stopped" } });
    const out = detail(page, c, l) + whole(page, l);
    assert.match(out, /trust-activation-trust-0a1b2c3d-1/);
    assert.match(out, new RegExp(ty.waiting(l)));
    assert.doesNotMatch(out, new RegExp(`>${ty.state(l, "activated")}<`));
    assert.doesNotMatch(out, /data-action=|m02-propose/);
    assert.match(out, /data-condition="hold"/);
  }
});

test("a superseded change links to the activated one", () => {
  const winner = changeFixture({ id: "trust-0a1b2c3d-2", number: 2, state: "activated" });
  const c = changeFixture({ state: "superseded", superseded_by: winner.id, superseded_epoch: { value: "2", generation: 0, number: 2 } });
  const out = detail(pageFixture({}), c);
  assert.match(out, /Another change was activated on the same epoch/);
  assert.match(out, /Superseded by/);
  assert.match(out, /data-opens="trust-0a1b2c3d-2"/);
  assert.doesNotMatch(out, /Change 2/);
});

test("a blocked validation names the blocked checks in plain words", () => {
  const blockedItems = items.map((i) => (i.item === "recovery_path" ? { ...i, result: "blocked", reason: "recovery_unverified", detail: "missing:trust_activate" } : i));
  const c = changeFixture({ validator: "sample-admin-2", validation: { result: "blocked", at: "2026-11-18T09:00:00Z", evidence_current: true, items: blockedItems, evidence_digest: "e".repeat(64) },
    can: { validate: true, cancel: true }, why_not: { approve: "not_validated", activate: "needs_approval" } });
  const out = detail(pageFixture({}), c);
  assert.match(out, /Validation found blocked checks, so this change stays a draft and cannot be approved or activated\./);
  assert.match(out, /1 check blocked/);
  assert.match(out, /No administrator who can sign in under the proposed rules holds: activate/);
  assert.match(out, /recovery_unverified/);
  assert.equal((out.match(/class="live-primary"/g) ?? []).length, 1);
  assert.match(out, /data-action="validate"/);
});

test("the enrolment block replaces the holders line", () => {
  const out = whole(pageFixture({ holders: { qualifying_people: 1, members: 1, needed: 3, enrolment: "not_decided", ready: false, missing: [] }, rights: { propose: true, validate: true, approve: true, activate: true } }));
  assert.match(out, /data-enrolment="blocked"/);
  assert.match(out, /<strong>Trust changes need three different administrators who hold the propose, approve and activate rights, plus one who holds the validate right\.<\/strong> Administrators who can use at least one trust right now: 1\./);
  const three = whole(pageFixture({ holders: { qualifying_people: 3, members: 3, needed: 3, enrolment: "not_decided", ready: false, missing: ["trust_propose", "trust_approve", "trust_activate"] } }));
  assert.match(three, /data-enrolment="blocked"/);
  assert.match(three, /No administrator who can sign in under the proposed rules holds: propose, approve, activate\. Administrators who can use at least one trust right now: 3\./);
  const ok = whole(pageFixture());
  assert.match(ok, /Administrators who can use at least one trust right now: 3\. A trust change needs three different administrators/);
  assert.match(ok, /m02-propose/);
});

test("isolation shows the problem line and removes trust actions", () => {
  const c = changeFixture({ can: {} , why_not: { approve: "trust_unavailable", activate: "trust_unavailable" } });
  const page = pageFixture({ ledger: { state: "isolated", isolation_reason: "integrity_failed", problem_line: 9 }, trust: { origin: "ledger_replay" } });
  const out = whole(page) + detail(page, c);
  assert.match(out, /The decision ledger cannot be verified\./);
  assert.match(out, /\(problem at line 9\)/);
  assert.match(out, /Trust changes are unavailable right now/);
  assert.match(out, /Current values may be older than the last activated change/);
  assert.match(out, /are not kept/);
  assert.doesNotMatch(out, /m02-propose|data-action=/);
});

test("versions render as g.n inside an ltr isolate in every language", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const out = whole(pageFixture({ trust: { epoch: "12884901888000001", epoch_generation: 3000000, epoch_number: 1 } }), l);
    assert.match(out, /<bdi dir="ltr" class="live-mono m02-epoch">3000000\.1<\/bdi>/);
    assert.doesNotMatch(out, glyphs);
  }
});

test("rationales use an auto-direction span, never bdi", () => {
  const c = changeFixture({ history: [{ state: "draft", by: "sample-admin", at: "2026-11-18T09:00:00Z", rationale: "تقليل مدة الجلسة" }] });
  const out = detail(pageFixture({}), c, "ar");
  assert.match(out, /<span class="m02-free" dir="auto">تقليل مدة الجلسة<\/span>/);
});

test("confirmations C1 to C6 show the common facts, the bound values and their boundary", () => {
  const ch = changeFixture({ state: "independently_approved", change_revision: 2, validator: "sample-admin-2", approver: "sample-admin-2",
    validation: { result: "validated", evidence_current: true, evidence_digest: "e".repeat(64), items } });
  const page = pageFixture({ ledger: { state: "ok", generation: 0 } });
  const body = (k: Confirming) => renderToStaticMarkup(<ConfirmBody l="en" page={page} k={k} expectedDigest={"f".repeat(64)} workspace="sample-workspace" />);
  const c1 = body({ kind: "propose", change: null, proposed: { maxAge: 60, step: 10 } });
  assert.match(c1, /sample-workspace/);
  assert.match(c1, /Local password/);
  assert.match(c1, new RegExp("f".repeat(64) + ".*expected"));
  assert.match(c1, /everyone in the workspace is signed out/);
  assert.match(c1, /Creates a draft only/);
  const revise = body({ kind: "propose", change: ch, proposed: { maxAge: 45, step: 0 } });
  assert.match(revise, /Change revision 2 becomes 3/);
  const c2 = body({ kind: "validate", change: ch, proposed: null });
  assert.match(c2, /Your current session is used as the test login/);
  assert.match(c2, /Provider diagnostics: No identity provider is connected\./);
  const c3 = body({ kind: "approve", change: ch, proposed: null });
  assert.match(c3, new RegExp("e".repeat(64)));
  assert.match(c3, /You hold the approval right and did not propose this change/);
  const c4 = body({ kind: "activate", change: ch, proposed: null });
  assert.match(c4, /0\.2<\/bdi> <span class="m02-sub">\(expected\)/);
  assert.match(c4, /including you/);
  assert.match(c4, /Undoing this needs a new trust change with a newer version/);
  assert.match(c4, /<p class="m02-boundary live-strong">After you confirm, this change cannot be rejected or cancelled\.<\/p>/);
  assert.match(body({ kind: "reject", change: ch, proposed: null }), /A rejected change cannot be reopened/);
  assert.match(body({ kind: "cancel", change: ch, proposed: null }), /A cancelled proposal cannot be reopened/);
  for (const k of ["validate", "approve", "activate", "reject", "cancel"] as const) {
    const out = body({ kind: k, change: ch, proposed: null });
    assert.match(out, /trust-0a1b2c3d-1/, k);
    assert.match(out, /Change revision<\/dt><dd>2/, k);
    assert.doesNotMatch(out, glyphs, k);
  }
});

test("a revoked view holds no list, detail, form, dialog or selection", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const out = renderToStaticMarkup(<Revoked l={l} onReload={noop} />);
    assert.match(out, new RegExp(ty.revokedTitle(l)));
    assert.doesNotMatch(out, /m02-table|m02-detail|m02-form|<dialog|m02-tag|trust-0a1b2c3d/);
  }
});

test("the after-activation view trusts only a completed activated answer", () => {
  const view = (out: { status: string; state: string; job_id: string; fields: Record<string, string> }) =>
    renderToStaticMarkup(<AfterActivation l="en" out={out} onSignIn={noop} />);
  const done = view({ status: "completed", state: "activated", job_id: "", fields: { epoch_generation: "0", epoch_number: "2", ledger_sequence: "5", sessions_ended: "3", sessions_remaining: "0" } });
  assert.match(done, /data-after="activated"/);
  assert.match(done, /0\.2/);
  assert.match(done, /Your session has ended\. Sign in again to see the remaining invalidation progress/);
  const waiting = view({ status: "accepted", state: "", job_id: "trust-activation-trust-0a1b2c3d-1", fields: {} });
  assert.match(waiting, /data-after="activation_unknown"/);
  assert.doesNotMatch(waiting, /Activated|retry/i);
  for (const odd of [{ status: "completed", state: "draft" }, { status: "accepted", state: "" }, { status: "", state: "activated" }]) {
    const out = view({ ...odd, job_id: "", fields: { epoch: "2" } });
    assert.match(out, /data-after="unclear"/, JSON.stringify(odd));
    assert.match(out, /did not confirm whether the change was activated\. Sign in again/);
    assert.match(out, /<button type="button" class="live-primary">Sign in again<\/button>/);
    assert.doesNotMatch(out, /0\.2|Activated</);
  }
  const partial = view({ status: "completed", state: "activated", job_id: "", fields: {} });
  assert.match(partial, /Not recorded/);
});

test("dialog details after a refusal and the C3/C4 wording", () => {
  const ch = changeFixture({ state: "independently_approved", approver: "sample-admin-2", validator: "sample-admin-2",
    validation: { result: "validated", at: "2026-11-18T09:00:00Z", evidence_current: true, evidence_digest: "e".repeat(64), items } });
  const page = pageFixture({});
  const body = (k: Confirming, dead = false) => renderToStaticMarkup(<ConfirmBody l="en" page={page} k={k} expectedDigest="" workspace="w" dead={dead} />);
  const c3 = body({ kind: "approve", change: { ...ch, state: "validated" }, proposed: null });
  assert.doesNotMatch(c3, /2026-11-18T09:00:00Z/);
  assert.match(c3, /Lockout risk<\/dt><dd>Verified/);
  const dead = body({ kind: "approve", change: { ...ch, state: "validated" }, proposed: null }, true);
  assert.doesNotMatch(dead, /Your authority|Lockout risk/);
  assert.doesNotMatch(body({ kind: "activate", change: ch, proposed: null }, true), /Lockout risk|Test login/);
  assert.match(body({ kind: "activate", change: ch, proposed: null }), /To be activated by<\/dt><dd>You/);
  assert.equal(ty.confirmTitle("en", "Propose trust change", 0), "Propose trust change");
  assert.equal(ty.confirmTitle("en", "Validate trust change", 3), "Validate trust change: Change 3");
  assert.equal(ty.confirmTitle("ko", "신뢰 변경 검증", 3), "신뢰 변경 검증: 변경 3");
  const missing = body({ kind: "reject", change: { ...ch, proposer: "" }, proposed: null });
  assert.match(missing, /Proposed by<\/dt><dd><span class="live-quiet">Not recorded/);
});

test("an activation that is not confirmed never says activated", () => {
  const c = changeFixture({ state: "activation_unknown", activator: "sample-admin-3", job: { id: "trust-activation-x", state: "waiting", reason: "commit_unknown" } });
  for (const l of ["en", "ko", "ar"] as const) {
    const out = detail(pageFixture({ trust: { hold: { reason: "commit_unknown", since: "t" } } }), c, l);
    assert.doesNotMatch(out, new RegExp(ty.activatedBy(l) + "<"), l);
    assert.match(out, new RegExp(ty.attemptedBy(l)), l);
  }
});

test("round-two details: lockout reason, stale text per command, kept approval, empty reasons, the hold shown once", () => {
  const ch = changeFixture({ state: "validated", validation: { result: "validated", at: "2026-11-18T09:00:00Z", evidence_current: true, evidence_digest: "e".repeat(64), items } });
  const page = pageFixture({});
  const c3 = renderToStaticMarkup(<ConfirmBody l="en" page={page} k={{ kind: "approve", change: ch, proposed: null }} expectedDigest="" workspace="w" />);
  assert.match(c3, /Lockout risk<\/dt><dd>Verified · Three different administrators who can sign in under the proposed rules/);
  for (const l of ["ko", "ar"] as const) {
    assert.match(renderToStaticMarkup(<ConfirmBody l={l} page={page} k={{ kind: "activate", change: ch, proposed: null }} expectedDigest="" workspace="w" />), new RegExp(ty.reason(l, "recovery_verified")));
  }
  const stale = new AdminError("evidence_stale");
  assert.equal(trustFailure("en", stale, "approve").body, ty.sentence("en", "stale_validated"));
  assert.equal(trustFailure("en", stale, "activate").body, ty.fail.evidence_stale("en"));
  const kept = changeFixture({ state: "independently_approved", history: [{ state: "revalidated_approval_kept", by: "sample-admin-2", at: "2026-11-18T09:00:00Z", rationale: "" }] });
  assert.match(detail(page, kept), /Validated again, approval kept/);
  const out = detail(page, ch);
  assert.doesNotMatch(out, /data-label="Reason"><\/td>/);
  assert.match(out, /<tr role="row" data-item="test_login" data-result="verified"><td role="cell">Test login<\/td><td role="cell" data-label="Result" class="m02-word">Verified<\/td><td role="cell"><\/td><\/tr>/);
  const unknown = changeFixture({ state: "activation_unknown", job: { id: "trust-activation-x", state: "waiting", reason: "commit_unknown" } });
  const held = pageFixture({ trust: { hold: { reason: "commit_unknown", since: "t" } } });
  const split = detail(held, unknown);
  assert.equal((split.match(/could not confirm that the activation record was saved/g) ?? []).length, 0);
  assert.match(split, /listed in the notice above/);
  const step = renderToStaticMarkup(<><ShortConditions l="en" page={held} skipHold /><Detail l="en" page={held} c={unknown} split={false} onClose={noop} onAction={noop} onOpen={noop} /></>);
  assert.equal((step.match(/could not confirm that the activation record was saved/g) ?? []).length, 1);
});

test("the validated line keeps the validator ID in one isolated unit", () => {
  const c = changeFixture({ state: "validated", validator: "sample-admin-2", validation: { result: "validated", at: "2026-11-18T09:00:00Z", evidence_current: true, items, evidence_digest: "e".repeat(64) } });
  for (const l of ["en", "ko", "ar"] as const) {
    const html = detail(pageFixture(), c, l);
    assert.match(html, /<p class="m02-result">[^<]*<bdi dir="ltr" class="live-mono m02-actor">sample-admin-2<\/bdi>/);
    assert.doesNotMatch(html, /⁣/);
  }
});

test("an impact that was not computed says so and shows no count, while a computed zero is shown as zero", () => {
  const impactOf = (out: string) => /<section class="m02-impact"[\s\S]*?<\/section>/.exec(out)?.[0] ?? "";
  for (const l of ["en", "ko", "ar"] as const) {
    const missing = impactOf(detail(pageFixture(), changeFixture({ impact: null }), l));
    assert.ok(missing.includes(ty.impactNotComputed(l)), l);
    assert.ok(missing.includes(ty.signsOut(l)), l);
    assert.doesNotMatch(missing, /m02-sum/);
    assert.doesNotMatch(missing, /<dd>\d/);
    const zero = impactOf(detail(pageFixture(), changeFixture({ impact: { members: 0, sessions: 0, affected_ids: [], weakens_mfa: false, signs_out_everyone: false } }), l));
    assert.ok(!zero.includes(ty.impactNotComputed(l)), l);
    assert.match(zero, /<dd>0<\/dd>/);
  }
  assert.equal(impactOf(detail(pageFixture(), changeFixture({ state: "activation_unknown", impact: null }))), "", "no impact section while the activation is not confirmed");
});
