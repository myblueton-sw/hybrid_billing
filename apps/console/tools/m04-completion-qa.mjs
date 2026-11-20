import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import { actionCompletion } from "./m04-completion-actions.mjs";
import { navigationCompletion } from "./m04-completion-navigation.mjs";
import { declarationCompletion } from "./m04-completion-declarations.mjs";

const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (all, key, i, values) =>
        key.startsWith("--") ? [...all, [key.slice(2), values[i + 1]]] : all,
      [],
    ),
);
assert.ok(
  args.url && args.activation,
  "Provide owned sample --url and --activation",
);
const group = args.group || "actions";
assert.ok(
  ["actions", "navigation", "declaration", "remaining"].includes(group),
);
const base = args.url.replace(/\/?$/, "/");
assert.ok(["127.0.0.1", "localhost", "[::1]"].includes(new URL(base).hostname));
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
});
const page = await context.newPage();
page.setDefaultTimeout(15000);
const login = args.login || "admin@example.invalid";
let owner = args.owner || "";
const workspace = args.workspace || "sample-workspace";
const output = args.output || "output/validation/m04-completion";
const password = randomBytes(18).toString("base64url");
const redactions = new Set([password]);
const safe = (message) => {
  let result = String(message);
  for (const value of redactions)
    if (value) result = result.replaceAll(value, "[redacted]");
  return result;
};
const prefix = "complete-" + randomBytes(5).toString("hex");
const results = [],
  captures = [],
  errors = [];
let csrf = "";
page.on("pageerror", (error) => errors.push(safe(error.message)));
page.on("dialog", (dialog) => dialog.accept());
const requestId = () => randomBytes(16).toString("hex");
async function read(path) {
  const response = await context.request.get(base + "api/v1" + path, {
    headers: { "X-CSRF-Token": csrf, "X-Request-Id": requestId() },
  });
  assert.equal(response.status(), 200, "actual canonical read failed: " + path);
  return response.json();
}
async function command(id, target, body) {
  const request = requestId();
  const revision = (await read("/identity/services?limit=1")).revision;
  const response = await context.request.post(base + "api/v1/commands/" + id, {
    headers: {
      "X-CSRF-Token": csrf,
      "X-Request-Id": request,
      Origin: new URL(base).origin,
    },
    data: {
      request_id: request,
      target_id: target,
      idempotency_key: requestId(),
      expected_revision: revision,
      rationale: "Synthetic bounded navigation fixture setup",
      body,
    },
  });
  assert.equal(
    response.status(),
    200,
    "Accepted synthetic setup command: " + id,
  );
  const result = await response.json();
  if (result.fields?.secret_ref) redactions.add(result.fields.secret_ref);
  assert.equal(result.status, "completed");
  return result;
}
async function navigate(view, extra = {}, locale = "en") {
  await page.evaluate(
    ({ view, extra, locale }) => {
      location.hash =
        "/service-accounts?" +
        new URLSearchParams({ lang: locale, view, ...extra });
    },
    { view, extra, locale },
  );
  await page
    .locator(`[data-security-screen="M04"][lang="${locale}"]`)
    .waitFor();
}
async function capture(locale, width, state, focus) {
  assert.equal(
    await page
      .locator("#svc-secret,#reauth-password,#signin-password,#link-new")
      .count(),
    0,
  );
  await page.evaluate(() => document.fonts.ready);
  const actual = await page.evaluate(() => {
    const screen = document.querySelector('[data-security-screen="M04"]');
    return {
      locale: screen.lang,
      direction: screen.dir,
      fits: document.documentElement.scrollWidth <= innerWidth,
      font: getComputedStyle(screen).fontFamily,
      focus: document.activeElement?.id,
    };
  });
  assert.equal(actual.locale, locale);
  assert.equal(actual.direction, locale === "ar" ? "rtl" : "ltr");
  assert.ok(actual.fits && actual.font.includes("IBM Plex"));
  if (focus) assert.equal(actual.focus, focus);
  await mkdir(output, { recursive: true });
  await page.screenshot({
    path: `${output}/${locale}-${width}-${state}.png`,
    fullPage: true,
  });
  captures.push({ locale, width, state, ...actual });
}
async function scenario(name, run) {
  try {
    await run();
    results.push({ name, status: "PASS" });
    console.log("PASS " + name);
  } catch (error) {
    results.push({
      name,
      status: "FAIL",
      error: safe(error.message),
    });
    throw error;
  }
}
async function create(suffix) {
  const name = prefix + "-" + suffix;
  await navigate("accounts");
  await page
    .getByRole("button", { name: "Create service account", exact: true })
    .click();
  const form = page.locator("form").filter({
    has: page.getByRole("heading", {
      name: "Create service account",
      exact: true,
    }),
  });
  await form.locator("#svc-name").fill(name);
  await form.locator("#svc-owner-select").selectOption(owner);
  await form
    .locator("#svc-purpose")
    .fill("Synthetic finite M04 completion journey");
  await form
    .locator("#svc-organization-select")
    .selectOption("sample-le-seller");
  await form
    .getByRole("group", { name: "Audience", exact: true })
    .locator("select")
    .selectOption("test-service-api");
  await form
    .getByRole("group", { name: "Declared scopes", exact: true })
    .locator("select")
    .selectOption("read");
  await form.getByRole("button", { name: "Add scope", exact: true }).click();
  await form
    .locator("#live-reason")
    .fill("Synthetic M04 completion account setup");
  const accepted = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname === "/api/v1/commands/service.create",
  );
  await form.locator("button[type=submit]").click();
  assert.equal((await accepted).status(), 200);
  const id = "svc-" + name;
  await page.getByRole("heading", { name: id, exact: true }).waitFor();
  const canonical = await read("/identity/services/" + id);
  assert.equal(canonical.row.declaration.version, "1");
  return id;
}
async function issue(service) {
  await navigate("credentials", {
    svc_c_service: service,
    svc_c_issue: service,
  });
  const form = page
    .locator("form")
    .filter({ has: page.locator("#svc-issue-title") });
  await form.locator("#svc-days").fill("1");
  await form
    .locator("#live-reason")
    .fill("Synthetic completion credential setup");
  const accepted = page.waitForResponse(
    (response) =>
      new URL(response.url()).pathname ===
      "/api/v1/commands/service.issue_credential",
  );
  await form.locator("button[type=submit]").click();
  const response = await accepted;
  assert.equal(response.status(), 200);
  const result = await response.json();
  assert.equal(result.status, "completed");
  assert.equal(result.business_state, "credential_issued");
  const credential = result.fields.credential;
  assert.ok(
    typeof credential === "string" && /^cred-[a-f0-9]{16}$/.test(credential),
    "Issued credential identifier shape",
  );
  await form
    .getByRole("button", { name: "Show secret", exact: true })
    .waitFor();
  return credential;
}

try {
  const line = (await readFile(args.activation, "utf8"))
    .trim()
    .split("\n")
    .find((value) => value.startsWith(login + " "));
  assert.ok(line, "Synthetic activation entry unavailable");
  const link = new URL(line.split(" ")[1]);
  const activationURL =
    base + link.pathname.replace(/^\//, "") + "?lang=en" + link.hash;
  for (const value of [
    link.href,
    activationURL,
    link.hash,
    link.hash.slice(1),
    ...new URLSearchParams(link.hash.slice(1)).values(),
  ])
    if (value.length > 4) redactions.add(value);
  await page.goto(activationURL);
  await page.locator("#link-new").fill(password);
  await page.locator("#link-confirm").fill(password);
  await page.getByRole("button", { name: "Set password", exact: true }).click();
  await page.locator("a.service-button-link").waitFor();
  await page.goto(base + "?service=api&lang=en");
  await page.locator("#signin-workspace").fill(workspace);
  await page.locator("#signin-email").fill(login);
  await page.locator("#signin-password").fill(password);
  await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  for (const cookie of await context.cookies(base))
    redactions.add(cookie.value);
  const sessionResponse = await context.request.get(base + "api/v1/session", {
    headers: { "X-HB-Client": "console" },
  });
  assert.equal(sessionResponse.status(), 200, "Actual current session read");
  const currentSession = await sessionResponse.json();
  csrf = currentSession.csrf_token;
  redactions.add(csrf);
  owner ||= currentSession.user;
  assert.equal(currentSession.workspace, workspace);
  assert.ok(csrf);
  await page.goto(
    base + "?service=api&lang=en#/service-accounts?lang=en&view=accounts",
  );
  const service = await create("primary");
  const credential = await issue(service);
  const helpers = {
    page,
    context,
    base,
    password,
    service,
    credential,
    navigate,
    read,
    capture,
    scenario,
    command,
    workspace,
  };
  if (group === "actions") await actionCompletion(helpers);
  else {
    for (let i = 1; i < 27; i++) {
      const result = await command("service.issue_credential", service, {
        days: "1",
      });
      assert.equal(result.business_state, "credential_issued");
    }
    const initial = await read("/identity/services/" + service);
    for (let version = 1; version < 11; version++) {
      const result = await command("service.declare", service, {
        purpose: "Synthetic retained declaration " + (version + 1),
        organization: initial.row.declaration.organization,
        catalog_version: initial.catalog_version,
        scopes: JSON.stringify(initial.row.declaration.scopes),
        declaration_version: String(version),
      });
      assert.equal(result.business_state, "service_declared");
      assert.equal(result.fields.declaration_version, String(version + 1));
    }
    const populated = await read(
      "/identity/service-credentials?service=" + service + "&limit=100",
    );
    assert.equal(populated.count.value, "27");
    assert.equal(populated.rows.length, 27);
    if (group !== "declaration") await navigationCompletion(helpers);
    if (group !== "navigation") await declarationCompletion(helpers);
  }
  assert.deepEqual(errors, []);
} catch (error) {
  console.log("FAIL " + safe(error.message));
  process.exitCode = 1;
  if (
    !(await page
      .locator("#svc-secret,#reauth-password,#signin-password,#link-new")
      .count())
  ) {
    await mkdir(output, { recursive: true });
    await page.screenshot({ path: output + "/failure.png", fullPage: true });
  }
} finally {
  await mkdir(output, { recursive: true });
  await writeFile(
    output + "/report.json",
    JSON.stringify({ results, captures, errors }, null, 2),
  );
  await browser.close();
}
