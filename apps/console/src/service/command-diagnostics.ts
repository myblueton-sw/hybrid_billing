import { refusalDetail } from "./refusal";
import { AdminError, type AdminClient } from "./admin";

export type RowDiagnostic = { command: string; detail: CommandDiagnostic };

export type CommandDiagnostic = { code: string; request: string; client: string; target: string; cancelled: boolean };

function token(v: string): string {
  return /^[A-Za-z0-9._:-]{1,128}$/.test(v) ? v : "";
}

export function diagnosticOf(error: unknown, target = "", client = ""): CommandDiagnostic {
  const response = refusalDetail(error);
  return {
    code: response?.code ?? "",
    request: response?.request ?? "",
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
