import { defineConfig } from "vite";
import { createHash } from "node:crypto";
import { readdirSync, readFileSync, writeFileSync } from "node:fs";
import { resolve, relative } from "node:path";
import { fileURLToPath } from "node:url";
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
].sort();
const revision = hash(
  inputs
    .map(
      (p) =>
        `${relative(root, p).replaceAll("\\", "/")}\n${hash(readFileSync(p).toString().replaceAll("\r\n", "\n"))}`,
    )
    .join("\n"),
);
const proxy = {
  "/api/focus-overview": {
    target: "http://127.0.0.1:8806",
    changeOrigin: true,
  },
};
const licensePackages = [
  "react",
  "react-dom",
  "scheduler",
  "@fontsource/ibm-plex-sans",
  "@fontsource/ibm-plex-sans-kr",
  "@fontsource/ibm-plex-sans-arabic",
  "@fontsource/ibm-plex-mono",
];
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
    {
      name: "provenance-and-notices",
      generateBundle() {
        for (const name of licensePackages) {
          this.emitFile({
            type: "asset",
            fileName: `licenses/${name.replaceAll("/", "-")}.txt`,
            source: readFileSync(
              resolve(root, "node_modules", name, "LICENSE"),
              "utf8",
            ),
          });
        }
      },
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
