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
