import { AdminError, type Attempt, type Outcome, type Secret } from "../../../../service/admin";
import { leaving, onDeliberateLeave } from "../../../../shared/unload-intent";
import { attemptState, nextPhase, outsideWindow, type Track } from "./live-model";
import { secretCollection, type Collect } from "./secret-collection";
import type { ServiceCredential } from "../../../../service/service-query-types";

export type Phase = "sending" | "unknown" | "issued" | "lapsed";
export type Issued = { ref: string; credential: string; until: string };
export type Rotation = { credential: string; overlap: string };
export type Entry = {
  service: string;
  track: Track;
  days: string;
  reason: string;
  expected: string;
  replay: number;
  phase: Phase;
  retry: boolean;
  firstSentMono: number;
  firstSentWall: number;
  error: AdminError | null;
  issued?: Issued;
  issuedAfter: number;
  rotate?: Rotation;
};
export type Refusal = { error: unknown; retry: boolean };
export type Clocks = { mono: () => number; wall: () => number };
export type Target = Pick<Window, "addEventListener" | "removeEventListener">;
export type IssueValues = Readonly<{ days: string; reason: string; rotate?: Readonly<Rotation> }>;
export type Marker = { service: string; credential: string };
export type Observation = Pick<ServiceCredential, "id" | "service" | "current_state" | "secret_pickup">;
export type Exec = (attempt: Attempt, expected: string, values: IssueValues) => Promise<Outcome>;
export type Sent =
  | { kind: "refused" }
  | { kind: "lapsed" }
  | { kind: "issued"; issued: Issued }
  | { kind: "unknown" | "removed" | "kept"; error: unknown };

const credentialPattern = /^cred-[0-9a-f]{16}$/;
const refPattern = /^[A-Za-z0-9_-]{16,64}$/;

export function issueAttempts(fresh: () => Attempt, clocks: Clocks, target?: Target, admitted: () => boolean = () => true, visible: () => boolean = () => true) {
  const entries = new Map<string, Map<string, Entry>>();
  const refusals = new Map<string, Map<string, Refusal>>();
  const marks = new Map<string, Map<string, string>>();
  const listeners = new Set<() => void>();
  let reads = 0;
  let version = 0;
  let listening = false;
  let active = false;
  let disposed = false;
  let unsubscribeLeave: (() => void) | null = null;
  const current = () => active && !disposed && admitted();
  const of = (user: string) => {
    let found = entries.get(user);
    if (!found) entries.set(user, (found = new Map()));
    return found;
  };
  const guarded = () => {
    if (!current()) return false;
    for (const list of entries.values()) for (const e of list.values()) if (e.phase !== "lapsed") return true;
    for (const list of marks.values()) if (list.size > 0) return true;
    return false;
  };
  const warn = (e: BeforeUnloadEvent) => {
    if (leaving() || !guarded()) return;
    e.preventDefault();
  };
  const changed = () => {
    version += 1;
    const want = guarded();
    if (target && want !== listening) {
      if (want) target.addEventListener("beforeunload", warn as EventListener);
      else target.removeEventListener("beforeunload", warn as EventListener);
      listening = want;
    }
    for (const fn of listeners) fn();
  };
  const elapsed = (e: Entry) => Math.max(clocks.mono() - e.firstSentMono, clocks.wall() - e.firstSentWall);
  const refuse = (user: string, service: string, refusal: Refusal | null) => {
    let found = refusals.get(user);
    if (!found) refusals.set(user, (found = new Map()));
    if (refusal) found.set(service, refusal);
    else found.delete(service);
  };
  const release = (user?: string) => {
    collection.release(user);
    for (const [who, list] of entries) if (user === undefined || who === user) list.clear();
    for (const who of marks.keys()) if (user === undefined || who === user) marks.delete(who);
    refusals.clear();
    changed();
  };
  const mark = (user: string, credential: string, service: string) => {
    if (!current()) return;
    let list = marks.get(user);
    if (!list) marks.set(user, (list = new Map()));
    list.set(credential, service);
    changed();
  };
  const collection = secretCollection({ current, visible, changed,
    issued: (user, service) => { const entry = entries.get(user)?.get(service); return entry?.phase === "issued" ? entry.issued : undefined; },
    returned: (user, service, credential, acknowledged) => {
      if (!current()) return;
      const list = entries.get(user);
      if (list?.get(service)?.issued?.credential === credential) list.delete(service);
      if (!acknowledged) mark(user, credential, service);
      else changed();
    },
  });
  const dispose = () => {
    if (disposed) return;
    disposed = true;
    active = false;
    unsubscribeLeave?.();
    unsubscribeLeave = null;
    release();
    listeners.clear();
  };
  return {
    version: () => version,
    activate() {
      if (disposed || !admitted()) return false;
      if (!active) { active = true; unsubscribeLeave = onDeliberateLeave(dispose); }
      return true;
    },
    current,
    disposed: () => disposed,
    dispose,
    subscribe(fn: () => void) {
      if (disposed) return () => {};
      listeners.add(fn);
      return () => { listeners.delete(fn); };
    },
    list(user: string): Entry[] {
      if (!current()) return [];
      return [...of(user).values()].sort((a, b) => a.firstSentMono - b.firstSentMono);
    },
    get(user: string, service: string): Entry | undefined {
      if (!current()) return undefined;
      return of(user).get(service);
    },
    refusal(user: string, service: string): Refusal | undefined {
      if (!current()) return undefined;
      return refusals.get(user)?.get(service);
    },
    clearRefusal(user: string, service: string) {
      if (!current() || !refusals.get(user)?.has(service)) return;
      refuse(user, service, null);
      changed();
    },
    guarded,
    listening: () => listening,
    readStarted(): number {
      if (!current()) return 0;
      reads += 1;
      return reads;
    },
    async submit(user: string, service: string, values: { days: string; reason: string; rotate?: Rotation }, revision: string, window: number, exec: Exec): Promise<Sent> {
      if (!current()) return { kind: "refused" };
      const list = of(user);
      let e = list.get(service);
      if (e && e.phase !== "unknown") return { kind: "refused" };
      if (e && outsideWindow(elapsed(e), e.replay)) {
        e.phase = "lapsed";
        changed();
        return { kind: "lapsed" };
      }
      if (!e) {
        e = { service, track: attemptState(fresh), days: values.days, reason: values.reason, expected: revision, replay: window, phase: "sending", retry: false,
          firstSentMono: clocks.mono(), firstSentWall: clocks.wall(), error: null, issuedAfter: 0, ...(values.rotate ? { rotate: { ...values.rotate } } : {}) };
        list.set(service, e);
      } else e.retry = true;
      const entry = e;
      entry.phase = "sending";
      refuse(user, service, null);
      const sent = entry.track.send(entry.expected);
      changed();
      let outcome: Outcome | null = null;
      let failure: unknown = null;
      try {
        outcome = await exec(sent.attempt, sent.expected, Object.freeze({ days: entry.days, reason: entry.reason, ...(entry.rotate ? { rotate: Object.freeze({ ...entry.rotate }) } : {}) }));
      } catch (err) {
        failure = err;
      }
      entry.track.idle();
      if (!current() || list.get(service) !== entry) return { kind: "refused" };
      if (outcome) {
        const credential = outcome.fields.credential ?? "";
        const ref = outcome.fields.secret_ref ?? "";
        if (outcome.status === "completed" && outcome.state === (entry.rotate ? "credential_rotated" : "credential_issued") && outcome.job_id === "" && credentialPattern.test(credential) && refPattern.test(ref)) {
          entry.track.succeeded();
          const until = new Date(clocks.wall() + Number(entry.days) * 86400000).toISOString();
          entry.issued = { ref, credential, until };
          entry.issuedAfter = reads;
          entry.phase = "issued";
          entry.error = null;
          changed();
          return { kind: "issued", issued: entry.issued };
        }
        failure = new AdminError("unavailable");
      }
      const next = nextPhase(entry.retry, failure, outsideWindow(elapsed(entry), entry.replay, true));
      if (next === "remove") {
        list.delete(service);
        refuse(user, service, { error: failure, retry: entry.retry });
        changed();
        return { kind: "removed", error: failure };
      }
      entry.phase = next;
      entry.error = failure instanceof AdminError ? failure : null;
      if (next === "unknown") entry.track.keep(entry.expected);
      changed();
      return next === "lapsed" ? { kind: "lapsed" } : { kind: "unknown", error: failure };
    },
    discard(user: string, service: string): boolean {
      if (!current()) return false;
      const list = of(user);
      const e = list.get(service);
      if (!e || (e.phase !== "unknown" && e.phase !== "lapsed")) return false;
      list.delete(service);
      changed();
      return true;
    },
    open(user: string, service: string): Issued | null {
      if (!current()) return null;
      const entry = entries.get(user)?.get(service);
      return entry?.phase === "issued" && entry.issued ? { ...entry.issued } : null;
    },
    collect(user: string, service: string, issued: Pick<Issued, "credential" | "ref">, exec: Collect, retry = false): Promise<Secret> {
      return collection.start(user, service, issued, exec, retry);
    },
    collectionPhase: collection.phase,
    revoked(user: string, credential: string) {
      if (!current()) return;
      const list = of(user);
      for (const [service, entry] of list) if (entry.phase === "issued" && entry.issued?.credential === credential) list.delete(service);
      collection.cancel(user, credential);
      marks.get(user)?.delete(credential);
      changed();
    },
    observe(user: string, row: Observation | null, readSeq: number) {
      if (!row || !current()) return;
      const list = entries.get(user);
      const entry = list?.get(row.service);
      if (!entry || entry.phase !== "issued" || entry.issued?.credential !== row.id || readSeq <= entry.issuedAfter) return;
      if (row.current_state === "revoked" || row.current_state === "expired") {
        list!.delete(row.service);
        collection.cancel(user, row.id);
        changed();
      } else if (row.secret_pickup === "collected") {
        list!.delete(row.service);
        mark(user, row.id, row.service);
      }
    },
    marks(user: string): Marker[] {
      return current() ? [...(marks.get(user)?.entries() ?? [])].map(([credential, service]) => ({ credential, service })) : [];
    },
    mark(user: string, credential: string, service: string) { mark(user, credential, service); },
    unmark(user: string, credential: string) {
      if (!current()) return;
      marks.get(user)?.delete(credential);
      collection.acknowledge(user, credential);
      changed();
    },
    release,
  };
}

export type IssueAttempts = ReturnType<typeof issueAttempts>;
