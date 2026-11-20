import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (pairs, value, index, all) =>
        value.startsWith("--")
          ? [...pairs, [value.slice(2), all[index + 1]]]
          : pairs,
      [],
    ),
);
if (!args.activation || !args.url)
  throw new Error(
    "Provide --url and --activation for an owned synthetic sample runtime.",
  );
const origin = new URL(args.url);
if (!["127.0.0.1", "localhost", "[::1]"].includes(origin.hostname))
  throw new Error("This sample fixture requires an owned loopback runtime.");
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const name = "consumer-" + randomBytes(6).toString("hex");
const service = "svc-" + name;
const login = args.login || "admin@example.invalid";
const owner = args.owner || "sample-admin";
const workspace = args.workspace || "sample-workspace";
const output = args.output || "output/validation/m04-consumer";
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
});
const page = await context.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("dialog", (d) => d.accept());
const base = args.url.replace(/\/?$/, "/");
const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
const activation = lines.find((line) => line.startsWith(login + " "));
if (!activation) throw new Error("Synthetic activation entry is unavailable.");
const link = new URL(activation.split(" ")[1]);
const password = randomBytes(18).toString("base64url");
try {
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
  await page.locator("#svc-list-title").waitFor();
  await page
    .getByRole("button", { name: "Create service account", exact: true })
    .click();
  const form = page.locator("form").filter({
    has: page.getByRole("heading", {
      name: "Create service account",
      exact: true,
    }),
  });
  await form
    .getByLabel("Service account name", { exact: false })
    .count()
    .then(async (n) => {
      if (n)
        await form
          .getByLabel("Service account name", { exact: false })
          .fill(name);
      else await form.locator("input").first().fill(name);
    });
  await form
    .getByRole("group", { name: "Responsible owner", exact: true })
    .locator("select")
    .selectOption(owner);
  await form
    .getByLabel("Purpose", { exact: true })
    .fill("Synthetic bounded management verification");
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
    .fill("Synthetic consumer regression verification");
  const request = page.waitForRequest((r) =>
    r.url().endsWith("/commands/service.create"),
  );
  await form
    .getByRole("button", { name: "Create service account", exact: true })
    .click();
  const sent = JSON.parse((await request).postData());
  if (
    Object.keys(sent.body).sort().join(",") !==
    "catalog_version,name,organization,owner,purpose,scopes"
  )
    throw Error("incomplete create payload");
  await page.getByRole("heading", { name: service, exact: true }).waitFor();

  console.log(
    "PASS actual authenticated create full declaration payload and canonical read-back",
  );
  await page
    .getByRole("button", { name: "Edit declaration", exact: true })
    .click();
  const edit = page.locator("form").filter({
    has: page.getByRole("heading", { name: "Edit declaration", exact: true }),
  });
  await edit
    .getByLabel("Purpose", { exact: true })
    .fill("Updated synthetic processing purpose");
  await edit
    .locator("#live-reason")
    .fill("Synthetic declaration regression verification");
  const declarationRequest = page.waitForRequest((r) =>
    r.url().endsWith("/commands/service.declare"),
  );
  await edit
    .getByRole("button", { name: "Edit declaration", exact: true })
    .click();
  const declarationBody = JSON.parse(
    (await declarationRequest).postData(),
  ).body;
  if (
    declarationBody.owner !== undefined ||
    declarationBody.declaration_version !== "1" ||
    declarationBody.purpose !== "Updated synthetic processing purpose"
  )
    throw Error("declaration mutation contract");
  await page.getByRole("heading", { name: service, exact: true }).waitFor();
  console.log(
    "PASS current declaration editing preserves owner and expected declaration version",
  );
  await page
    .getByRole("button", { name: "Manage credentials", exact: true })
    .click();
  const issueForm = page
    .locator("form")
    .filter({ has: page.locator("#svc-issue-title") });
  await page.evaluate(() => {
    window.__qaRealNow = Date.now;
    Date.now = () => window.__qaRealNow() - 86400000;
  });
  await issueForm.locator("#svc-days").fill("1");
  await issueForm
    .locator("#live-reason")
    .fill("Synthetic credential issuance verification");
  await issueForm.locator("button[type=submit]").click();
  await issueForm
    .getByRole("button", { name: "Show secret", exact: true })
    .waitFor();
  const expiryRead = page.waitForResponse(
    (response) =>
      /\/identity\/service-credentials\/cred-[a-f0-9]{16}\?/.test(
        response.url(),
      ) && response.status() === 200,
  );
  await issueForm
    .getByRole("button", { name: "Show secret", exact: true })
    .click();
  const canonicalExpiry = (await (await expiryRead).json()).row.not_after;
  await page.locator("#svc-secret").waitFor();
  if (!(await page.locator("#svc-secret").inputValue()).startsWith("hbsk_"))
    throw Error("secret not actually returned");
  const expectedExpiry = await page.evaluate(
    (iso) =>
      new Intl.DateTimeFormat("en-US", {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(iso)),
    canonicalExpiry,
  );
  await page.waitForFunction(
    (expected) =>
      [...document.querySelectorAll(".live-step dl > div")].some(
        (row) =>
          row.querySelector("dt")?.textContent === "Valid" &&
          row.querySelector("dd")?.textContent?.includes(expected),
      ),
    expectedExpiry,
    { timeout: 5000 },
  );
  console.log(
    "PASS authoritative credential expiry replaces simulated skewed browser estimate",
  );
  await page.locator("#svc-ack").check();
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await page.locator("#svc-secret").waitFor({ state: "detached" });
  await page.evaluate(() => {
    Date.now = window.__qaRealNow;
    delete window.__qaRealNow;
  });
  console.log(
    "PASS explicit account issuance and original-session one-time reveal acknowledgement",
  );
  await page
    .locator("table")
    .first()
    .getByRole("button", { name: /View details/ })
    .first()
    .click();
  await page
    .getByRole("button", { name: "Revoke", exact: true })
    .first()
    .click();
  const revokeForm = page.locator("form").filter({
    has: page.getByRole("heading", {
      name: "Revoke credential",
      exact: true,
    }),
  });
  await revokeForm
    .locator("#live-reason")
    .fill("Synthetic credential revocation verification");
  await revokeForm.locator("button[type=submit]").click();
  await page
    .getByRole("heading", { name: "Revoke credential", exact: true })
    .waitFor({ state: "detached" });
  console.log("PASS canonical credential detail and revocation");
  const variants = [];
  await mkdir(output, { recursive: true });
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 1280, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        base +
          `?service=api&lang=${locale}#/service-accounts?lang=${locale}&view=accounts&svc_a_selected=${service}&svc_a_q=not-matching`,
      );
      await page.locator("#svc-list-title").waitFor();
      await page.locator(".live-panel h2").waitFor();
      await page.waitForFunction(
        ({ locale, width }) => {
          const screen = document.querySelector('[data-security-screen="M04"]');
          const inner = screen?.querySelector(".live-users .live-users");
          return (
            screen?.getAttribute("lang") === locale &&
            inner?.getAttribute("data-wide") ===
              (width >= 1280 ? "true" : "false")
          );
        },
        { locale, width },
      );
      await page.evaluate(async () => {
        await document.fonts.ready;
        await new Promise((done) =>
          requestAnimationFrame(() => requestAnimationFrame(done)),
        );
      });
      const check = await page.evaluate(() => ({
        body: document.documentElement.scrollWidth <= innerWidth,
        dir: document
          .querySelector('[data-security-screen="M04"]')
          .getAttribute("dir"),
        panel: document.querySelector(".live-panel").getBoundingClientRect()
          .width,
      }));
      if (!check.body) throw Error(`body overflow ${locale}/${width}`);
      if (check.dir !== (locale === "ar" ? "rtl" : "ltr"))
        throw Error("locale direction");
      await page.screenshot({
        path: `${output}/${locale}-${width}-known-account.png`,
        fullPage: true,
      });
      variants.push({ locale, width, ...check });
    }
  console.log(JSON.stringify({ variants }));

  await mkdir(output, { recursive: true });

  console.log(JSON.stringify({ errors }));
} catch (error) {
  console.log("FAIL " + error.message.replaceAll(password, "[redacted]"));
  if (!(await page.locator("#svc-secret").count()))
    await page.screenshot({ path: output + "/failure.png", fullPage: true });
  process.exitCode = 1;
} finally {
  await browser.close();
}
