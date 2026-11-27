import { test } from "node:test";
import assert from "node:assert/strict";
import { afterSignIn, asciiLower, emailValid, localeOf, loginValid, passwordProblem, serviceMode, serviceUrl, shellLocale, utf8Length } from "./helpers";
import { retryPhrase } from "./copy";
test("service mode is entered only by /activate or service=api", () => {
  assert.equal(serviceMode({ pathname: "/activate", search: "" }), true);
  assert.equal(serviceMode({ pathname: "/", search: "?service=api" }), true);
  assert.equal(serviceMode({ pathname: "/", search: "?service=local" }), false);
  assert.equal(serviceMode({ pathname: "/", search: "" }), false);
  assert.equal(serviceMode({ pathname: "/activate/x", search: "" }), false);
});
test("locales default to Korean and follow the shell hash", () => {
  assert.equal(localeOf("?lang=en"), "en");
  assert.equal(localeOf("?lang=ar"), "ar");
  assert.equal(localeOf("?lang=fr"), "ko");
  assert.equal(shellLocale("#/home?lang=ar", "en"), "ar");
  assert.equal(shellLocale("#/home", "en"), "en");
  assert.equal(serviceUrl("help", "en", "signed-out"), "/?service=api&lang=en&view=help&status=signed-out");
  assert.equal(serviceUrl("signin", "ko"), "/?service=api&lang=ko");
});
test("after sign-in the route and its query are kept and the language is set", () => {
  assert.equal(afterSignIn("", "en"), "#/home?lang=en");
  assert.equal(afterSignIn("#/b09?period=2026-10&lang=ko", "ar"), "#/b09?period=2026-10&lang=ar");
  assert.equal(afterSignIn("#garbage", "ko"), "#/home?lang=ko");
});
test("password rules follow the server in UTF-8 bytes and ASCII lower-casing", () => {
  assert.equal(utf8Length("가나다라"), 12);
  assert.equal(passwordProblem("en", "", "a@b"), "Enter a new password");
  assert.equal(passwordProblem("en", "a".repeat(11), "a@b"), "Password must be at least 12 characters");
  assert.equal(passwordProblem("en", "가나다라", "a@b"), "");
  assert.equal(passwordProblem("en", "a".repeat(257), "a@b"), "Password is too long. Use a shorter one");
  assert.equal(passwordProblem("en", "a".repeat(256), "a@b"), "");
  assert.equal(passwordProblem("en", "admin@example.invalid", "Admin@Example.Invalid"), "Password must be different from your email address");
  assert.equal(passwordProblem("en", "émile@example.org", "Émile@example.org"), "");
  assert.equal(asciiLower("ÉA"), "Éa");
});
test("activation logins and contact emails match the server's form rules", () => {
  assert.equal(loginValid("ab"), false);
  assert.equal(loginValid("abc"), true);
  assert.equal(loginValid("a".repeat(200)), true);
  assert.equal(loginValid("a".repeat(201)), false);
  assert.equal(loginValid("a\u0001b"), false);
  assert.equal(loginValid("a\u0085b"), false);
  assert.equal(emailValid("admin"), false);
  assert.equal(emailValid("a@b"), true);
  assert.equal(emailValid("a@" + "b".repeat(252)), true);
  assert.equal(emailValid("a@" + "b".repeat(253)), false);
});
test("retry phrases use seconds under a minute and rounded-up minutes", () => {
  assert.equal(retryPhrase("en", 1), "in 1 second");
  assert.equal(retryPhrase("en", 59), "in 59 seconds");
  assert.equal(retryPhrase("en", 61), "in 2 minutes");
  assert.ok(retryPhrase("ko", 30).includes("30"));
});

test("signing in keeps the screen and its filters but drops every list position", async () => {
  const { afterSignIn } = await import("./helpers");
  const next = afterSignIn("#/my-security?view=sessions&sq=abc&safter=c1&op_a=o2&tc_oa=x&acc_after=y&svc_a_limit=10", "ko");
  const q = new URLSearchParams(next.split("?")[1]);
  assert.equal(next.split("?")[0], "#/my-security");
  assert.equal(q.get("view"), "sessions");
  assert.equal(q.get("sq"), "abc");
  assert.equal(q.get("svc_a_limit"), "10");
  assert.equal(q.get("lang"), "ko");
  for (const name of ["safter", "op_a", "tc_oa", "acc_after"]) assert.equal(q.get(name), null);
});

test("signing in drops all ten list positions and the M04 selections and open forms, and keeps sizes", async () => {
  const { afterSignIn, listCursors, signInDropped } = await import("./helpers");
  assert.equal(listCursors.length, 10);
  const kept = { acc_limit: "10", req_el: "50", svc_c_limit: "25", tc_ol: "10" };
  const all = new URLSearchParams({ ...Object.fromEntries([...listCursors, ...signInDropped].map((n) => [n, "x"])), ...kept });
  const q = new URLSearchParams(afterSignIn("#/service-accounts?" + all.toString(), "en").split("?")[1]);
  for (const name of [...listCursors, ...signInDropped]) assert.equal(q.get(name), null, name);
  for (const [name, value] of Object.entries(kept)) assert.equal(q.get(name), value, name);
});
