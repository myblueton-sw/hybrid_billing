import { AdminError, type Secret } from "../../../../service/admin";
import type { Issued } from "./issue-attempts";

export type CollectionPhase = "idle" | "pending" | "unknown";
export type Collect = (reference: string) => Promise<Secret>;
type Pending = { service: string; ref: string; promise: Promise<Secret>; cancelled: boolean; acknowledged: boolean };
type Hooks = {
  current: () => boolean;
  visible: () => boolean;
  issued: (user: string, service: string) => Issued | undefined;
  returned: (user: string, service: string, credential: string, acknowledged: boolean) => void;
  changed: () => void;
};

export function secretCollection(hooks: Hooks) {
  const pending = new Map<string, Pending>();
  const unknown = new Set<string>();
  const key = (user: string, credential: string) => JSON.stringify([user, credential]);
  const unavailable = () => new AdminError("unavailable");
  const signedOut = () => new AdminError("unauthenticated");
  return {
    start(user: string, service: string, target: Pick<Issued, "credential" | "ref">, collect: Collect, retry = false): Promise<Secret> {
      if (!hooks.current()) return Promise.reject(signedOut());
      if (!hooks.visible()) return Promise.reject(unavailable());
      const { credential, ref: reference } = target;
      const id = key(user, credential);
      const found = pending.get(id);
      if (found) return found.service === service && found.ref === reference ? found.promise : Promise.reject(unavailable());
      const issued = hooks.issued(user, service);
      if (!issued || issued.credential !== credential || issued.ref !== reference || unknown.has(id) && !retry) return Promise.reject(unavailable());
      let operation: Pending;
      let dispatched = false;
      const promise = Promise.resolve().then(() => {
        if (!hooks.current() || operation.cancelled) throw signedOut();
        if (!hooks.visible()) throw unavailable();
        dispatched = true;
        return collect(reference);
      }).then((result) => {
        if (!hooks.current() || operation.cancelled) throw signedOut();
        if (!result || result.credential !== credential || typeof result.secret !== "string" || !/^hbsk_[A-Za-z0-9_-]{43}$/.test(result.secret)) throw unavailable();
        unknown.delete(id);
        hooks.returned(user, service, credential, operation.acknowledged);
        return result;
      }).catch((error: unknown) => {
        if (dispatched && hooks.current() && !operation.cancelled) unknown.add(id);
        throw error;
      }).finally(() => {
        if (pending.get(id) === operation) pending.delete(id);
        if (hooks.current()) hooks.changed();
      });
      operation = { service, ref: reference, promise, cancelled: false, acknowledged: false };
      pending.set(id, operation);
      hooks.changed();
      return promise;
    },
    phase(user: string, credential: string): CollectionPhase {
      if (!hooks.current()) return "idle";
      const id = key(user, credential);
      if (pending.has(id)) return "pending";
      return unknown.has(id) ? "unknown" : "idle";
    },
    acknowledge(user: string, credential: string) {
      if (!hooks.current()) return;
      const found = pending.get(key(user, credential));
      if (found) found.acknowledged = true;
    },
    cancel(user: string, credential: string) {
      const id = key(user, credential);
      const found = pending.get(id);
      if (found) found.cancelled = true;
      unknown.delete(id);
    },
    release(user?: string) {
      for (const [id, operation] of pending) if (user === undefined || JSON.parse(id)[0] === user) {
        operation.cancelled = true;
        pending.delete(id);
      }
      for (const id of unknown) if (user === undefined || JSON.parse(id)[0] === user) unknown.delete(id);
    },
  };
}
