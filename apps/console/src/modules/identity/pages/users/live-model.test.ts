import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type UserRow } from "../../../../service/admin";
import { accountName, failureText, offers, roleName, setup, when } from "./live-model";
import { w } from "./live-copy";
import { c } from "../../../../service/copy";

const base: UserRow = {
  user_id: "u1", login: "a@example.invalid", email: "a@example.invalid", state: "active", epoch: 1,
  membership: { role: "member", state: "active", version: 1 }, password_set: true, invitation_expires: "",
  link_purpose: "", link_expires: "", installation_administrator: false,
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

test("membership actions follow the membership state", () => {
  assert.deepEqual(offers("en", base, "x").membership, ["change_role", "suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: { role: "member", state: "suspended", version: 2 } }), "x").membership, ["reinstate_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: null }), "x").membership, ["invite"]);
  assert.deepEqual(offers("en", row({ membership: null, state: "suspended" }), "x").membership, []);
  assert.deepEqual(offers("en", row({ link_purpose: "reset", link_expires: "2026-11-20T00:00:00Z" }), "x").membership, ["suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: { role: "administrator", state: "active", version: 1 }, link_purpose: "set" }), "x").membership, ["change_role", "suspend_membership", "remove_membership"]);
  assert.deepEqual(offers("en", row({ membership: null, invitation_expires: "2026-11-20T00:00:00Z" }), "x").membership, []);
});

test("sign-in actions follow the password and role", () => {
  assert.deepEqual(offers("en", row({ password_set: false }), "x").sign, ["issue_set_link"]);
  assert.deepEqual(offers("en", row({ password_set: false, login: "" }), "x").sign, []);
  assert.deepEqual(offers("en", base, "x").sign, ["issue_reset_link"]);
  const admin = offers("en", row({ membership: { role: "administrator", state: "active", version: 1 } }), "x");
  assert.deepEqual(admin.sign, []);
  assert.ok(admin.notes.includes(w.adminResetReason("en")));
  const suspended = offers("en", row({ state: "suspended" }), "x");
  assert.deepEqual(suspended.sign, []);
  assert.ok(suspended.notes.includes(w.suspendedResetReason("en")));
  assert.deepEqual(offers("en", row({ password_set: false, invitation_expires: "2026-11-20T00:00:00Z" }), "x").sign, ["issue_set_link", "revoke_invitation"]);
  assert.ok(offers("en", row({ link_purpose: "reset", link_expires: "2026-11-20T00:00:00Z" }), "x").notes.includes(w.promotionReason("en")));
});

test("setup text names the outstanding link before the password state", () => {
  assert.match(setup("en", row({ link_purpose: "set", link_expires: "2026-11-20T09:00:00Z" })), /^Set link issued, expires /);
  assert.match(setup("en", row({ link_purpose: "reset", link_expires: "2026-11-20T09:00:00Z" })), /^Reset link issued/);
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
