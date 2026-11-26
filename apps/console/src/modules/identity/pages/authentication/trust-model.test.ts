import { test } from "node:test";
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { AdminError } from "../../../../service/admin";
import { changeFixture, pageFixture } from "./trust-fixtures";
import { ty } from "./trust-copy";
import {
  actions, asTrustFailure, canPropose, candidateOf, conditions, conditionText, configDigest, configEncoding, currentVersion, enrolled, expectedNext, fatal, findingReason, nextStep, revoked,
  rawEpochText as epochText, rightsList, stateSentence, trustFailure, unavailable, valuesFor, versionOf, whyNot,
} from "./trust-model";

test("versions come from the two integer fields, never from the raw decimal", () => {
  const big = pageFixture({ trust: { epoch: "12884901888000001", epoch_generation: 3000000, epoch_number: 1 }, ledger: { state: "ok", generation: 3000000 } });
  assert.equal(currentVersion(big), "3000000.1");
  assert.equal(versionOf({ generation: 2, number: 7 }), "2.7");
  assert.equal(expectedNext(big).text, "3000000.2");
  assert.equal(expectedNext(pageFixture({ trust: { epoch_generation: 0, epoch_number: 4 }, ledger: { state: "ok", generation: 2 } })).text, "2.1");
  assert.equal(expectedNext(pageFixture({ trust: { epoch_generation: 1, epoch_number: 0 }, ledger: { state: "ok", generation: 1 } })).text, "1.1");
  assert.equal(epochText("4294967298"), "1.2");
  assert.equal(epochText("12884901888000001"), "3000000.1");
  assert.equal(epochText("x"), "x");
});

test("the expected candidate digest uses the server's canonical encoding", async () => {
  const c = candidateOf(60, 10);
  const enc = configEncoding(c);
  const want = createHash("sha256").update(enc).digest("hex");
  assert.equal(await configDigest(c), want);
  assert.equal(enc.length, 8 + 15 + 8 + 8 + 8 + 8 + 4 + 8 + 8 + 8 + 13 + 8 + 8);
  assert.notEqual(await configDigest(candidateOf(60, 0)), want);
  assert.equal(Buffer.from(enc.slice(0, 8)).readBigUInt64BE(), 15n);
  assert.equal(want, "af179a8e130d4fcde866be8626e1614ac50a6e1c122cd678dbd666a15fb54b02");
  assert.equal(await configDigest(candidateOf(720, 0)), "7a1d6f2ae7ddad9dc92e57607f6837ff477ea563ed1d23dbee0dc3c5fb4a2fa2");
});

test("next steps and state sentences per state", () => {
  const cases: [string, Record<string, unknown>, string][] = [
    ["draft", {}, ty.next.draft("en")],
    ["draft", { validation: { result: "blocked", items: [] } }, ty.next.blocked("en")],
    ["validated", { validation: { result: "validated", evidence_current: true, items: [] } }, ty.next.validated("en")],
    ["validated", { validation: { result: "validated", evidence_current: false, items: [] } }, ty.next.stale("en")],
    ["independently_approved", { validation: { result: "validated", evidence_current: true, items: [] } }, ty.next.approved("en")],
    ["activation_unknown", {}, ty.next.unknown("en")],
    ["activated", {}, ty.next.activated("en")],
    ["activated", { invalidation: { state: "in_progress", items: [] } }, ty.next.progress("en")],
    ["superseded", {}, ty.next.closed("en")],
    ["rejected", {}, ty.next.closed("en")],
  ];
  for (const [state, over, want] of cases) assert.equal(nextStep("en", changeFixture({ state, ...over })), want, state);
  assert.equal(stateSentence("en", changeFixture({ validation: { result: "blocked", items: [] } })), ty.sentence("en", "blocked"));
  assert.match(stateSentence("en", changeFixture({ state: "validated", validation: { result: "validated", evidence_current: false, items: [] } })), /cannot be approved\. Validate it again\.$/);
  assert.match(stateSentence("en", changeFixture({ state: "independently_approved", validation: { result: "validated", evidence_current: false, items: [] } })), /recorded approval cannot be used to activate/);
  assert.deepEqual(whyNot("en", changeFixture({ state: "independently_approved", why_not: { approve: "not_validated", activate: "approver" }, validation: { result: "validated", evidence_current: true, items: [] } })), [ty.why.approver("en")]);
  assert.deepEqual(whyNot("en", changeFixture({ state: "validated", why_not: { approve: "evidence_stale", activate: "needs_approval" }, validation: { result: "validated", evidence_current: false, items: [] } })), []);
  assert.match(stateSentence("en", changeFixture({ state: "superseded" })), /Another change was activated on the same epoch/);
  assert.doesNotMatch(stateSentence("en", changeFixture({ state: "activation_unknown" })), /Activated/);
  for (const l of ["en", "ko", "ar"] as const) assert.doesNotMatch(ty.state(l, "activation_unknown"), new RegExp(ty.state(l, "activated")));
});

test("actions follow the server flags with one primary", () => {
  assert.deepEqual(actions(changeFixture({ can: { validate: true, reject: true, revise: true, cancel: true } })), { primary: "validate", secondary: ["revise", "reject", "cancel"] });
  assert.deepEqual(actions(changeFixture({ can: { activate: true, validate: true } })), { primary: "activate", secondary: ["validate"] });
  assert.deepEqual(actions(changeFixture({ can: { approve: true } })), { primary: "approve", secondary: [] });
  assert.deepEqual(actions(changeFixture({})), { primary: null, secondary: [] });
  assert.deepEqual(whyNot("en", changeFixture({ state: "independently_approved", why_not: { approve: "not_validated", activate: "self" } })), [ty.why.self("en")]);
  assert.deepEqual(whyNot("en", changeFixture({ state: "draft", why_not: { approve: "not_validated", activate: "self" } })), [ty.why.not_validated("en"), ty.why.self("en")]);
  assert.deepEqual(whyNot("en", changeFixture({ state: "activated", why_not: { approve: "self", activate: "self" } })), []);
  assert.deepEqual(whyNot("en", changeFixture({ why_not: { approve: "evidence_stale", activate: "evidence_stale" } })), [ty.why.evidence_stale("en")]);
});

test("conditions, unavailability, holders and rights", () => {
  assert.deepEqual(conditions(pageFixture()), []);
  const held = pageFixture({ trust: { hold: { reason: "commit_unknown", since: "t" } }, ledger: { state: "stopped" } });
  assert.deepEqual(conditions(held).map((c) => c.kind), ["hold"]);
  assert.equal(unavailable(held), true);
  assert.equal(canPropose(held), false);
  const iso = pageFixture({ ledger: { state: "isolated", isolation_reason: "integrity_failed", problem_line: 7 }, trust: { origin: "carried", carried_line: 4, reset_at: "2026-11-18T08:00:00Z" } });
  assert.deepEqual(conditions(iso), [{ kind: "isolated", line: 7, older: false }, { kind: "reset", line: 0 }, { kind: "carried", line: 4 }]);
  assert.deepEqual(conditions(pageFixture({ ledger: { state: "isolated", isolation_reason: "missing" } })), [{ kind: "isolated", line: 0, older: false }]);
  assert.equal(conditionText("en", { kind: "isolated", line: 0, older: true }, "").length, 3);
  assert.deepEqual(conditions(pageFixture({ ledger: { state: "stopped", not_durable: true } })).map((c) => c.kind), ["durable"]);
  assert.deepEqual(conditions(pageFixture({ ledger: { state: "stopped" } })).map((c) => c.kind), ["stopped"]);
  assert.equal(unavailable(pageFixture({ ledger: { state: "full" } })), false);
  assert.equal(canPropose(pageFixture({ rights: { propose: false } })), false);
  assert.equal(enrolled(pageFixture({ holders: { qualifying_people: 1, needed: 3, ready: false } })), false);
  assert.equal(enrolled(pageFixture({ holders: { qualifying_people: 3, needed: 3, ready: false } })), false);
  assert.equal(enrolled(pageFixture({ holders: { qualifying_people: 3, needed: 3, ready: true } })), true);
  assert.equal(rightsList("en", pageFixture({ rights: { propose: true, validate: true } })), "Your trust rights: propose, validate.");
  assert.equal(rightsList("en", pageFixture({ rights: {} })), ty.noRights("en"));
});

test("findings name missing roles and blocked test logins", () => {
  assert.equal(findingReason("en", { item: "recovery_path", result: "blocked", reason: "recovery_unverified", detail: "missing:trust_activate,trust_approve" }), "No administrator who can sign in under the proposed rules holds: activate, approve.");
  assert.equal(findingReason("en", { item: "recovery_path", result: "blocked", reason: "recovery_unverified", detail: "three_distinct_people" }), ty.reason("en", "recovery_unverified"));
  assert.equal(findingReason("en", { item: "test_login", result: "blocked", reason: "test_login_failed", detail: "mfa_required" }), ty.reason("en", "test_login_failed"));
  assert.equal(findingReason("en", { item: "issuer_subject", result: "not_applicable", reason: "local_password", detail: "" }), ty.reason("en", "local_password"));
  assert.equal(findingReason("en", { item: "record_size", result: "verified", reason: "", detail: "" }), "");
});

test("command bodies carry the bound digests, epochs and revisions", () => {
  const ch = changeFixture({ state: "independently_approved", change_revision: 2, validation: { result: "validated", evidence_digest: "e".repeat(64), items: [] } });
  const page = pageFixture({ trust: { epoch: "4294967297", epoch_generation: 1, epoch_number: 1, policy_number: 4 } });
  assert.deepEqual(valuesFor(page, { kind: "propose", change: null, proposed: { maxAge: 60, step: 10 } }),
    { max_age_minutes: "60", step_up_minutes: "10", expected_epoch: "4294967297", expected_policy: "4" });
  assert.equal(valuesFor(page, { kind: "propose", change: ch, proposed: { maxAge: 60, step: 0 } }).expected_change_revision, "2");
  assert.deepEqual(valuesFor(page, { kind: "validate", change: ch, proposed: null }), { expected_change_revision: "2", change_digest: ch.change_digest, expected_epoch: "4294967297" });
  assert.deepEqual(valuesFor(page, { kind: "approve", change: ch, proposed: null }), { change_digest: ch.change_digest, evidence_digest: "e".repeat(64), expected_epoch: "4294967297" });
  assert.deepEqual(valuesFor(page, { kind: "activate", change: ch, proposed: null }), { change_digest: ch.change_digest, evidence_digest: "e".repeat(64), expected_epoch: "4294967297", expected_policy: "4" });
  assert.deepEqual(valuesFor(page, { kind: "cancel", change: ch, proposed: null }), { expected_change_revision: "2" });
});

test("refusals in every language", () => {
  const held = pageFixture({ trust: { hold: { reason: "commit_unknown", since: "t" } } });
  const shown = asTrustFailure(new AdminError("busy"), held);
  assert.equal(shown instanceof AdminError && shown.failure, "trust_unavailable");
  assert.equal(asTrustFailure(new AdminError("busy"), pageFixture()) instanceof AdminError && (asTrustFailure(new AdminError("busy"), pageFixture()) as AdminError).failure, "busy");
  for (const l of ["en", "ko", "ar"] as const) {
    const bodies = (["self", "epoch", "approval_stale", "evidence_stale", "needs_approval", "trust_unavailable", "stage", "denied", "unauthenticated"] as const).map((f) => trustFailure(l, new AdminError(f)).body);
    assert.equal(new Set(bodies).size, bodies.length, l);
    assert.notEqual(trustFailure(l, new AdminError("busy")).body, "");
    assert.notEqual(trustFailure(l, new Error("x")).title, "");
  }
  assert.equal(fatal(new AdminError("evidence_stale")), true);
  assert.equal(fatal(new AdminError("busy")), false);
  assert.equal(fatal(new Error("x")), false);
  assert.equal(revoked(new AdminError("denied")), true);
  assert.equal(revoked(new AdminError("stage")), false);
});
