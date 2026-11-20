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

const { CommandFailure } = await import("./command-failure");
test("M04 command refusals use service context and preserve safe server references", () => {
  const messages = {
    en: "its expected revision no longer matches",
    ko: "예상 리비전이 현재 리비전과 일치하지 않아",
    ar: "المراجعة المتوقعة لم تعد متطابقة",
  };
  for (const l of ["en", "ko", "ar"] as const) {
    const html = renderToStaticMarkup(
      <CommandFailure
        l={l}
        error={
          new AdminError(
            "conflict",
            [],
            0,
            "",
            "revision_conflict",
            "qa-command-reference",
          )
        }
        outcome="refused"
      />,
    );
    assert.ok(html.includes(t(l, "refused")) && html.includes(messages[l]));
    assert.ok(
      html.includes("revision_conflict") &&
        html.includes("qa-command-reference"),
    );
    assert.ok(!html.includes(w.saveFailed(l)));
  }
});
test("an uncertain original remains uncertain after a definitive replay refusal", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    for (const failure of ["conflict", "denied", "invalid"] as const) {
      const html = renderToStaticMarkup(
        <CommandFailure
          l={l}
          error={
            new AdminError(
              failure,
              [],
              0,
              "",
              "revision_conflict",
              "qa-replay-reference",
            )
          }
          outcome="unknown"
        />,
      );
      assert.ok(html.includes(t(l, "unknown")));
      assert.ok(!html.includes(t(l, "refused")));
      assert.ok(!html.includes("preparing a new request"));
      assert.ok(!html.includes(v.denied(l)));
      assert.ok(html.includes("qa-replay-reference"));
    }
  }
});
test("M04 command diagnostics suppress unsafe tokens and raw failure messages", () => {
  const error = new AdminError(
    "invalid",
    ["PRIVATE_FIELD_VALUE"],
    0,
    "PRIVATE_OPERATION",
    "<unsafe-code>",
    "PRIVATE REQUEST VALUE",
  );
  error.message = "PRIVATE_RAW_MESSAGE";
  const html = renderToStaticMarkup(
    <CommandFailure l="en" error={error} outcome="refused" />,
  );
  for (const privateValue of [
    "PRIVATE_FIELD_VALUE",
    "PRIVATE_OPERATION",
    "unsafe-code",
    "PRIVATE REQUEST VALUE",
    "PRIVATE_RAW_MESSAGE",
  ])
    assert.ok(!html.includes(privateValue));
  assert.ok(html.includes("Not available"));
});
test("password confirmation cancellation differs from unverified transport without save copy", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const cancelled = renderToStaticMarkup(
      <CommandFailure
        l={l}
        error={new AdminError("reauth_cancelled")}
        outcome="confirmation"
      />,
    );
    const unavailable = renderToStaticMarkup(
      <CommandFailure
        l={l}
        error={new AdminError("unavailable")}
        outcome="confirmation"
      />,
    );
    assert.notEqual(cancelled, unavailable);
    assert.ok(
      !cancelled.includes(w.saveFailed(l)) &&
        !unavailable.includes(w.saveFailed(l)),
    );
    assert.ok(!unavailable.includes("was cancelled"));
  }
});
