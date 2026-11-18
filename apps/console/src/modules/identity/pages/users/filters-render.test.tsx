import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import { fc } from "./filter-copy";
import { noFilters, type UserFilters } from "./filter-model";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { FilterBand, PagingArea, ScopeLine, chipText } = await import("./filters");

const locales: Locale[] = ["en", "ko", "ar"];
const noop = () => undefined;
const plain = (html: string) => html.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/\s+/g, " ");
const auth = { connected: ["password"], not_connected: ["saml", "oidc", "ldap"] };
const band = (l: Locale, view: string, filters: UserFilters = noFilters, wide = true, open = false) =>
  renderToStaticMarkup(<FilterBand l={l} view={view} wide={wide} open={open} filters={filters} days={14} auth={auth} onToggle={noop} onChange={noop} onRemove={noop} onClearAll={noop} />);

test("the band has seven controls and states the sign-in method as text in every language", () => {
  for (const l of locales) {
    const html = band(l, "users");
    const text = plain(html);
    for (const id of ['id="m01-q"', 'id="m01-st"', 'id="m01-ms"', 'id="m01-role"', 'id="m01-per"', 'id="m01-si"', 'id="m01-adm"', 'role="search"', 'dir="auto"']) assert.ok(html.includes(id), `${l} ${id}`);
    assert.equal((html.match(/<option value="/g) ?? []).length, 5 * 1 + 3 + 4 + 2 + 4 + 4, l);
    assert.ok(text.includes(fc.methodLabel(l)) && text.includes(fc.method(l, auth)), l);
    assert.ok(!/<p class="m01-method">.*?(<select|<input|<button)/.test(html), l);
    assert.ok(text.includes(fc.periodSoon(l, 14)) && text.includes(fc.periodBounded(l, 14)) && text.includes(fc.membershipEnded(l)), l);
    assert.ok(!html.includes("m01-applied") && !html.includes("m01-filters-toggle") && !html.includes("hidden=\"\""), l);
  }
});

test("chips name each filter, offer removal and clear all, and keep the fixed invitations chip", () => {
  const f = { ...noFilters, q: "김민", role: "member", per: "ends_soon", adm: true };
  for (const l of locales) {
    const html = band(l, "membership", f);
    const text = plain(html);
    assert.ok(text.includes(fc.appliedFilters(l)) && text.includes(fc.clearAll(l)), l);
    for (const k of ["q", "role", "per", "adm"]) assert.ok(html.includes(`id="m01-chip-${k}"`), `${l} ${k}`);
    assert.ok(html.includes(`aria-label="${fc.removeFilter(l, chipText(l, "membership", "role", f, 14)).replace(/"/g, "&quot;")}"`), l);
    assert.ok(chipText(l, "membership", "per", f, 14).endsWith(fc.withMembership(l)) && !chipText(l, "users", "per", f, 14).endsWith(fc.withMembership(l)), l);
    assert.ok(html.includes('<bdi dir="auto">김민</bdi>'), l);
    const inv = band(l, "invitations");
    assert.ok(inv.includes("m01-chip-fixed") && !inv.includes("m01-clear-all") && !inv.includes("<button type=\"button\" id=\"m01-chip"), l);
  }
});

test("below 1280 the band collapses behind a toggle that counts applied filters, and chips stay visible", () => {
  for (const l of locales) {
    const html = band(l, "users", { ...noFilters, st: "active" }, false, false);
    assert.ok(html.includes('aria-expanded="false"') && html.includes('hidden=""') && html.includes(fc.filtersApplied(l, 1)), l);
    assert.ok(html.indexOf("m01-applied") > html.indexOf("m01-filter-body"), l);
    assert.ok(band(l, "users", noFilters, false, true).includes('aria-expanded="true"'), l);
  }
});

test("the paging area words counts, scan stops, failures and restarts truthfully", () => {
  for (const l of locales) {
    const area = (over: Partial<Parameters<typeof PagingArea>[0]>) => plain(renderToStaticMarkup(<PagingArea l={l} next="" nextLogin="" stopped={false} paging="idle" changed={false} count="c"
      compact={false} onMore={noop} onReload={noop} onRestart={noop} {...over} />));
    assert.equal(area({}).trim(), "");
    const more = area({ next: "a:u1" });
    assert.ok(more.includes("Show more users") === (l === "en") && !more.includes(fc.continueSearching(l)), l);
    const stopped = area({ next: "a:u1", nextLogin: "ann@example.invalid", stopped: true });
    const squash = (t: string) => t.replace(/\s+/g, "");
    assert.ok(squash(stopped).includes(squash(fc.searchedUpTo(l).replace("{login}", "ann@example.invalid"))) && stopped.includes(fc.continueSearching(l)), l);
    const departed = area({ next: "d:user-9", nextLogin: "", stopped: true });
    assert.ok(squash(departed).includes(squash(fc.searchedUser(l).replace("{id}", "user-9"))), l);
    assert.ok(area({ next: "a:u1", paging: "failed" }).includes(fc.nextFailed(l)), l);
    const restart = area({ next: "a:u1", paging: "restart" });
    assert.ok(restart.includes(fc.restart(l)[0]) && restart.includes(fc.startAgain(l)), l);
    const changed = area({ next: "a:u1", changed: true });
    assert.ok(fc.changed(l).every((s) => changed.includes(s)), l);
    assert.ok(area({ next: "a:u1", compact: true }).includes("c"), l);
    for (const n of [0, 1, 7]) assert.ok(!/of \d|total|\ball\b/i.test(fc.shownMore(l, n) + fc.shownAll(l, n)), l);
    assert.ok(plain(renderToStaticMarkup(<ScopeLine l={l} workspace="sample-workspace" />)).includes(fc.scopeLabel(l)), l);
  }
});

test("copy follows the technical-writing review", () => {
  assert.equal(fc.method("en", auth), "Everyone signs in with a password set at this installation. Single sign-on (SAML, OIDC) and directory sign-in (LDAP) are not connected.");
  assert.equal(fc.method("ko", auth), "모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다. SSO(SAML·OIDC)와 디렉터리 로그인(LDAP)은 연결되어 있지 않습니다.");
  assert.equal(fc.method("ar", auth), "يسجّل الجميع الدخول بكلمة مرور معيّنة في هذا التثبيت. الدخول الموحد (SAML وOIDC) والدخول عبر الدليل (LDAP) غير متصلَين.");
  assert.equal(fc.method("en", { connected: ["password", "oidc"], not_connected: ["ldap"] }), "Connected sign-in methods: PASSWORD and OIDC. Directory sign-in (LDAP) is not connected.");
  assert.equal(fc.method("en", { connected: [], not_connected: [] }), "");
  assert.equal(fc.periodSoon("en", 1), "Ends within 1 day");
  assert.equal(fc.periodSoon("en", 14), "Ends within 14 days");
  assert.equal(fc.periodSoon("ar", 1), "تنتهي خلال يوم واحد");
  assert.equal(fc.periodSoon("ar", 3), "تنتهي خلال 3 أيام");
  assert.equal(fc.periodSoon("ar", 14), "تنتهي خلال 14 يومًا");
  assert.equal(fc.periodBounded("en", 14), "Ends in more than 14 days");
  assert.equal(fc.periodBounded("ko", 14), "14일 이후 종료");
  assert.equal(fc.updated("en", "3 shown; end of list", 1), "List updated. 3 shown; end of list. Filters applied: 1.");
  assert.equal(fc.jobsNote("en"), "Filters apply to the Users, Invitations and Membership tabs.");
  assert.equal(fc.jobsNote("ko"), "필터는 '사용자', '초대', '소속·역할' 탭에 적용됩니다.");
  assert.equal(fc.jobsNote("ar"), "تُطبَّق عوامل التصفية على علامات التبويب «المستخدمون» و«الدعوات» و«العضوية».");
});

test("round 2 copy: no double stops, KO particles, AR agreement, set link times, active option", async () => {
  const { optionName } = await import("./filters");
  for (const l of locales) {
    for (const count of [fc.noneShown(l), fc.shownAll(l, 1), fc.shownMore(l, 7)]) {
      for (const k of [0, 1, 2]) {
        for (const said of [fc.updated(l, count, k), fc.removed(l, count)]) assert.ok(!/\.\s*\.|。\s*\./.test(said), `${l} ${said}`);
      }
    }
    assert.ok(/[.]$/.test(fc.jobsNote(l)) || /다\.$/.test(fc.jobsNote(l)), l);
  }
  assert.equal(fc.periodBounded("ko", 14), "14일 이후 종료");
  assert.equal(fc.setLinkAt("en"), "Password set link issued, expires {at}");
  assert.equal(fc.setLinkAt("ko"), "비밀번호 설정 링크 발급됨 · {at} 만료");
  assert.equal(fc.setLinkAt("ar"), "صدر رابط تعيين كلمة المرور، ينتهي في {at}");
  assert.equal(fc.resetLinkAt("en"), "Password reset link issued, expires {at}");
  assert.equal(optionName("ar", "ms", "active", 14), "نشطة");
  assert.equal(optionName("en", "ms", "active", 14), "Active");
  const one = { connected: ["password"], not_connected: ["ldap"] };
  const three = { connected: ["password"], not_connected: ["saml", "ldap", "kerberos"] };
  assert.equal(fc.method("en", one), "Everyone signs in with a password set at this installation. Directory sign-in (LDAP) is not connected.");
  assert.equal(fc.method("ko", one), "모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다. 디렉터리 로그인(LDAP)은 연결되어 있지 않습니다.");
  assert.equal(fc.method("ar", one), "يسجّل الجميع الدخول بكلمة مرور معيّنة في هذا التثبيت. الدخول عبر الدليل (LDAP) غير متصل.");
  assert.equal(fc.method("ar", auth).endsWith("غير متصلَين."), true);
  assert.equal(fc.method("ar", three).endsWith("غير متصلة."), true);
  assert.equal(fc.method("ko", { connected: ["password"], not_connected: ["saml"] }), "모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다. SSO(SAML)는 연결되어 있지 않습니다.");
  assert.equal(fc.method("ko", three), "모든 사용자는 이 설치에서 설정한 비밀번호로 로그인합니다. SSO(SAML), 디렉터리 로그인(LDAP)과 KERBEROS는 연결되어 있지 않습니다.");
});

test("the select-loaded control is not rendered when nothing is loaded", async () => {
  const { SelectionBar, HeaderCheck } = await import("./Selection");
  const bar = (loaded: string[]) => renderToStaticMarkup(<SelectionBar l="en" selected={new Set(["u1"])} loaded={loaded} limit={500} atLimit={false} names={{}} onPrepare={noop} onClear={noop} onRemove={noop} onAll={noop} />);
  assert.ok(bar(["u2"]).includes("data-select-loaded"));
  assert.ok(!bar([]).includes("data-select-loaded") && bar([]).includes("m01-notloaded") && bar([]).includes("m01-selnote"));
  const cell = renderToStaticMarkup(<table><thead><tr><HeaderCheck l="en" selected={new Set()} loaded={[]} onAll={noop} cell /></tr></thead></table>);
  assert.ok(cell.includes('<th scope="col" class="m01-check"></th>'), cell);
});
