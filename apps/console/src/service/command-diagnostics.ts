import { AdminError, type AdminClient } from "./admin";

export type RowDiagnostic = { command: string; detail: CommandDiagnostic };

export type CommandDiagnostic = { code: string; request: string; client: string; target: string; cancelled: boolean };

function token(v: string): string {
  return /^[A-Za-z0-9._:-]{1,128}$/.test(v) ? v : "";
}

export function diagnosticOf(error: unknown, target = "", client = ""): CommandDiagnostic {
  return {
    code: error instanceof AdminError && /^[a-z][a-z0-9_]{0,95}$/.test(error.code) ? error.code : "",
    request: error instanceof AdminError ? token(error.request) : "",
    client: token(client),
    target: token(target),
    cancelled: error instanceof AdminError && error.failure === "reauth_cancelled",
  };
}

export function observeCommands(api: AdminClient, publish: (v: CommandDiagnostic | null) => void, current: () => boolean): AdminClient {
  let sequence = 0;
  return {
    ...api,
    async command(...args: Parameters<AdminClient["command"]>) {
      const own = ++sequence;
      if (current()) publish(null);
      try {
        return await api.command(...args);
      } catch (error) {
        if (current() && own === sequence) publish(diagnosticOf(error, args[1], args[5].request));
        throw error;
      }
    },
  };
}
