import { AdminError } from "./admin";
import type { RowDiagnostic } from "./command-diagnostics";

export type RefusalDetail = { code: string; request: string };

export function refusalDetail(error: unknown): RefusalDetail | undefined {
  if (!(error instanceof AdminError)) return undefined;
  const code = /^[a-z][a-z0-9_]{0,127}$/.test(error.code) ? error.code : "";
  const request = /^[A-Za-z0-9._:-]{1,128}$/.test(error.request) ? error.request : "";
  return code || request ? { code, request } : undefined;
}

function copy(detail: RowDiagnostic): RowDiagnostic {
  return { command: detail.command, detail: { ...detail.detail } };
}

export class Refusals<T> {
  private readonly entries = new Map<string, { value: T; detail?: RowDiagnostic }>();

  set(key: string, value: T, detail?: RowDiagnostic): void {
    this.entries.set(key, { value, detail: detail ? copy(detail) : undefined });
  }
  get(key: string): T | undefined { return this.entries.get(key)?.value; }
  detail(key: string): RowDiagnostic | undefined {
    const detail = this.entries.get(key)?.detail;
    return detail ? copy(detail) : undefined;
  }
  has(key: string): boolean { return this.entries.has(key); }
  delete(key: string): boolean { return this.entries.delete(key); }
  clear(): void { this.entries.clear(); }
  keys(): MapIterator<string> { return this.entries.keys(); }
  [Symbol.iterator](): MapIterator<string> { return this.keys(); }
}
