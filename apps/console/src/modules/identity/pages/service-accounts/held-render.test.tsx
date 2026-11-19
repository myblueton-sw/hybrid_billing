import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { AdminError } from "../../../../service/admin";
import { HeldBlock } from "./Held";

for (const l of ["en", "ko", "ar"] as const) {
  for (const id of ["svc", "svc-create"]) {
    test(`${l} ${id} unknown outcomes keep diagnostic references without claiming failure`, () => {
      const html = renderToStaticMarkup(<HeldBlock l={l} id={id} phase="unknown" lead="Outcome unknown" lapsed={<p>Lapsed</p>} focusKey={0}
        error={new AdminError("busy", [], 0, "", "dependency_unavailable")} request="qa-held-ref" />);
      assert.ok(html.includes("dependency_unavailable"));
      assert.ok(html.includes("qa-held-ref"));
      assert.ok(!html.includes('role="alert"'));
      assert.ok(html.includes(`id="${id}-held-lead"`));
    });
  }
}
