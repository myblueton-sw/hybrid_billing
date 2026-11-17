import { test } from "node:test";
import assert from "node:assert/strict";
import React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { AdminError, type AdminClient } from "./admin";
import { ServiceContext, clientFor, clientKey, useAdminClient, type ServiceSession } from "./context";
import { c } from "./copy";
import { cancellable, outcomeOf, settleOnce, stepUpGate } from "./reauth-model";
import { Reauthenticate } from "./Reauthenticate";

test("the gate opens once and shares one pending answer", async () => {
  let opened = 0;
  const gate = stepUpGate(() => { opened++; });
  const a = gate.request(), b = gate.request();
  assert.equal(opened, 1);
  assert.equal(gate.pending(), true);
  gate.settle(true);
  gate.settle(false);
  assert.deepEqual(await Promise.all([a, b]), [true, true]);
  assert.equal(gate.pending(), false);
  const c2 = gate.request();
  assert.equal(opened, 2);
  gate.settle(false);
  assert.equal(await c2, false);
  gate.settle(true);
});

test("a dialog settles once and cannot be cancelled while pending", () => {
  const seen: boolean[] = [];
  const settle = settleOnce((ok) => seen.push(ok));
  settle(true);
  settle(false);
  assert.deepEqual(seen, [true]);
  assert.equal(cancellable(true), false);
  assert.equal(cancellable(false), true);
});

test("endpoint results choose the dialog outcome", () => {
  assert.deepEqual(outcomeOf(new AdminError("wrong")), { kind: "wrong" });
  assert.deepEqual(outcomeOf(new AdminError("unauthenticated")), { kind: "signed-out" });
  assert.deepEqual(outcomeOf(new AdminError("conflict")), { kind: "changed" });
  for (const f of ["busy", "unavailable", "limited", "denied"] as const) assert.deepEqual(outcomeOf(new AdminError(f)), { kind: "failed" }, f);
  assert.deepEqual(outcomeOf(new Error("x")), { kind: "failed" });
});

test("the admin client is keyed on the token and the step-up function", async () => {
  const stepUp = async () => true;
  const service: ServiceSession = { session: { user: "u", workspace: "w", role: "administrator", login: "a@example.invalid", csrf: "tok" }, signOut: () => undefined, stepUp };
  const [csrf, fn] = clientKey(service);
  assert.equal(csrf, "tok");
  assert.equal(fn, stepUp);
  assert.deepEqual(clientKey(null), ["", undefined]);
  const prompted: string[] = [];
  const replies = [403, 200];
  const original = globalThis.fetch;
  globalThis.fetch = (async (_url: string, init: RequestInit) => {
    prompted.push(String((init.headers as Record<string, string>)["X-CSRF-Token"]));
    const status = replies.shift() ?? 200;
    return new Response(JSON.stringify(status === 403 ? { code: "reauthentication_required" } : { business_state: "ok" }), { status });
  }) as typeof fetch;
  try {
    let api: AdminClient | null = null;
    let asked = 0;
    const Probe = () => { api = useAdminClient(); return null; };
    renderToStaticMarkup(React.createElement(ServiceContext.Provider, { value: { ...service, stepUp: async () => { asked++; return true; } } }, React.createElement(Probe)));
    const out = await (api as unknown as AdminClient).command("identity.suspend_user", "u1", "r1", "x", {}, { request: "q", key: "k" });
    assert.equal(out.state, "ok");
    assert.equal(asked, 1);
    assert.deepEqual(prompted, ["tok", "tok"]);
  } finally {
    globalThis.fetch = original;
  }
});

test("the dialog markup has no stored password and names the account", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const html = renderToStaticMarkup(React.createElement(Reauthenticate, { locale: l, login: "a@example.invalid", csrf: "t", opener: null, onDone: () => undefined, onSignedOut: () => undefined }));
    assert.match(html, /<dialog[^>]*aria-labelledby="reauth-title"/);
    assert.match(html, /aria-describedby="reauth-body"/);
    assert.match(html, new RegExp(`dir="${l === "ar" ? "rtl" : "ltr"}"`));
    assert.match(html, /autoComplete="current-password"|autocomplete="current-password"/);
    assert.match(html, /autoComplete="username"|autocomplete="username"/);
    assert.ok(html.includes(c.reauthTitle(l)) && html.includes(c.reauthBody(l)));
    assert.doesNotMatch(html, /name="/);
    assert.doesNotMatch(html, /type="password"[^>]*value=/);
    assert.ok(html.includes("a@example.invalid"));
    assert.ok(html.includes(c.reauthHint(l)) && html.includes(c.signInEmail(l)));
    assert.match(html, /id="reauth-password"[^>]*aria-describedby="reauth-hint"/);
  }
  for (const l of ["en", "ko", "ar"] as const) {
    for (const key of ["reauthNotDone", "reauthWrong", "reauthChanged", "reauthFailed", "reauthRefused", "reauthEmpty", "reauthConfirm", "reauthCancel", "reauthHint"] as const) assert.notEqual(c[key](l), "");
  }
  assert.notEqual(c.reauthNotDone("ko"), c.reauthNotDone("en"));
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(c.reauthText(l, "reauth"), c.reauthRefused(l));
    assert.equal(c.reauthText(l, "reauth_cancelled"), c.reauthNotDone(l));
    assert.notEqual(c.reauthRefused(l), c.reauthNotDone(l));
    assert.ok(!/password/i.test(c.ended(l)) && !c.ended(l).includes("비밀번호") && !c.ended(l).includes("كلمة"));
  }
  assert.notEqual(c.reauthNotDone("ar"), c.reauthNotDone("en"));
});

test("useAdminClient returns the same client across re-renders and a new one when the key changes", () => {
  const stepUp = async () => true;
  const service: ServiceSession = { session: { user: "u", workspace: "w", role: "administrator", login: "a@example.invalid", csrf: "tok-memo" }, signOut: () => undefined, stepUp };
  const seen: AdminClient[] = [];
  const Probe = () => { seen.push(useAdminClient()); return null; };
  const tree = (value: ServiceSession) => React.createElement(ServiceContext.Provider, { value }, React.createElement(Probe));
  renderToStaticMarkup(tree(service));
  renderToStaticMarkup(tree({ ...service, signOut: () => undefined }));
  assert.equal(seen.length, 2);
  assert.equal(seen[0], seen[1]);
  renderToStaticMarkup(tree({ ...service, session: { ...service.session, csrf: "tok-other" } }));
  assert.notEqual(seen[2], seen[0]);
  renderToStaticMarkup(tree({ ...service, session: { ...service.session, csrf: "tok-other" }, stepUp: async () => false }));
  assert.notEqual(seen[3], seen[2]);
  assert.equal(clientFor("tok-x", stepUp), clientFor("tok-x", stepUp));
});
