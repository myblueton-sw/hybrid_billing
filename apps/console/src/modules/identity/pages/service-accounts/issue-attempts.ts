import { AdminError, type Attempt, type Outcome, type ServicesPage } from "../../../../service/admin";
import { leaving, onDeliberateLeave } from "../../../../shared/unload-intent";
import { attemptState, nextPhase, outsideWindow, type Track } from "./live-model";

export type Phase = "sending" | "unknown" | "issued" | "lapsed";
export type Issued = { ref: string; credential: string; until: string };
export type Rotation = { credential: string; overlap: string };
export type Entry = {
  service: string;
  track: Track;
  days: string;
  reason: string;
  expected: string;
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
export type Exec = (attempt: Attempt, expected: string) => Promise<Outcome>;
export type Sent =
  | { kind: "refused" }
  | { kind: "lapsed" }
  | { kind: "issued"; issued: Issued }
  | { kind: "unknown" | "removed" | "kept"; error: unknown };

const credentialPattern = /^cred-[0-9a-f]{16}$/;
const refPattern = /^[A-Za-z0-9_-]{16,64}$/;

export function issueAttempts(fresh: () => Attempt, clocks: Clocks, target?: Target) {
  const entries = new Map<string, Map<string, Entry>>();
  const refusals = new Map<string, Map<string, Refusal>>();
  const marks = new Map<string, string[]>();
  const listeners = new Set<() => void>();
  let reads = 0;
  let version = 0;
  let listening = false;
  const of = (user: string) => {
    let found = entries.get(user);
    if (!found) entries.set(user, (found = new Map()));
    return found;
  };
  const guarded = () => {
    for (const list of entries.values()) for (const e of list.values()) if (e.phase !== "lapsed") return true;
    for (const list of marks.values()) if (list.length > 0) return true;
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
    for (const [who, list] of entries) if (user === undefined || who === user) list.clear();
    for (const who of marks.keys()) if (user === undefined || who === user) marks.set(who, []);
    refusals.clear();
    changed();
  };
  onDeliberateLeave(() => release());
  return {
    version: () => version,
    subscribe(fn: () => void) {
      listeners.add(fn);
      return () => { listeners.delete(fn); };
    },
    list(user: string): Entry[] {
      return [...of(user).values()].sort((a, b) => a.firstSentMono - b.firstSentMono);
    },
    get(user: string, service: string): Entry | undefined {
      return of(user).get(service);
    },
    refusal(user: string, service: string): Refusal | undefined {
      return refusals.get(user)?.get(service);
    },
    clearRefusal(user: string, service: string) {
      if (!refusals.get(user)?.has(service)) return;
      refuse(user, service, null);
      changed();
    },
    guarded,
    listening: () => listening,
    readStarted(): number {
      reads += 1;
      return reads;
    },
    async submit(user: string, service: string, values: { days: string; reason: string; rotate?: Rotation }, revision: string, window: number, exec: Exec): Promise<Sent> {
      const list = of(user);
      let e = list.get(service);
      if (e && e.phase !== "unknown") return { kind: "refused" };
      if (e && outsideWindow(elapsed(e), window)) {
        e.phase = "lapsed";
        changed();
        return { kind: "lapsed" };
      }
      if (!e) {
        e = { service, track: attemptState(fresh), days: values.days, reason: values.reason, expected: revision, phase: "sending", retry: false,
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
        outcome = await exec(sent.attempt, sent.expected);
      } catch (err) {
        failure = err;
      }
      entry.track.idle();
      if (list.get(service) !== entry) return { kind: "refused" };
      if (outcome) {
        const credential = outcome.fields.credential ?? "";
        const ref = outcome.fields.secret_ref ?? "";
        if (credentialPattern.test(credential) && refPattern.test(ref)) {
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
      const next = nextPhase(entry.retry, failure, outsideWindow(elapsed(entry), window, true));
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
      const list = of(user);
      const e = list.get(service);
      if (!e || (e.phase !== "unknown" && e.phase !== "lapsed")) return false;
      list.delete(service);
      changed();
      return true;
    },
    consume(user: string, service: string): Issued | null {
      const list = of(user);
      const e = list.get(service);
      if (!e || e.phase !== "issued" || !e.issued) return null;
      list.delete(service);
      changed();
      return e.issued;
    },
    revoked(user: string, credential: string) {
      const list = of(user);
      for (const [service, e] of list) if (e.phase === "issued" && e.issued?.credential === credential) list.delete(service);
      changed();
    },
    reconcile(user: string, page: ServicesPage, readSeq: number) {
      const list = of(user);
      let dropped = false;
      for (const [service, e] of list) {
        if (e.phase !== "issued" || !e.issued || readSeq <= e.issuedAfter) continue;
        const row = page.services.find((s) => s.id === service)?.credentials.find((c) => c.id === e.issued?.credential);
        if (row?.state === "active" && row.secret_pickup === "open") continue;
        list.delete(service);
        dropped = true;
      }
      if (dropped) changed();
    },
    marks(user: string): string[] {
      return marks.get(user) ?? [];
    },
    mark(user: string, id: string) {
      const now = marks.get(user) ?? [];
      if (!now.includes(id)) marks.set(user, [...now, id]);
      changed();
    },
    unmark(user: string, id: string) {
      marks.set(user, (marks.get(user) ?? []).filter((x) => x !== id));
      changed();
    },
    release,
  };
}

export type IssueAttempts = ReturnType<typeof issueAttempts>;
