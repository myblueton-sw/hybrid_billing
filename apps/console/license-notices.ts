import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { resolve } from "node:path";
import type { Plugin } from "vite";

export const licensePackages = [
  "react",
  "react-dom",
  "scheduler",
  "@fontsource/ibm-plex-sans",
  "@fontsource/ibm-plex-sans-kr",
  "@fontsource/ibm-plex-sans-arabic",
  "@fontsource/ibm-plex-mono",
] as const;

export function noticeAssets(root: string) {
  const lock = JSON.parse(readFileSync(resolve(root, "package-lock.json"), "utf8"));
  const assets = new Map<string, Buffer>();
  const packages = licensePackages.map((name) => {
    const installed = JSON.parse(readFileSync(resolve(root, "node_modules", name, "package.json"), "utf8"));
    const locked = lock.packages?.[`node_modules/${name}`];
    if (installed.name !== name || typeof installed.version !== "string" || !installed.version || installed.version !== locked?.version) {
      throw new Error(`Notice package identity/version mismatch: ${name}`);
    }
    const bytes = readFileSync(resolve(root, "node_modules", name, "LICENSE"));
    if (!bytes.length) throw new Error(`Empty original notice: ${name}`);
    const path = `licenses/${name.replaceAll("/", "-")}.txt`;
    assets.set(path, bytes);
    return { name, version: installed.version, source_file: "LICENSE", path, bytes: bytes.length, sha256: createHash("sha256").update(bytes).digest("hex") };
  });
  assets.set("licenses/manifest.json", Buffer.from(JSON.stringify({ schema: 1, scope: "frontend-notice-set", packages }, null, 2) + "\n"));
  return assets;
}

export function licenseNotices(root: string): Plugin {
  const assets = noticeAssets(root);
  return {
    name: "original-license-notices",
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        const path = (request.url ?? "").split("?", 1)[0];
        if (!path.startsWith("/licenses/")) return next();
        response.setHeader("X-Content-Type-Options", "nosniff");
        const bytes = assets.get(path.slice(1));
        if (!bytes) {
          response.statusCode = 404;
          response.end();
          return;
        }
        if (request.method !== "GET" && request.method !== "HEAD") {
          response.statusCode = 405;
          response.setHeader("Allow", "GET, HEAD");
          response.end();
          return;
        }
        response.setHeader("Content-Type", path === "/licenses/manifest.json" ? "application/json; charset=utf-8" : "text/plain; charset=utf-8");
        response.setHeader("Content-Length", bytes.length);
        response.end(request.method === "HEAD" ? undefined : bytes);
      });
    },
    generateBundle() {
      for (const [fileName, source] of assets) this.emitFile({ type: "asset", fileName, source });
    },
  };
}
