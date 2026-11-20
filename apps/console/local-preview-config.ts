import { loopbackHost, type LocalPreviewLogin } from "./src/service/local-preview.ts";

export function previewLogin(command: string, isPreview: boolean | undefined, env: Record<string, string | undefined>): LocalPreviewLogin | null {
  if (command !== "serve" || isPreview) return null;
  const workspace = env.HB_LOCAL_PREVIEW_WORKSPACE?.trim();
  const email = env.HB_LOCAL_PREVIEW_EMAIL?.trim();
  const password = env.HB_LOCAL_PREVIEW_PASSWORD;
  return workspace && email && password ? { workspace, email, password } : null;
}

export function requireLocalPreviewHost(host: unknown, login: LocalPreviewLogin | null): void {
  if (login && !loopbackHost(host)) throw new Error("Local preview sign-in requires an explicit loopback listening host.");
}
