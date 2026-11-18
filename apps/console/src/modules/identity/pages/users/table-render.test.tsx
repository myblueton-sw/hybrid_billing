import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { UserRow } from "../../../../service/admin";
import { w } from "./live-copy";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { Table } = await import("./Live");

const langs: Locale[] = ["en", "ko", "ar"];
const none: UserRow = { user_id: "u-none", login: "nomember", email: "", state: "active", epoch: 1, membership: null, password_set: true, invitation_expires: "", link_purpose: "", link_expires: "", installation_administrator: false, revocation: null };
const sel = { on: false, selected: { mode: "ids", ids: [] }, self: "", atLimit: false, loaded: [], onToggle: () => {}, onAll: () => {} } as never;
const cells = (html: string) => [...html.matchAll(/<td role="cell" data-label="([^"]*)"[^>]*>(.*?)<\/td>/g)].map((x) => ({ label: x[1], body: x[2] }));
const hidden = (s: string) => `<span aria-hidden="true">—</span><span class="visually-hidden">${s}</span>`;

test("the Membership tab role cells show a dash and a hidden No role when there is no membership", () => {
  for (const l of langs) {
    const html = renderToStaticMarkup(<Table l={l} view="membership" users={[none]} selected="" onOpen={() => {}} sel={sel} serverTime="2026-11-19T10:00:00Z" />);
    const row = cells(html);
    const role = row.find((c) => c.label === w.workspaceRole(l));
    assert.ok(role, l);
    assert.equal(role.body, hidden(w.noRole(l)), l);
    const status = row.find((c) => c.label === w.membershipStatus(l));
    assert.equal(status?.body, w.noMembership(l), l);
    assert.equal(row.filter((c) => c.body.includes(w.noMembership(l))).length, 1, l);
    const combined = row[row.length - 1];
    assert.ok(combined.body.startsWith(hidden(w.noRole(l))), l);
  }
});

test("the Users tab keeps No membership in the role cell because it has no membership status column", () => {
  for (const l of langs) {
    const html = renderToStaticMarkup(<Table l={l} view="users" users={[none]} selected="" onOpen={() => {}} sel={sel} serverTime="2026-11-19T10:00:00Z" />);
    const role = cells(html).find((c) => c.label === w.workspaceRole(l));
    assert.equal(role?.body, w.noMembership(l), l);
  }
});

test("No role reads the same in every language", () => {
  assert.equal(w.noRole("en"), "No role");
  assert.equal(w.noRole("ko"), "역할 없음");
  assert.equal(w.noRole("ar"), "بلا دور");
});
