import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import { AdminError } from "../../../../service/admin";
import { c as common } from "../../../../service/copy";
import { qcopy } from "../service-accounts/query-copy";
import { w } from "../users/live-copy";
import { ty } from "./trust-copy";
import { tl } from "./trust-list-copy";
import { changeFixture, pageFixture } from "./trust-fixtures";
import { ChangesHead, Conditions, Detail } from "./TrustChanges";
import { DetailPending, goneFrom, shownRead, type DetailRead } from "./TrustDetail";

const noop = () => undefined;
const langs = ["en", "ko", "ar"] as const;
const pending = (read: Exclude<DetailRead, { state: "ready" }>, l: (typeof langs)[number] = "en", split = true, number: number | null = 7) =>
  renderToStaticMarkup(<DetailPending l={l} opened={{ id: "trust-0a1b2c3d-7", number }} read={read} split={split} onClose={noop} onRetry={noop} onSignOut={noop} />);

test("M02 detail: which answer is shown for the opened change", () => {
  const a = {}, b = {};
  const ready: DetailRead = { state: "ready", change: changeFixture({ id: "x" }) };
  assert.deepEqual(shownRead("", a, { id: "x", token: a, read: ready }), { state: "loading" });
  assert.deepEqual(shownRead("x", a, null), { state: "loading" });
  assert.equal(shownRead("x", a, { id: "x", token: a, read: ready }), ready);
  assert.equal(shownRead("x", b, { id: "x", token: a, read: ready }), ready, "an earlier answer for the same change stays while it is read again");
  assert.deepEqual(shownRead("y", b, { id: "x", token: a, read: ready }), { state: "loading" }, "another change's answer is never shown");
  assert.deepEqual(shownRead("x", b, { id: "x", token: a, read: { state: "error", error: new Error("e") } }), { state: "loading" }, "an earlier failure does not stay");
  assert.deepEqual(shownRead("x", b, { id: "x", token: a, read: { state: "gone" } }), { state: "loading" });
  assert.deepEqual(shownRead("x", a, { id: "x", token: a, read: { state: "gone" } }), { state: "gone" });
  assert.equal(goneFrom(new AdminError("invalid", [], 0, "", "invalid_request")), true);
  assert.equal(goneFrom(new AdminError("invalid")), false, "the console's own refusal of a malformed answer is a failure, not a removed change");
  assert.equal(goneFrom(new AdminError("unavailable", [], 0, "", "dependency_unavailable")), false);
  assert.equal(goneFrom(new Error("network")), false);
});

test("M02 detail: loading, failure and no-longer-kept each say one thing and offer the right buttons", () => {
  for (const l of langs) {
    const loading = pending({ state: "loading" }, l);
    assert.ok(loading.includes(`>${ty.change(l, 7)}</h2>`) && loading.includes(tl.detailLoading(l)) && loading.includes('role="status"') && loading.includes(`>${ty.close(l)}</button>`));
    assert.ok(!loading.includes(common.tryAgain(l)) && !loading.includes(tl.noLongerKept(l)));
    const failed = pending({ state: "error", error: new AdminError("unavailable", [], 0, "", "dependency_unavailable") }, l);
    assert.ok(failed.includes(`<h3>${tl.detailFailed(l)}</h3>`) && failed.includes('role="alert"') && failed.includes(qcopy(l, "readUnavailable")) && failed.includes(`>${common.tryAgain(l)}</button>`));
    assert.equal(failed.split(`>${ty.close(l)}</button>`).length, 2, "one Close beside Try again");
    assert.ok(!failed.includes("m02-close"));
    const gone = pending({ state: "gone" }, l);
    assert.ok(gone.includes(tl.noLongerKept(l)) && !gone.includes(common.tryAgain(l)) && !gone.includes('role="alert"'));
    assert.equal(gone.split(`>${ty.close(l)}</button>`).length, 2, "one Close and no retry");
  }
  const byId = pending({ state: "loading" }, "ar", true, null);
  assert.ok(byId.includes('<bdi dir="ltr" class="live-mono">trust-0a1b2c3d-7</bdi>'));
  const stepped = pending({ state: "loading" }, "en", false);
  assert.ok(!stepped.includes("m02-close"));
  const signedOut = pending({ state: "error", error: new AdminError("unauthenticated") });
  assert.ok(!signedOut.includes(common.tryAgain("en")) && signedOut.includes(`>${w.signInAgain("en")}</button>`));
});

test("M02 detail: the notice, the note and the successor no longer look through a list", () => {
  const hold = { trust: { hold: { reason: "commit_unknown", since: "2026-11-18T09:00:00Z" } }, ledger: { state: "stopped" } };
  for (const l of langs) {
    const named = renderToStaticMarkup(<Conditions l={l} page={pageFixture({ ...hold, waiting_activation: "trust-0a1b2c3d-9" })} onOpen={noop} />);
    assert.ok(named.includes('data-opens="trust-0a1b2c3d-9"') && named.includes(`>${tl.openWaiting(l)}</button>`), l);
    const winner = "trust-0a1b2c3d-2";
    const c = changeFixture({ state: "superseded", superseded_by: winner, superseded_epoch: { value: "2", generation: 0, number: 2 } });
    const out = renderToStaticMarkup(<Detail l={l} page={pageFixture()} c={c} split onClose={noop} onAction={noop} onOpen={noop} />);
    assert.ok(out.includes(ty.supersededBy(l)) && out.includes(`<bdi dir="ltr" class="live-mono">${winner}</bdi>`) && out.includes(`data-opens="${winner}"`) && out.includes(`>${tl.openReplacing(l)}</button>`), l);
  }
  const unnamed = renderToStaticMarkup(<Conditions l="en" page={pageFixture(hold)} onOpen={noop} />);
  assert.ok(unnamed.includes('data-condition="hold"') && !unnamed.includes("data-opens"), "the notice opens only what the policy read names");
  const kept = renderToStaticMarkup(<ChangesHead l="en" page={pageFixture({ earlier_run_kept: true })} onPropose={noop} />);
  const none = renderToStaticMarkup(<ChangesHead l="en" page={pageFixture()} onPropose={noop} />);
  assert.ok(kept.includes("m02-note") && !none.includes("m02-note"));
});
