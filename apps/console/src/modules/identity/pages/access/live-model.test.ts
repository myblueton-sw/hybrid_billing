import { test } from "node:test";
import assert from "node:assert/strict";
import type { AccessGrant, AccessPage } from "../../../../service/admin";
import { accessText, ceilingGrants, delegationsOf, maxDays, people, reasonText, scopeText, wholeDays } from "./live-model";
import { a, accessPhrase, actionLabel } from "./live-copy";

const grant = (change: Partial<AccessGrant>): AccessGrant => ({
  id: "g", subject: "u", client: "", action: "view", kind: "focus_data", fields: ["rows"],
  scope: { customer: "all", legal_entity: "all", contract: "all", period: "any" },
  not_before: "2026-11-16T00:00:00Z", not_after: "2026-12-16T00:00:00Z", state: "active", effective: true, version: 1, ...change,
});
const page = (grants: AccessGrant[]): AccessPage => ({ revision: "r1", grants, delegations: [], catalog: [], subjects: [], now: "2026-11-16T09:00:00Z" });

test("grants split into current, ended and ceiling", () => {
  const p = page([
    grant({ id: "c", state: "pending" }), grant({ id: "b", effective: false }), grant({ id: "a" }),
    grant({ id: "x", state: "revoked" }), grant({ id: "y", state: "expired" }), grant({ id: "k", subject: "", client: "console" }),
    grant({ id: "k2", subject: "", client: "console", state: "expired" }),
  ]);
  const split = people(p);
  assert.deepEqual(split.current.map((g) => g.id), ["a", "c", "b"]);
  assert.deepEqual(split.ended.map((g) => g.id), ["x", "y"]);
  assert.deepEqual(ceilingGrants(p).map((g) => g.id), ["k"]);
  const d = { ...grant({}), delegator: "a", delegate: "b", parent: "", redelegate: false };
  const ds = delegationsOf({ ...p, delegations: [{ ...d, id: "d1" }, { ...d, id: "d2", state: "revoked" }, { ...d, id: "d3", state: "pending" }] });
  assert.deepEqual(ds.current.map((x) => x.id), ["d1", "d3"]);
  assert.deepEqual(ds.ended.map((x) => x.id), ["d2"]);
});

test("days are floored and strict", () => {
  assert.equal(maxDays("2026-12-16T09:00:00Z", "2026-11-16T09:00:00Z"), 30);
  assert.equal(maxDays("2026-12-16T08:59:59Z", "2026-11-16T09:00:00Z"), 29);
  assert.equal(maxDays("bad", "2026-11-16T09:00:00Z"), 0);
  assert.equal(maxDays("2026-11-15T00:00:00Z", "2026-11-16T09:00:00Z"), 0);
  assert.equal(wholeDays("30", 30), 30);
  for (const raw of ["0", "31", "", "01", "1.5", "-1", "12345"]) assert.equal(wholeDays(raw, 30), null, raw);
});

test("labels and reasons", () => {
  assert.equal(accessText("en", "view", ["rows"]), "View · Rows");
  assert.equal(accessText("en", "issue", []), "Issue");
  assert.equal(accessText("en", "mystery", ["other"]), "mystery · other");
  assert.equal(scopeText("ko", "focus_data", "all"), "수집된 FOCUS 데이터 · 모든 고객");
  assert.equal(scopeText("en", "usage", "c1"), "usage · c1");
  assert.equal(actionLabel("ar", "issue"), "إصدار");
  assert.equal(accessPhrase("en", "view", "focus_data", ["rows"]), "view rows of collected FOCUS data");
  assert.equal(accessPhrase("en", "issue", "focus_data", []), null);
  const base = { action: "view", field: "rows", allowed: false, delegations: [] as string[] };
  assert.equal(reasonText("en", { ...base, allowed: true, reason: "allowed" }), a.whyAllowed("en"));
  assert.equal(reasonText("en", { ...base, allowed: true, reason: "allowed", delegations: ["d1", "d2"] }), a.whyChain("en", "d1 → d2"));
  const reasons = ["no_matching_grant", "client_not_allowed", "principal_not_current", "invalid_request", "stamp_not_current", "authority_unavailable", "odd"];
  const texts = reasons.map((reason) => reasonText("en", { ...base, reason }));
  assert.equal(new Set(texts).size, reasons.length);
  assert.match(texts[6], /odd\)$/);
});
