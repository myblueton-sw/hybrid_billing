import { useEffect, useMemo, useRef, useState, useSyncExternalStore, type ReactNode } from "react";
import { attempt } from "../../../../../service/admin";
import type { Session } from "../../../../../service/api";
import { useService } from "../../../../../service/context";
import type { Locale } from "../../../../../shared/types";
import { guardUnload } from "../../../../../shared/unload-intent";
import { ltr, stamp } from "../live-format";
import { SetupError } from "./context";
import type { ProtectedScope } from "./protected";
import { stepText as s, type StepWord } from "./step-copy";
import { anyStepDirty, sessionStepStore, type StepEntry, type StepIntent, type StepWriteStore } from "./step-store";
import { needsAuthentication } from "./write-copy";

export type ScopedIntent = StepIntent & Readonly<{ scope: ProtectedScope }>;
let guarded = false;
export const codeOf = (e: unknown) => e instanceof SetupError ? e.code : "service_unavailable";

export function useStepStore<I extends ScopedIntent, R>(kind: string, session: Session, send: () => (intent: I) => Promise<R>): StepWriteStore<I, R> | undefined {
 const store = useMemo(() => { try { return sessionStepStore<I, R>(kind, session, send); } catch { return undefined; } }, [kind, session.user, session.workspace, session.csrf]);
 useEffect(() => { if (guarded || typeof window === "undefined") return; guarded = true; window.addEventListener("beforeunload", guardUnload(anyStepDirty)); }, []);
 return store;
}

type Props<I extends ScopedIntent, R> = {
 l: Locale; id: string; title?: string; intro?: boolean; store: StepWriteStore<I, R>; contract: string; onSettled: () => void;
 describe: (entry: StepEntry<I, R>) => Readonly<{ action: string; target: ReactNode; state?: ReactNode; stateLabel?: string; note?: ReactNode }>;
 explain: (code: string, entry?: StepEntry<I, R>) => string; revision: (intent: I) => Promise<string>;
};
export function StepResults<I extends ScopedIntent, R>({ l, id, title, intro = true, store, contract, onSettled, describe, explain, revision }: Props<I, R>) {
 const state = useSyncExternalStore(store.subscribe, store.snapshot), service = useService(), heading = useRef<HTMLHeadingElement>(null), [error, setError] = useState(""), [working, setWorking] = useState(false);
 const entries = state.entries.filter(e => e.intent.scope.contract === contract || e.phase === "pending" || e.phase === "unknown").reverse();
 const busy = working || state.entries.some(e => e.phase === "pending"), latest = state.entries.at(-1);
 const settle = async (run: () => Promise<void>) => { if (working) return; setWorking(true); setError(""); heading.current?.focus(); try { await run(); } catch (e) { setError(codeOf(e)); return; } finally { setWorking(false); } onSettled(); };
 const recover = async (key: string, intent: I) => {
  let current: string;
  try { current = await revision(intent); } catch { throw new SetupError("recovery_read"); }
  await store.recover(key, attempt(), current);
 };
 if (entries.length === 0) return null;
 return <section className="o03-part o03-steps o03-results" aria-labelledby={id} aria-busy={busy || undefined}>
  <h4 id={id} ref={heading} tabIndex={-1}>{title ?? s(l, "results")}</h4>
  {intro && <p className="live-sub">{s(l, "sessionOnly")}</p>}
  {error && <div className="live-error" role="alert"><p>{explain(error)}</p><p className="live-mono">{s(l, "code")}: {ltr(error)}</p></div>}
  {service?.stepUp && needsAuthentication(error || (latest && latest.phase !== "recorded" && latest.phase !== "reconciled" ? latest.error ?? "" : "")) && <button type="button" className="live-secondary" disabled={busy} onClick={() => void service.stepUp?.().catch(() => setError("reauthentication_required"))}>{s(l, "auth")}</button>}
  <div aria-live="polite">{entries.map(entry => { const d = describe(entry); return <article className="o03-setup-outcome" key={entry.intent.attempt.key}>
   <h5>{d.action}</h5>
   <p>{d.target}<span className="live-sub">{s(l, "forContract")}: {ltr(entry.intent.scope.contract)} · {s(l, "settings")}: {ltr(entry.intent.scope.configuration)} · {s(l, "sentAt")}: {stamp(l, entry.at)}</span></p>
   <p>{s(l, (entry.phase === "unknown" && entry.recovers ? "phase_unknown_retry" : "phase_" + entry.phase) as StepWord)}</p>
   {entry.error && entry.phase === "refused" && <p>{explain(entry.error, entry)}</p>}
   {entry.error && (entry.phase === "refused" || entry.phase === "unknown") && <p className="live-mono">{s(l, "code")}: {ltr(entry.error)}</p>}
   {d.note && <p>{d.note}</p>}
   {d.state && <p>{d.stateLabel ?? s(l, "recordedState")}: {d.state}</p>}
   {entry.phase === "unknown" && <div className="o03-docs-pages"><button type="button" className="live-secondary" disabled={busy} onClick={() => void settle(() => store.retry(entry.intent.attempt.key))}>{s(l, "retry")}</button>{!entry.recovers && <button type="button" className="live-secondary" disabled={busy} onClick={() => void settle(() => recover(entry.intent.attempt.key, entry.intent))}>{s(l, "recover")}</button>}</div>}
  </article>; })}</div>
 </section>;
}
