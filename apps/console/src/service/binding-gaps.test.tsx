import React from "react";
import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup as render } from "react-dom/server";
import { detailOf } from "./identity-bulk";
import { installationOf } from "./installation";
import { integrationsOf } from "./integrations";
import { evidenceOf } from "./release-evidence";
import * as nodeModule from "node:module";
import { adminClient } from "./admin";
import { b } from "../modules/identity/pages/users/bulk-copy";
import { ClientsView } from "../modules/administration/pages/integrations/LiveClients";
import { k as clientsCopy } from "../modules/administration/pages/integrations/live-copy";
import { Readiness } from "../modules/administration/pages/release-evidence/LiveLoad";
import { k as evidenceCopy } from "../modules/administration/pages/release-evidence/live-copy";
import { CurrentRules } from "../modules/identity/pages/authentication/TrustChanges";
import { pageFixture } from "../modules/identity/pages/authentication/trust-fixtures";
import { ty } from "../modules/identity/pages/authentication/trust-copy";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { PreviewBody } = await import("../modules/identity/pages/users/BulkPrepare");

const noop = () => undefined;
const api = adminClient("synthetic");
const escape = (v: string) => v.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;").replaceAll("'", "&#x27;");

for (const l of ["en", "ko", "ar"] as const) {
  test(`${l}: bulk impact requires explicit not-applicable states`, () => {
    for (const value of [undefined, null, false, 0, "future", "not_applicable"]) {
      const detail = detailOf({ impact: { api_keys: value, pending_jobs: value } });
      const html = render(<PreviewBody l={l} api={api} prepared={{ detail, targets: null }} onMore={noop} />);
      assert.equal(html.includes(escape(b.noKeys(l))), value === "not_applicable");
      assert.equal(html.includes(escape(b.noJobs(l))), value === "not_applicable");
      if (value !== "not_applicable") assert.ok(html.includes(escape(b.impactUnknown(l))));
    }
  });
  test(`${l}: client section state prevents contradictory child disclosure`, () => {
    for (const state of ["not_permitted", "not_configured", "future", "", "configured"]) {
      const page = integrationsOf({ clients: { state, clients: [{ id: "hidden-client" }], service_users: [{ id: "hidden-service" }] } });
      const html = render(<ClientsView l={l} page={page} />);
      assert.equal(html.includes("hidden-client"), state === "configured");
      assert.equal(html.includes("hidden-service"), state === "configured");
      if (state === "not_permitted") assert.ok(html.includes(escape(clientsCopy.clientsDenied(l))));
      if (state === "not_configured") assert.ok(html.includes(escape(clientsCopy.clientsUnconfigured(l))));
    }
  });
  test(`${l}: build failure count preserves zero and unknown`, () => {
    for (const failures of [0, 7, null, -1, "7"]) {
      const page = evidenceOf({ load: { readiness: { gates: [{ gate: "trial", status: "not_run", profiles: [{ profile: "synthetic-profile", status: "not_run", run: 0, failures }], retired: [] }] } } });
      const html = render(<Readiness l={l} page={page} />);
      assert.ok(html.includes(escape(evidenceCopy.failuresSinceApproval(l))));
      assert.ok(html.includes(escape(evidenceCopy.failureCounterScope(l))));
      if (failures === 0 || failures === 7) assert.ok(html.includes(`>${failures}</bdi>`));
      else assert.ok(html.includes(escape(evidenceCopy.unknown(l))));
      assert.ok(html.includes(escape(evidenceCopy.noRunAtBuild(l))));
    }
  });
  test(`${l}: restored rule list keeps empty, unknown and named fields distinct`, () => {
    for (const fields of [undefined, [], ["max_age", "step_up"], ["future-field"]]) {
      const page = pageFixture({ trust: { stricter_fields: fields } });
      const html = render(<CurrentRules l={l} page={page} loading={false} onReload={noop} />);
      assert.ok(html.includes(escape(ty.stricterFields(l))));
      if (!fields) assert.ok(html.includes(escape(ty.fieldsUnknown(l))));
      else if (!fields.length) assert.ok(html.includes(escape(ty.noStricterFields(l))));
      else if (fields.length > 1) assert.ok(html.includes("<br/>"));
      else if (fields[0] === "future-field") assert.match(html, /<bdi[^>]*dir="ltr"[^>]*>future-field<\/bdi>/);
    }
  });
}

test("installation summaries count only correctly shaped history and backup collections", () => {
  const row = { kind: "install", from: "", to: "1", run: "r1", backup: "", runs: [] };
  for (const [history, backups, count] of [[[], [], 0], [[row], ["b1"], 1], [[null], [1], null], [undefined, undefined, null], [[{ ...row, runs: [null] }], {}, null]] as const) {
    const page = installationOf({ installed: { history, backups } });
    assert.equal(page.installed.history, count);
    assert.equal(page.installed.backups, count);
  }
});
