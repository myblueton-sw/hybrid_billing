import assert from "node:assert/strict";
import { request } from "node:http";
import { readFileSync, mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import test from "node:test";
import { build, createServer } from "vite";
import { licenseNotices, licensePackages, noticeAssets } from "./license-notices";

const root = fileURLToPath(new URL(".", import.meta.url));

test("development serves only exact original license bytes and rejects unsupported requests", async () => {
  const server = await createServer({ configFile: false, root, plugins: [licenseNotices(root)], server: { host: "127.0.0.1", port: 0 } });
  await server.listen();
  try {
    const address = server.httpServer!.address();
    assert.ok(address && typeof address === "object");
    const get = (path: string, method = "GET") => new Promise<{ status: number; headers: Record<string, unknown>; body: Buffer }>((done, reject) => {
      request({ host: "127.0.0.1", port: address.port, path, method }, (response) => {
        const parts: Buffer[] = [];
        response.on("data", (part) => parts.push(part));
        response.on("end", () => done({ status: response.statusCode!, headers: response.headers, body: Buffer.concat(parts) }));
      }).on("error", reject).end();
    });
    const manifest = await get("/licenses/manifest.json");
    assert.equal(manifest.status, 200);
    assert.equal(manifest.headers["content-type"], "application/json; charset=utf-8");
    assert.deepEqual(manifest.body, noticeAssets(root).get("licenses/manifest.json"));
    const head = await get("/licenses/manifest.json", "HEAD");
    assert.equal(head.body.length, 0);
    assert.equal(head.headers["content-length"], String(manifest.body.length));
    assert.equal((await get("/licenses/manifest.json", "POST")).status, 405);
    for (const name of licensePackages) {
      const path = `/licenses/${name.replaceAll("/", "-")}.txt`;
      const original = readFileSync(resolve(root, "node_modules", name, "LICENSE"));
      const result = await get(path + "?download=1");
      assert.equal(result.status, 200);
      assert.equal(result.headers["content-type"], "text/plain; charset=utf-8");
      assert.equal(result.headers["x-content-type-options"], "nosniff");
      assert.deepEqual(result.body, original);
      const head = await get(path, "HEAD");
      assert.equal(head.status, 200);
      assert.equal(head.headers["content-length"], String(original.length));
      assert.equal(head.body.length, 0);
      const post = await get(path, "POST");
      assert.equal(post.status, 405);
      assert.equal(post.headers.allow, "GET, HEAD");
    }
    for (const path of ["/licenses/missing.txt", "/licenses/../package.json", "/licenses/%2e%2e/package.json", "/licenses/react.txt/extra"]) {
      const result = await get(path);
      assert.equal(result.status, 404, path);
      assert.equal(result.body.length, 0);
    }
  } finally {
    await server.close();
  }
});

test("missing original license fails instead of emitting fabricated notice", () => {
  assert.throws(() => licenseNotices(resolve(root, "nonexistent-notice-root")), /ENOENT/);
});


test("manifest binds each original to its locked version and actual bytes", () => {
  const assets = noticeAssets(root);
  const manifest = JSON.parse(assets.get("licenses/manifest.json")!.toString());
  const lock = JSON.parse(readFileSync(resolve(root, "package-lock.json"), "utf8"));
  assert.equal(manifest.schema, 1);
  assert.equal(manifest.scope, "frontend-notice-set");
  assert.deepEqual(manifest.packages.map((p: { name: string }) => p.name), [...licensePackages]);
  for (const p of manifest.packages) {
    const bytes = assets.get(p.path)!;
    assert.equal(p.version, lock.packages[`node_modules/${p.name}`].version);
    assert.equal(p.source_file, "LICENSE");
    assert.equal(p.bytes, bytes.length);
    assert.equal(p.sha256, createHash("sha256").update(bytes).digest("hex"));
    assert.deepEqual(bytes, readFileSync(resolve(root, "node_modules", p.name, "LICENSE")));
  }
  assert.deepEqual(noticeAssets(root), assets);
});

test("absent or mismatched evidence and empty originals fail closed", () => {
  const fixture = mkdtempSync(resolve(tmpdir(), "hb-notice-"));
  const directory = resolve(fixture, "node_modules/react");
  mkdirSync(directory, { recursive: true });
  const metadata = (name: string, version: string) => writeFileSync(resolve(directory, "package.json"), JSON.stringify({ name, version }));
  const lock = (packages: object) => writeFileSync(resolve(fixture, "package-lock.json"), JSON.stringify({ packages }));
  try {
    metadata("react", "1.0.0");
    lock({});
    assert.throws(() => noticeAssets(fixture), /identity.version mismatch/);
    lock({ "node_modules/react": { version: "2.0.0" } });
    assert.throws(() => noticeAssets(fixture), /identity.version mismatch/);
    metadata("wrong-package", "2.0.0");
    assert.throws(() => noticeAssets(fixture), /identity.version mismatch/);
    metadata("react", "2.0.0");
    assert.throws(() => noticeAssets(fixture), /ENOENT/);
    writeFileSync(resolve(directory, "LICENSE"), "");
    assert.throws(() => noticeAssets(fixture), /Empty original notice/);
  } finally { rmSync(fixture, { recursive: true, force: true }); }
});

test("production emits the same manifest and original bytes", async () => {
  const expected = noticeAssets(root);
  const result = await build({ configFile: false, root, logLevel: "silent", plugins: [licenseNotices(root)], build: { write: false, rollupOptions: { input: resolve(root, "license-notices.ts"), external: [/^node:/, "vite"] } } });
  assert.ok(!Array.isArray(result) && "output" in result);
  const emitted = new Map(result.output.filter((entry) => entry.type === "asset").map((entry) => [entry.fileName, Buffer.from(entry.source)]));
  assert.deepEqual(emitted, expected);
});
