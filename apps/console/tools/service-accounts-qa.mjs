import { createRequire } from "node:module";
import { readFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (a, v, i, all) =>
        v.startsWith("--") ? [...a, [v.slice(2), all[i + 1]]] : a,
      [],
    ),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const url = (args.url || "http://127.0.0.1:4173/").replace(/\/?$/, "/");
const workspace = args.workspace || "sample-workspace";
const login = "admin@example.invalid";
const password = randomBytes(18).toString("base64url");
const scenarios = [
  "flip", "narrow", "inflight", "cancel", "leave", "selector", "revokebtn", "backnarrow", "inflightleave", "reload", "remindercancel", "revealcancel",
  "typesending", "idemconflict", "replaypartial", "window", "issueddrop", "signoutheld", "create", "createleave",
];
const only = new Set((args.only || scenarios.join(",")).split(","));
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const results = [];
const check = (name, ok, detail = "") =>
  results.push(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " " + String(detail).replace(/\s+/g, " ") : ""}`);
const issuePath = "**/api/v1/commands/service.issue_credential";
const servicesRead = (r) =>
  r.url().split("?")[0].endsWith("/api/v1/identity/services");

async function activate() {
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const link = new URL(
    lines.find((l) => l.startsWith(login + " ")).split(" ")[1],
  );
  const page = await browser.newPage();
  await page.goto(
    url + link.pathname.replace(/^\//, "") + "?lang=en" + link.hash,
  );
  await page.locator("#link-new").fill(password);
  await page.locator("#link-confirm").fill(password);
  await page.getByRole("button", { name: "Set password" }).click();
  await page.locator("a.service-button-link").waitFor();
  await page.close();
}

async function signedIn() {
  const page = await browser.newPage({
    viewport: { width: 1440, height: 900 },
  });
  await page.goto(url + "?service=api&lang=en");
  await page.locator("#signin-workspace").fill(workspace);
  await page.locator("#signin-email").fill(login);
  await page.locator("#signin-password").fill(password);
  await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  return page;
}

const view = (page, name) =>
  page.goto(
    url + `?service=api&lang=en#/service-accounts?lang=en&view=${name}`,
  );
const submit = (page) =>
  page.locator(".live-panel form button[type=submit]").click();

async function createService(page, name) {
  await view(page, "accounts");
  await page.locator("#svc-list-title").waitFor();
  await page.locator("#svc-name").fill(name);
  await page.locator("#svc-owner").selectOption({ label: login });
  await page.locator("#live-reason").fill("Regression check");
  await submit(page);
  await page
    .locator(".service-status", { hasText: new RegExp(`svc-${name}\\W* created`) })
    .waitFor();
}

async function heldIssue(page, service, duringFirst, whileHeld) {
  await view(page, "credentials");
  await page.locator("#svc-cred-title").waitFor();
  await page.locator("#svc-select").selectOption(service);
  await page.locator("#svc-days").waitFor();
  const posts = [];
  const record = (r) => {
    if (r.url().endsWith("/commands/service.issue_credential"))
      posts.push(JSON.parse(r.postData() ?? "{}"));
  };
  page.on("request", record);
  let lost = false;
  await page.route(issuePath, async (r) => {
    if (lost) return r.continue();
    lost = true;
    await r.fetch();
    await duringFirst();
    await r.abort();
  });
  await page.locator("#svc-days").fill("1");
  await page.locator("#live-reason").fill("Lost response");
  const reloaded = page.waitForResponse((r) => lost && servicesRead(r));
  await submit(page);
  await page
    .locator(".live-panel button[type=submit]", { hasText: "Try again" })
    .waitFor();
  await reloaded;
  await whileHeld();
  const form = await page
    .locator(".live-panel form")
    .evaluate(
      (f) =>
        `${f.querySelector("#svc-days")?.value}|${f.querySelector("#live-reason")?.value}|${f.querySelector("button[type=submit]")?.textContent}`,
    )
    .catch(() => "no form");
  if (form !== "no form") await submit(page);
  const shown = await page
    .locator("#svc-secret")
    .waitFor({ timeout: 10000 })
    .then(() => true)
    .catch(() => false);
  await page.unroute(issuePath);
  page.off("request", record);
  const same =
    posts.length === 2 &&
    posts[0].request_id === posts[1].request_id &&
    posts[0].idempotency_key === posts[1].idempotency_key &&
    JSON.stringify(posts[0]) === JSON.stringify(posts[1]);
  if (shown) {
    await page
      .locator(".live-panel button", { hasText: "I didn't store it" })
      .click();
    await page.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
    await submit(page);
    await page.locator(".service-status", { hasText: "revoked" }).waitFor();
  }
  return { form, shown, same, posts: posts.length };
}

const report = (name, r) => {
  check(`${name}: form kept`, r.form === "1|Lost response|Try again", r.form);
  check(`${name}: same request_id and idempotency_key on the retry`, r.same, `posts=${r.posts}`);
  check(`${name}: secret shown after the retry`, r.shown);
};


const safe = (handle) => async (r) => {
  try {
    await handle(r);
  } catch {
    return;
  }
};

const tab = (p, name) => p.locator(".planning-tabs button", { hasText: new RegExp(`^${name}$`) }).click();

async function signIn(p) {
  await p.goto(url + "?service=api&lang=en");
  await p.locator("#signin-workspace").fill(workspace);
  await p.locator("#signin-email").fill(login);
  await p.locator("#signin-password").fill(password);
  await p.locator("form button[type=submit]").click();
  await p.locator(".service-session").waitFor();
}

function watch(p, errors) {
  const state = { refs: [], keys: [], issueBodies: [], dialogs: [], armed: null, logs: [] };
  p.on("pageerror", (e) => errors.push(e.message));
  p.on("console", (m) => state.logs.push(m.text()));
  p.on("dialog", async (d) => {
    if (state.armed) {
      const take = state.armed;
      state.armed = null;
      take(d.type());
      await d.accept();
      return;
    }
    state.dialogs.push(d.type());
    await d.dismiss();
  });
  p.on("request", (r) => {
    if (r.method() === "POST" && r.url().includes("/commands/")) {
      const body = JSON.parse(r.postData() ?? "{}");
      if (body.idempotency_key) state.keys.push(body.idempotency_key);
      if (r.url().endsWith("/commands/service.issue_credential")) state.issueBodies.push(r.postData());
    }
  });
  p.on("response", async (r) => {
    if (!r.url().endsWith("/commands/service.issue_credential")) return;
    const body = await r.json().catch(() => ({}));
    if (body.fields?.secret_ref) state.refs.push(body.fields.secret_ref);
  });
  return state;
}

async function opened(ctx, errors, width = 1440) {
  const p = await ctx.newPage();
  await p.setViewportSize({ width, height: 900 });
  const st = watch(p, errors);
  await p.goto(url + "?service=api&lang=en#/service-accounts?lang=en&view=credentials");
  await p.locator("#svc-cred-title").waitFor();
  return { p, st };
}

async function mark(p) {
  const token = randomBytes(8).toString("hex");
  await p.evaluate((t) => { window.__hb333 = t; }, token);
  return async () => (await p.evaluate(() => window.__hb333)) === token;
}

async function clean(name, p, st) {
  const found = await p.evaluate((refs) => {
    const copy = document.body.cloneNode(true);
    copy.querySelector("#svc-secret")?.remove();
    const html = copy.outerHTML;
    const stores = [JSON.stringify({ ...localStorage }), JSON.stringify({ ...sessionStorage })];
    return [html, ...stores].some((t) => t.includes("hbsk_") || refs.some((r) => t.includes(r)));
  }, st.refs).catch(() => false);
  const logged = st.logs.some((t) => t.includes("hbsk_") || st.refs.some((r) => t.includes(r)) || st.keys.some((k) => t.includes(k)));
  check(`${name}: no secret, pickup reference or key outside the secret field, in storage or in the console`, !found && !logged);
  check(`${name}: no unexpected dialog`, st.dialogs.length === 0, st.dialogs.join(","));
}

async function newService(p, name) {
  const size = p.viewportSize();
  if (size.width < 1280) await p.setViewportSize({ width: 1440, height: 900 });
  await tab(p, "Accounts");
  await p.locator("#svc-list-title").waitFor();
  if (!(await p.locator("#svc-name").isVisible())) await p.locator(".svc-create-open").click();
  await p.locator("#svc-name").fill(name);
  await p.locator("#svc-owner").selectOption({ label: login });
  await p.locator("#live-reason").fill("Regression check");
  await p.locator(".live-panel form button[type=submit]").click();
  await p.locator(".service-status", { hasText: new RegExp(`svc-${name}\\W* created`) }).waitFor();
  await tab(p, "Credentials");
  await p.locator("#svc-cred-title").waitFor();
  await p.locator("#svc-select").selectOption(`svc-${name}`);
  if (size.width < 1280) {
    await p.setViewportSize(size);
    await p.locator(".live-users[data-wide=false]").waitFor();
  }
  return `svc-${name}`;
}

async function issueForm(p) {
  if (!(await p.locator("#svc-days").isVisible())) await p.locator(".svc-issue-open").click();
  await p.locator("#svc-days").waitFor();
}

async function lose(p, path, during = async () => {}) {
  let n = 0;
  const sent = [];
  await p.route(path, safe(async (r) => {
    n += 1;
    if (n > 1) return r.continue();
    const real = await r.fetch();
    sent.push(await real.json().catch(() => ({})));
    await during();
    await r.abort();
  }));
  return { sent, done: () => p.unroute(path) };
}

async function lostIssue(p, service, days = "1", reason = "Lost response") {
  await p.locator("#svc-select").selectOption(service);
  await issueForm(p);
  const lost = await lose(p, issuePath);
  await p.locator("#svc-days").fill(days);
  await p.locator("#live-reason").fill(reason);
  await p.locator(".live-panel form button[type=submit]").click();
  await p.locator(".live-held h3", { hasText: "Outcome unknown" }).waitFor();
  await lost.done();
  return lost.sent[0]?.fields?.credential ?? "";
}

async function retried(name, p, st) {
  const before = st.keys.length;
  await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
  const shown = await p.locator("#svc-secret").waitFor({ timeout: 10000 }).then(() => true).catch(() => false);
  check(`${name}: the retry resends the same idempotency key`, st.keys.length === before + 1 && st.keys.at(-1) === st.keys.at(-2), st.keys.slice(-2).join(" "));
  check(`${name}: the secret is shown after the retry`, shown);
  check(`${name}: the complete issue body stays byte-identical on retry`, st.issueBodies.length >= 2 && st.issueBodies.at(-1) === st.issueBodies.at(-2));
}

const attemptRow = (p, service) => p.locator(`[data-attempt="${service}"]`);
const focused = (p) => p.evaluate(() => {
  const a = document.activeElement;
  return a ? `${a.tagName}#${a.id}[${["data-credential", "data-unstored-revoke", "data-never-revoke", "data-attempt-revoke"].map((n) => a.getAttribute(n) ?? "").join("|")}]` : "";
});

async function heldForm(name, p) {
  const form = await p.locator(".live-panel form").evaluate((f) =>
    `${f.querySelector("#svc-days")?.value}|${f.querySelector("#svc-days")?.readOnly}|${f.querySelector("#live-reason")?.value}|${f.querySelector("#live-reason")?.readOnly}|${f.querySelector("button[type=submit]")?.textContent}`).catch(() => "no form");
  check(`${name}: the held form is restored with locked values`, form === "1|true|Lost response|true|Try again", form);
}

async function newScenarios(errors) {
  const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const first = await ctx.newPage();
  await signIn(first);
  await first.close();
  const run = async (name, body, width = 1440) => {
    if (!only.has(name)) return;
    const { p, st } = await opened(ctx, errors, width);
    try {
      await body(p, st);
    } catch (e) {
      check(`${name}: completed`, false, String(e).split("\n").slice(0, 3).join(" "));
      if (args.shots) await p.screenshot({ path: `${args.shots}/${name}-${width}.png`, fullPage: true }).catch(() => {});
    }
    await clean(name, p, st);
    await p.close();
  };
  await run("leave", async (p, st) => {
    const svc = await newService(p, "leave");
    const kept = await mark(p);
    await lostIssue(p, svc);
    await tab(p, "Accounts");
    const note = await p.locator(".live-attempts-note").waitFor({ timeout: 5000 }).then(() => true).catch(() => false);
    await tab(p, "Credentials");
    const row = await attemptRow(p, svc).innerText().catch(() => "");
    check("leave: the accounts view notes the held issue", note);
    check("leave: the reminder lists the unknown outcome", row.includes("outcome unknown"), row);
    await attemptRow(p, svc).locator("button", { hasText: "Open issue form" }).click();
    await heldForm("leave", p);
    check("leave: no reload", await kept());
    await retried("leave", p, st);
  });
  await run("selector", async (p, st) => {
    const other = await newService(p, "other");
    const svc = await newService(p, "selector");
    const kept = await mark(p);
    await lostIssue(p, svc);
    await p.locator("#svc-select").selectOption(other);
    check("selector: the reminder lists the held issue", await attemptRow(p, svc).isVisible());
    await p.locator("#svc-select").selectOption(svc);
    await heldForm("selector", p);
    check("selector: no reload", await kept());
    await retried("selector", p, st);
  });
  await run("revokebtn", async (p, st) => {
    const svc = await newService(p, "revokebtn");
    await issueForm(p);
    await p.locator("#svc-days").fill("2");
    await p.locator("#live-reason").fill("Existing");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator("#svc-secret").waitFor();
    await p.locator("#svc-ack").check();
    await p.locator(".live-panel button", { hasText: "Done" }).click();
    const existing = await p.locator("[data-credential]").first().getAttribute("data-credential");
    const kept = await mark(p);
    await lostIssue(p, svc);
    await p.locator(`[data-credential="${existing}"]`).click();
    await p.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
    await p.locator(".live-panel button", { hasText: "Cancel" }).click();
    await p.waitForTimeout(200);
    check("revokebtn: Cancel returns focus to the row's Revoke button", (await focused(p)).includes(`[${existing}|`), await focused(p));
    await heldForm("revokebtn", p);
    check("revokebtn: no reload", await kept());
    await retried("revokebtn", p, st);
  });
  await run("backnarrow", async (p, st) => {
    const svc = await newService(p, "backnarrow");
    const kept = await mark(p);
    await lostIssue(p, svc);
    await p.locator(".live-back").click();
    const row = await attemptRow(p, svc).isVisible();
    check("backnarrow: Back keeps the held issue as a reminder", row);
    await attemptRow(p, svc).locator("button", { hasText: "Open issue form" }).click();
    await heldForm("backnarrow", p);
    check("backnarrow: no reload", await kept());
    await retried("backnarrow", p, st);
  }, 1000);
  await run("inflightleave", async (p) => {
    const svc = await newService(p, "inflight-leave");
    const kept = await mark(p);
    await issueForm(p);
    let release;
    const gate = new Promise((y) => { release = y; });
    let readRelease;
    const readGate = new Promise((y) => { readRelease = y; });
    let heldRead = false;
    await p.route(issuePath, safe(async (r) => { await gate; await r.continue(); }));
    await p.locator("#svc-days").fill("1");
    await p.locator("#live-reason").fill("In flight");
    const issued = p.waitForResponse((r) => r.url().endsWith("/commands/service.issue_credential"));
    await p.locator(".live-panel form button[type=submit]").click();
    await p.route("**/api/v1/identity/services", safe(async (r) => {
      if (heldRead) return r.continue();
      heldRead = true;
      const real = await r.fetch();
      await readGate;
      await r.fulfill({ response: real });
    }));
    await tab(p, "Accounts");
    await p.waitForTimeout(300);
    release();
    await issued;
    await p.locator(".live-attempts-note").waitFor();
    readRelease();
    await p.waitForTimeout(500);
    check("inflightleave: a read that started before the outcome does not drop the issued entry", await p.locator(".live-attempts-note").isVisible());
    await p.unroute("**/api/v1/identity/services");
    await p.unroute(issuePath);
    await p.locator(".live-attempts-note button").click();
    await p.waitForTimeout(200);
    check("inflightleave: focus moves to the reminder heading", (await focused(p)).startsWith("H3#svc-attempts-title"), await focused(p));
    const row = await attemptRow(p, svc).innerText();
    check("inflightleave: the issued credential waits for Show secret", row.includes("has not been shown"), row);
    await attemptRow(p, svc).locator("button", { hasText: "Show secret" }).click();
    check("inflightleave: Show secret shows the secret", await p.locator("#svc-secret").waitFor({ timeout: 10000 }).then(() => true).catch(() => false));
    check("inflightleave: no reload", await kept());
  });
  await run("reload", async (p, st) => {
    const svc = await newService(p, "reload");
    const credential = await lostIssue(p, svc);
    const guarded = await p.evaluate(() => { const e = new Event("beforeunload", { cancelable: true }); dispatchEvent(e); return e.defaultPrevented; });
    check("reload: the page guards against unload while the outcome is unknown", guarded);
    const seen = new Promise((y) => { st.armed = y; });
    await p.reload();
    check("reload: the browser asked before reloading", (await Promise.race([seen, new Promise((y) => setTimeout(() => y("none"), 3000))])) === "beforeunload");
    await p.locator("#svc-cred-title").waitFor();
    await p.locator("#svc-select").selectOption(svc);
    const cell = await p.locator(`tr:has([data-credential="${credential}"])`).innerText().catch(() => "");
    check("reload: the row says the secret is not collected yet", cell.includes("Secret not collected yet") && cell.includes("estimated to end"), cell);
    check("reload: Active still shows", cell.includes("Active"), cell);
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    await signIn(p);
    await p.goto(url + "?service=api&lang=en#/service-accounts?lang=en&view=credentials");
    await p.locator("#svc-cred-title").waitFor();
    await p.locator("#svc-select").selectOption(svc);
    const after = await p.locator(`tr:has([data-credential="${credential}"])`).innerText().catch(() => "");
    check("reload: after sign-out the row says the secret was never collected", after.includes("Secret never collected"), after);
    check("reload: the never-collected region lists it", await p.locator(`[data-never="${credential}"]`).isVisible());
    await p.locator(`[data-never-revoke="${credential}"]`).click();
    await p.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
    check("reload: the revoke reason is prefilled", (await p.locator("#live-reason").inputValue()) === "Secret never collected");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator(".service-status", { hasText: "revoked" }).waitFor();
    await p.waitForTimeout(200);
    check("reload: focus reaches the credentials heading", (await focused(p)).startsWith("H2#svc-cred-title"), await focused(p));
    check("reload: the region is gone", (await p.locator(`[data-never="${credential}"]`).count()) === 0);
  });
  for (const width of [1440, 390]) {
    await run("remindercancel", async (p) => {
      const a = await newService(p, `mark-a-${width}`);
      const b = await newService(p, `mark-b-${width}`);
      await p.locator("#svc-select").selectOption(a);
      await issueForm(p);
      await p.locator("#svc-days").fill("1");
      await p.locator("#live-reason").fill("Unstored");
      await p.locator(".live-panel form button[type=submit]").click();
      await p.locator("#svc-secret").waitFor();
      const id = (await p.locator(".live-panel .live-facts").innerText()).match(/cred-[0-9a-f]{16}/)?.[0] ?? "";
      if (width < 1280) await p.locator(".live-back").click();
      await p.locator("#svc-select").selectOption(b);
      await p.locator(`[data-unstored-revoke="${id}"]`).click();
      await p.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
      await p.locator(".live-panel button", { hasText: "Cancel" }).click();
      await p.waitForTimeout(200);
      check(`remindercancel at ${width}: focus returns to the reminder's Revoke button`, (await focused(p)).includes(`[|${id}|`), await focused(p));
    }, width);
  }
  await run("revealcancel", async (p) => {
    const svc = await newService(p, "revealcancel");
    await issueForm(p);
    await p.locator("#svc-days").fill("1");
    await p.locator("#live-reason").fill("Reveal cancel");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator("#svc-secret").waitFor();
    const id = (await p.locator(".live-panel .live-facts").innerText()).match(/cred-[0-9a-f]{16}/)?.[0] ?? "";
    await p.locator(".live-panel button", { hasText: "I didn't store it" }).click();
    await p.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
    await p.locator(".live-panel button", { hasText: "Cancel" }).click();
    await p.waitForTimeout(200);
    check("revealcancel: focus returns to the credential row", (await focused(p)).includes(`[${id}|`), `${svc} ${await focused(p)}`);
  });
  await run("typesending", async (p, st) => {
    const svc = await newService(p, "typesending");
    await issueForm(p);
    let release;
    let calls = 0;
    await p.route(issuePath, safe(async (r) => {
      calls += 1;
      if (calls > 2) return r.continue();
      await new Promise((y) => { release = y; });
      const real = await r.fetch();
      if (calls === 2) return r.fulfill({ response: real });
      await r.abort();
    }));
    await p.locator("#svc-days").fill("1");
    await p.locator("#live-reason").fill("Typed while sending");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator(".live-held", { hasText: "Sending this request" }).waitFor();
    await p.locator("#svc-days").focus();
    await p.keyboard.type("9");
    await p.locator("#live-reason").focus();
    await p.keyboard.type(" more");
    const during = await p.evaluate(() => `${document.querySelector("#svc-days").readOnly}|${document.querySelector("#svc-days").value}|${document.querySelector("#live-reason").value}`);
    check("typesending: inputs are read-only and unchanged while sending", during === "true|1|Typed while sending", during);
    check("typesending: no Discard on a first send", (await p.locator(".live-panel button", { hasText: "Discard and start over" }).count()) === 0);
    release();
    await p.locator(".live-held h3", { hasText: "Outcome unknown" }).waitFor();
    await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
    await p.locator(".live-held", { hasText: "Sending this request" }).waitFor();
    check("typesending: Discard is shown but disabled while a retry is sending", await p.locator(".live-panel button", { hasText: "Discard and start over" }).isDisabled());
    release();
    check("typesending: the secret is shown", await p.locator("#svc-secret").waitFor({ timeout: 10000 }).then(() => true).catch(() => false));
    check("typesending: the retry resent the first key", st.keys.length >= 2 && st.keys.at(-1) === st.keys.at(-2), st.keys.join(" "));
    await p.unroute(issuePath);
    await tab(p, "Accounts");
    await p.locator("#svc-list-title").waitFor();
    if (!(await p.locator("#svc-name").isVisible())) await p.locator(".svc-create-open").click();
    let cRelease;
    let cCalls = 0;
    await p.route("**/api/v1/commands/service.create", safe(async (r) => {
      cCalls += 1;
      if (cCalls > 1) return r.continue();
      await new Promise((y) => { cRelease = y; });
      await r.fetch();
      await r.abort();
    }));
    await p.locator("#svc-name").fill("typed-create");
    await p.locator("#svc-owner").selectOption({ label: login });
    await p.locator("#live-reason").fill("Create while sending");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator(".live-held", { hasText: "Sending this request" }).waitFor();
    await p.locator("#svc-name").focus();
    await p.keyboard.type("zz");
    const create = await p.evaluate(() => `${document.querySelector("#svc-name").readOnly}|${document.querySelector("#svc-name").value}|${document.querySelector("#svc-owner").tagName}|${document.querySelector("#svc-owner").readOnly}`);
    check("typesending: the create form is locked with a read-only owner input", create === "true|typed-create|INPUT|true", create);
    cRelease();
    await p.locator(".live-held h3", { hasText: "Outcome unknown" }).waitFor();
    const keys = st.keys.length;
    await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
    await p.locator(".service-status", { hasText: /svc-typed-create\W* created/ }).waitFor();
    check("typesending: the create retry resent the first key", st.keys.length === keys + 1 && st.keys.at(-1) === st.keys.at(-2));
    await p.unroute("**/api/v1/commands/service.create");
  });
  for (const [name, status, body] of [
    ["idemconflict", 409, { code: "idempotency_conflict", retry: false, next_action: "none", request_id: "synthetic" }],
    ["replaypartial", 500, { code: "internal_error", retry: false, next_action: "none", request_id: "synthetic", operation_id: "op-synthetic" }],
  ]) {
    await run(name, async (p) => {
      const svc = await newService(p, name);
      await lostIssue(p, svc);
      await p.route(issuePath, (r) => r.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) }));
      await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
      await p.locator(".live-held h3", { hasText: "Outcome can no longer be checked" }).waitFor({ timeout: 5000 }).catch(() => {});
      const panel = await p.locator(".live-panel").innerText();
      check(`${name} (synthetic ${status}): the attempt lapses`, panel.includes("Outcome can no longer be checked") && !panel.includes("Try again"), panel.slice(0, 200));
      check(`${name}: only Discard and start over is offered`, (await p.locator(".live-panel button", { hasText: "Discard and start over" }).count()) === 1);
      check(`${name}: the reminder keeps a lapsed row`, (await attemptRow(p, svc).innerText()).includes("can no longer be checked"));
      for (const banned of ["Could not save this change", "nothing was saved", "so you don't issue twice"]) {
        check(`${name}: no "${banned}"`, !(await p.locator("body").innerText()).includes(banned));
      }
      await p.unroute(issuePath);
    });
  }
  await run("window", async (p, st) => {
    await p.route("**/api/v1/identity/services", safe(async (r) => {
      const real = await r.fetch();
      const body = await real.json();
      body.limits.replay_seconds = 10;
      await r.fulfill({ response: real, json: body });
    }));
    await p.locator("#svc-cred-title").waitFor();
    const svc = await newService(p, "window");
    await lostIssue(p, svc);
    await p.waitForTimeout(10500);
    const keys = st.keys.length;
    await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
    await p.locator(".live-held h3", { hasText: "Outcome can no longer be checked" }).waitFor({ timeout: 5000 }).catch(() => {});
    check("window (synthetic replay_seconds 10): no request after the window", st.keys.length === keys);
    check("window: the attempt lapses", await p.locator(".live-held h3", { hasText: "Outcome can no longer be checked" }).isVisible());
    await p.unroute("**/api/v1/identity/services");
  });
  await run("issueddrop", async (p) => {
    const svc = await newService(p, "issueddrop");
    await issueForm(p);
    let release;
    await p.route(issuePath, safe(async (r) => { await new Promise((y) => { release = y; }); await r.continue(); }));
    await p.locator("#svc-days").fill("1");
    await p.locator("#live-reason").fill("Dropped");
    const done = p.waitForResponse((r) => r.url().endsWith("/commands/service.issue_credential"));
    await p.locator(".live-panel form button[type=submit]").click();
    await tab(p, "Accounts");
    release();
    const credential = (await (await done).json()).fields.credential;
    await p.locator(".live-attempts-note").waitFor();
    await p.unroute(issuePath);
    const second = await ctx.newPage();
    await second.goto(url + "?service=api&lang=en#/service-accounts?lang=en&view=credentials");
    await second.locator("#svc-select").selectOption(svc);
    await second.locator(`[data-credential="${credential}"]`).click();
    await second.locator(".live-panel form button[type=submit]").waitFor();
    await second.locator("#live-reason").fill("Revoked elsewhere");
    await second.locator(".live-panel form button[type=submit]").click();
    await second.locator(".service-status", { hasText: "revoked" }).waitFor();
    await second.close();
    await p.locator(".live-tools button", { hasText: "Reload" }).click();
    await p.waitForTimeout(500);
    check("issueddrop: a fresh read drops the issued entry of a revoked credential", (await p.locator(".live-attempts-note").count()) === 0);
    check("issueddrop: no unload guard remains", !(await p.evaluate(() => { const e = new Event("beforeunload", { cancelable: true }); dispatchEvent(e); return e.defaultPrevented; })));
  });
  await run("signoutheld", async (p, st) => {
    const svc = await newService(p, "signoutheld");
    const credential = await lostIssue(p, svc);
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    check("signoutheld: signing out with an unknown outcome raises no dialog", st.dialogs.length === 0, st.dialogs.join(","));
    await signIn(p);
    await p.goto(url + "?service=api&lang=en#/service-accounts?lang=en&view=credentials");
    await p.locator("#svc-select").selectOption(svc);
    const after = await p.locator(`tr:has([data-credential="${credential}"])`).innerText().catch(() => "");
    check("signoutheld: after signing in again the secret reads never collected", after.includes("Secret never collected"), after);
    await issueForm(p);
    await p.locator("#svc-days").fill("1");
    await p.locator("#live-reason").fill("Shown then sign out");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator("#svc-secret").waitFor();
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    check("signoutheld: signing out with a secret on screen raises no dialog", st.dialogs.length === 0, st.dialogs.join(","));
    await signIn(p);
  });
  await run("create", async (p, st) => {
    await tab(p, "Accounts");
    await p.locator("#svc-list-title").waitFor();
    const lost = await lose(p, "**/api/v1/commands/service.create");
    await p.locator("#svc-name").fill("lost-create");
    await p.locator("#svc-owner").selectOption({ label: login });
    await p.locator("#live-reason").fill("Lost create");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator(".live-held h3", { hasText: "Outcome unknown" }).waitFor();
    await lost.done();
    const keys = st.keys.length;
    await p.locator(".live-panel form button[type=submit]", { hasText: "Try again" }).click();
    const created = await p.locator(".service-status", { hasText: /svc-lost-create\W* created/ }).waitFor({ timeout: 10000 }).then(() => true).catch(() => false);
    check("create: the retry replays the committed create", created && st.keys.length === keys + 1 && st.keys.at(-1) === st.keys.at(-2));
    check("create: no name-taken error", !(await p.locator("body").innerText()).includes("already used"));
  });
  await run("createleave", async (p) => {
    await tab(p, "Accounts");
    await p.locator("#svc-list-title").waitFor();
    const lost = await lose(p, "**/api/v1/commands/service.create");
    await p.locator("#svc-name").fill("left-create");
    await p.locator("#svc-owner").selectOption({ label: login });
    await p.locator("#live-reason").fill("Lost create");
    await p.locator(".live-panel form button[type=submit]").click();
    await p.locator(".live-held h3", { hasText: "Outcome unknown" }).waitFor();
    await lost.done();
    await tab(p, "Credentials");
    await tab(p, "Accounts");
    await p.locator("#svc-name").fill("left-create");
    await p.locator("#svc-owner").selectOption({ label: login });
    await p.locator("#live-reason").fill("Again");
    await p.locator(".live-panel form button[type=submit]").click();
    const error = await p.locator("#svc-name-error").innerText({ timeout: 10000 }).catch(() => "");
    check("createleave: the name error says you created it", error.includes("already exists and you created it"), error);
  });
  await ctx.close();
}

try {
  await activate();
  const page = await signedIn();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
  if (only.has("flip")) {
    await createService(page, "flip");
    report(
      "held issue across 1440 to 1000 to 1440 px",
      await heldIssue(page, "svc-flip", async () => {}, async () => {
        await page.setViewportSize({ width: 1000, height: 900 });
        await page.locator(".live-users[data-wide=false]").waitFor();
        await page.setViewportSize({ width: 1440, height: 900 });
        await page.locator(".live-users[data-wide=true]").waitFor();
      }),
    );
  }
  if (only.has("narrow")) {
    await createService(page, "narrow");
    let visible = false;
    report(
      "held issue retried at 1000 px",
      await heldIssue(page, "svc-narrow", async () => {}, async () => {
        await page.setViewportSize({ width: 1000, height: 900 });
        await page.locator(".live-users[data-wide=false]").waitFor();
        visible = await page
          .locator(".live-panel button[type=submit]", { hasText: "Try again" })
          .isVisible();
      }),
    );
    check("held issue stays on screen at 1000 px", visible);
    await page.setViewportSize({ width: 1440, height: 900 });
  }
  if (only.has("inflight")) {
    await createService(page, "inflight");
    report(
      "reload while the issue request is in flight",
      await heldIssue(
        page,
        "svc-inflight",
        async () => {
          const read = page.waitForResponse(servicesRead);
          await page
            .locator(".live-tools button", { hasText: "Reload" })
            .click();
          await read;
        },
        async () => {},
      ),
    );
  }
  if (only.has("cancel")) {
    await createService(page, "cancel");
    await page.setViewportSize({ width: 390, height: 900 });
    await view(page, "credentials");
    await page.locator("#svc-cred-title").waitFor();
    await page.locator("#svc-select").selectOption("svc-cancel");
    await page.locator(".svc-issue-open").click();
    await page.locator("#svc-days").fill("1");
    await page.locator("#live-reason").fill("Narrow cancel");
    await submit(page);
    await page.locator("#svc-secret").waitFor();
    await page
      .locator(".live-panel button", { hasText: "I didn't store it" })
      .click();
    await page.locator(".live-step h2", { hasText: "Revoke credential" }).waitFor();
    const id = (await page.locator(".live-step .live-who").innerText()).trim();
    await page.locator(".live-panel button", { hasText: "Cancel" }).click();
    await page.locator("#svc-cred-title").waitFor({ timeout: 5000 }).catch(() => {});
    check(
      "cancelled revoke at 390 px returns to the list",
      await page.locator("#svc-cred-title").isVisible(),
    );
    check(
      "cancelled revoke at 390 px focuses the credential row",
      await page.evaluate(
        (x) => document.activeElement?.getAttribute("data-credential") === x,
        id,
      ),
      id,
    );
  }
  await newScenarios(errors);
  check("no page errors", errors.length === 0, JSON.stringify(errors));
} finally {
  await browser.close();
}
console.log(results.join("\n"));
process.exitCode = results.some((r) => r.startsWith("FAIL")) ? 1 : 0;
