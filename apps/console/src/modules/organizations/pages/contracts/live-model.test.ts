import { test } from "node:test";
import assert from "node:assert/strict";
import type { ContractDetail, ContractRow } from "../../../../service/contracts";
import { contractDetailOf } from "../../../../service/contracts";
import {
  actionOf, activeFilters, actorText, addDays, clearDraft, coversEnd, dayProblem, draftOf, filtered, filtersOf, forget, idProblem, keepDraft,
  keepsAttempt, knownBlockers, legalEntities, lifecycleBounds, minDay, offered, openGaps, pendingFor, pendingNow, periodOf, react, reasonTooLong, revisionText,
  setScope, settle, settleBounds, discard, bodyFor, commandFor, controlFor, canWithdraw, endPassed, spanDays, utf8Bytes, validDay, withdrawable,
} from "./live-model";
import { k, tables } from "./live-copy";

const row = (over: Partial<ContractRow>): ContractRow => ({
  id: "c", name: "", profile: "", profile_name: "", buyer: "", payer: "", issuer: "", path: "", currency: "", timezone: "", cycle: "", state: "", lifecycle_revision: null,
  settlement: "", sequence: null, start: "", end: "", open_gaps: null, ...over,
});

test("filters read only known values and combine", () => {
  const f = filtersOf(new URLSearchParams("state=active&settlement=bogus&path=external_invoice&buyer=le-1"));
  assert.deepEqual(f, { state: "active", settlement: "", path: "external_invoice", buyer: "le-1" });
  assert.equal(filtersOf(new URLSearchParams("buyer=has space")).buyer, "");
  assert.equal(activeFilters(f), 3);
  const rows = [row({ id: "a", state: "active", path: "external_invoice", buyer: "le-1" }), row({ id: "b", state: "active", path: "external_invoice", buyer: "le-2" }), row({ id: "c", state: "ended" })];
  assert.deepEqual(filtered(rows, f).map((r) => r.id), ["a"]);
  assert.deepEqual(filtered(rows, { state: "", settlement: "", path: "", buyer: "" }).length, 3);
  assert.deepEqual(legalEntities(rows), ["le-1", "le-2"]);
});

test("dates and periods are validated as UTC calendar days", () => {
  assert.equal(validDay("2026-02-29"), false);
  assert.equal(validDay("2028-02-29"), true);
  assert.equal(validDay("2026-1-01"), false);
  assert.equal(addDays("2026-12-31", 1), "2027-01-01");
  assert.equal(spanDays("2026-01-01", "2027-01-01"), 365);
  assert.deepEqual(periodOf(new URLSearchParams("")), { from: "", until: "", problem: "" });
  assert.equal(periodOf(new URLSearchParams("from=x")).problem, "from");
  assert.equal(periodOf(new URLSearchParams("from=2026-01-01&until=y")).problem, "until");
  assert.equal(periodOf(new URLSearchParams("from=2026-02-01&until=2026-02-01")).problem, "order");
  assert.equal(periodOf(new URLSearchParams("from=2026-01-01&until=2027-01-03")).problem, "span");
  assert.equal(periodOf(new URLSearchParams("from=2026-01-01&until=2027-01-02")).problem, "");
  assert.equal(periodOf(new URLSearchParams("until=2026-03-01")).problem, "");
});

test("bounds come from the server today and the hints", () => {
  assert.equal(minDay("2026-11-16", "2026-12-01"), "2026-12-01");
  assert.equal(minDay("2026-11-16", "2026-11-01"), "2026-11-16");
  assert.equal(minDay("2026-11-16", ""), "2026-11-16");
  assert.equal(minDay("", "2026-12-01"), "2026-12-01");
  assert.deepEqual(lifecycleBounds("2026-11-16", "2026-11-20"), { min: "2026-11-20", max: addDays("2026-11-16", 3660) });
  assert.deepEqual(lifecycleBounds("bad", ""), { min: "bad", max: "" });
  assert.deepEqual(settleBounds("2026-11-16", "2026-10-01"), { min: "2026-10-01", max: "2026-11-16" });
  assert.deepEqual(settleBounds("2026-11-16", ""), { min: "", max: "2026-11-16" });
  assert.equal(dayProblem("x", "", ""), "format");
  assert.equal(dayProblem("2026-01-01", "2026-02-01", ""), "min");
  assert.equal(dayProblem("2026-03-01", "", "2026-02-01"), "max");
  assert.equal(dayProblem("2026-02-01", "2026-02-01", "2026-02-01"), "");
  assert.equal(coversEnd("2026-10-01", "2026-10-01"), true);
  assert.equal(coversEnd("2026-09-30", "2026-10-01"), false);
  assert.equal(coversEnd("", "2026-10-01"), false);
  assert.equal(coversEnd("", ""), true);
});

test("hints fail closed", () => {
  const detail = contractDetailOf({ actions: { "contract.suspend": { allowed: true, blocked: "", earliest: "2026-11-16" }, "contract.end": { allowed: true, blocked: "", earliest: "" } } });
  assert.equal(offered(actionOf(detail, "contract.suspend"), true), true);
  assert.equal(offered(actionOf(detail, "contract.end"), true), false);
  assert.equal(offered(actionOf(detail, "contract.end"), false), true);
  assert.equal(offered(actionOf(detail, "contract.resume")), false);
  assert.equal(offered(undefined), false);
  assert.equal(actionOf(null, "contract.settle").blocked, "missing");
  const odd = { ...detail, actions: { ...detail.actions, "contract.settle": { allowed: true, blocked: "open_gaps", earliest: "" } } } as ContractDetail;
  assert.deepEqual(actionOf(odd, "contract.settle"), { allowed: false, blocked: "open_gaps", earliest: "" });
  const gone = { ...detail, actions: {} } as unknown as ContractDetail;
  assert.equal(actionOf(gone, "contract.renew").blocked, "missing");
});

test("every blocker has a sentence and unknown codes fall back", () => {
  for (const b of knownBlockers) {
    assert.ok(tables.blockerWords[b], b);
    for (const l of ["en", "ko", "ar"] as const) assert.ok(!k.blocked(l, "contract.settle", b).includes("for this contract now"), b + l);
  }
  assert.match(k.blocked("en", "contract.suspend", "brand_new"), /not available for this contract now/);
  assert.match(k.blocked("en", "contract.suspend", ""), /not available for this contract now/);
  assert.match(k.blocked("en", "contract.suspend", "unknown"), /not available for this contract now/);
  assert.match(k.eventBlocked("en", "missing"), /Not available for this event now/);
  assert.match(k.blocked("en", "contract.suspend", "not_active"), /^Suspend is not available: the contract is not active/);
});

test("zero and empty values are words, never 0", () => {
  assert.equal(revisionText(0), "na");
  assert.equal(revisionText(null), "unknown");
  assert.equal(revisionText(2), "2");
  assert.equal(actorText("", "seed"), "seed");
  assert.equal(actorText("a@x", "u1"), "a@x");
  assert.equal(k.state("en", "weird"), "Unknown state (weird)");
  assert.equal(k.settlement("en", "none"), "Not applicable");
  assert.equal(k.path("en", ""), "Not recorded");
  assert.equal(k.path("ko", "resale"), "재판매");
  assert.equal(k.gap("ar", "x"), "سبب غير معروف (x)");
  assert.equal(k.record("en", "batch"), "Lifecycle change");
  assert.equal(k.cycleWord("en", ""), "Not recorded");
  assert.equal(k.cycleWord("en", "monthly"), "Monthly");
  assert.equal(k.cycleWord("en", "weekly"), "weekly");
});

test("failure reactions and field targets", () => {
  assert.equal(react("invalid", ["from"]), "field");
  assert.equal(react("invalid", []), "reason");
  assert.equal(react("refused", []), "hint");
  assert.equal(react("stage", []), "hint");
  assert.equal(react("conflict", []), "conflict");
  assert.equal(react("denied", []), "denied");
  assert.equal(react("unauthenticated", []), "signin");
  assert.equal(react("limited", []), "wait");
  assert.equal(react("busy", []), "retry");
  assert.equal(react("unavailable", []), "retry");
  assert.equal(react("stale", []), "renew");
  assert.equal(react("partial", []), "partial");
  assert.equal(react("other", []), "failed");
  assert.deepEqual([keepsAttempt("retry"), keepsAttempt("wait"), keepsAttempt("hint"), keepsAttempt("conflict")], [true, false, false, false]);
});

test("reason limit counts UTF-8 bytes", () => {
  assert.equal(utf8Bytes("abc"), 3);
  assert.equal(utf8Bytes("가"), 3);
  assert.equal(utf8Bytes("ع"), 2);
  assert.equal(reasonTooLong("가".repeat(85)), false);
  assert.equal(reasonTooLong("가".repeat(86)), true);
  assert.equal(reasonTooLong("a".repeat(256)), false);
  assert.equal(idProblem("cw-03-gap"), false);
  assert.equal(idProblem("https://x"), true);
  assert.equal(idProblem(""), true);
  assert.equal(idProblem("a".repeat(129)), true);
});

test("attempts survive remounts and reloads until a definite answer", () => {
  forget();
  setScope("ws", "u1", "s1");
  let n = 0;
  const fresh = () => ({ request: "r" + ++n, key: "k" + n });
  const first = pendingFor("contract.suspend", "c1", { from: "2026-12-01", rationale: "x" }, "r10", fresh);
  settle("contract.suspend", "c1", react("busy", []));
  const again = pendingFor("contract.suspend", "c1", { from: "2026-12-01", rationale: "x" }, "r11", fresh);
  assert.deepEqual([again.key, again.expected], [first.key, "r10"]);
  assert.ok(pendingNow("contract.suspend", "c1"));
  const edited = pendingFor("contract.suspend", "c1", { from: "2026-12-02", rationale: "x" }, "r11", fresh);
  assert.notEqual(edited.key, first.key);
  assert.equal(edited.expected, "r11");
  settle("contract.suspend", "c1", react("stage", []));
  assert.equal(pendingNow("contract.suspend", "c1"), undefined);
  pendingFor("contract.end", "c1", { from: "2026-12-05" }, "r12", fresh);
  settle("contract.end", "c1", "ok");
  assert.equal(pendingNow("contract.end", "c1"), undefined);
  pendingFor("contract.end", "c2", { from: "2026-12-05" }, "r12", fresh);
  settle("contract.end", "c2", react("busy", []));
  assert.ok(pendingNow("contract.end", "c2"));
  setScope("ws", "u1", "s1");
  assert.ok(pendingNow("contract.end", "c2"));
  setScope("ws", "u1", "s2");
  assert.equal(pendingNow("contract.end", "c2"), undefined);
  pendingFor("contract.end", "c2", { from: "2026-12-05" }, "r12", fresh);
  setScope("ws", "u2", "s2");
  assert.equal(pendingNow("contract.end", "c2"), undefined);
  pendingFor("contract.end", "c3", { from: "2026-12-05" }, "r12", fresh);
  settle("contract.end", "c3", react("limited", []));
  assert.equal(pendingNow("contract.end", "c3"), undefined);
  pendingFor("contract.end", "c4", { from: "2026-12-05" }, "r12", fresh);
  discard("contract.end", "c4", { from: "2026-12-05" });
  assert.ok(pendingNow("contract.end", "c4"));
  discard("contract.end", "c4", { from: "2026-12-06" });
  assert.equal(pendingNow("contract.end", "c4"), undefined);
  forget();
});

test("drafts survive remounts per command and contract and clear on sign-out", () => {
  forget();
  setScope("ws", "u1", "s1");
  assert.deepEqual(draftOf("contract.settle", "c1"), {});
  keepDraft("contract.settle", "c1", { as_of: "2026-10-01" });
  assert.deepEqual(draftOf("contract.settle", "c1"), { as_of: "2026-10-01" });
  assert.deepEqual(draftOf("contract.settle", "c2"), {});
  clearDraft("contract.settle", "c1");
  assert.deepEqual(draftOf("contract.settle", "c1"), {});
  keepDraft("contract.reopen", "c1", { reason: "correction" });
  forget();
  assert.deepEqual(draftOf("contract.reopen", "c1"), {});
});

test("withdrawable events and open gaps come from the server fields", () => {
  const d = contractDetailOf({
    events: [{ id: "e1", withdraw_blocked: "" }, { id: "e2", withdraw_blocked: "settled" }, { id: "e3" }],
    evidence: [{ write: "a", kind: "missing", resolved: false }, { write: "b", kind: "missing", resolved: true }, { write: "c", kind: "secured" }],
  });
  assert.deepEqual(d.events.map(withdrawable), [true, false, false]);
  assert.deepEqual(openGaps(d.evidence).map((e) => e.write), ["a"]);
});

test("command bodies match the API for every form", () => {
  const ev = { id: "e9", kind: "end", from: "2026-12-01", revision: 0, withdraw_blocked: "", cancel_required: true };
  for (const key of ["contract.suspend", "contract.resume", "contract.renew", "contract.end"] as const) {
    assert.deepEqual(bodyFor(key, { from: "2026-12-01", rationale: "r" }), { from: "2026-12-01" });
    assert.equal(commandFor(key), key);
  }
  assert.deepEqual(bodyFor("contract.withdraw", { cancel: "contract_continues" }, ev), { event: "e9", cancel: "contract_continues" });
  assert.deepEqual(bodyFor("contract.withdraw", { cancel: "contract_continues" }, { ...ev, cancel_required: false }), { event: "e9" });
  assert.deepEqual(bodyFor("contract.withdraw", {}), { event: "" });
  assert.deepEqual(bodyFor("contract.evidence.secured", { reference: "r1", covered_until: "2026-10-01", resolves: "", reason: "x", follow_up: "y" }), { kind: "secured", reference: "r1", covered_until: "2026-10-01" });
  assert.deepEqual(bodyFor("contract.evidence.secured", { reference: "r1", covered_until: "2026-10-01", resolves: "a,b" }).resolves, "a,b");
  assert.deepEqual(bodyFor("contract.evidence.missing", { reason: "not_attempted", follow_up: "f", reference: "x" }), { kind: "missing", reason: "not_attempted", follow_up: "f" });
  assert.deepEqual(bodyFor("contract.settle", { as_of: "2026-10-01", rationale: "r" }), { as_of: "2026-10-01" });
  assert.deepEqual(bodyFor("contract.reopen", { reason: "correction", rationale: "r" }), { reason: "correction" });
  assert.deepEqual(bodyFor("contract.evidence.missing", {}), { kind: "missing", reason: "", follow_up: "" });
  assert.equal(commandFor("contract.evidence.secured"), "contract.evidence");
  assert.equal(commandFor("contract.evidence.missing"), "contract.evidence");
});

test("server fields map to form controls and the rationale", () => {
  for (const f of ["from", "cancel", "reference", "covered_until", "resolves", "reason", "follow_up", "as_of"]) assert.equal(controlFor(f), f);
  for (const f of ["kind", "rationale", "expected_revision", "target_id", ""]) assert.equal(controlFor(f), "rationale");
});

test("withdraw needs both the event and the action hint", () => {
  const d = contractDetailOf({ events: [{ id: "e1", withdraw_blocked: "" }, { id: "e2", withdraw_blocked: false }], actions: { "contract.withdraw": { allowed: true, blocked: "" } } });
  assert.deepEqual(d.events.map((e) => canWithdraw(d, e)), [true, false]);
  assert.equal(d.events[1].withdraw_blocked, "missing");
  const blocked = contractDetailOf({ events: [{ id: "e1", withdraw_blocked: "" }], actions: { "contract.withdraw": { allowed: false, blocked: "write_limit" } } });
  assert.equal(canWithdraw(blocked, blocked.events[0]), false);
});

test("an end day reads as ended only once it has passed", () => {
  assert.equal(endPassed("2026-11-16", "2026-11-16"), true);
  assert.equal(endPassed("2026-11-17", "2026-11-16"), false);
  assert.equal(endPassed("", "2026-11-16"), false);
  assert.equal(endPassed("2026-11-16", ""), false);
});

test("names fall back to the ID, never blank", async () => {
  const { nameOf } = await import("./live-format");
  const parties = [{ id: "le-1", name: "North", kind: "legal_entity" as const, parent: "", state: "active" }, { id: "le-2", name: "", kind: "legal_entity" as const, parent: "", state: "active" }];
  assert.equal(nameOf("le-1", parties), "North");
  assert.equal(nameOf("le-2", parties), "le-2");
  assert.equal(nameOf("le-9", parties), "le-9");
  assert.equal(nameOf("", parties), "");
});

test("a step-up refusal or cancellation keeps the attempt and the form", () => {
  for (const f of ["reauth", "reauth_cancelled"]) {
    assert.equal(react(f, []), "reauth");
    assert.equal(keepsAttempt(react(f, [])), true);
  }
});
