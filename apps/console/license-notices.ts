import { readFileSync } from "node:fs";
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
  return new Map(licensePackages.map((name) => [
    `licenses/${name.replaceAll("/", "-")}.txt`,
    readFileSync(resolve(root, "node_modules", name, "LICENSE")),
  ]));
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
        response.setHeader("Content-Type", "text/plain; charset=utf-8");
        response.setHeader("Content-Length", bytes.length);
        response.end(request.method === "HEAD" ? undefined : bytes);
      });
    },
    generateBundle() {
      for (const [fileName, source] of assets) this.emitFile({ type: "asset", fileName, source });
    },
  };
}
