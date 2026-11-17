import { AdminError } from "./admin";

export type StepUpGate = {
  request(): Promise<boolean>;
  settle(ok: boolean): void;
  pending(): boolean;
};

export function stepUpGate(open: () => void): StepUpGate {
  let current: { promise: Promise<boolean>; resolve: (ok: boolean) => void } | null = null;
  return {
    request() {
      if (current) return current.promise;
      let resolve: (ok: boolean) => void = () => undefined;
      const promise = new Promise<boolean>((r) => { resolve = r; });
      current = { promise, resolve };
      open();
      return promise;
    },
    settle(ok) {
      const settling = current;
      if (!settling) return;
      current = null;
      settling.resolve(ok);
    },
    pending() {
      return current !== null;
    },
  };
}

export type DialogOutcome =
  | { kind: "done" }
  | { kind: "wrong" }
  | { kind: "signed-out" }
  | { kind: "changed" }
  | { kind: "failed" };

export function outcomeOf(e: unknown): DialogOutcome {
  if (!(e instanceof AdminError)) return { kind: "failed" };
  switch (e.failure) {
    case "wrong":
      return { kind: "wrong" };
    case "unauthenticated":
      return { kind: "signed-out" };
    case "conflict":
      return { kind: "changed" };
  }
  return { kind: "failed" };
}

export function cancellable(pending: boolean): boolean {
  return !pending;
}

export function settleOnce(settle: (ok: boolean) => void): (ok: boolean) => void {
  let done = false;
  return (ok) => {
    if (done) return;
    done = true;
    settle(ok);
  };
}
