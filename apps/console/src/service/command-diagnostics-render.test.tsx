import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { CommandDiagnostics, ErrorDiagnostics } from "./CommandDiagnostics";
import { AdminError, adminClient } from "./admin";

for (const l of ["en", "ko", "ar"] as const) {
  test(`${l} diagnostics distinguish response, client and missing references`, () => {
    const html = renderToStaticMarkup(<CommandDiagnostics l={l} detail={{ code: "prerequisite_failed", request: "server-330", client: "client-330", target: "plugin:sample", cancelled: false }} />);
    for (const value of ["prerequisite_failed", "server-330", "client-330", "plugin:sample"]) assert.ok(html.includes(`>${value}</bdi>`));
    assert.ok(html.includes('dir="ltr"'));
    assert.ok(html.includes('role="status"'));
    const lost = renderToStaticMarkup(<ErrorDiagnostics l={l} error={new Error("PRIVATE exception")} />);
    assert.ok(!lost.includes("PRIVATE"));
    assert.ok(!lost.includes(">unavailable<"));
    assert.ok(lost.includes({ en: "Not available", ko: "확인할 수 없음", ar: "غير متاح" }[l]));
    assert.equal(renderToStaticMarkup(<CommandDiagnostics l={l} detail={null} />), "");
  });
}

test("non-object JSON failures never fabricate response diagnostic fields", async () => {
  for (const body of [null, [], 42, "plain text"]) {
    const fetcher = (async () => new Response(JSON.stringify(body), { status: 503, headers: { "X-Request-ID": "not-a-json-reference" } })) as typeof fetch;
    await assert.rejects(adminClient("t", fetcher).integrations(), (e: AdminError) => e.code === "" && e.request === "");
  }
});
