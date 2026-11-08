import { readdirSync, readFileSync } from "node:fs";
import { resolve, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";
import assert from "node:assert/strict";
const root = fileURLToPath(new URL("../dist/", import.meta.url));
function files(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
    e.isDirectory() ? files(resolve(dir, e.name)) : [resolve(dir, e.name)],
  );
}
const manifest = JSON.parse(
  readFileSync(resolve(root, "build-manifest.json"), "utf8"),
);
const actual = files(root)
  .map((p) => relative(root, p).replaceAll("\\", "/"))
  .filter((p) => p !== "build-manifest.json")
  .sort();
assert.deepEqual(manifest.files.map((f) => f.path).sort(), actual);
assert(actual.includes("index.html"));
assert.equal(actual.filter((p) => p.startsWith("licenses/")).length, 7);
for (const entry of manifest.files)
  assert.equal(
    createHash("sha256")
      .update(readFileSync(resolve(root, entry.path)))
      .digest("hex"),
    entry.sha256,
    entry.path,
  );
console.log(
  `Verified ${actual.length} final artifact hashes, including HTML and licenses.`,
);
