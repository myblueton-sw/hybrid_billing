import type { Locale } from "../../../../shared/types";
import type { IsolationRow } from "../../../../service/recovery";
import { k } from "./live-copy";
import {
  basisContext, basisFrom, basisOf, blockerPolicy, callsOf, generationNumber, installTarget, more, pendingNow, refusedNow, rowId, stoppedOf, type Basis,
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

function LockLog({ l, r }: { l: Locale; r: IsolationRow }) {
  return (
    <section className="p03-locklog" aria-label={k.lockLog(l)}>
      <h4>{k.lockLog(l)}</h4>
      <ListOr l={l} items={r.locks} empty={k.noLocks(l)}>
        {(list) => (
          <div className="live-table">
            <table className="p03-table">
              <colgroup><col className="p03-c24" /><col className="p03-c30" /><col className="p03-c46" /></colgroup>
              <thead><tr><th scope="col">{k.colTime(l)}</th><th scope="col">{k.colAdmin(l)}</th><th scope="col">{k.colReason(l)}</th></tr></thead>
              <tbody>
                {list.map((x, n) => (
                  <tr key={n} className="p03-lock-row">
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
  const { open, toggle, close, focusNote, mark } = useOpen<"lock">();
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
      <RowNote kept={kept} blockers={blockers} focus={focusNote} />
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
                <LockLog l={l} r={r} />
                <Actions {...shared} r={r} />
              </article>
            );
          })}
        </>
      )}
    </ListOr>
  );
}
