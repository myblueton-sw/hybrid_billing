import { freeze } from "../../../../issuance/pages/approval-inbox/results/contract";
import type { Session } from "../../../../../service/api";
import { SetupError } from "./context";
import type { Configuration, ConfigurationPage } from "./history";

export const setupParts = ["settings", "rules", "channels", "changes", "statements"] as const;
export type SetupPart = typeof setupParts[number];
export const partOf = (value: string | null): SetupPart => setupParts.includes(value as SetupPart) ? value as SetupPart : "settings";

export type SetupScope = Readonly<{ configuration: string; digest: string; setupOn: string; contractRevision: string; billingRevision: string }>;
export const scopeEligible = (row: Configuration): boolean => row.state === "confirmed" && row.module_observation === "active" && !!row.setup_on;

export function scopeOf(row: Configuration): SetupScope {
  if (!scopeEligible(row)) throw new SetupError("invalid_setup_request");
  return freeze({ configuration: row.configuration, digest: row.digest, setupOn: row.setup_on!, contractRevision: row.contract_revision, billingRevision: row.billing_revision });
}

export function reconcileScope(current: SetupScope | undefined, page: ConfigurationPage | undefined, after: string): SetupScope | undefined {
  if (!page) return current;
  const whole = !after && !page.next;
  let kept = current;
  if (current) {
    const row = page.rows.find(r => r.configuration === current.configuration);
    if (row ? !scopeEligible(row) || row.digest !== current.digest : whole) kept = undefined;
  }
  if (kept || !whole) return kept;
  const eligible = page.rows.filter(scopeEligible);
  return eligible.length === 1 ? scopeOf(eligible[0]) : undefined;
}

export class SetupScopeStore {
  private value: SetupScope | undefined;
  private listeners = new Set<() => void>();
  getSnapshot = () => this.value;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private set(next: SetupScope | undefined) { if (next === this.value) return; this.value = next; this.listeners.forEach(listener => listener()); }
  choose(row: Configuration) { this.set(scopeOf(row)); }
  reconcile(page: ConfigurationPage | undefined, after: string) { this.set(reconcileScope(this.value, page, after)); }
}

const stores = new Map<string, SetupScopeStore>();
export function sessionScopeStore(session: Pick<Session, "user" | "workspace" | "csrf">, contract: string, on: string): SetupScopeStore {
  const key = JSON.stringify([session.user, session.workspace, session.csrf, contract, on]);
  let store = stores.get(key);
  if (store) stores.delete(key);
  else store = new SetupScopeStore();
  stores.set(key, store);
  while (stores.size > 32) stores.delete(stores.keys().next().value!);
  return store;
}
