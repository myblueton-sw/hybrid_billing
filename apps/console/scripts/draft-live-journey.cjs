const assert = require("node:assert/strict");
const { createRequire } = require("node:module");
const runnerRequire = process.env.HB_PLAYWRIGHT_PACKAGE ? createRequire(process.env.HB_PLAYWRIGHT_PACKAGE) : require;
const { chromium } = runnerRequire("playwright");
async function input() {
 let raw = "";
 for await (const part of process.stdin) raw += part;
 return JSON.parse(raw);
}
(async () => {
 const seed = await input();
 const browser = await chromium.launch({ headless: true });
 const errors = [];
 try {
  async function actor(cookie) {
   const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
   await context.addCookies([{ name: cookie.name, value: cookie.value, url: seed.origin, httpOnly: true, sameSite: "Strict" }]);
   const page = await context.newPage();
   page.setDefaultTimeout(15000);
   page.on("pageerror", error => errors.push(error.message));
   page.on("response", response => { if(response.status()>=400)process.stderr.write(`HTTP ${response.status()} ${new URL(response.url()).pathname}\n`); });
   return { context, page };
  }
  const submitter = await actor(seed.submitter);
  await submitter.page.goto(`${seed.origin}/?service=api#/approvals?view=history&lang=en&approval-q=${encodeURIComponent(seed.request)}`);
  try { await submitter.page.locator(`[data-b09-record="${seed.request}"]`).filter({ visible: true }).click(); } catch(error) { process.stderr.write((await submitter.page.locator("[role=alert]").allTextContents()).join("\n")+"\n");throw error; }
  const create = submitter.page.getByRole("button", { name: "Create draft", exact: true });
  await create.waitFor();
  assert.equal(await create.isDisabled(), true);
  await submitter.page.getByLabel("Document language", { exact: true }).selectOption("en");
  await submitter.page.locator(".b09-draft-create").getByLabel("Reason for this action", { exact: true }).fill("Verified approved source and reusable configuration");
  await create.click();
  await submitter.page.locator(".b09-draft-outcomes").getByText("Draft submitted", { exact: true }).waitFor();
  const reviewer = await actor(seed.reviewer);
  await reviewer.page.goto(`${seed.origin}/?service=api#/approvals?view=drafts&lang=en`);
  await reviewer.page.locator(".b09-draft-list button").click();
  const save = reviewer.page.getByRole("button", { name: "Save review", exact: true });
  await save.waitFor();
  assert.equal(await save.isDisabled(), true);
  await reviewer.page.getByRole("radio").check();
  await reviewer.page.getByLabel("Reason for this action", { exact: true }).fill("Verified retained source and applicable payment terms");
  await save.click();
  await reviewer.page.getByRole("heading", { name: "Saved review evidence", exact: true }).waitFor();
  const approve = reviewer.page.getByRole("button", { name: "Approve draft", exact: true });
  await approve.waitFor();
  await reviewer.page.getByLabel("Reason for this action", { exact: true }).fill("Independent approval of the saved review");
  await approve.click();
  await reviewer.page.locator(".b09-draft-detail").getByText("Draft approved", { exact: true }).waitFor();
  assert.deepEqual(errors, []);
  await submitter.context.close();
  await reviewer.context.close();
  process.stdout.write("PASS: actual browser/API creation, independent review, approval and retained inspection\n");
 } finally { await browser.close(); }
})().catch(error => { process.stderr.write(String(error.message) + "\n"); process.exitCode = 1; });
