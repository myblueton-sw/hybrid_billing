import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import type { HistoryEvent } from "../../../../service/access-history";

const assets = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => assets.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assets}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { HistoryRow } = await import("./AccessHistory");
const event: HistoryEvent = {
  at: "2026-11-19T12:34:56Z", kind: "signed_in", reason: "", workspace: "w", count: 0,
  session: { handle: "0123456789abcdef", current: true, active: true }, device: { browser: "chrome", os: "macos" },
};
for (const l of ["en", "ko", "ar"] as const) {
  test(`${l} history row isolates date and time into independent runs`, () => {
    const html = renderToStaticMarkup(<HistoryRow l={l} e={event} />);
    assert.equal((html.match(/<bdi dir="ltr" class="mysec-nw">/g) ?? []).length, 2);
    assert.ok(!/[\u061c\u200e\u200f]/.test(html), 'generated LTR dates must not carry embedded directional marks');
    assert.ok(html.includes('role="row"'));
    assert.equal((html.match(/role="cell"/g) ?? []).length, 4);
  });
}
