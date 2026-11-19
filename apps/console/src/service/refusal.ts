import { AdminError } from "./admin";

export type RefusalDetail = { code: string; request: string };

export function refusalDetail(error: unknown): RefusalDetail | undefined {
  if (!(error instanceof AdminError) || error.failure === "reauth_cancelled") return undefined;
  const code = /^[a-z][a-z0-9_]{0,127}$/.test(error.code) ? error.code : "";
  const request = /^[A-Za-z0-9._:-]{1,128}$/.test(error.request) ? error.request : "";
  return code || request ? { code, request } : undefined;
}

export class Refusals<T> {
  private readonly entries = new Map<string, { value: T; detail?: RefusalDetail }>();

  set(key: string, value: T, detail?: RefusalDetail): void {
    this.entries.set(key, { value, detail: detail ? { ...detail } : undefined });
  }

  get(key: string): T | undefined { return this.entries.get(key)?.value; }
  detail(key: string): RefusalDetail | undefined {
    const detail = this.entries.get(key)?.detail;
    return detail ? { ...detail } : undefined;
  }
  has(key: string): boolean { return this.entries.has(key); }
  delete(key: string): boolean { return this.entries.delete(key); }
  clear(): void { this.entries.clear(); }
  keys(): MapIterator<string> { return this.entries.keys(); }
  [Symbol.iterator](): MapIterator<string> { return this.keys(); }
}
