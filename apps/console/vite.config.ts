import { defineConfig } from "vite";
import { createHash } from "node:crypto";
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { licenseNotices } from "./license-notices";
import { proxyFor } from "./proxy-config";
const root = fileURLToPath(new URL(".", import.meta.url));
const foundation = resolve(root, "../../packages/design-system");
const hash = (data: string | Buffer) =>
  createHash("sha256").update(data).digest("hex");
function files(dir: string): string[] {
  return readdirSync(dir, { withFileTypes: true })
    .sort((a, b) => (a.name < b.name ? -1 : 1))
    .flatMap((e) =>
      e.isDirectory() ? files(resolve(dir, e.name)) : [resolve(dir, e.name)],
    );
}
const provenance = JSON.parse(
  readFileSync(resolve(foundation, "provenance.json"), "utf8"),
);
for (const asset of provenance.assets)
  if (hash(readFileSync(resolve(foundation, asset.path))) !== asset.sha256)
    throw Error(`Foundation hash mismatch: ${asset.path}`);
const inputs = [
  ...files(resolve(root, "src")),
  ...files(foundation),
  resolve(root, "package-lock.json"),
  resolve(root, "package.json"),
  resolve(root, "tsconfig.json"),
  resolve(root, "index.html"),
  resolve(root, "vite.config.ts"),
  resolve(root, "proxy-config.ts"),
  resolve(root, "license-notices.ts"),
].sort();
const revision = hash(
  inputs
    .map(
      (p) =>
        `${relative(root, p).replaceAll("\\", "/")}\n${hash(readFileSync(p).toString().replaceAll("\r\n", "\n"))}`,
    )
    .join("\n"),
);
const proxy = proxyFor(process.env.HB_GATED === "1");
export default defineConfig({
  base: "./",
  define: { __SOURCE_REVISION__: JSON.stringify(revision) },
  preview: { host: "127.0.0.1", proxy },
  server: { proxy, host: "127.0.0.1", fs: { allow: [resolve(root, "../..")] } },
  build: {
    target: "es2022",
    outDir: "dist",
    emptyOutDir: true,
    assetsInlineLimit: 0,
  },
  plugins: [
    licenseNotices(root),
    {
      name: "provenance-and-notices",
      closeBundle() {
        const output = resolve(root, "dist");
        writeFileSync(
          resolve(output, "build-manifest.json"),
          JSON.stringify(
            {
              revision,
              mode: "frontend-design-increment",
              profiles: ["standard", "evidence"],
              files: files(output)
                .filter((p) => relative(output, p) !== "build-manifest.json")
                .map((p) => ({
                  path: relative(output, p).replaceAll("\\", "/"),
                  sha256: hash(readFileSync(p)),
                })),
            },
            null,
            2,
          ),
        );
      },
    },
  ],
});
