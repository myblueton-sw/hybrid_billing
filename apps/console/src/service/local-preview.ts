export type LocalPreviewLogin = { workspace: string; email: string; password: string };
export const loopbackHost = (host: unknown): boolean => host === "127.0.0.1" || host === "localhost" || host === "::1" || host === "[::1]";

export function localPreviewLogin(host: string, value: LocalPreviewLogin | null): LocalPreviewLogin | null {
  return loopbackHost(host) && value?.workspace.trim() && value.email.trim() && value.password ? value : null;
}
