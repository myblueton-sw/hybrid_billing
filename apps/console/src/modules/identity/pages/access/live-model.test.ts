import { test } from "node:test";
import assert from "node:assert/strict";
import { ledgerOf, type AccessGrant, type AccessPage } from "../../../../service/admin";
import { accessText, ceilingGrants, closed, delegationsOf, maxDays, people, reasonText, revocable, scopeText, stateLine, wholeDays } from "./live-model";
import { a, accessPhrase, actionLabel } from "./live-copy";

const grant = (change: Partial<AccessGrant>): AccessGrant => ({
  id: "g", subject: "u", client: "", action: "view", kind: "focus_data", fields: ["rows"],
  scope: { customer: "all", legal_entity: "all", contract: "all", period: "any" },
  not_before: "2026-11-16T00:00:00Z", not_after: "2026-12-16T00:00:00Z", state: "active", effective: true, version: 1, ...change,
});
const page = (grants: AccessGrant[]): AccessPage => ({
  revision: "r1", grants, delegations: [], catalog: [], subjects: [], now: "2026-11-16T09:00:00Z", you: "", started_at: "", settings: [], requests: [], revocations: [], ledger: ledgerOf({ state: "ok" }),
});

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
  const ds = delegationsOf({ ...p, delegations: [{ ...d, id: "d1" }, { ...d, id: "d2", state: "revoked" }, { ...d, id: "d3", state: "pending" }, { ...d, id: "d4", state: "ended" }] });
  assert.deepEqual(ds.current.map((x) => x.id), ["d1", "d3"]);
  assert.deepEqual(ds.ended.map((x) => x.id), ["d2", "d4"]);
});

test("unconfirmed revocations stay open for grants and delegations, and unknown states read unconfirmed", () => {
  const p = page([grant({ id: "a" }), grant({ id: "r", state: "revocation_unconfirmed" }), grant({ id: "x", state: "revoked" }), grant({ id: "q", state: "future" as AccessGrant["state"] })]);
  const split = people(p);
  assert.deepEqual(split.current.map((g) => g.id), ["a", "q", "r"]);
  assert.deepEqual(split.ended.map((g) => g.id), ["x"]);
  const d = { ...grant({}), delegator: "a", delegate: "b", parent: "", redelegate: false };
  const ds = delegationsOf({ ...p, delegations: [{ ...d, id: "d1", state: "revocation_unconfirmed" }, { ...d, id: "d2", state: "revoked" }, { ...d, id: "d3", state: "odd" as "ended" }] });
  assert.deepEqual(ds.current.map((x) => x.id), ["d1", "d3"]);
  assert.deepEqual(ds.ended.map((x) => x.id), ["d2"]);
  const want = { en: "Revocation not confirmed", ko: "회수 미확정", ar: "الإبطال غير مؤكَّد" } as const;
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(stateLine(l, grant({ state: "revocation_unconfirmed" }))[0], want[l]);
    assert.equal(stateLine(l, grant({ state: "something-new" as AccessGrant["state"] }))[0], want[l]);
    assert.equal(stateLine(l, { ...d, state: "revocation_unconfirmed" })[0], want[l]);
  }
  assert.equal(stateLine("en", grant({}))[0], "Active");
  assert.equal(stateLine("en", grant({ effective: false }))[0], "Does not apply");
  assert.equal(stateLine("en", grant({ state: "revoked" }))[0], "Revoked");
  assert.equal(stateLine("en", grant({ state: "expired" }))[0], "Expired");
  assert.equal(stateLine("en", grant({ state: "pending" }))[0], "Not yet valid");
  assert.ok(revocable("active") && revocable("pending") && !revocable("revocation_unconfirmed") && !revocable("revoked"));
  assert.ok(closed("ended") && !closed("revocation_unconfirmed"));
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

test("counts and day hints read with counted nouns", () => {
  assert.equal(a.activeGrants("en", 1), "1 active grant");
  assert.equal(a.activeGrants("en", 2), "2 active grants");
  assert.equal(a.activeDelegations("en", 1), "1 active delegation");
  assert.equal(a.activeDelegations("ar", 2), "تفويضان ساريان");
  assert.equal(a.activeGrants("ar", 11), "11 صلاحية سارية");
  assert.equal(a.activeGrants("ko", 3), "유효 3건");
  assert.equal(a.days("ar"), "عدد الأيام");
  assert.match(a.daysHint("en", 1, "d"), /^1 to 1 day\./);
  assert.match(a.daysHint("ar", 29, "d"), /من 1 إلى 29 يومًا/);
  assert.match(a.daysHint("ar", 2, "d"), /من 1 إلى يومين\./);
  assert.match(a.daysHint("ko", 29, "d"), /^1~29일\./);
  assert.equal(a.effectFor("ar", "x"), "الأثر على x");
});


test("candidate cost access has localized scope and an exact request phrase", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const text = accessText(l, "view_cost", ["charge_evidence"]);
    assert.ok(!text.includes("charge_evidence"));
    assert.ok(!scopeText(l, "charge_candidate", "").includes("charge_candidate"));
    assert.ok(accessPhrase(l, "view_cost", "charge_candidate", ["charge_evidence"]));
    assert.equal(accessPhrase(l, "view", "charge_candidate", ["charge_evidence"]), null);
    assert.equal(accessPhrase(l, "view_cost", "charge_candidate", ["charge_evidence", "rows"]), null);
  }
});


test("grant capabilities preserve identity across reorder and refuse removed or expired selection", async () => {
  const { capabilityKey, grantCapabilities, selectedCapability } = await import("./live-model");
  const focus = { action: "view", kind: "focus_data", fields: ["rows"], until: "2026-12-16T00:00:00Z" };
  const cost = { action: "view_cost", kind: "charge_candidate", fields: ["charge_evidence"], until: "2026-12-01T00:00:00Z" };
  const current = { now: "2026-11-16T00:00:00Z", catalog: [cost, focus] };
  assert.equal(selectedCapability(current, ""), undefined);
  assert.equal(selectedCapability(current, capabilityKey(focus)), focus);
  assert.equal(selectedCapability({ ...current, catalog: [focus, cost] }, capabilityKey(focus)), focus);
  assert.equal(selectedCapability({ ...current, catalog: [cost] }, capabilityKey(focus)), undefined);
  const later = { ...cost, until: "2026-12-10T00:00:00Z" };
  assert.deepEqual(grantCapabilities({ ...current, catalog: [cost, later, focus] }), [later, focus]);
  assert.deepEqual(grantCapabilities({ ...current, catalog: [later, cost, focus] }), [later, focus]);
  const expired = { ...cost, until: current.now };
  assert.deepEqual(grantCapabilities({ ...current, catalog: [expired, focus] }), [focus]);
  assert.equal(selectedCapability({ ...current, catalog: [expired, focus] }, capabilityKey(cost)), undefined);
  assert.deepEqual(grantCapabilities({ ...current, catalog: [{ ...cost, action: "issue" }, focus] }), [focus]);
  assert.equal(capabilityKey({ ...focus, fields: ["b", "a"] }), capabilityKey({ ...focus, fields: ["a", "b"] }));
});


test("approval review is a distinct read capability alongside FOCUS and candidate costs", async () => {
  const { capabilityKey, grantCapabilities, selectedCapability } = await import("./live-model");
  const approval = { action: "view", kind: "approval_review", fields: ["approval_evidence"], until: "2026-12-16T00:00:00Z" };
  const focus = { ...approval, kind: "focus_data", fields: ["rows"] };
  const cost = { ...approval, action: "view_cost", kind: "charge_candidate", fields: ["charge_evidence"] };
  const page = { now: "2026-11-16T00:00:00Z", catalog: [approval, cost, focus] };
  assert.equal(grantCapabilities(page).length, 3);
  for (const capability of page.catalog) assert.equal(selectedCapability(page, capabilityKey(capability)), capability);
  for (const l of ["en", "ko", "ar"] as const) {
    assert.ok(!accessText(l, "view", ["approval_evidence"]).includes("approval_evidence"));
    assert.ok(!scopeText(l, "approval_review", "").includes("approval_review"));
    assert.ok(accessPhrase(l, "view", "approval_review", ["approval_evidence"]));
    assert.equal(accessPhrase(l, "request_approval", "approval_review", ["approval_evidence"]), null);
    assert.equal(accessPhrase(l, "view_cost", "approval_review", ["approval_evidence"]), null);
    assert.equal(accessPhrase(l, "view", "approval_review", ["charge_evidence"]), null);
  }
});
test("synthetic input confirmation is labeled without enabling generic requests", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    assert.ok(!accessText(l, "confirm_input", []).includes("confirm_input"));
    assert.ok(!scopeText(l, "synthetic_input_evidence", "").includes("synthetic_input_evidence"));
    assert.equal(accessPhrase(l, "confirm_input", "synthetic_input_evidence", []), null);
 assert.equal(accessPhrase(l, "view_cost", "synthetic_input_evidence", ["input_review_evidence"]), null);
 assert.ok(!accessText(l, "view_cost", ["input_review_evidence"]).includes("input_review_evidence"));
  }
});


test("pricing grant labels remain separate from generic grant offers", () => {
 for (const l of ["en", "ko", "ar"] as const) {
  for (const action of ["propose_pricing_policy", "approve_pricing_policy", "revoke_pricing_policy", "use_pricing_policy"]) {
   assert.ok(!accessText(l, action, []).includes(action));
   assert.equal(accessPhrase(l, action, "synthetic_pricing_policy", []), null);
  }
  assert.ok(!scopeText(l, "synthetic_pricing_policy", "").includes("synthetic_pricing_policy"));
  assert.ok(!accessText(l, "view_cost", ["pricing_policy_evidence"]).includes("pricing_policy_evidence"));
  assert.equal(accessPhrase(l, "view_cost", "synthetic_pricing_policy", ["pricing_policy_evidence"]), null);
 }
});


test("document authority uses explicit localized action and evidence labels", () => {
 for (const l of ["en", "ko", "ar"] as const) {
  for (const action of ["propose_document_evidence", "confirm_document_evidence", "revoke_document_evidence", "prepare_erp_record"]) {
   assert.ok(!accessText(l, action, []).includes(action));
  }
  assert.ok(!accessText(l, "view_cost", ["document_evidence"]).includes("document_evidence"));
  assert.equal(accessPhrase(l, "view_cost", "synthetic_document_evidence", ["document_evidence"]), null);
 }
});

test("ERP result authority uses explicit localized action and evidence labels", () => {
 for (const l of ["en", "ko", "ar"] as const) {
  for (const action of ["propose_erp_result", "confirm_erp_result", "revoke_erp_result", "apply_erp_result"]) {
   assert.ok(!accessText(l, action, []).includes(action));
  }
  assert.ok(!accessText(l, "view_cost", ["erp_result_evidence"]).includes("erp_result_evidence"));
  assert.equal(accessPhrase(l, "view_cost", "synthetic_erp_result_evidence", ["erp_result_evidence"]), null);
 }
});
