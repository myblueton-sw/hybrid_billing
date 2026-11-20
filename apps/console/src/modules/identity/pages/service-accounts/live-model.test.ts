import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type CredentialRow } from "../../../../service/admin";
import { attemptState, credentialState, formVisible, keepsHold, nextPhase, outsideWindow, panelShown, pickupLabel, pinIssue, returnFocus, serviceState, settleFailure, validName, wholeDays } from "./live-model";
import { v } from "./live-copy";

const cred = (change: Partial<CredentialRow>): CredentialRow => ({ id: "cred-1", not_before: "2026-11-16T00:00:00Z", not_after: "2026-12-16T00:00:00Z", state: "active", usable: true, ...change });
test("names mirror the API rule", () => {
  for (const ok of ["abc", "a-b-c", "0x9", "a".repeat(48)]) assert.equal(validName(ok), true, ok);
  for (const bad of ["", "ab", "a".repeat(49), "-ab", "ab-", "a--b", "Abc", "a_b", "a b", "é12"]) assert.equal(validName(bad), false, bad);
});

test("days are whole and bounded", () => {
  assert.equal(wholeDays("30", 30), 30);
  assert.equal(wholeDays("1", 30), 1);
  for (const raw of ["0", "31", "", "01", "1.5", "-1", "1234"]) assert.equal(wholeDays(raw, 30), null, raw);
});

test("credential labels follow usable and the account state", () => {
  assert.equal(credentialState("en", cred({}))[0], v.cActive("en"));
  assert.equal(credentialState("en", cred({ usable: false }))[0], v.cOnHold("en"));
  assert.equal(credentialState("en", cred({ usable: false }), true)[0], v.cRevokeRequired("en"));
  assert.equal(credentialState("en", cred({ state: "revoked" }), true)[0], v.cRevoked("en"));
  assert.equal(credentialState("en", cred({ state: "expired" }))[0], v.cExpired("en"));
});

test("an unknown issue outcome holds the attempt and only a definitive answer releases it", () => {
  for (const f of ["busy", "unavailable"]) {
    assert.equal(keepsHold(f, false), true, f);
    assert.equal(keepsHold(f, true), true, f);
  }
  assert.equal(keepsHold("partial", false), true);
  assert.equal(keepsHold("partial", true), false);
  for (const f of ["limited", "unauthenticated"]) {
    assert.equal(keepsHold(f, true), true, f);
    assert.equal(keepsHold(f, false), false, f);
  }
  for (const f of ["conflict", "stale", "refused", "stage", "denied", "invalid"]) {
    assert.equal(keepsHold(f, true), false, f);
    assert.equal(keepsHold(f, false), false, f);
  }
});

test("a submitted issue stays on screen when the layout narrows", () => {
  const pinned = pinIssue("svc-a");
  assert.equal(pinned.chosen, "svc-a");
  for (const wide of [true, false]) assert.equal(panelShown(wide, true, pinned.open), true, String(wide));
  assert.equal(panelShown(false, true, false), false);
  assert.equal(panelShown(false, false, false), true);
});

const counter = () => {
  let n = 0;
  return () => { n += 1; return { request: `r${n}`, key: `k${n}` }; };
};

test("a reload while an issue is in flight keeps its attempt for the retry", () => {
  const t = attemptState(counter());
  const first = t.send("rev-1");
  t.revised();
  t.keep("rev-1");
  t.idle();
  t.revised();
  const retry = t.send("rev-2");
  assert.deepEqual(retry.attempt, first.attempt);
  assert.equal(retry.expected, "rev-1");
});

test("an attempt is renewed by a reload when idle, after a success and on renew", () => {
  const t = attemptState(counter());
  const a = t.send("rev-1").attempt;
  t.idle();
  t.revised();
  const b = t.send("rev-1").attempt;
  assert.notDeepEqual(b, a);
  t.succeeded();
  t.idle();
  const c = t.send("rev-2").attempt;
  assert.notDeepEqual(c, b);
  t.succeeded();
  t.keep("rev-2");
  t.idle();
  assert.deepEqual(t.send("rev-3"), { attempt: c, expected: "rev-2" });
  t.idle();
  t.renew();
  const d = t.send("rev-3");
  assert.notDeepEqual(d.attempt, c);
  assert.equal(d.expected, "rev-3");
});

const fail = (failure: string, code = "") => new AdminError(failure as AdminError["failure"], [], 0, "", code);

test("a user renew is refused while a request is in flight, so the retry keeps the key (F1)", () => {
  const t = attemptState(counter());
  const first = t.send("rev-1");
  assert.equal(t.sending(), true);
  assert.equal(t.renew(), false);
  t.keep("rev-1");
  t.idle();
  const retry = t.send("rev-2");
  assert.deepEqual(retry.attempt, first.attempt);
  assert.equal(retry.expected, "rev-1");
});

test("the failure path leaves sending before it renews, for every form (N3)", () => {
  for (const [form, hold] of [["IssueForm", true], ["CreateForm", true], ["StateStep", false], ["RevokeStep", false]] as const) {
    const t = attemptState(counter());
    const a = t.send("rev-1").attempt;
    assert.equal(settleFailure(t, fail("refused"), hold, "rev-1"), "remove", form);
    assert.equal(t.sending(), false, form);
    assert.notDeepEqual(t.send("rev-2").attempt, a, form);
  }
  const held = attemptState(counter());
  const a = held.send("rev-1").attempt;
  assert.equal(settleFailure(held, fail("busy"), true, "rev-1"), "unknown");
  assert.deepEqual(held.send("rev-2"), { attempt: a, expected: "rev-1" });
  assert.equal(held.renew(), false);
  const plain = attemptState(counter());
  const p = plain.send("rev-1").attempt;
  assert.equal(settleFailure(plain, fail("reauth_cancelled"), false, "rev-1"), "kept");
  assert.deepEqual(plain.send("rev-1").attempt, p);
  assert.equal(settleFailure(plain, fail("busy"), false, "rev-1"), "remove");
});

test("every answer has a next phase after a first send and after a retry (F2)", () => {
  const rows: [string, string, string, string, string][] = [
    ["busy", "", "unknown", "unknown", "unknown"],
    ["unavailable", "", "unknown", "unknown", "unknown"],
    ["partial", "", "unknown", "lapsed", "lapsed"],
    ["limited", "", "unknown", "unknown", "unknown"],
    ["unauthenticated", "", "unknown", "unknown", "unknown"],
    ["reauth", "reauthentication_required", "remove", "unknown", "unknown"],
    ["reauth_cancelled", "", "remove", "unknown", "unknown"],
    ["conflict", "revision_conflict", "remove", "remove", "lapsed"],
    ["stale", "idempotency_conflict", "lapsed", "lapsed", "lapsed"],
    ["stale", "idempotency_key_expired", "lapsed", "lapsed", "lapsed"],
    ["refused", "", "remove", "remove", "lapsed"],
    ["stage", "", "remove", "remove", "lapsed"],
    ["invalid", "", "remove", "remove", "lapsed"],
    ["denied", "", "remove", "lapsed", "lapsed"],
    ["wrong", "", "lapsed", "lapsed", "lapsed"],
    ["locked", "", "lapsed", "lapsed", "lapsed"],
    ["form", "", "lapsed", "lapsed", "lapsed"],
    ["self", "", "lapsed", "lapsed", "lapsed"],
    ["epoch", "", "lapsed", "lapsed", "lapsed"],
    ["approval_stale", "", "lapsed", "lapsed", "lapsed"],
    ["evidence_stale", "", "lapsed", "lapsed", "lapsed"],
    ["needs_approval", "", "lapsed", "lapsed", "lapsed"],
  ];
  for (const [failure, code, first, inside, outside] of rows) {
    const e = fail(failure, code);
    assert.equal(nextPhase(false, e, false), first, `${failure} ${code} first`);
    assert.equal(nextPhase(true, e, false), inside, `${failure} ${code} retry inside`);
    assert.equal(nextPhase(true, e, true), outside, `${failure} ${code} retry outside`);
  }
  assert.equal(nextPhase(true, new TypeError("x"), true), "unknown");
});

test("the replay window keeps a margin and an unknown window lapses a refused retry", () => {
  assert.equal(outsideWindow(3539999, 3600), false);
  assert.equal(outsideWindow(3540000, 3600), true);
  assert.equal(outsideWindow(8999, 10), false);
  assert.equal(outsideWindow(9000, 10), true);
  assert.equal(outsideWindow(1e12, 0), false);
  assert.equal(outsideWindow(0, 0, true), true);
});

test("focus returns to the button that opened the revoke, else the view heading (F7)", () => {
  for (const from of ['[data-credential="cred-1"]', '[data-unstored-revoke="cred-1"]', '[data-never-revoke="cred-1"]', '[data-attempt-revoke="svc-a"]']) {
    assert.deepEqual(returnFocus(from, true), [from, "#svc-cred-title"]);
    assert.deepEqual(returnFocus(from, false), [from, "#svc-list-title"]);
  }
  assert.deepEqual(returnFocus("", true), ["#svc-cred-title"]);
});

test("the pickup sub-line and the never-collected list", () => {
  const at = "2026-11-19T10:10:00Z";
  assert.equal(pickupLabel("en", cred({ secret_pickup: "never_collected" }))?.never, true);
  const open = pickupLabel("ko", cred({ secret_pickup: "open", secret_pickup_until: at }));
  assert.equal(open?.never, false);
  assert.ok(open?.time);
  assert.equal(pickupLabel("en", cred({ secret_pickup: "collected" })), null);
  assert.equal(pickupLabel("en", cred({})), null);
  assert.equal(pickupLabel("en", cred({ state: "revoked", secret_pickup: "never_collected" })), null);
  assert.equal(pickupLabel("en", cred({ secret_pickup: "open" })), null);

});

test("the issue form is visible only when all five conditions hold (N7)", () => {
  const base = { view: "credentials", mode: "form", current: "svc-a", service: "svc-a", wide: true, open: false, visibility: "visible" };
  assert.equal(formVisible(base), true);
  assert.equal(formVisible({ ...base, wide: false, open: true }), true);
  for (const change of [{ view: "accounts" }, { mode: "reveal" }, { current: "svc-b" }, { wide: false }, { visibility: "hidden" }]) {
    assert.equal(formVisible({ ...base, ...change }), false, JSON.stringify(change));
  }
});
