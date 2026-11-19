import { test } from "node:test";
import assert from "node:assert/strict";
import type { CredentialRow, ServiceRow, ServicesPage } from "../../../../service/admin";
import { attemptState, credentials, credentialState, initialService, keepsHold, issueBlock, liveCount, panelShown, pinIssue, serviceState, split, validName, wholeDays } from "./live-model";
import { v } from "./live-copy";

const cred = (change: Partial<CredentialRow>): CredentialRow => ({ id: "cred-1", not_before: "2026-11-16T00:00:00Z", not_after: "2026-12-16T00:00:00Z", state: "active", usable: true, ...change });
const svc = (change: Partial<ServiceRow>): ServiceRow => ({
  id: "svc-a", state: "active", version: 1, creator: "u1", creator_login: "a@x", owner: "u1", owner_login: "a@x", credentials: [], ...change,
});
const page = (services: ServiceRow[]): ServicesPage => ({ revision: "r", services, owners: [], limits: { credential_days: 30, credentials_per_service: 2, services: 200 }, now: "2026-11-16T00:00:00Z" });

test("names mirror the API rule", () => {
  for (const ok of ["abc", "a-b-c", "0x9", "a".repeat(48)]) assert.equal(validName(ok), true, ok);
  for (const bad of ["", "ab", "a".repeat(49), "-ab", "ab-", "a--b", "Abc", "a_b", "a b", "é12"]) assert.equal(validName(bad), false, bad);
});

test("days are whole and bounded", () => {
  assert.equal(wholeDays("30", 30), 30);
  assert.equal(wholeDays("1", 30), 1);
  for (const raw of ["0", "31", "", "01", "1.5", "-1", "1234"]) assert.equal(wholeDays(raw, 30), null, raw);
});

test("states, groups and limits", () => {
  const live = svc({ id: "svc-b", credentials: [cred({}), cred({ id: "cred-2", state: "revoked" }), cred({ id: "cred-3", state: "expired" })] });
  const p = page([svc({ id: "svc-r", state: "retired" }), svc({ id: "svc-s", state: "suspended" }), live]);
  assert.deepEqual(split(p).current.map((s) => s.id), ["svc-s", "svc-b"]);
  assert.deepEqual(split(p).retired.map((s) => s.id), ["svc-r"]);
  assert.equal(liveCount(live), 1);
  assert.deepEqual(credentials(live).current.map((c) => c.id), ["cred-1"]);
  assert.deepEqual(credentials(live).ended.map((c) => c.id), ["cred-2", "cred-3"]);
  assert.equal(issueBlock(svc({ state: "retired" }), 2), "retired");
  assert.equal(issueBlock(svc({ state: "suspended" }), 2), "suspended");
  assert.equal(issueBlock(svc({ credentials: [cred({}), cred({ id: "c2" })] }), 2), "limit");
  assert.equal(issueBlock(live, 2), "");
  assert.equal(serviceState("en", "retired"), v.stRetired("en"));
  assert.equal(serviceState("ko", "suspended"), v.stSuspended("ko"));
  assert.equal(serviceState("ar", "active"), v.stActive("ar"));
});

test("credential labels follow usable and the account state", () => {
  assert.equal(credentialState("en", cred({}))[0], v.cActive("en"));
  assert.equal(credentialState("en", cred({ usable: false }))[0], v.cOnHold("en"));
  assert.equal(credentialState("en", cred({ usable: false }), true)[0], v.cRevokeRequired("en"));
  assert.equal(credentialState("en", cred({ state: "revoked" }), true)[0], v.cRevoked("en"));
  assert.equal(credentialState("en", cred({ state: "expired" }))[0], v.cExpired("en"));
});

test("the initial service prefers the wanted one, then an active one", () => {
  const p = page([svc({ id: "svc-r", state: "retired" }), svc({ id: "svc-s", state: "suspended" }), svc({ id: "svc-a" })]);
  assert.equal(initialService(p, "svc-r"), "svc-r");
  assert.equal(initialService(p, "ghost"), "svc-a");
  assert.equal(initialService(page([svc({ id: "svc-r", state: "retired" }), svc({ id: "svc-s", state: "suspended" })]), ""), "svc-s");
  assert.equal(initialService(page([svc({ id: "svc-r", state: "retired" })]), ""), "svc-r");
  assert.equal(initialService(page([]), ""), "");
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
  t.renew();
  t.idle();
  const d = t.send("rev-3");
  assert.notDeepEqual(d.attempt, c);
  assert.equal(d.expected, "rev-3");
});
