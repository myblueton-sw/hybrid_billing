import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import { detailOf } from "../../../../service/identity-bulk";
import { b } from "./bulk-copy";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { PreviewBody } = await import("./BulkPrepare");

const detail = detailOf({ job: { id: "bulk-revocation-2", state: "prepared", effect: "suspend_membership", preparer: "user-a", counts: { targets: 3 } }, impact: {}, exclusions: {}, outside_scope: [] });
const api = {} as AdminClient;

test("a list read by someone other than its preparer shows no false count of people", () => {
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(b.ownerOnlyPeople(l).replace(/'/g, "&#x27;")), l);
    assert.ok(!html.includes(b.toRevokeN(l, 0)), l);
    assert.ok(!html.includes(b.showing(l, 0, 0)), l);
  }
});

test("the preparer sees the people on the list", () => {
  const html = renderToStaticMarkup(<PreviewBody l="en" api={api} prepared={{ detail, targets: { total: 1, next: "", rows: [{ user_id: "u1", login: "ann@example.invalid", role: "member", role_until: "", user_state: "active", membership_state: "active", installation_administrator: false }] } }} onMore={() => {}} />);
  assert.ok(html.includes("ann@example.invalid") && html.includes(b.toRevokeN("en", 1)) && !html.includes(b.ownerOnlyPeople("en")));
});


test("input confirmation permission impact has a localized label, count and recheck meaning", () => {
  const labels = { en: "Input data permission requests", ko: "입력 데이터 권한 요청", ar: "طلبات صلاحية بيانات الإدخال" };
  const withInput = { ...detail, impact: { ...detail.impact, pending: [{ store: "input_authority_proposals", count: 2, rechecked: true }] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withInput, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">2</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(!html.includes("input_authority_proposals"), l);
  }
});

test("pricing policy permission impact has a localized label, count and recheck meaning", () => {
  const labels = { en: "Pricing policy permission requests", ko: "가격 정책 권한 요청", ar: "طلبات صلاحية سياسة التسعير" };
  const withInput = { ...detail, impact: { ...detail.impact, pending: [{ store: "pricing_authority_proposals", count: 2, rechecked: true }] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withInput, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">2</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(!html.includes("pricing_authority_proposals"), l);
  }
});

test("pricing policy proposal impact has a localized label, count and recheck meaning", () => {
  const labels = { en: "Pricing policy proposals", ko: "가격 정책 제안", ar: "مقترحات سياسة التسعير" };
  const withInput = { ...detail, impact: { ...detail.impact, pending: [{ store: "pricing_policy_proposals", count: 2, rechecked: true }] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withInput, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">2</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(!html.includes("pricing_policy_proposals"), l);
  }
});

test("charge approval policy proposal impact has a localized label, count and recheck meaning", () => {
  const labels = { en: "Charge approval policy proposals", ko: "청구 승인 정책 제안", ar: "مقترحات سياسة اعتماد الرسوم" };
  const withInput = { ...detail, impact: { ...detail.impact, pending: [{ store: "financial_policy_proposals", count: 2, rechecked: true }] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withInput, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">2</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(!html.includes("financial_policy_proposals"), l);
  }
});

test("ERP final policy impact remains distinct from charge policy in every locale", () => {
  const labels = { en: "ERP final-amount approval policy proposals", ko: "ERP 최종 금액 승인 정책 제안", ar: "مقترحات سياسة اعتماد المبلغ النهائي في ERP" };
  const withPolicies = { ...detail, impact: { ...detail.impact, pending: [
    { store: "erp_final_policy_proposals", count: 1, rechecked: true },
    { store: "financial_policy_proposals", count: 2, rechecked: true },
  ] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withPolicies, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">1</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(html.includes(`<dt>${b.store(l, "financial_policy_proposals")}</dt><dd><span class="m01-line">2</span>`), l);
    assert.ok(!html.includes("erp_final_policy_proposals"), l);
  }
});
test("ERP final limit impact remains distinct from charge policy in every locale", () => {
  const labels = { en: "ERP final-amount approval limit proposals", ko: "ERP 최종 금액 승인 한도 제안", ar: "مقترحات حدود اعتماد المبلغ النهائي في ERP" };
  const withPolicies = { ...detail, impact: { ...detail.impact, pending: [
    { store: "erp_final_authority_proposals", count: 1, rechecked: true },
    { store: "financial_authority_proposals", count: 2, rechecked: true },
  ] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withPolicies, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">1</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(html.includes(`<dt>${b.store(l, "financial_authority_proposals")}</dt><dd><span class="m01-line">2</span>`), l);
    assert.ok(!html.includes("erp_final_authority_proposals"), l);
  }
});

test("charge approval limit proposal impact has a localized label, count and recheck meaning", () => {
  const labels = { en: "Charge approval limit proposals", ko: "청구 승인 한도 제안", ar: "مقترحات حدود اعتماد الرسوم" };
  const withInput = { ...detail, impact: { ...detail.impact, pending: [{ store: "financial_authority_proposals", count: 2, rechecked: true }] } };
  for (const l of ["en", "ko", "ar"] as Locale[]) {
    const html = renderToStaticMarkup(<PreviewBody l={l} api={api} prepared={{ detail: withInput, targets: null }} onMore={() => {}} />);
    assert.ok(html.includes(`<dt>${labels[l]}</dt><dd><span class="m01-line">2</span><span class="m01-line">${b.rechecked(l)}</span></dd>`), l);
    assert.ok(!html.includes("financial_authority_proposals"), l);
  }
});
