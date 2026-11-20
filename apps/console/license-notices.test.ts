import assert from "node:assert/strict";
import { request } from "node:http";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { resolve } from "node:path";
import test from "node:test";
import { createServer } from "vite";
import { licenseNotices, licensePackages } from "./license-notices";

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
