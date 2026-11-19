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
const only = new Set((args.only || "flip,narrow,inflight,cancel").split(","));
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const results = [];
const check = (name, ok, detail = "") =>
  results.push(`${ok ? "PASS" : "FAIL"} ${name}${detail ? " " + detail : ""}`);
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
    posts[0].idempotency_key === posts[1].idempotency_key;
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
  check("no page errors", errors.length === 0, JSON.stringify(errors));
} finally {
  await browser.close();
}
console.log(results.join("\n"));
process.exitCode = results.some((r) => r.startsWith("FAIL")) ? 1 : 0;
