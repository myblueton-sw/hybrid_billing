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
const organization = args.organization || "sample-le-seller";
const audience = args.audience || "test-service-api";
const scope = args.scope || "read";
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
const note = (name, why) => results.push(`NOTE ${name}: ${why}`);
const issuePath = "**/api/v1/commands/service.issue_credential";
const createPath = "**/api/v1/commands/service.create";
const isIssue = (r) => r.url().endsWith("/commands/service.issue_credential");
const accountList = (u) => u.pathname === "/api/v1/identity/services";
const accountDetail = (u) => u.pathname.startsWith("/api/v1/identity/services/");
const route = (params) => "#/service-accounts?" + new URLSearchParams({ lang: "en", ...params });
const credentialsView = (params = {}) => url + "?service=api&lang=en" + route({ view: "credentials", ...params });

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

const safe = (handle) => async (r) => {
  try {
    await handle(r);
  } catch {
    return;
  }
};
const gate = () => {
  let open;
  const shut = new Promise((y) => { open = y; });
  return { shut, open: () => open() };
};

const tab = (p, name) => p.locator(".planning-tabs button", { hasText: new RegExp(`^${name}$`) }).click();
const issueForm = (p) => p.locator("form:has(#svc-issue-title)");
const tryAgain = (p) => issueForm(p).getByRole("button", { name: "Try again", exact: true });
const createForm = (p) => p.locator("form").filter({ has: p.getByRole("heading", { name: "Create service account", exact: true }) });
const panel = (p) => p.locator(".live-panel");
const revokeHeading = (p) => p.getByRole("heading", { name: "Revoke credential", exact: true });
const reminder = (p, service) => p.locator(".live-attempts li").filter({ hasText: service });
const pending = (p) => p.locator("section").filter({ has: p.getByRole("heading", { name: "Credentials awaiting collection", exact: true }) }).locator("li");
const credentialRow = (p, credential) => p.locator("tr").filter({ hasText: credential });
const guarded = (p) => p.evaluate(() => { const e = new Event("beforeunload", { cancelable: true }); dispatchEvent(e); return e.defaultPrevented; });
const visible = (locator, timeout = 10000) => locator.waitFor({ timeout }).then(() => true).catch(() => false);

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
      if (isIssue(r)) state.issueBodies.push(r.postData());
    }
  });
  p.on("response", async (r) => {
    if (!isIssue(r)) return;
    const body = await r.json().catch(() => ({}));
    if (body.fields?.secret_ref) state.refs.push(body.fields.secret_ref);
  });
  return state;
}

async function opened(ctx, errors, width = 1440) {
  const p = await ctx.newPage();
  await p.setViewportSize({ width, height: 900 });
  const st = watch(p, errors);
  await p.goto(credentialsView());
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

async function fillCreate(p, name, reason = "Regression check") {
  await tab(p, "Accounts");
  await p.locator("#svc-list-title").waitFor();
  const form = createForm(p);
  if (!(await form.count())) await p.locator("[data-svc-create]").click();
  await form.locator("#svc-name").fill(name);
  await form.getByRole("group", { name: "Responsible owner", exact: true }).locator("select").selectOption({ label: login });
  await form.locator("#svc-purpose").fill("Regression check of held credential issues");
  await form.getByRole("group", { name: "Owning organization", exact: true }).locator("select").selectOption(organization);
  await form.getByRole("group", { name: "Audience", exact: true }).locator("select").selectOption(audience);
  await form.getByRole("group", { name: "Declared scopes", exact: true }).locator("select").selectOption(scope);
  await form.getByRole("button", { name: "Add scope", exact: true }).click();
  await form.locator("#live-reason").fill(reason);
  return form;
}

async function newService(p, name) {
  const size = p.viewportSize();
  if (size.width < 1280) await p.setViewportSize({ width: 1440, height: 900 });
  const form = await fillCreate(p, name);
  await form.locator("button[type=submit]").click();
  await p.getByRole("heading", { name: `svc-${name}`, exact: true }).waitFor();
  await p.getByRole("button", { name: "Manage credentials", exact: true }).click();
  await issueForm(p).locator("#svc-days").waitFor();
  if (size.width < 1280) {
    await p.setViewportSize(size);
    await p.waitForFunction((w) => innerWidth === w, size.width);
  }
  return `svc-${name}`;
}

async function pick(p, service) {
  const group = p.getByRole("group", { name: "Account for credential issuance", exact: true });
  await group.locator("input").fill(service);
  await group.getByRole("button", { name: "Search", exact: true }).click();
  await group.locator("select").selectOption(service);
  await p.locator("#svc-issue-title + p", { hasText: service }).waitFor();
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

async function sendIssue(p, days, reason) {
  await issueForm(p).locator("#svc-days").fill(days);
  await issueForm(p).locator("#live-reason").fill(reason);
  await issueForm(p).locator("button[type=submit]").click();
}

async function lostIssue(p, days = "1", reason = "Lost response") {
  const lost = await lose(p, issuePath);
  await sendIssue(p, days, reason);
  await tryAgain(p).waitFor();
  await lost.done();
  return lost.sent[0]?.fields?.credential ?? "";
}

async function reveal(p) {
  const show = issueForm(p).getByRole("button", { name: "Show secret", exact: true });
  if (!(await visible(show))) return false;
  await show.click();
  return visible(p.locator("#svc-secret"));
}

async function issued(p, days, reason) {
  await sendIssue(p, days, reason);
  await reveal(p);
  return (await panel(p).innerText()).match(/cred-[0-9a-f]{16}/)?.[0] ?? "";
}

async function retried(name, p, st) {
  const before = st.keys.length;
  await tryAgain(p).click();
  const shown = await reveal(p);
  check(`${name}: the retry resends the same idempotency key`, st.keys.length === before + 1 && st.keys.at(-1) === st.keys.at(-2), st.keys.slice(-2).join(" "));
  check(`${name}: the secret is shown after the retry`, shown);
  check(`${name}: the complete issue body stays byte-identical on retry`, st.issueBodies.length >= 2 && st.issueBodies.at(-1) === st.issueBodies.at(-2));
}

const formState = (p) => issueForm(p).evaluate((f) => {
  const days = f.querySelector("#svc-days"), reason = f.querySelector("#live-reason");
  return `${days?.value}|${days?.matches(":disabled")}|${reason?.value}|${reason?.matches(":disabled")}|${f.querySelector("button[type=submit]")?.textContent}`;
}).catch(() => "no form");

async function heldForm(name, p) {
  const form = await formState(p);
  check(`${name}: the held form is restored with locked values`, form === "1|true|Lost response|true|Try again", form);
}

const focused = (p) => p.evaluate(() => {
  const a = document.activeElement;
  const text = (e) => (e?.innerText ?? e?.textContent ?? "").replace(/\s+/g, " ").trim();
  return {
    tag: a?.tagName ?? "",
    id: a?.id ?? "",
    text: text(a).slice(0, 60),
    row: text(a?.closest("tr")),
    li: a?.closest(".live-attempts li") ? text(a.closest("li")) : "",
    step: text(a?.closest(".live-step")?.querySelector("h2,h3")),
  };
});
const shownFocus = (f) => `${f.tag}#${f.id} "${f.text}" row="${f.row.slice(0, 40)}" li="${f.li.slice(0, 40)}" step="${f.step}"`;

async function heldIssue(p, duringFirst, whileHeld) {
  const posts = [];
  const record = (r) => {
    if (r.method() === "POST" && isIssue(r)) posts.push(JSON.parse(r.postData() ?? "{}"));
  };
  p.on("request", record);
  const lost = await lose(p, issuePath, duringFirst);
  await sendIssue(p, "1", "Lost response");
  await tryAgain(p).waitFor();
  await lost.done();
  await whileHeld();
  const state = await formState(p);
  const form = state === "no form" ? state : state.split("|").filter((_, i) => i % 2 === 0).join("|");
  if (state !== "no form") await tryAgain(p).click();
  const shown = await reveal(p);
  p.off("request", record);
  const same =
    posts.length === 2 &&
    posts[0].request_id === posts[1].request_id &&
    posts[0].idempotency_key === posts[1].idempotency_key &&
    JSON.stringify(posts[0]) === JSON.stringify(posts[1]);
  if (shown) {
    await panel(p).getByRole("button", { name: "I didn't store it: revoke", exact: true }).click();
    await revokeHeading(p).waitFor();
    await panel(p).locator("form button[type=submit]").click();
    await revokeHeading(p).waitFor({ state: "detached" });
  }
  return { form, shown, same, posts: posts.length };
}

const report = (name, r) => {
  check(`${name}: form kept`, r.form === "1|Lost response|Try again", r.form);
  check(`${name}: same request_id and idempotency_key on the retry`, r.same, `posts=${r.posts}`);
  check(`${name}: secret shown after the retry`, r.shown);
};

const resize = async (p, width) => {
  await p.setViewportSize({ width, height: 900 });
  await p.waitForFunction((w) => innerWidth === w, width);
};

async function heldScenarios(run) {
  await run("flip", async (p) => {
    await newService(p, "flip");
    report(
      "held issue across 1440 to 1000 to 1440 px",
      await heldIssue(p, async () => {}, async () => {
        await resize(p, 1000);
        await resize(p, 1440);
      }),
    );
  });
  await run("narrow", async (p) => {
    await newService(p, "narrow");
    let shown = false;
    report(
      "held issue retried at 1000 px",
      await heldIssue(p, async () => {}, async () => {
        await resize(p, 1000);
        shown = await tryAgain(p).isVisible();
      }),
    );
    check("held issue stays on screen at 1000 px", shown);
  });
  await run("inflight", async (p) => {
    await newService(p, "inflight");
    report(
      "reload while the issue request is in flight",
      await heldIssue(
        p,
        async () => {
          const read = p.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/identity/service-credentials");
          await p.locator(".live-toolbar:has(#svc-cred-title) button", { hasText: "Reload" }).click();
          await read;
        },
        async () => {},
      ),
    );
  });
  await run("cancel", async (p) => {
    await newService(p, "cancel");
    await resize(p, 390);
    const id = await issued(p, "1", "Narrow cancel");
    await panel(p).getByRole("button", { name: "I didn't store it: revoke", exact: true }).click();
    await revokeHeading(p).waitFor();
    await panel(p).getByRole("button", { name: "Cancel", exact: true }).click();
    await p.locator("#svc-cred-title").waitFor({ timeout: 5000 }).catch(() => {});
    await p.waitForTimeout(200);
    check("cancelled revoke at 390 px returns to the list", await p.locator("#svc-cred-title").isVisible());
    const f = await focused(p);
    check("cancelled revoke at 390 px focuses the credential row", !!id && f.row.includes(id), `${id} ${shownFocus(f)}`);
  }, 390);
}

async function scenariosIn(errors) {
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
  await heldScenarios(run);
  await run("leave", async (p, st) => {
    const svc = await newService(p, "leave");
    const kept = await mark(p);
    await lostIssue(p);
    await tab(p, "Accounts");
    await p.locator("#svc-list-title").waitFor();
    check("leave: the accounts view shows the held issue reminder", await visible(reminder(p, svc), 5000));
    await tab(p, "Credentials");
    const row = await reminder(p, svc).innerText().catch(() => "");
    check("leave: the reminder lists the unknown outcome", /outcome unknown/i.test(row), row);
    await reminder(p, svc).getByRole("button", { name: "Open issue form", exact: true }).click();
    await heldForm("leave", p);
    check("leave: no reload", await kept());
    await retried("leave", p, st);
  });
  await run("selector", async (p, st) => {
    const other = await newService(p, "other");
    const svc = await newService(p, "selector");
    const kept = await mark(p);
    await lostIssue(p);
    await pick(p, other);
    check("selector: the reminder lists the held issue", await reminder(p, svc).isVisible());
    check("selector: the other account offers a fresh issue form", (await formState(p)).endsWith("|false|Issue credential"), await formState(p));
    await pick(p, svc);
    await heldForm("selector", p);
    check("selector: no reload", await kept());
    await retried("selector", p, st);
  });
  await run("revokebtn", async (p, st) => {
    const svc = await newService(p, "revokebtn");
    const existing = await issued(p, "2", "Existing");
    await p.locator("#svc-ack").check();
    await panel(p).getByRole("button", { name: "Done", exact: true }).click();
    await p.locator("#svc-secret").waitFor({ state: "detached" });
    await issueForm(p).locator("#svc-days").waitFor();
    const kept = await mark(p);
    await lostIssue(p);
    await credentialRow(p, existing).getByRole("button", { name: /View details/ }).click();
    const step = p.locator(".live-step").filter({ has: p.getByRole("heading", { name: existing, exact: true }) });
    await step.getByRole("button", { name: "Revoke", exact: true }).click();
    await revokeHeading(p).waitFor();
    await panel(p).getByRole("button", { name: "Cancel", exact: true }).click();
    await p.waitForTimeout(200);
    const f = await focused(p);
    check("revokebtn: Cancel returns focus to the credential's Revoke button", f.text === "Revoke" && f.step === existing, `${svc} ${shownFocus(f)}`);
    await heldForm("revokebtn", p);
    check("revokebtn: no reload", await kept());
    await retried("revokebtn", p, st);
  });
  await run("backnarrow", async (p, st) => {
    const svc = await newService(p, "backnarrow");
    const kept = await mark(p);
    await lostIssue(p);
    await p.goBack();
    await p.locator("#svc-list-title").waitFor();
    check("backnarrow: browser Back keeps the held issue as a reminder", await visible(reminder(p, svc), 5000));
    await reminder(p, svc).getByRole("button", { name: "Open issue form", exact: true }).click();
    await issueForm(p).locator("#svc-days").waitFor();
    await heldForm("backnarrow", p);
    check("backnarrow: no reload", await kept());
    await retried("backnarrow", p, st);
  }, 1000);
  await run("inflightleave", async (p) => {
    const svc = await newService(p, "inflight-leave");
    const kept = await mark(p);
    const issue = gate(), read = gate();
    let heldRead = false;
    await p.route(issuePath, safe(async (r) => { await issue.shut; await r.continue(); }));
    const outcome = p.waitForResponse(isIssue);
    await sendIssue(p, "1", "In flight");
    const holdRead = safe(async (r) => {
      if (heldRead) return r.continue();
      heldRead = true;
      const real = await r.fetch();
      await read.shut;
      await r.fulfill({ response: real });
    });
    await p.route(accountList, holdRead);
    await tab(p, "Accounts");
    await p.waitForTimeout(300);
    issue.open();
    await outcome;
    await reminder(p, svc).filter({ hasText: "has not been shown" }).waitFor();
    read.open();
    await p.waitForTimeout(500);
    check("inflightleave: a read that started before the outcome does not drop the issued entry", heldRead && await reminder(p, svc).isVisible());
    await p.unroute(accountList, holdRead);
    await p.unroute(issuePath);
    note("inflightleave: focus moves to the reminder heading", "no longer exists by design: the reminder region itself renders above both the Accounts and Credentials views (Live.tsx:147 <RecoveryRegion> outside the view switch), so there is no separate accounts note with a jump button (live-copy.ts accountsAttemptsNote has no reader)");
    const row = await reminder(p, svc).innerText();
    check("inflightleave: the issued credential waits for Show secret", row.includes("has not been shown"), row);
    await reminder(p, svc).getByRole("button", { name: "Show secret", exact: true }).click();
    check("inflightleave: Show secret shows the secret", await visible(p.locator("#svc-secret")));
    check("inflightleave: no reload", await kept());
  });
  await run("reload", async (p, st) => {
    const svc = await newService(p, "reload");
    const credential = await lostIssue(p);
    check("reload: the page guards against unload while the outcome is unknown", await guarded(p));
    const seen = new Promise((y) => { st.armed = y; });
    await p.reload();
    check("reload: the browser asked before reloading", (await Promise.race([seen, new Promise((y) => setTimeout(() => y("none"), 3000))])) === "beforeunload");
    await p.locator("#svc-cred-title").waitFor();
    await p.goto(credentialsView({ svc_c_service: svc }));
    await p.locator("#svc-cred-title").waitFor();
    await credentialRow(p, credential).waitFor({ timeout: 10000 }).catch(() => {});
    const cell = await credentialRow(p, credential).innerText().catch(() => "");
    check("reload: the row says the secret is not collected yet", cell.includes("Secret not collected yet") && cell.includes("estimated to end"), cell);
    check("reload: Active still shows", cell.includes("Active"), cell);
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    await signIn(p);
    await p.goto(credentialsView({ svc_c_service: svc }));
    await p.locator("#svc-cred-title").waitFor();
    await credentialRow(p, credential).waitFor({ timeout: 10000 }).catch(() => {});
    const after = await credentialRow(p, credential).innerText().catch(() => "");
    check("reload: after sign-out the row says the secret was not returned", after.includes("Secret not returned"), after);
    const listed = pending(p).filter({ hasText: credential });
    check("reload: the awaiting-collection region lists it", await visible(listed, 5000));
    await listed.getByRole("button", { name: "Revoke", exact: true }).click();
    await revokeHeading(p).waitFor();
    const prefilled = await panel(p).locator("#live-reason").inputValue();
    note("reload: the revoke reason is prefilled", `no longer exists by design: the awaiting-collection list opens revoke without a reason (Live.tsx:805 select={(row) => revoke(row.id, row.service)}); prefill is kept only for the reveal and unstored-secret reminders (Live.tsx:1133, 1169); observed "${prefilled}"`);
    if (!prefilled) await panel(p).locator("#live-reason").fill("Secret never collected");
    await panel(p).locator("form button[type=submit]").click();
    await revokeHeading(p).waitFor({ state: "detached" });
    await p.waitForTimeout(200);
    const f = await focused(p);
    check("reload: focus reaches the credentials heading", f.tag === "H2" && f.id === "svc-cred-title", shownFocus(f));
    await p.waitForTimeout(300);
    check("reload: the region is gone", (await pending(p).filter({ hasText: credential }).count()) === 0);
  });
  for (const width of [1440, 390]) {
    await run("remindercancel", async (p) => {
      await newService(p, `mark-a-${width}`);
      const id = await issued(p, "1", "Unstored");
      const unstored = p.locator(".live-attempts li").filter({ hasText: id });
      await unstored.getByRole("button", { name: "Revoke", exact: true }).click();
      await revokeHeading(p).waitFor();
      await panel(p).getByRole("button", { name: "Cancel", exact: true }).click();
      await p.waitForTimeout(200);
      const f = await focused(p);
      check(`remindercancel at ${width}: focus returns to the reminder's Revoke button`, !!id && f.text === "Revoke" && f.li.includes(id), `${id} ${shownFocus(f)}`);
    }, width);
  }
  await run("revealcancel", async (p) => {
    const svc = await newService(p, "revealcancel");
    const id = await issued(p, "1", "Reveal cancel");
    await panel(p).getByRole("button", { name: "I didn't store it: revoke", exact: true }).click();
    await revokeHeading(p).waitFor();
    await panel(p).getByRole("button", { name: "Cancel", exact: true }).click();
    await p.waitForTimeout(200);
    const f = await focused(p);
    check("revealcancel: focus returns to the credential row", !!id && f.row.includes(id), `${svc} ${id} ${shownFocus(f)}`);
  });
  await run("typesending", async (p, st) => {
    await newService(p, "typesending");
    const firstSend = gate(), retrySend = gate();
    let calls = 0;
    await p.route(issuePath, safe(async (r) => {
      calls += 1;
      if (calls > 2) return r.continue();
      await (calls === 1 ? firstSend : retrySend).shut;
      const real = await r.fetch();
      if (calls === 2) return r.fulfill({ response: real });
      await r.abort();
    }));
    const discard = p.getByRole("button", { name: "Discard", exact: true });
    const sending = issueForm(p).locator("[role=status]", { hasText: "Sending this request" });
    await sendIssue(p, "1", "Typed while sending");
    await sending.waitFor();
    await p.locator("#svc-days").focus().catch(() => {});
    await p.keyboard.type("9");
    await p.locator("#live-reason").focus().catch(() => {});
    await p.keyboard.type(" more");
    const during = await p.evaluate(() => `${document.querySelector("#svc-days").matches(":disabled")}|${document.querySelector("#svc-days").value}|${document.querySelector("#live-reason").value}`);
    check("typesending: inputs are locked and unchanged while sending", during === "true|1|Typed while sending", during);
    check("typesending: no Discard on a first send", (await discard.count()) === 0);
    firstSend.open();
    await tryAgain(p).waitFor();
    await tryAgain(p).click();
    await sending.waitFor();
    const usable = await discard.evaluateAll((all) => all.filter((b) => !b.disabled).length);
    check("typesending: Discard cannot be used while a retry is sending", usable === 0, `enabled=${usable} shown=${await discard.count()}`);
    retrySend.open();
    check("typesending: the secret is shown", await reveal(p));
    check("typesending: the retry resent the first key", st.keys.length >= 2 && st.keys.at(-1) === st.keys.at(-2), st.keys.join(" "));
    await p.unroute(issuePath);
    await p.locator("#svc-ack").check();
    await panel(p).getByRole("button", { name: "Done", exact: true }).click();
    await p.locator("#svc-secret").waitFor({ state: "detached" });
    const createSend = gate();
    let cCalls = 0;
    await p.route(createPath, safe(async (r) => {
      cCalls += 1;
      if (cCalls > 1) return r.continue();
      await createSend.shut;
      await r.fetch();
      await r.abort();
    }));
    const form = await fillCreate(p, "typed-create", "Create while sending");
    await form.locator("button[type=submit]").click();
    await form.locator(".live-held").waitFor();
    await form.locator("#svc-name").focus().catch(() => {});
    await p.keyboard.type("zz");
    const create = await form.evaluate((f) => `${f.querySelector("#svc-name").matches(":disabled")}|${f.querySelector("#svc-name").value}|${f.querySelector("#svc-owner select")?.matches(":disabled")}`);
    check("typesending: the create form is locked with a locked owner picker", create === "true|typed-create|true", create);
    createSend.open();
    const retry = form.getByRole("button", { name: "Try again", exact: true });
    await retry.waitFor();
    const keys = st.keys.length;
    await retry.click();
    await p.getByRole("heading", { name: "svc-typed-create", exact: true }).waitFor();
    check("typesending: the create retry resent the first key", st.keys.length === keys + 1 && st.keys.at(-1) === st.keys.at(-2));
    await p.unroute(createPath);
  });
  for (const [name, status, body] of [
    ["idemconflict", 409, { code: "idempotency_conflict", retry: false, next_action: "none", request_id: "synthetic" }],
    ["replaypartial", 500, { code: "internal_error", retry: false, next_action: "none", request_id: "synthetic", operation_id: "op-synthetic" }],
  ]) {
    await run(name, async (p) => {
      const svc = await newService(p, name);
      await lostIssue(p);
      await p.route(issuePath, (r) => r.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) }));
      await tryAgain(p).click();
      await issueForm(p).locator("[role=status]", { hasText: "Outcome can no longer be checked" }).waitFor({ timeout: 5000 }).catch(() => {});
      const shown = await issueForm(p).innerText();
      check(`${name} (synthetic ${status}): the attempt lapses`, shown.includes("Outcome can no longer be checked") && !shown.includes("Try again"), shown.slice(0, 200));
      check(`${name}: only Discard is offered`, (await issueForm(p).getByRole("button", { name: "Discard", exact: true }).count()) === 1);
      check(`${name}: the reminder keeps a lapsed row`, (await reminder(p, svc).innerText()).includes("can no longer be checked"));
      for (const banned of ["Could not save this change", "nothing was saved", "so you don't issue twice"]) {
        check(`${name}: no "${banned}"`, !(await p.locator("body").innerText()).includes(banned));
      }
      await p.unroute(issuePath);
    });
  }
  await run("window", async (p, st) => {
    const shorten = safe(async (r) => {
      const real = await r.fetch();
      const body = await real.json();
      if (body.limits?.replay_seconds) body.limits.replay_seconds = 10;
      await r.fulfill({ response: real, json: body });
    });
    await p.route(accountDetail, shorten);
    await newService(p, "window");
    await lostIssue(p);
    await p.waitForTimeout(10500);
    const keys = st.keys.length;
    await tryAgain(p).click();
    const lapsed = issueForm(p).locator("[role=status]", { hasText: "Outcome can no longer be checked" });
    await lapsed.waitFor({ timeout: 5000 }).catch(() => {});
    check("window (synthetic replay_seconds 10): no request after the window", st.keys.length === keys);
    check("window: the attempt lapses", await lapsed.isVisible());
    await p.unroute(accountDetail, shorten);
  });
  await run("issueddrop", async (p) => {
    const svc = await newService(p, "issueddrop");
    const release = gate();
    await p.route(issuePath, safe(async (r) => { await release.shut; await r.continue(); }));
    const done = p.waitForResponse(isIssue);
    await sendIssue(p, "1", "Dropped");
    await tab(p, "Accounts");
    release.open();
    const credential = (await (await done).json()).fields.credential;
    await reminder(p, svc).filter({ hasText: "has not been shown" }).waitFor();
    await p.unroute(issuePath);
    const second = await ctx.newPage();
    await second.goto(credentialsView({ svc_c_service: svc, svc_c_selected: credential }));
    const step = second.locator(".live-step").filter({ has: second.getByRole("heading", { name: credential, exact: true }) });
    await step.getByRole("button", { name: "Revoke", exact: true }).click();
    await revokeHeading(second).waitFor();
    await panel(second).locator("#live-reason").fill("Revoked elsewhere");
    await panel(second).locator("form button[type=submit]").click();
    await revokeHeading(second).waitFor({ state: "detached" });
    await second.close();
    await p.locator(".live-toolbar:has(#svc-list-title) button", { hasText: "Reload" }).click();
    await p.waitForTimeout(500);
    check("issueddrop: a fresh read drops the issued entry of a revoked credential", (await reminder(p, svc).count()) === 0);
    check("issueddrop: no unload guard remains", !(await guarded(p)));
  });
  await run("signoutheld", async (p, st) => {
    const svc = await newService(p, "signoutheld");
    const credential = await lostIssue(p);
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    check("signoutheld: signing out with an unknown outcome raises no dialog", st.dialogs.length === 0, st.dialogs.join(","));
    await signIn(p);
    await p.goto(credentialsView({ svc_c_service: svc, svc_c_issue: svc }));
    await credentialRow(p, credential).waitFor({ timeout: 10000 }).catch(() => {});
    const after = await credentialRow(p, credential).innerText().catch(() => "");
    check("signoutheld: after signing in again the secret reads not returned", after.includes("Secret not returned"), after);
    await issued(p, "1", "Shown then sign out");
    await p.locator(".service-signout").click();
    await p.locator("#signin-workspace").waitFor();
    check("signoutheld: signing out with a secret on screen raises no dialog", st.dialogs.length === 0, st.dialogs.join(","));
    await signIn(p);
  });
  await run("create", async (p, st) => {
    const lost = await lose(p, createPath);
    const form = await fillCreate(p, "lost-create", "Lost create");
    await form.locator("button[type=submit]").click();
    const retry = form.getByRole("button", { name: "Try again", exact: true });
    await retry.waitFor();
    await lost.done();
    const keys = st.keys.length;
    await retry.click();
    const created = await visible(p.getByRole("heading", { name: "svc-lost-create", exact: true }));
    check("create: the retry replays the committed create", created && st.keys.length === keys + 1 && st.keys.at(-1) === st.keys.at(-2));
    check("create: no name-taken error", !(await p.locator("body").innerText()).includes("already used"));
  });
  await run("createleave", async (p, st) => {
    const lost = await lose(p, createPath);
    const form = await fillCreate(p, "left-create", "Lost create");
    await form.locator("button[type=submit]").click();
    await form.getByRole("button", { name: "Try again", exact: true }).waitFor();
    await lost.done();
    await tab(p, "Credentials");
    await tab(p, "Accounts");
    await p.locator("#svc-list-title").waitFor();
    await p.locator("[data-svc-create]").click();
    const held = await createForm(p).evaluate((f) => `${f.querySelector("#svc-name")?.value}|${f.querySelector("#svc-name")?.matches(":disabled")}|${f.querySelector("button[type=submit]")?.textContent}`).catch(() => "no form");
    check("createleave: returning keeps the held create locked with Try again", held === "left-create|true|Try again", held);
    note("createleave: the name error says you created it", "no longer exists by design: a held create keeps the form locked to the original request until it is retried or discarded (DeclarationForm.tsx:196 locked = !!entry, :411 fieldset disabled), so a second same-name submit cannot be made while the outcome is unknown; live-copy.ts:228 errNameTakenMine has no reader");
    await p.getByRole("button", { name: "Discard retained intent", exact: true }).first().click();
    const keys = st.keys.length;
    const answered = p.waitForResponse((r) => r.url().endsWith("/commands/service.create"));
    await createForm(p).locator("button[type=submit]").click();
    const refusedStatus = (await answered).status();
    const refused = await visible(createForm(p).locator(".live-held", { hasText: "Command refused" }), 5000);
    check("createleave: a new create with the committed name is refused by the server", st.keys.length === keys + 1 && refusedStatus >= 400 && refused, `status=${refusedStatus}`);
    check("createleave: no second account opens", (await p.getByRole("heading", { name: "svc-left-create", exact: true }).count()) === 0);
  });
  await ctx.close();
}

try {
  await activate();
  const errors = [];
  await scenariosIn(errors);
  check("no page errors", errors.length === 0, JSON.stringify(errors));
} finally {
  await browser.close();
}
console.log(results.join("\n"));
process.exitCode = results.some((r) => r.startsWith("FAIL")) ? 1 : 0;
