import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type Membership, type UserRow } from "../../../../service/admin";
import { accountName, failureText, offers, roleName, setup, when } from "./live-model";
import { w } from "./live-copy";
import { c } from "../../../../service/copy";

const mem = (role: "member" | "administrator", state: "active" | "suspended", version: number, extra: Partial<Membership> = {}): Membership => ({
  role, state, version, role_from: "", role_until: "", role_period: "open", suspended_for: "", ledger: "", ...extra,
});

const base: UserRow = {
  user_id: "u1", login: "a@example.invalid", email: "a@example.invalid", state: "active", epoch: 1,
  membership: mem("member", "active", 1), password_set: true, invitation_expires: "",
  link_purpose: "", link_expires: "", installation_administrator: false, revocation: null,
};
const row = (change: Partial<UserRow>): UserRow => ({ ...base, ...change });

test("self and departed accounts offer nothing", () => {
  assert.equal(offers("en", base, "u1").blocked, w.selfReason("en"));
  assert.equal(offers("en", row({ state: "departed", membership: null }), "x").blocked, w.departedReason("en"));
});

test("account actions follow the account state", () => {
  assert.deepEqual(offers("en", base, "x").account, ["suspend_user", "depart_user"]);
  assert.deepEqual(offers("en", row({ state: "suspended" }), "x").account, ["reinstate_user", "depart_user"]);
});

test("an unrecognised account or membership state reads as unknown and offers no action", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(accountName(l, "locked"), w.unknownState(l));
    assert.notEqual(accountName(l, "locked"), w.active(l));
    assert.equal(accountName(l, "active"), w.active(l));
  }
  const odd = offers("en", row({ state: "locked" as UserRow["state"] }), "x");
  assert.deepEqual([odd.account, odd.membership, odd.sign, odd.blocked], [[], [], [], w.unknownStateReason("en")]);
  const member = offers("en", row({ membership: { ...mem("member", "active", 1), state: "unknown" } }), "x");
  assert.deepEqual(member.membership, []);
  assert.ok(member.notes.includes(w.unknownStateReason("en")));
  assert.deepEqual(member.account, ["suspend_user", "depart_user"]);
  const ended = offers("en", row({ membership: { ...mem("member", "active", 1), state: "unknown", role_period: "ended" } }), "x");
  assert.deepEqual(ended.membership, []);
});

test("an unrecognised role reads as unknown and offers no membership or sign-in link action", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(roleName(l, "unknown"), w.unknownRole(l));
    assert.equal(roleName(l, ""), w.noMembership(l));
    assert.equal(roleName(l, "owner"), w.unknownRole(l));
  }
  const odd = offers("en", row({ membership: { ...mem("member", "active", 1), role: "unknown" } }), "x");
  assert.deepEqual([odd.membership, odd.sign, odd.notes], [[], [], [w.unknownRoleReason("en")]]);
  assert.deepEqual(odd.account, ["suspend_user", "depart_user"]);
  const unset = offers("en", row({ password_set: false, membership: { ...mem("member", "active", 1), role: "unknown" } }), "x");
  assert.deepEqual(unset.sign, []);
});

test("membership actions follow the membership state", () => {
  assert.deepEqual(offers("en", base, "x").membership, ["change_role", "set_role_period", "suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: mem("member", "suspended", 2) }), "x").membership, ["reinstate_membership", "set_role_period", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: null }), "x").membership, ["invite"]);
  assert.deepEqual(offers("en", row({ membership: null, state: "suspended" }), "x").membership, []);
  assert.deepEqual(offers("en", row({ link_purpose: "reset", link_expires: "2026-11-20T00:00:00Z" }), "x").membership, ["set_role_period", "suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: mem("administrator", "active", 1), link_purpose: "set" }), "x").membership, ["change_role", "set_role_period", "suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: null, invitation_expires: "2026-11-20T00:00:00Z" }), "x").membership, []);
});

test("sign-in actions follow the password and role", () => {
  assert.deepEqual(offers("en", row({ password_set: false }), "x").sign, ["issue_set_link"]);
  assert.deepEqual(offers("en", row({ password_set: false, login: "" }), "x").sign, []);
  assert.deepEqual(offers("en", base, "x").sign, ["issue_reset_link"]);
  const admin = offers("en", row({ membership: mem("administrator", "active", 1) }), "x");
  assert.deepEqual(admin.sign, []);
  assert.ok(admin.notes.includes(w.adminResetReason("en")));
  const suspended = offers("en", row({ state: "suspended" }), "x");
  assert.deepEqual(suspended.sign, []);
  assert.ok(suspended.notes.includes(w.suspendedResetReason("en")));
  assert.deepEqual(offers("en", row({ password_set: false, invitation_expires: "2026-11-20T00:00:00Z" }), "x").sign, ["issue_set_link", "revoke_invitation"]);
  assert.ok(offers("en", row({ link_purpose: "reset", link_expires: "2026-11-20T00:00:00Z" }), "x").notes.includes(w.promotionReason("en")));
});

test("setup text names the outstanding link before the password state", () => {
  assert.match(setup("en", row({ link_purpose: "set", link_expires: "2026-11-20T09:00:00Z" })), /^Password set link issued, expires /);
  assert.match(setup("en", row({ link_purpose: "reset", link_expires: "2026-11-20T09:00:00Z" })), /^Password reset link issued, expires /);
  assert.equal("setIssued" in w || "resetIssued" in w, false);
  assert.match(setup("en", row({ password_set: false, invitation_expires: "2026-11-20T09:00:00Z" })), /^Invitation pending until /);
  assert.equal(setup("en", base), w.passwordSet("en"));
  assert.equal(setup("en", row({ password_set: false })), w.noPassword("en"));
});

test("names and dates", () => {
  assert.equal(roleName("ko", "administrator"), "관리자");
  assert.equal(roleName("ar", "member"), "عضو");
  assert.equal(roleName("en", ""), "No membership");
  assert.equal(accountName("en", "suspended"), "Suspended");
  assert.equal(accountName("en", "departed"), "Departed");
  assert.equal(when("en", "not a date"), "");
  assert.notEqual(when("ko", "2026-11-20T09:00:00Z"), "");
});

test("failures have distinct text", () => {
  const kinds = ["conflict", "refused", "stage", "denied", "unauthenticated", "limited", "partial", "unavailable", "busy", "stale"] as const;
  const bodies = kinds.map((k) => failureText("en", new AdminError(k, [], 3, "op-1")).body);
  assert.equal(new Set(bodies).size, kinds.length);
  assert.match(failureText("en", new AdminError("partial", [], 0, "op-9")).body, /op-9/);
  assert.equal(failureText("en", new AdminError("refused")).title, w.refusedTitle("en"));
  assert.equal(failureText("en", new AdminError("invalid")).body, "");
  assert.equal(failureText("en", new Error("x")).title, w.saveFailed("en"));
  assert.equal(failureText("en", new AdminError("conflict")).title, w.saveFailed("en"));
  assert.equal(failureText("en", new AdminError("denied")).title, w.deniedTitle("en"));
  assert.equal(failureText("en", new AdminError("invalid")).title, "Check the following");
});

test("step-up refusals and cancellations have neutral copy in every locale", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    for (const f of ["reauth", "reauth_cancelled"] as const) {
      assert.deepEqual(failureText(l, new AdminError(f)), { title: w.saveFailed(l), body: f === "reauth" ? c.reauthRefused(l) : c.reauthNotDone(l) });
    }
    assert.notEqual(failureText(l, new AdminError("reauth")).body, failureText(l, new AdminError("reauth_cancelled")).body);
  }
});

test("a period-ended membership offers a new period and removal only", () => {
  for (const m of [mem("member", "suspended", 3, { suspended_for: "role_period_ended", role_period: "ended" }), mem("administrator", "active", 2, { role_period: "ended" })]) {
    assert.deepEqual(offers("en", row({ membership: m }), "x").membership, ["set_role_period", "remove_membership"]);
  }
});
