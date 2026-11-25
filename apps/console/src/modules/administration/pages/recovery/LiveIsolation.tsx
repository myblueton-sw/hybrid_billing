import type { Locale } from "../../../../shared/types";
import { useEffect, useRef, useState } from "react";
import { AdminError } from "../../../../service/admin";
import type { IsolationRow, LockEntry, LocksPage } from "../../../../service/recovery";
import { k } from "./live-copy";
import {
  basisContext, basisFrom, basisOf, blockerPolicy, callsOf, generationNumber, installTarget, lockRows, more, pendingNow, refusedNow, rowId, stoppedOf, type Basis,
} from "./live-model";
import { Code, Known, ListOr, Prose, Rich, Stamp, Who, ltr } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";

function Calls({ l, r }: { l: Locale; r: IsolationRow }) {
  const c = callsOf(r);
  return (
    <section className="p03-calls" aria-label={k.openCalls(l)}>
      <h4>{k.openCalls(l)}</h4>
      {c.kind === "not_shown" && <p className="live-empty">{k.callsNotShown(l)}</p>}
      {c.kind === "empty" && <p className="live-empty">{k.noCalls(l)}</p>}
      {c.kind === "unknown" && <p className="live-empty">{c.code ? <>{k.unknown(l)} ({ltr(c.code)})</> : k.unknown(l)}</p>}
      {c.kind === "listed" && r.open_calls && (
        <>
          {more(r.open_calls.length, r.open_calls_total) && <p className="p03-small live-quiet">{k.showingOf(l, r.open_calls_total ?? 0)}</p>}
          <div className="live-table">
            <table className="p03-table">
              <colgroup><col className="p03-c16" /><col className="p03-c22" /><col className="p03-c22" /><col className="p03-c40" /></colgroup>
              <thead><tr><th scope="col">{k.colCall(l)}</th><th scope="col">{k.colWriter(l)}</th><th scope="col">{k.colDeadline(l)}</th><th scope="col">{k.colTarget(l)}</th></tr></thead>
              <tbody>
                {r.open_calls.map((call, n) => (
                  <tr key={call.id || n} className="p03-call">
                    <td className="p03-first"><Known l={l} value={call.id} /></td>
                    <td data-label={k.colWriter(l)}><Known l={l} value={call.writer} /></td>
                    <td data-label={k.colDeadline(l)}><Stamp l={l} at={call.deadline} /></td>
                    <td data-label={k.colTarget(l)} className="p03-wrap">
                      {!call.key ? k.unknown(l) : call.series && call.number === null ? <>{ltr(call.key + " · " + call.series + " #", true)} {k.unknown(l)}</> : ltr(call.key + (call.series ? " · " + call.series + " #" + call.number : ""), true)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      <p className="p03-small live-quiet p03-call-note">
        {r.other_scope_present === null ? <>{k.otherScopeUnknown(l)} </> : r.other_scope_present ? <>{k.otherScope(l)} </> : null}
        {k.deadlineNote(l)}
      </p>
    </section>
  );
}

type OlderLocks = { key: string; rows: LockEntry[]; next: string | null; total: number | null; status: "idle" | "loading" | "failed" };

function LockLog({ l, r, revision, read, onRefused }: { l: Locale; r: IsolationRow; revision: string; read?: (installation: string, after: string) => Promise<LocksPage>; onRefused?: (e: unknown) => boolean }) {
  const key = r.installation + "\u0000" + revision;
  const blank = (): OlderLocks => ({ key, rows: [], next: null, total: null, status: "idle" });
  const [state, setState] = useState<OlderLocks>(blank);
  const [reset, setReset] = useState(false);
  const token = useRef(0);
  const loaded = useRef({ key, rows: 0 });
  const [landed, setLanded] = useState(0);
  const countRef = useRef<HTMLParagraphElement>(null);
  useEffect(() => { if (landed > 0) countRef.current?.focus(); }, [landed]);
  useEffect(() => {
    token.current += 1;
    if (loaded.current.key !== key && loaded.current.rows > 0) setReset(true);
    loaded.current = { key, rows: 0 };
    setState((was) => (was.key === key ? was : blank()));
  }, [key]); // eslint-disable-line react-hooks/exhaustive-deps
  const current = state.key === key ? state : blank();
  const facts = r.lock_history;
  const next = facts && read ? current.next ?? facts.next : "";
  const rows = r.locks === null ? null : lockRows(r.locks, current.rows);
  const total = Math.max(current.total ?? facts?.total ?? 0, rows?.length ?? 0);
  const failed = current.status === "failed";
  const load = async () => {
    if (current.status === "loading" || !next || !read) return;
    const mine = ++token.current;
    setReset(false);
    setState({ ...current, status: "loading" });
    try {
      const got = await read(r.installation, next);
      if (mine !== token.current) return;
      loaded.current = { key, rows: current.rows.length + got.rows.length };
      setState({ key, rows: [...current.rows, ...got.rows], next: got.next, total: got.total, status: "idle" });
      if (!got.next) setLanded((n) => n + 1);
    } catch (err) {
      if (mine !== token.current) return;
      setState({ ...current, status: onRefused?.(err) ? "idle" : "failed" });
    }
  };
  const cut = rows !== null && facts !== null && total > (r.locks?.length ?? 0);
  return (
    <section className="p03-locklog" aria-label={k.lockLog(l)}>
      <h4>{k.lockLog(l)}</h4>
      {cut && <p className="p03-small live-quiet p03-lock-count" role="status" ref={countRef} tabIndex={-1}>{!next && rows.length >= total ? k.lockAll(l, total) : k.lockShown(l, rows.length, total)}</p>}
      {reset && <p className="p03-small p03-lock-reset" role="status">{k.lockReset(l)}</p>}
      <ListOr l={l} items={rows} empty={k.noLocks(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p03-table">
              <colgroup><col className="p03-c24" /><col className="p03-c30" /><col className="p03-c46" /></colgroup>
              <thead><tr><th scope="col">{k.colTime(l)}</th><th scope="col">{k.colAdmin(l)}</th><th scope="col">{k.colReason(l)}</th></tr></thead>
              <tbody>
                {list.map((x, n) => (
                  <tr key={x.number ?? "n" + n} className="p03-lock-row">
                    <td className="p03-first"><Stamp l={l} at={x.at} /></td>
                    <td data-label={k.colAdmin(l)}><Who l={l} login={x.by_login} id={x.by} /></td>
                    <td data-label={k.colReason(l)}><Prose l={l} value={x.rationale} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      {next && (
        <div className="p03-lock-more">
          {failed && <p role="alert">{k.lockFailed(l)}</p>}
          <button type="button" className="live-secondary p03-lock-more-button" aria-disabled={current.status === "loading" || undefined} onClick={load}>
            {failed ? k.lockTryAgain(l) : current.status === "loading" ? k.lockLoading(l) : k.lockMore(l)}
          </button>
        </div>
      )}
    </section>
  );
}

export function LockFacts({ l, b }: { l: Locale; b: Basis }) {
  const g = generationNumber(b);
  return (
    <>
      <div><dt>{k.factInstallation(l)}</dt><dd><Known l={l} value={b.installation} /></dd></div>
      <div><dt>{k.factState(l)}</dt><dd><Code l={l} value={b.state} say={k.installState} /></dd></div>
      {g !== null && <div><dt>{k.colGeneration(l)}</dt><dd>{ltr(g + " → " + (g + 1))}</dd></div>}
    </>
  );
}

export function LockImpact({ l, b }: { l: Locale; b: Basis }) {
  const begins = b.begins ? b.begins.split(",") : [];
  return (
    <>
      <p><strong>{k.lockStrong(l)}</strong> <Rich text={k.lockImpact(l, b.installation, generationNumber(b))} /></p>
      {begins.map((n) => <p key={n}>{k.beginRetired(l, n)}</p>)}
    </>
  );
}

function Actions({ r, ...shared }: Shared & { r: IsolationRow }) {
  const { l, page, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"lock">();
  const target = installTarget(r.installation);
  const pending = target ? pendingNow("recovery.lock", target) : undefined;
  const usable = canCommand && !!target && (r.can_lock || !!pending);
  const blockers: Blocker[] = [];
  if (!target) blockers.push(line("", k.narrow(l, "no_command")));
  if (target && !usable && !r.can_lock && blockerPolicy(r.lock_blocked, "lock") === "explain") blockers.push(line(k.lock(l), k.blocker(l, r.lock_blocked)));
  const kept = target && refusedNow("recovery.lock", target) ? k.kept(l, "recovery.lock") : "";
  const b = pending ? basisFrom(pending.context) : basisOf(page, r.installation);
  return (
    <div className="p03-actions">
      <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} />
      {usable && !open && (
        <div className="live-buttons">
          <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("lock")} onClick={() => toggle("lock")}>{k.lock(l)}</button>
        </div>
      )}
      {open && usable && (
        <CommandForm
          {...shared} command="recovery.lock" target={target} values={{}}
          title={k.confirmLock(l)}
          facts={<LockFacts l={l} b={b} />}
          impact={<LockImpact l={l} b={b} />}
          verb={k.lockVerb(l, r.installation)}
          context={basisContext(b)}
          done={(fields, state) => (state === "locked" ? k.lockedDone(l, fields.installation || r.installation) : k.resultUnknown(l, state))}
          onClose={close}
        />
      )}
    </div>
  );
}

export function IsolationView(shared: Shared) {
  const { l, page, selection } = shared;
  return (
    <ListOr l={l} items={page.isolation.rows} empty={k.notConfigured(l)}>
      {(rows) => (
        <>
          {rows.map((r, n) => {
            const current = !!r.installation && selection.installation === r.installation;
            const stopped = stoppedOf(r.state, r.stopped_at);
            const head = "p03-iso-h-" + n;
            return (
              <article
                key={r.installation || n} className={"p03-install" + (current ? " live-selected" : "")} id={r.installation ? rowId("iso", r.installation) : undefined}
                aria-labelledby={head} aria-current={current || undefined}
              >
                <div className="p03-install-head">
                  <h3 id={head}><Known l={l} value={r.installation} /></h3>
                  <span className="p03-tag"><Code l={l} value={r.state} say={k.installState} /></span>
                </div>
                <dl className="p03-facts p03-facts-5">
                  <div><dt>{k.colGeneration(l)}</dt><dd>{r.generation === null ? k.unknown(l) : <bdi dir="ltr">{r.generation}</bdi>}</dd></div>
                  <div><dt>{k.stoppedAt(l)}</dt><dd>{stopped.kind === "time" ? <Stamp l={l} at={stopped.at} /> : stopped.kind === "not_stopped" ? k.notStopped(l) : k.unknown(l)}</dd></div>
                  <div>
                    <dt>{k.colWriters(l)}</dt>
                    <dd>
                      <ListOr l={l} items={r.writers} empty={k.noWriters(l)}>{(list) => list.map((w) => <span key={w} className="live-sub">{ltr(w)}</span>)}</ListOr>
                      <span className="live-sub live-quiet p03-small">{k.noWriterRemoval(l)}</span>
                    </dd>
                  </div>
                  <div><dt>{k.colVerifier(l)}</dt><dd>{r.verifier ? ltr(r.verifier) : k.noVerifier(l)}</dd></div>
                  <div>
                    <dt>{k.permits(l)}</dt>
                    <dd>
                      <Code l={l} value={r.permits} say={k.shown} />
                      {r.permits === "not_shown" && <span className="live-sub live-quiet p03-small">{k.permitsNever(l)}</span>}
                    </dd>
                  </div>
                </dl>
                <Calls l={l} r={r} />
                <LockLog l={l} r={r} revision={shared.page.revision} read={shared.api.recoveryLocks} onRefused={(e) => {
                  if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
                  if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
                  return false;
                }} />
                <Actions {...shared} r={r} />
              </article>
            );
          })}
        </>
      )}
    </ListOr>
  );
}
