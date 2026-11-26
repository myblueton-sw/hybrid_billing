import { createRequire } from "node:module";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import assert from "node:assert/strict";
import { formsMatrix } from "./m04-form-qa.mjs";
import { stateMatrix } from "./m04-state-matrix.mjs";
import { commandMatrix } from "./m04-command-matrix.mjs";
import { authMatrix } from "./m04-auth-matrix.mjs";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (all, key, i, values) =>
        key.startsWith("--") ? [...all, [key.slice(2), values[i + 1]]] : all,
      [],
    ),
);
if (!args.activation || !args.url)
  throw Error("Provide an owned sample --url and --activation.");
const base = args.url.replace(/\/?$/, "/");
if (!["127.0.0.1", "localhost", "[::1]"].includes(new URL(base).hostname))
  throw Error("An owned loopback sample is required.");
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const login = args.login || "admin@example.invalid",
  owner = args.owner || "sample-admin",
  workspace = args.workspace || "sample-workspace";
const output = args.output || "output/validation/m04-errors";
const password = randomBytes(18).toString("base64url");
const prefix = "fault-" + randomBytes(5).toString("hex");
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
});
const page = await context.newPage();
const results = [],
  errors = [];
let variants = [];
let authVariants = [];
const commandVariants = [];
let formVariants = [];
page.on("pageerror", (error) => errors.push(error.message));
page.on("dialog", (dialog) => dialog.accept());
const accountPath = (url) =>
  new URL(url).pathname === "/api/v1/identity/services";
const frame = () =>
  page.evaluate(
    () =>
      new Promise((done) =>
        requestAnimationFrame(() => requestAnimationFrame(done)),
      ),
  );
async function navigate(view, extra = {}, target = page, lang = "en") {
  await target.evaluate(
    ({ view, extra, lang }) => {
      location.hash =
        "/service-accounts?" + new URLSearchParams({ lang, view, ...extra });
    },
    { view, extra, lang },
  );
  await target.waitForSelector(`[data-security-screen="M04"][lang="${lang}"]`);
}
async function create(name, target = page, unknown = false) {
  await navigate("accounts", {}, target);
  await target
    .getByRole("button", { name: "Create service account", exact: true })
    .click();
  const form = target.locator("form").filter({
    has: target.getByRole("heading", {
      name: "Create service account",
      exact: true,
    }),
  });
  await form.locator("input").first().fill(name);
  await form
    .getByRole("group", { name: "Responsible owner", exact: true })
    .locator("select")
    .selectOption(owner);
  await form
    .getByLabel("Purpose", { exact: true })
    .fill("Synthetic failure recovery verification");
  await form
    .getByRole("group", { name: "Owning organization", exact: true })
    .locator("select")
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
    .fill("Synthetic failure recovery verification");
  await form.locator("button[type=submit]").click();
  if (unknown)
    await target
      .getByRole("heading", { name: "Command outcome unknown", exact: true })
      .first()
      .waitFor();
  else
    await target
      .getByRole("heading", { name: "svc-" + name, exact: true })
      .waitFor();
  return "svc-" + name;
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
      error: String(error.message).replaceAll(password, "[redacted]"),
    });
    throw error;
  }
}
async function row(id) {
  await page
    .locator("table tbody tr")
    .filter({ has: page.getByText(id, { exact: true }) })
    .waitFor();
}
async function nextPage() {
  const pending = page.waitForResponse(
    (response) =>
      accountPath(response.url()) &&
      !!new URL(response.url()).searchParams.get("after") &&
      response.status() === 200,
  );
  await page.getByRole("button", { name: "Next", exact: true }).click();
  const result = await (await pending).json();
  assert.equal(result.rows.length, 1);
  await row(result.rows[0].id);
  assert.equal(await page.locator("table tbody tr").count(), 1);
}

try {
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const line = lines.find((value) => value.startsWith(login + " "));
  if (!line) throw Error("Synthetic activation is unavailable.");
  const link = new URL(line.split(" ")[1]);
  await page.goto(
    base + link.pathname.replace(/^\//, "") + "?lang=en" + link.hash,
  );
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
  await page.goto(
    base + "?service=api&lang=en#/service-accounts?lang=en&view=accounts",
  );
  const a = await create(prefix + "-a"),
    b = await create(prefix + "-b");
  let retainedCredentialB;
  await scenario(
    "M04 invalid form fields link errors and keep command submission blocked",
    async () => {
      formVariants = await formsMatrix({
        page,
        navigate,
        output: output + "/forms",
        service: a,
        otherService: b,
      });
    },
  );
  await scenario(
    "unsent state rationale survives canonical refresh and view remount",
    async () => {
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      const reason =
        "Synthetic unsent state rationale retained across canonical refresh";
      await form.locator("#live-reason").fill(reason);
      const refreshed = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/identity/services/" + b && response.status() === 200,
      );
      await page
        .locator('section[aria-labelledby="svc-list-title"] > .live-toolbar')
        .getByRole("button", { name: "Reload", exact: true })
        .click();
      await refreshed;
      await form.locator("#live-reason").waitFor();
      assert.equal(
        await form.locator("#live-reason").inputValue(),
        reason,
        "canonical refresh discarded the unsent state rationale",
      );
      await navigate("credentials");
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      await form.locator("#live-reason").waitFor();
      assert.equal(
        await form.locator("#live-reason").inputValue(),
        reason,
        "view remount discarded the unsent state rationale",
      );
      const writer = await context.newPage();
      try {
        await writer.goto(base + "?service=api&lang=en");
        await writer.locator(".service-session").waitFor();
        await create(prefix + "-revision-conflict", writer);
      } finally {
        await writer.close();
      }
      const refused = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/commands/service.suspend" && response.status() === 409,
      );
      await form.locator("button[type=submit]").click();
      await refused;
      await form
        .getByRole("heading", { name: "Command refused", exact: true })
        .first()
        .waitFor();
      assert.equal(
        await form
          .getByRole("heading", { name: "Command refused", exact: true })
          .count(),
        1,
        "M04 refusal heading must appear once",
      );
      assert.equal(
        await form
          .getByRole("heading", {
            name: "Could not save this change",
            exact: true,
          })
          .count(),
        0,
      );
      assert.equal(
        await form.getByText("revision_conflict", { exact: true }).count(),
        1,
        "safe canonical refusal code is missing",
      );
      commandVariants.push(
        ...(await commandMatrix({
          page,
          navigate,
          output: output + "/command-errors",
          outcome: "refused",
        })),
      );
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      await form.locator("#live-reason").waitFor();
      assert.equal(await form.locator("#live-reason").inputValue(), reason);
      await form
        .getByRole("button", { name: "Discard retained intent", exact: true })
        .click();
      assert.equal(
        await form.locator("#live-reason").inputValue(),
        reason,
        "definitive refusal erased the unsent rationale",
      );
      assert.equal(await form.locator("#live-reason").isEditable(), true);
      await form.getByRole("button", { name: "Cancel", exact: true }).click();
      await page
        .getByRole("button", { name: "Retire service account", exact: true })
        .click();
      await page
        .getByText(
          "An unsent change is retained. Reopen its original target and action to continue, or discard the draft. Cancel closes this form and keeps the draft.",
          { exact: true },
        )
        .waitFor();
      assert.equal(await page.locator("#live-reason").count(), 0);
      await mkdir(output + "/action-drafts", { recursive: true });
      for (const [locale, title, notice] of [
        [
          "en",
          "Retire service account",
          "An unsent change is retained. Reopen its original target and action to continue, or discard the draft. Cancel closes this form and keeps the draft.",
        ],
        [
          "ko",
          "서비스 계정 사용 종료",
          "아직 제출하지 않은 변경 초안이 보존되어 있습니다. 원래 대상과 작업을 다시 열어 계속하거나 초안을 버리세요. 취소하면 이 양식만 닫히고 초안은 유지됩니다.",
        ],
        [
          "ar",
          "إيقاف حساب الخدمة نهائيًا",
          "توجد مسودة محفوظة لتغيير لم يُرسل بعد. أعد فتح الهدف والإجراء الأصليين للمتابعة، أو تجاهل المسودة. يؤدي الإلغاء إلى إغلاق هذا النموذج مع الاحتفاظ بالمسودة.",
        ],
      ]) {
        await page.setViewportSize({ width: 390, height: 1000 });
        if (locale !== "en") {
          await navigate("accounts", { svc_a_selected: b }, page, locale);
          await page.getByRole("button", { name: title, exact: true }).click();
        }
        await page.getByText(notice, { exact: true }).waitFor();
        await page.evaluate(() => document.fonts.ready);
        assert.equal(
          await page
            .locator('[data-security-screen="M04"]')
            .getAttribute("dir"),
          locale === "ar" ? "rtl" : "ltr",
        );
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
          true,
        );
        assert.equal(
          await page
            .locator("#svc-secret,#reauth-password,#signin-password")
            .count(),
          0,
        );
        await page.screenshot({
          path: output + "/action-drafts/" + locale + "-390-incompatible.png",
          fullPage: true,
        });
      }
      await page.setViewportSize({ width: 1440, height: 1000 });
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Retire service account", exact: true })
        .click();
      await page
        .getByRole("button", { name: "Discard previous draft", exact: true })
        .click();
      const retired = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Retire service account",
          exact: true,
        }),
      });
      await retired.locator("#live-reason").waitFor();
      assert.equal(await retired.locator("#live-reason").inputValue(), "");
      await retired
        .getByRole("button", { name: "Cancel", exact: true })
        .click();
    },
  );
  await scenario(
    "a refused replay cannot erase the uncertainty of an accepted original command",
    async () => {
      const service = await create(prefix + "-uncertain");
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      await form
        .locator("#live-reason")
        .fill("Synthetic uncertain original command");
      const bodies = [];
      const handler = async (route) => {
        bodies.push(route.request().postData());
        if (bodies.length === 1) {
          const response = await route.fetch();
          assert.equal(response.status(), 200);
          return route.abort("failed");
        }
        return route.fulfill({
          status: 409,
          contentType: "application/json",
          body: JSON.stringify({
            code: "revision_conflict",
            request_id: "qa-uncertain-replay",
            message: "PRIVATE_UNRENDERED_MESSAGE",
          }),
        });
      };
      await page.route("**/api/v1/commands/service.suspend", handler);
      try {
        await form.locator("button[type=submit]").click();
        await form
          .getByRole("heading", {
            name: "Command outcome unknown",
            exact: true,
          })
          .first()
          .waitFor();
        const refused = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname ===
              "/api/v1/commands/service.suspend" && response.status() === 409,
        );
        await Promise.all([
          refused,
          form.locator("button[type=submit]").click(),
        ]);
        await form.getByText("qa-uncertain-replay", { exact: true }).waitFor();
        assert.equal(
          await form
            .getByRole("heading", {
              name: "Command outcome unknown",
              exact: true,
            })
            .count(),
          1,
        );
        assert.equal(
          await form
            .getByRole("heading", { name: "Command refused", exact: true })
            .count(),
          0,
        );
        assert.equal(
          await form.locator("button[type=submit]").isDisabled(),
          true,
        );
        assert.equal(bodies.length, 2);
        assert.ok(Buffer.from(bodies[0]).equals(Buffer.from(bodies[1])));
        commandVariants.push(
          ...(await commandMatrix({
            page,
            navigate,
            output: output + "/command-errors",
            outcome: "unknown",
          })),
        );
        await navigate("credentials");
        await page
          .getByRole("button", { name: "Discard retained intent", exact: true })
          .click();
        const current = page.waitForResponse(
          (response) =>
            new URL(response.url()).pathname ===
              "/api/v1/identity/services/" + service &&
            response.status() === 200,
        );
        await navigate("accounts", { svc_a_selected: service });
        assert.equal((await (await current).json()).row.state, "suspended");
      } finally {
        await page.unroute("**/api/v1/commands/service.suspend", handler);
      }
    },
  );
  await scenario(
    "late old query cannot replace a newer bounded page",
    async () => {
      let release, arrived, delivered;
      const completed = new Promise((done) => {
        delivered = done;
      });
      const gate = new Promise((done) => {
          release = done;
        }),
        seen = new Promise((done) => {
          arrived = done;
        });
      const handler = async (route) => {
        const url = new URL(route.request().url());
        if (!accountPath(url) || url.searchParams.get("q") !== a)
          return route.continue();
        const response = await route.fetch();
        arrived();
        await gate;
        await route.fulfill({ response });
        delivered();
      };
      await page.route("**/api/v1/identity/services?*", handler);
      await navigate("accounts", { svc_a_q: a });
      await seen;
      const newer = page.waitForResponse(
        (response) =>
          accountPath(response.url()) &&
          new URL(response.url()).searchParams.get("q") === b &&
          response.status() === 200,
      );
      await navigate("accounts", { svc_a_q: b });
      await newer;
      await row(b);
      release();
      await completed;
      await frame();
      assert.equal(
        await page.locator("table tbody tr").filter({ hasText: a }).count(),
        0,
      );
      assert.equal(await page.locator("table tbody tr").count(), 1);
      await page.unroute("**/api/v1/identity/services?*", handler);
    },
  );
  await scenario(
    "a pending new query hides prior rows until its own response",
    async () => {
      let release, arrived;
      const gate = new Promise((done) => {
          release = done;
        }),
        seen = new Promise((done) => {
          arrived = done;
        });
      const handler = async (route) => {
        if (new URL(route.request().url()).searchParams.get("q") !== a)
          return route.continue();
        const response = await route.fetch();
        arrived();
        await gate;
        await route.fulfill({ response });
      };
      await page.route("**/api/v1/identity/services?*", handler);
      await navigate("accounts", { svc_a_q: a });
      await seen;
      await frame();
      assert.equal(await page.locator("table tbody tr").count(), 0);
      assert.ok(
        await page
          .getByRole("status")
          .filter({ hasText: /Loading/ })
          .count(),
      );
      release();
      await row(a);
      await page.unroute("**/api/v1/identity/services?*", handler);
    },
  );
  await scenario(
    "actual source conflict restarts first while preserving filters",
    async () => {
      await navigate("accounts", { svc_a_q: prefix, svc_a_limit: "1" });
      await nextPage();
      await page.waitForFunction(() =>
        new URLSearchParams(location.hash.split("?")[1]).has("svc_a_after"),
      );
      await page.locator("table tbody tr").waitFor();
      const other = await context.newPage();
      other.on("dialog", (dialog) => dialog.accept());
      await other.goto(
        base + "?service=api&lang=en#/service-accounts?lang=en&view=accounts",
      );
      await create(prefix + "-c", other);
      await other.close();
      const conflict = page.waitForResponse(
        (response) => accountPath(response.url()) && response.status() === 409,
      );
      await page
        .getByRole("button", { name: "Reload", exact: true })
        .first()
        .click();
      await conflict;
      await page.waitForFunction(
        () => document.querySelectorAll("table tbody tr").length === 0,
      );
      await page.getByRole("button", { name: "First", exact: true }).waitFor();
      assert.equal(await page.locator("table tbody tr").count(), 0);
      await page.getByRole("button", { name: "First", exact: true }).click();
      await page.locator("table tbody tr").waitFor();
      const params = await page.evaluate(() =>
        Object.fromEntries(new URLSearchParams(location.hash.split("?")[1])),
      );
      assert.equal(params.svc_a_after, undefined);
      assert.equal(params.svc_a_q, prefix);
      assert.equal(params.svc_a_limit, "1");
      assert.equal(await page.locator("table tbody tr").count(), 1);
    },
  );
  await scenario(
    "transport retry retains the exact current signed cursor",
    async () => {
      await nextPage();
      await page.locator("table tbody tr").waitFor();
      let failed = false;
      const sent = [];
      const handler = async (route) => {
        sent.push(route.request().url());
        if (!failed) {
          failed = true;
          return route.fulfill({
            status: 503,
            contentType: "application/json",
            body: JSON.stringify({
              code: "dependency_unavailable",
              request_id: "synthetic-failure",
              retry: true,
              next_action: "none",
            }),
          });
        }
        return route.continue();
      };
      await page.route("**/api/v1/identity/services?*", handler);
      const failedResponse = page.waitForResponse(
        (response) => accountPath(response.url()) && response.status() === 503,
      );
      await page
        .getByRole("button", { name: "Reload", exact: true })
        .first()
        .click();
      await failedResponse;
      await page.waitForFunction(
        () => document.querySelectorAll("table tbody tr").length === 0,
      );
      await page.getByRole("button", { name: "First", exact: true }).waitFor();
      const recovered = page.waitForResponse(
        (response) => accountPath(response.url()) && response.status() === 200,
      );
      await page
        .getByRole("button", { name: "Reload", exact: true })
        .last()
        .click();
      const recoveredPage = await (await recovered).json();
      await row(recoveredPage.rows[0].id);
      assert.equal(sent.length, 2);
      assert.equal(sent[0], sent[1]);
      assert.ok(new URL(sent[0]).searchParams.get("after"));
      await page.unroute("**/api/v1/identity/services?*", handler);
    },
  );
  await scenario(
    "replaying revoke A from B preserves B's original pickup",
    async () => {
      let collections = 0;
      const count = (request) => {
        if (
          new URL(request.url()).pathname.startsWith("/api/v1/identity/links/")
        )
          collections++;
      };
      page.on("request", count);
      const issue = async (service) => {
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
          .fill("Synthetic independent pickup retention verification");
        const response = page.waitForResponse(
          (response) =>
            response.url().endsWith("/commands/service.issue_credential") &&
            response.status() === 200,
        );
        await form.locator("button[type=submit]").click();
        const credential = (await (await response).json()).fields.credential;
        await page
          .locator(".live-attempts li")
          .filter({ hasText: credential })
          .waitFor();
        return credential;
      };
      const credentialA = await issue(a),
        credentialB = await issue(b);
      retainedCredentialB = credentialB;
      const retainedB = () =>
        page.locator(".live-attempts li").filter({ hasText: credentialB });
      const bodies = [];
      const handler = async (route) => {
        bodies.push(route.request().postData());
        if (bodies.length === 1) {
          const response = await route.fetch();
          assert.equal(response.status(), 200);
          return route.abort("failed");
        }
        return route.continue();
      };
      await page.route("**/api/v1/commands/service.revoke_credential", handler);
      await page
        .locator(".live-attempts li")
        .filter({ hasText: credentialA })
        .getByRole("button", { name: "Revoke", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Revoke credential",
          exact: true,
        }),
      });
      await form
        .locator("#live-reason")
        .fill("Synthetic original credential revocation verification");
      await form.locator("button[type=submit]").click();
      await page
        .getByRole("heading", { name: "Command outcome unknown", exact: true })
        .first()
        .waitFor();
      const targetRead = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/identity/service-credentials/" + credentialB &&
          response.status() === 200,
      );
      await retainedB()
        .getByRole("button", { name: "Revoke", exact: true })
        .click();
      await targetRead;
      await page.locator(".live-panel .live-held h3").waitFor();
      assert.equal(
        await page
          .locator("form")
          .filter({
            has: page.getByRole("heading", {
              name: "Revoke credential",
              exact: true,
            }),
          })
          .count(),
        0,
      );
      await page
        .getByRole("button", { name: "Try again", exact: true })
        .first()
        .click();
      await page
        .getByRole("heading", { name: "Command completed", exact: true })
        .first()
        .waitFor();
      assert.equal(bodies.length, 2);
      assert.ok(
        Buffer.from(bodies[0]).equals(Buffer.from(bodies[1])),
        "original command envelope changed",
      );
      assert.equal(JSON.parse(bodies[1]).target_id, credentialA);
      await retainedB()
        .getByRole("button", { name: "Show secret", exact: true })
        .waitFor();
      assert.equal(collections, 0);
      await page
        .getByRole("button", { name: "Discard retained intent", exact: true })
        .first()
        .click();
      await page.getByRole("button", { name: "Cancel", exact: true }).click();
      await page.locator("#svc-cred-title").waitFor();
      await page
        .getByRole("button", { name: "Reload", exact: true })
        .first()
        .click();
      await page.waitForFunction(
        (credential) =>
          ![...document.querySelectorAll(".live-attempts li")].some((row) =>
            row.textContent.includes(credential),
          ),
        credentialA,
      );
      await retainedB()
        .getByRole("button", { name: "Show secret", exact: true })
        .click();
      await page.locator("#svc-secret").waitFor();
      assert.ok(
        (await page.locator("#svc-secret").inputValue()).startsWith("hbsk_"),
      );
      assert.equal(collections, 1);
      await page.locator("#svc-ack").check();
      await page.getByRole("button", { name: "Done", exact: true }).click();
      await page.locator("#svc-secret").waitFor({ state: "detached" });
      await page.unroute(
        "**/api/v1/commands/service.revoke_credential",
        handler,
      );
      page.off("request", count);
    },
  );
  await scenario(
    "unsent revocation rationale survives canonical refresh without declaring storage",
    async () => {
      await navigate("credentials", {
        svc_c_service: b,
        svc_c_selected: retainedCredentialB,
      });
      await page
        .getByRole("heading", { name: retainedCredentialB, exact: true })
        .waitFor();
      let unavailable = true;
      const detailPath =
        "/api/v1/identity/service-credentials/" + retainedCredentialB;
      const handler = async (route) => {
        const response = await route.fetch();
        if (response.status() !== 200 || !unavailable)
          return route.fulfill({ response });
        const body = await response.json();
        body.eligibility.step_up_state = "unavailable";
        body.eligibility.can_revoke = false;
        return route.fulfill({ response, json: body });
      };
      await page.route("**" + detailPath + "?*", handler);
      await page
        .locator("section.live-step")
        .filter({
          has: page.getByRole("heading", {
            name: retainedCredentialB,
            exact: true,
          }),
        })
        .getByRole("button", { name: "Revoke", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Revoke credential",
          exact: true,
        }),
      });
      await form.locator("#live-reason").waitFor();
      assert.equal(await form.locator("#live-reason").inputValue(), "");
      const reason = "Synthetic unsubmitted revocation rationale";
      await form.locator("#live-reason").fill(reason);
      const refreshed = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/identity/service-credentials/" + retainedCredentialB &&
          response.status() === 200,
      );
      unavailable = false;
      await Promise.all([
        refreshed,
        form.getByRole("button", { name: "Reload", exact: true }).click(),
      ]);
      await page.unroute("**" + detailPath + "?*", handler);
      await form.locator("#live-reason").waitFor();
      assert.equal(await form.locator("#live-reason").inputValue(), reason);
      await form.getByRole("button", { name: "Cancel", exact: true }).click();
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      await page
        .getByRole("button", { name: "Discard previous draft", exact: true })
        .click();
      const state = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      await state.locator("#live-reason").waitFor();
      assert.equal(await state.locator("#live-reason").inputValue(), "");
      await state.getByRole("button", { name: "Cancel", exact: true }).click();
    },
  );
  await scenario(
    "unknown state retry stays bound to original action and target across locale/view",
    async () => {
      await navigate("accounts", { svc_a_selected: a });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      await form
        .locator("#live-reason")
        .fill("Synthetic original suspend verification");
      const bodies = [];
      const handler = async (route) => {
        bodies.push(route.request().postData());
        if (bodies.length === 1) {
          const response = await route.fetch();
          assert.equal(response.status(), 200);
          return route.abort("failed");
        }
        return route.continue();
      };
      await page.route("**/api/v1/commands/service.suspend", handler);
      await form.locator("button[type=submit]").click();
      await page
        .getByRole("heading", { name: "Command outcome unknown", exact: true })
        .first()
        .waitFor();
      await navigate("credentials", {}, page, "ko");
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Retire service account", exact: true })
        .click();
      await page.locator(".live-panel .live-held h3").waitFor();
      assert.equal(
        await page
          .locator("form")
          .filter({
            has: page.getByRole("heading", {
              name: "Retire service account",
              exact: true,
            }),
          })
          .count(),
        0,
      );
      await page
        .getByRole("button", { name: "Try again", exact: true })
        .first()
        .click();
      await page
        .getByRole("heading", { name: "Command completed", exact: true })
        .first()
        .waitFor();
      assert.equal(bodies.length, 2);
      assert.ok(
        Buffer.from(bodies[0]).equals(Buffer.from(bodies[1])),
        "original command envelope changed",
      );
      assert.equal(JSON.parse(bodies[1]).target_id, a);
      await page
        .getByRole("button", { name: "Discard retained intent", exact: true })
        .first()
        .click();
      await page.waitForFunction(
        () =>
          document.activeElement?.tagName === "H2" &&
          document.activeElement.textContent === "Retire service account",
        undefined,
        { timeout: 2000 },
      );
      await page.unroute("**/api/v1/commands/service.suspend", handler);
      await navigate("accounts", { svc_a_q: b });
      await row(b);
      assert.ok(
        (await page.locator("table tbody tr").textContent()).includes("Active"),
      );
    },
  );
  await scenario(
    "discarding another account draft focuses the newly available editor",
    async () => {
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Edit declaration", exact: true })
        .click();
      await page
        .getByLabel("Purpose", { exact: true })
        .fill("Retained synthetic purpose");
      await page.getByRole("button", { name: "Close", exact: true }).click();
      await navigate("accounts", { svc_a_selected: "svc-" + prefix + "-c" });
      await page
        .getByRole("button", { name: "Edit declaration", exact: true })
        .click();
      await page
        .getByRole("button", { name: "Discard previous draft", exact: true })
        .click();
      await page.waitForFunction(
        () =>
          document.activeElement?.tagName === "H2" &&
          document.activeElement.textContent === "Edit declaration",
        undefined,
        { timeout: 2000 },
      );
      await page.getByRole("button", { name: "Close", exact: true }).click();
    },
  );
  await scenario(
    "keyboard opening moves focus to canonical detail and closing returns it",
    async () => {
      await navigate("accounts", { svc_a_q: b });
      await row(b);
      const button = page
        .locator("table tbody tr")
        .filter({ hasText: b })
        .getByRole("button", { name: /View details/ });
      await button.focus();
      await page.keyboard.press("Enter");
      const heading = page.getByRole("heading", { name: b, exact: true });
      await heading.waitFor();
      await page.waitForFunction(
        (id) =>
          document.activeElement?.tagName === "H2" &&
          document.activeElement.textContent === id,
        b,
        { timeout: 2000 },
      );
      await page.getByRole("button", { name: "Close", exact: true }).click();
      await page.waitForFunction(
        (id) =>
          document.activeElement?.tagName === "BUTTON" &&
          document.activeElement.textContent?.includes("View details") &&
          document.activeElement.closest("tr")?.textContent?.includes(id),
        b,
        { timeout: 2000 },
      );
    },
  );
  await scenario(
    "suspended account credentials retain lifecycle but show their current hold",
    async () => {
      await navigate("accounts", { svc_a_selected: b });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      const form = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      await form
        .locator("#live-reason")
        .fill("Synthetic suspended credential metadata verification");
      const suspended = page.waitForResponse(
        (response) =>
          response.url().endsWith("/commands/service.suspend") &&
          response.status() === 200,
      );
      await form.locator("button[type=submit]").click();
      await suspended;
      await page
        .getByRole("button", { name: "Manage credentials", exact: true })
        .waitFor();
      const read = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/identity/service-credentials" &&
          new URL(response.url()).searchParams.get("service") === b &&
          response.status() === 200,
      );
      await navigate("credentials", {
        svc_c_service: b,
        svc_c_selected: retainedCredentialB,
      });
      const data = await (await read).json();
      const credential = data.rows.find(
        (item) => item.id === retainedCredentialB,
      );
      assert.ok(credential, "canonical issued credential missing");
      assert.equal(credential.current_state, "active");
      assert.equal(credential.usable, false);
      assert.equal(credential.service_state, "suspended");
      await row(retainedCredentialB);
      const loaded = page
        .locator("table tbody tr")
        .filter({ hasText: retainedCredentialB });
      await loaded
        .getByText("On hold", { exact: true })
        .waitFor({ timeout: 2000 });
      await loaded.getByText("Suspended", { exact: true }).waitFor();
      const known = page.locator("section.live-step").filter({
        has: page.getByRole("heading", {
          name: retainedCredentialB,
          exact: true,
        }),
      });
      await known.getByText("On hold", { exact: true }).waitFor();
      await known.getByText("Suspended", { exact: true }).waitFor();
    },
  );
  await scenario(
    "unknown complete creation survives canonical read-back and locale/view changes",
    async () => {
      const bodies = [];
      const handler = async (route) => {
        bodies.push(route.request().postData());
        if (bodies.length === 1) {
          const response = await route.fetch();
          assert.equal(response.status(), 200);
          return route.abort("failed");
        }
        return route.continue();
      };
      await page.route("**/api/v1/commands/service.create", handler);
      const created = await create(prefix + "-unknown", page, true);
      await navigate("credentials", {}, page, "ko");
      await page.locator(".live-held h3").first().waitFor();
      await navigate("accounts", { svc_a_selected: created });
      await page.getByRole("heading", { name: created, exact: true }).waitFor();
      await page
        .getByRole("heading", { name: "Command outcome unknown", exact: true })
        .first()
        .waitFor();
      assert.equal(bodies.length, 1);
      await page
        .getByRole("button", { name: "Try again", exact: true })
        .first()
        .click();
      await page
        .getByRole("heading", { name: "Command completed", exact: true })
        .first()
        .waitFor();
      assert.equal(bodies.length, 2);
      assert.ok(
        Buffer.from(bodies[0]).equals(Buffer.from(bodies[1])),
        "complete creation envelope changed",
      );
      const original = JSON.parse(bodies[0]);
      assert.equal(original.body.owner, owner);
      assert.equal(original.body.organization, "sample-le-seller");
      assert.equal(
        original.body.purpose,
        "Synthetic failure recovery verification",
      );
      assert.equal(
        original.body.scopes,
        '[{"audience":"test-service-api","scope":"read"}]',
      );
      assert.match(original.body.catalog_version, /^[a-f0-9]{64}$/);
      await page
        .getByRole("button", { name: "Discard retained intent", exact: true })
        .first()
        .click();
      await page.unroute("**/api/v1/commands/service.create", handler);
    },
  );
  await scenario(
    "in-flight issuance is localized and unknown replay survives new ineligibility",
    async () => {
      const service = "svc-" + prefix + "-c";
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
        .fill("Synthetic original issuance verification");
      let release, arrived, delivered;
      const gate = new Promise((done) => {
        release = done;
      });
      const seen = new Promise((done) => {
        arrived = done;
      });
      const completed = new Promise((done) => {
        delivered = done;
      });
      const bodies = [];
      const handler = async (route) => {
        bodies.push(route.request().postData());
        if (bodies.length === 1) {
          const response = await route.fetch();
          assert.equal(response.status(), 200);
          arrived();
          await gate;
          await route.abort("failed");
          delivered();
          return;
        }
        return route.continue();
      };
      await page.route("**/api/v1/commands/service.issue_credential", handler);
      await form.locator("button[type=submit]").click();
      await seen;
      try {
        await navigate(
          "credentials",
          { svc_c_service: service, svc_c_issue: service },
          page,
          "ko",
        );
        await page
          .locator(".live-attempts li")
          .filter({ hasText: service })
          .getByText(
            "요청을 보내는 중입니다. 서버가 응답할 때까지 값을 바꿀 수 없습니다.",
            { exact: false },
          )
          .waitFor({ timeout: 2000 });
        assert.equal(await page.locator("#svc-days").isDisabled(), true);
      } finally {
        release();
      }
      await completed;
      await navigate("credentials", { svc_c_service: b, svc_c_issue: b });
      const otherDraft = page
        .locator("form")
        .filter({ has: page.locator("#svc-issue-title") });
      await otherDraft.locator("#svc-days").fill("1");
      await otherDraft
        .locator("#live-reason")
        .fill("Synthetic unrelated unsubmitted issuance draft");
      await navigate("accounts", { svc_a_selected: service });
      await page
        .getByRole("button", { name: "Suspend service account", exact: true })
        .click();
      const suspendForm = page.locator("form").filter({
        has: page.getByRole("heading", {
          name: "Suspend service account",
          exact: true,
        }),
      });
      await suspendForm
        .locator("#live-reason")
        .fill("Synthetic current ineligibility after accepted issue");
      const suspended = page.waitForResponse(
        (response) =>
          response.url().endsWith("/commands/service.suspend") &&
          response.status() === 200,
      );
      await suspendForm.locator("button[type=submit]").click();
      await suspended;
      await page
        .getByRole("button", { name: "Manage credentials", exact: true })
        .waitFor();
      await navigate("credentials", {
        svc_c_service: service,
        svc_c_issue: service,
      });
      const issueForm = page
        .locator("form")
        .filter({ has: page.locator("#svc-issue-title") });
      await issueForm
        .getByRole("button", { name: "Try again", exact: true })
        .waitFor();
      assert.equal(
        await issueForm
          .getByRole("button", { name: "Try again", exact: true })
          .isEnabled(),
        true,
      );
      let collections = 0;
      const count = (request) => {
        if (
          new URL(request.url()).pathname.startsWith("/api/v1/identity/links/")
        )
          collections++;
      };
      page.on("request", count);
      await issueForm
        .getByRole("button", { name: "Try again", exact: true })
        .click();
      await page
        .locator(".live-attempts li")
        .filter({ hasText: service })
        .getByRole("button", { name: "Show secret", exact: true })
        .waitFor();
      assert.equal(bodies.length, 2);
      assert.ok(
        Buffer.from(bodies[0]).equals(Buffer.from(bodies[1])),
        "original issuance envelope changed",
      );
      assert.equal(collections, 0);
      await navigate("credentials", { svc_c_service: b, svc_c_issue: b });
      await otherDraft.locator("#live-reason").waitFor();
      assert.equal(await otherDraft.locator("#svc-days").inputValue(), "1");
      assert.equal(
        await otherDraft.locator("#live-reason").inputValue(),
        "Synthetic unrelated unsubmitted issuance draft",
        "original retry cleared another account's draft",
      );
      await navigate("credentials", {
        svc_c_issue: "svc-" + prefix + "-unknown",
      });
      await page
        .getByRole("button", { name: "Discard previous draft", exact: true })
        .click();
      await page.locator("#svc-days").waitFor();
      await page
        .locator(".live-attempts li")
        .filter({ hasText: service })
        .getByRole("button", { name: "Show secret", exact: true })
        .waitFor();
      assert.equal(collections, 0);
      await page.unroute(
        "**/api/v1/commands/service.issue_credential",
        handler,
      );
      page.off("request", count);
    },
  );
  await scenario(
    "authorized administrator missing-ID refusal does not imply global authority loss",
    async () => {
      const absent = "svc-" + prefix + "-absent";
      const refused = page.waitForResponse(
        (response) =>
          new URL(response.url()).pathname ===
            "/api/v1/identity/services/" + absent && response.status() === 403,
      );
      await navigate("accounts", { svc_a_q: b, svc_a_selected: absent });
      await refused;
      await row(b);
      const failure = page.locator(".live-panel .service-summary[role=alert]");
      await failure
        .getByText(
          "This data is unavailable with the current access and context.",
          { exact: true },
        )
        .waitFor();
      assert.ok(
        !(await failure.textContent()).includes(
          "permission to manage service accounts in this installation",
        ),
      );
      await page
        .locator(".live-panel")
        .getByText("permission_denied", { exact: true })
        .waitFor();
    },
  );
  await scenario(
    "EN KO AR state and computed-border matrix at four viewport widths",
    async () => {
      variants = await stateMatrix({
        page,
        navigate,
        output: output + "/states",
        service: b,
        credential: retainedCredentialB,
        prefix,
      });
      assert.equal(variants.length, 60);
    },
  );
  await scenario(
    "EN KO AR authentication recovery preserves unsubmitted issuance at four widths",
    async () => {
      authVariants = await authMatrix({
        page,
        navigate,
        output: output + "/authentication",
        service: "svc-" + prefix + "-unknown",
        password,
        otherService: b,
        login,
        workspace,
      });
      assert.equal(authVariants.length, 64);
    },
  );
  assert.deepEqual(errors, []);
} catch (error) {
  console.log(
    "FAIL " + String(error.message).replaceAll(password, "[redacted]"),
  );
  process.exitCode = 1;
  if (
    !(await page
      .locator("#svc-secret, #signin-password, #link-new, #reauth-password")
      .count())
  ) {
    await mkdir(output, { recursive: true });
    await page.screenshot({ path: output + "/failure.png", fullPage: true });
  }
} finally {
  await mkdir(output, { recursive: true });
  await writeFile(
    output + "/report.json",
    JSON.stringify(
      {
        results,
        errors,
        variants,
        authVariants,
        commandVariants,
        formVariants,
      },
      null,
      2,
    ),
  );
  await browser.close();
}
