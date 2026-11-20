import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import { qcopy as t } from "./query-copy";
import { w } from "../users/live-copy";
import { v } from "./live-copy";
import { AdminError } from "../../../../service/admin";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function")
  nodeModule.registerHooks({
    load: (url, context, next) =>
      assetStub.test(url)
        ? { format: "module", source: "export default ''", shortCircuit: true }
        : next(url, context),
  });
else
  nodeModule.register(
    "data:text/javascript," +
      encodeURIComponent(
        `export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`,
      ),
  );
const { Reauthenticate } = await import("./command-ui");
const { ReadBoundary } = await import("./query-ui");

test("M04 read failures describe reads and service accounts while retaining safe server references", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    for (const failure of ["unavailable", "denied"] as const) {
      const code =
        failure === "unavailable"
          ? "dependency_unavailable"
          : "permission_denied";
      const html = renderToStaticMarkup(
        <ReadBoundary
          l={l}
          read={{
            state: "error",
            error: new AdminError(
              failure,
              [],
              0,
              "",
              code,
              "qa-read-reference",
            ),
          }}
          reload={() => {}}
          restart={() => {}}
        >
          {() => <p>Must not display prior rows</p>}
        </ReadBoundary>,
      );
      assert.ok(
        html.includes(v.loadFailed(l).replaceAll("'", "&#x27;")),
        "read title describes another operation",
      );
      if (failure === "denied")
        assert.ok(
          !html.includes(v.denied(l)),
          "uniform missing/foreign detail refusal must not claim installation-wide authority loss",
        );
      assert.ok(
        html.includes(code) && html.includes("qa-read-reference"),
        "safe read diagnostic lost",
      );
      assert.ok(html.includes(w.reload(l)) && html.includes(t(l, "first")));
      assert.ok(!html.includes("Must not display prior rows"));
    }
  }
});

test("M04 prerequisite loss and unavailable evidence offer distinct passive recovery in all locales", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    let refreshes = 0;
    const render = (
      state:
        | "ready"
        | "reauthentication_required"
        | "prerequisite_failed"
        | "unavailable",
    ) =>
      renderToStaticMarkup(
        <Reauthenticate
          l={l}
          state={state}
          refresh={() => {
            refreshes++;
          }}
        />,
      );
    assert.equal(render("ready"), "");
    const prerequisite = render("prerequisite_failed"),
      unavailable = render("unavailable"),
      reauth = render("reauthentication_required");
    assert.notEqual(
      prerequisite,
      unavailable,
      "prerequisite loss and evidence failure were conflated",
    );
    for (const html of [prerequisite, unavailable]) {
      assert.ok(
        html.includes(w.reload(l)),
        "blocked metadata needs explicit refresh",
      );
      assert.ok(
        !html.includes(">" + t(l, "reauth") + "</button>"),
        "blocked metadata must not offer an authentication loop",
      );
    }
    assert.ok(reauth.includes(t(l, "reviewAgain")));
    assert.equal(refreshes, 0, "render must never refresh or submit");
  }
});
