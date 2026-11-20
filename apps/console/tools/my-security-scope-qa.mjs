import { createRequire } from "node:module";
import { createServer } from "node:http";
import { createHash } from "node:crypto";
import { mkdtemp, mkdir, readFile, writeFile, rm } from "node:fs/promises";
import { dirname, resolve, join, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import { build } from "vite";

const options = new Map();
for (let i = 2; i < process.argv.length; i += 2) {
  const key = process.argv[i], value = process.argv[i + 1];
  if (!["--playwright-module", "--chromium", "--output"].includes(key) || !value) throw Error("Expected --playwright-module, --chromium and optional --output paths");
  options.set(key, value);
}
if (!options.has("--playwright-module") || !options.has("--chromium")) throw Error("Playwright module and Chromium executable paths are required");
const { chromium } = createRequire(import.meta.url)(resolve(options.get("--playwright-module")));
const consoleRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const output = resolve(options.get("--output") ?? join(tmpdir(), "hb-m05-scope-qa"));
await mkdir(output, { recursive: true });
const run = await mkdtemp(join(output, "run-"));
const work = await mkdtemp(join(tmpdir(), "hb-m05-scope-build-"));
const sources = ["src/modules/identity/pages/my-security/Live.tsx", "src/modules/identity/pages/my-security/scoped-read.ts"];
const sourceHashes = await Promise.all(sources.map(async path => ({ path, sha256: createHash("sha256").update(await readFile(join(consoleRoot, path))).digest("hex") })));
const pathImport = path => JSON.stringify(join(consoleRoot, path).replaceAll("\\", "/"));
const entry = `import React,{useState,useLayoutEffect} from "react";
import {createRoot} from "react-dom/client";import {flushSync} from "react-dom";
import Live from ${pathImport(sources[0])};import {ServiceContext} from ${pathImport("src/service/context.ts")};
window.observations=[];
function Fixture(){const[scope,setScope]=useState('a');window.switchScope=()=>flushSync(()=>setScope('b'));useLayoutEffect(()=>{window.observations.push({scope,text:document.body.textContent});},[scope]);const session={user:scope,workspace:'w',role:'member',login:scope+'@example.invalid',csrf:'synthetic-'+scope};return <ServiceContext.Provider value={{session,signOut:()=>{}}}><Live locale="en" view="account" session={session} onSignOut={()=>{}} onView={()=>{}}/></ServiceContext.Provider>}
createRoot(document.getElementById('root')).render(<Fixture/>);`;
await writeFile(join(work, "entry.tsx"), entry);
await writeFile(join(work, "index.html"), '<!doctype html><html><head><meta charset="UTF-8"></head><body><div id="root"></div><script type="module" src="/entry.tsx"></script></body></html>');
let browser, server;
let report = { sourceHashes, pass: false };
try {
  await build({ configFile: false, root: work, resolve: { alias: { react: join(consoleRoot, "node_modules/react"), "react-dom": join(consoleRoot, "node_modules/react-dom") } }, build: { outDir: join(work, "dist") } });
  const dist = join(work, "dist");
  server = createServer(async (request, response) => {
    try {
      const pathname = decodeURIComponent(new URL(request.url, "http://localhost").pathname);
      const path = resolve(dist, "." + pathname.replace(/\/$/, "/index.html"));
      if (!path.startsWith(dist + sep)) { response.statusCode = 403; response.end(); return; }
      response.setHeader("content-type", ({ ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" })[extname(path)] ?? "application/octet-stream");
      response.end(await readFile(path));
    } catch { response.statusCode = 404; response.end(); }
  });
  await new Promise(done => server.listen(0, "127.0.0.1", done));
  const url = "http://127.0.0.1:" + server.address().port;
  browser = await chromium.launch({ headless: true, executablePath: resolve(options.get("--chromium")) });
  const page = await browser.newPage();
  const errors = [];
  page.on("pageerror", error => errors.push(error.message));
  await page.route("**/api/v1/me", async route => {
    const user = route.request().headers()["x-csrf-token"] === "synthetic-a" ? "a" : "b";
    await route.fulfill({ json: { user_id: user, login: user + "@example.invalid", email: user + "@example.invalid", workspace: "w", role: "member", identities: [], password_set: false, installation_administrator: false,
      current_session: { handle: user.repeat(16), current: true, signed_in: "2026-11-20T00:00:00Z", ends: "2026-11-20T12:00:00Z", state: "active", can_end: false }, recorded_session_count: 1, accepted_operation_count: 0,
      sessions_generation: "a".repeat(32), idle_minutes: 30, max_age_seconds: 0, signed_in_at: "2026-11-20T00:00:00Z", authenticated_at: "2026-11-20T00:00:00Z", now: "2026-11-20T01:00:00Z" } });
  });
  await page.goto(url);
  await page.getByText("a@example.invalid", { exact: true }).first().waitFor();
  await page.evaluate(() => window.switchScope());
  await page.getByText("b@example.invalid", { exact: true }).first().waitFor();
  const observations = await page.evaluate(() => window.observations.map(item => ({ scope: item.scope, containedPriorIdentity: item.scope === "b" && item.text.includes("a@example.invalid") })));
  report = { sourceHashes, pass: !observations.some(item => item.containedPriorIdentity) && errors.length === 0, observations, errors };
} catch (error) { report.error = error.message; }
finally {
  await browser?.close();
  if (server) await new Promise(done => server.close(done));
  await rm(work, { recursive: true, force: true });
  await writeFile(join(run, "report.json"), JSON.stringify(report, null, 2) + "\n");
}
console.log(JSON.stringify({ pass: report.pass, report: join(run, "report.json") }));
if (!report.pass) process.exitCode = 1;
