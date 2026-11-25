import type { Locale } from "../../../../shared/types";
import { runOutcomeKeys, type RetentionPage, type RunRow } from "../../../../service/retention";
import { useOlderRows } from "../../../../shared/older-rows";
import { AdminError } from "../../../../service/admin";
import { counted, plurals } from "./live-codes";
import { k } from "./live-copy";
import { artifactOf, blockerPolicy, keptKey, pendingNow, refusedNow, rowId, runNumber, runOf, runTarget, stateSentence, storesBalanced } from "./live-model";
import { Code, Free, MoreLine, Num, Rich, Who, href, ltr } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { Strip } from "./LiveExpiry";
import { Proposals } from "./LiveProposals";

function RunFacts({ l, ctx }: { l: Locale; ctx: Record<string, string> }) {
  return (
    <>
      <div><dt>{k.factRun(l)}</dt><dd>{ctx.run ? ltr(ctx.run) : k.unknown(l)}</dd></div>
      <div><dt>{k.factClass(l)}</dt><dd><Code l={l} value={ctx.class ?? ""} say={k.className} /></dd></div>
      <div><dt>{k.factState(l)}</dt><dd><Code l={l} value={ctx.state ?? ""} say={k.runState} /></dd></div>
      <div><dt>{k.factNotYetClaimed(l)}</dt><dd>{ctx.pending ? ltr(ctx.pending) : k.unknown(l)}</dd></div>
    </>
  );
}

function StateWords({ l, state }: { l: Locale; state: string }) {
  const note = k.runStateNote(l, state);
  return note ? <p className="p04-small">{note}</p> : null;
}

function Stores({ l, r }: { l: Locale; r: RunRow }) {
  const list = r.stores;
  const ok = storesBalanced(list);
  return (
    <>
      <h4 className="p04-per-store">{k.perStore(l)}</h4>
      {!ok && <p className="p04-warn">{k.sumWarn(l)}</p>}
      {list === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : list.length === 0 ? (
        <p className="live-empty">{k.none(l)}</p>
      ) : (
        <div className="live-table p04-cap-840">
          <table className="p04-table p04-stores">
            <caption className="visually-hidden">{k.storesCaption(l)}</caption>
            <colgroup><col className="p04-c24" /><col className="p04-c15" /><col className="p04-c15" /><col className="p04-c15" /><col className="p04-c15" /><col className="p04-c16" /></colgroup>
            <thead>
              <tr>
                <th scope="col">{k.colStore(l)}</th>
                <th scope="col" className="p04-num">{k.colTarget(l)}</th>
                <th scope="col" className="p04-num">{k.colDeleted(l)}</th>
                <th scope="col" className="p04-num">{k.colFailed(l)}</th>
                <th scope="col" className="p04-num">{k.colExcluded(l)}</th>
                <th scope="col" className="p04-num">{k.colPending(l)}</th>
              </tr>
            </thead>
            <tbody>
              {list.map((s, n) => (
                <tr key={s.store || n} className="p04-store-row">
                  <td className="p04-first">{s.store ? ltr(s.store) : k.unknown(l)}</td>
                  <td className="p04-num" data-label={k.colTarget(l)}><Num l={l} n={s.target} /></td>
                  <td className="p04-num" data-label={k.colDeleted(l)}><Num l={l} n={s.deleted} /></td>
                  <td className="p04-num" data-label={k.colFailed(l)}><Num l={l} n={s.failed} /></td>
                  <td className="p04-num" data-label={k.colExcluded(l)}><Num l={l} n={s.excluded} /></td>
                  <td className="p04-num" data-label={k.colPending(l)}><Num l={l} n={s.pending} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <p className="p04-small live-quiet">{k.sumNote(l)} {k.backupNote(l)}</p>
      <p className="p04-small">{k.outsideNote(l)}</p>
      <p className="p04-small">{k.aiNote(l)}</p>
    </>
  );
}

function Items({ l, r, page }: { l: Locale; r: RunRow; page: RetentionPage }) {
  const list = r.artifacts.rows;
  return (
    <>
      <h4>{k.itemsTitle(l)}</h4>
      <MoreLine l={l} more={r.artifacts.more} total={r.artifacts.total} forms={plurals.showingOf} />
      {list === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : list.length === 0 ? (
        <p className="live-empty">{k.none(l)}</p>
      ) : (
        <div className="live-table p04-cap-720">
          <table className="p04-table p04-run-items">
            <caption className="visually-hidden">{k.runItemsCaption(l)}</caption>
            <colgroup><col className="p04-c60" /><col className="p04-c40" /></colgroup>
            <thead><tr><th scope="col">{k.colItem(l)}</th><th scope="col">{k.colOutcome(l)}</th></tr></thead>
            <tbody>
              {list.map((x, n) => {
                const found = x.id ? artifactOf(page, x.id) : undefined;
                return (
                  <tr key={x.id || n}>
                    <td className="p04-first">
                      {!x.id ? k.unknown(l) : found ? <a className="p04-link" href={href(l, "expiry", { class: found.group.class, artifact: x.id })}>{ltr(x.id)}</a> : ltr(x.id)}
                    </td>
                    <td data-label={k.colOutcome(l)}><Code l={l} value={x.outcome} say={k.runOutcome} /></td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function Run({ r, ...shared }: Shared & { r: RunRow }) {
  const { l, page, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"stop">();
  const n = runNumber(r.run);
  const label = r.run && n ? r.run.label : "";
  const target = runTarget(r);
  const pending = target ? pendingNow("retention.stop_run", target) : undefined;
  const usable = canCommand && !!target && (r.can_stop || !!pending);
  const refused = target ? refusedNow("retention.stop_run", target) : "";
  const kept = refused ? k.kept(l, keptKey("retention.stop_run", refused)) : "";
  const blockers: Blocker[] = [];
  let state = "";
  if (!target) blockers.push(line("", k.narrow(l, "no_command")));
  else if (!usable && !r.can_stop) {
    if (blockerPolicy(r.stop_blocked, "stop_run") === "explain") blockers.push(line(k.stopRun(l), k.blocker(l, r.stop_blocked)));
    else if (stateSentence(r.stop_blocked, "stop_run")) state = k.stateSentence(l, stateSentence(r.stop_blocked, "stop_run"));
  }
  const current = selection.run !== "" && selection.run === n;
  const head = "p04-run-h-" + (n || "x");
  const chunks = r.chunks === null || r.next_chunk === null ? k.unknown(l) : counted(l, r.chunks, plurals.chunks, { next: String(r.next_chunk) });
  return (
    <article className={"p04-run" + (current ? " live-selected" : "")} id={n ? rowId("run", n) : undefined} aria-labelledby={head} aria-current={current || undefined}>
      <div className="p04-head">
        <h3 id={head}>{label ? ltr(label) : k.unknown(l)}</h3>
        <span className="p04-tag"><Code l={l} value={r.state} say={k.runState} /></span>
      </div>
      <StateWords l={l} state={r.state} />
      <dl className="p04-facts p04-facts-4">
        <div><dt>{k.factClass(l)}</dt><dd><Code l={l} value={r.class} say={k.className} /></dd></div>
        <div><dt>{k.factPolicyRevision(l)}</dt><dd><Num l={l} n={r.revision} /></dd></div>
        <div><dt>{k.factDecision(l)}</dt><dd>{r.decision === "not_connected" ? k.decisionNotConnected(l) : <Code l={l} value={r.decision} say={k.connection} />}</dd></div>
        <div><dt>{k.factChunks(l)}</dt><dd>{chunks}</dd></div>
        <div><dt>{k.factChunkSize(l)}</dt><dd><Num l={l} n={r.chunk_size} /></dd></div>
        <div><dt>{k.factLease(l)}</dt><dd>{r.lease_seconds === null ? k.unknown(l) : k.seconds(l, r.lease_seconds)}</dd></div>
        <div><dt>{k.factDamaged(l)}</dt><dd><Num l={l} n={r.damaged} /></dd></div>
        <div><dt>{k.factProposedBy(l)}</dt><dd><Who l={l} p={r.proposer} /></dd></div>
        <div><dt>{k.factApprovedBy(l)}</dt><dd><Who l={l} p={r.approver} /></dd></div>
        <div><dt>{k.factStoppedBy(l)}</dt><dd>{r.stopper_known ? <Who l={l} p={r.stopper} empty={k.notStopped(l)} /> : k.unknown(l)}</dd></div>
        <div className="p04-span"><dt>{k.factPlanReason(l)}</dt><dd><Free l={l} value={r.reason} /></dd></div>
      </dl>
      {r.stopper && <p className="p04-small"><Rich text={k.stoppedBy(l, r.stopper.login || r.stopper.id)} /></p>}
      <h4>{k.outcomeCounts(l)}</h4>
      <Strip l={l} counts={r.counts} keys={runOutcomeKeys} label={k.runOutcome} cls="p04-strip-9" />
      <div className="p04-actions">
        <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} state={state} />
        {usable && !open && (
          <div className="live-buttons">
            <button type="button" className="live-secondary p04-opener" aria-expanded={false} data-opener={mark("stop")} onClick={() => toggle("stop")}>{k.stopRun(l)}</button>
          </div>
        )}
        {open && usable && (
          <CommandForm
            {...shared} command="retention.stop_run" target={target} values={{}}
            heading={k.stopHeading(l, label)} title={k.confirmStop(l)} verb={k.stopVerb(l, label)}
            context={() => ({ view: "deletion", run: label, class: r.class, state: r.state, pending: r.counts.pending === null ? "" : String(r.counts.pending) })}
            facts={(ctx) => <RunFacts l={l} ctx={ctx} />}
            impact={() => <p><strong>{k.stopStrong(l)}</strong> {k.stopBody(l)}</p>}
            done={(fields, st) => {
              const stopped = /^(0|[1-9][0-9]*)$/.test(fields.stopped ?? "") ? Number(fields.stopped) : null;
              const run = fields.run || label;
              if (!st) return k.resultUnknown(l, st);
              return (got) => {
                const row = got && n ? runOf(got, n) : undefined;
                return k.stopDone(l, run, stopped, k.runState(l, row ? row.state : st));
              };
            }}
            onClose={close}
          />
        )}
      </div>
      <Stores l={l} r={r} />
      <Items l={l} r={r} page={page} />
    </article>
  );
}

export function DeletionView(shared: Shared) {
  const { l, page } = shared;
  const d = page.deletion;
  const facts = d.rows === null ? null : d.finished;
  const older = useOlderRows<RunRow>(page.revision, facts, d.rows, shared.api.retentionRuns, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  }, false);
  const rows = facts === null ? d.rows : older.rows;
  const finished = (rows ?? []).filter((r) => r.state !== "running");
  const running = (rows ?? []).filter((r) => r.state === "running");
  const cut = facts !== null && older.counted > (d.rows ?? []).filter((r) => r.state !== "running").length;
  const failed = older.status === "failed";
  return (
    <>
      <section className="p04-unconnected" aria-labelledby="p04-unconnected-title">
        <h3 id="p04-unconnected-title">{k.unconnectedTitle(l)}</h3>
        <dl className="p04-facts p04-facts-4">
          <div><dt>{k.worker(l)}</dt><dd><Code l={l} value={d.worker} say={k.connection} /></dd></div>
          <div><dt>{k.retry(l)}</dt><dd><Code l={l} value={d.retry} say={k.connection} /></dd></div>
          <div><dt>{k.backupExpiry(l)}</dt><dd><Code l={l} value={d.backup_expiry} say={k.connection} /></dd></div>
          <div><dt>{k.planProposals(l)}</dt><dd><Code l={l} value={d.propose_plan} say={k.connection} /></dd></div>
        </dl>
      </section>
      {facts === null && <MoreLine l={l} more={d.more} total={d.total} forms={plurals.showingNewest} />}
      {rows === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : rows.length === 0 && !older.more ? (
        <p className="live-empty">{d.total === 0 ? k.noRuns(l) : k.unknown(l)}</p>
      ) : facts === null ? (
        rows.map((r, n) => <Run key={r.run?.target || n} r={r} {...shared} />)
      ) : (
        <>
          <section className="p04-held-group p04-runs-running" aria-labelledby="p04-runs-running">
            <h4 id="p04-runs-running">{k.runsRunning(l, running.length)}</h4>
            {running.length === 0 ? <p className="live-empty">{k.runsNoneRunning(l)}</p> : running.map((r, n) => <Run key={r.run?.target || "u" + n} r={r} {...shared} />)}
          </section>
          <section className="p04-held-group p04-runs-finished" aria-labelledby="p04-runs-finished">
            <h4 id="p04-runs-finished">{k.runsFinished(l, finished.length)}</h4>
            {cut && <p className="p04-small live-quiet p04-runs-count" role="status" ref={older.countRef} tabIndex={-1}>{!older.more && finished.length >= older.counted ? k.runsAll(l, older.counted) : k.runsShown(l, finished.length, older.counted)}</p>}
            {older.reset && <p className="p04-small p04-runs-reset" role="status">{k.runsReset(l)}</p>}
            {finished.length === 0 && !older.more ? <p className="live-empty">{k.runsNoneFinished(l)}</p> : finished.map((r, n) => <Run key={r.run?.target || "d" + n} r={r} {...shared} />)}
            {older.more && (
              <div className="p04-held-more p04-runs-more">
                {failed && <p role="alert">{k.runsFailed(l)}</p>}
                <button type="button" className="live-secondary p04-held-more-button p04-runs-more-button" aria-disabled={older.status === "loading" || undefined} onClick={older.load}>
                  {failed ? k.heldTryAgain(l) : older.status === "loading" ? k.heldLoading(l) : k.runsMore(l)}
                </button>
              </div>
            )}
          </section>
        </>
      )}
      <Proposals kind="plan" {...shared} />
    </>
  );
}
