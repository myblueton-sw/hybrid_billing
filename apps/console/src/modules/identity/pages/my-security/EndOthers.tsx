import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type SelfEnd, type SelfEndRow, type SelfPage, type SelfSession } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { when } from "../authentication/live-model";
import { m } from "./live-copy";
import { datePart, e, stateSentence, timePart, whenSeconds } from "./end-copy";
import { goneCount, groupOf, groupOrder, heldShown, openByDefault, others, presentHandles, rowOutcome, submitOutcome, type ItemGroup } from "./live-model";
import "./end.css";

const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function DateTime({ l, iso }: { l: Locale; iso: string }) {
  return <><span className="mysec-nw">{datePart(l, iso)}</span> <span className="mysec-nw">{timePart(l, iso)}</span></>;
}

export type EndTarget = { kind: "others" } | { kind: "one"; handle: string };
export type EndResult =
  | { type: "done"; op: SelfEnd }
  | { type: "changed" }
  | { type: "gone" }
  | { type: "signedOut" }
  | { type: "cancel" }
  | { type: "reload" };

type Frozen = { generation: string; now: string; current: string; rows: SelfSession[] };

export function EndDialog({ l, api, page, target, onClose }: { l: Locale; api: AdminClient; page: SelfPage; target: EndTarget; onClose: (r: EndResult) => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const summary = useRef<HTMLDivElement>(null);
  const busyRef = useRef(false);
  const [busy, setBusy] = useState(false);
  const [frozen, setFrozen] = useState<Frozen | null>(null);
  const [reading, setReading] = useState<"reading" | "ready" | "failed">(target.kind === "others" ? "reading" : "ready");
  const [failure, setFailure] = useState("");
  const [unknown, setUnknown] = useState(false);
  const [refused, setRefused] = useState(false);
  const [focusKey, setFocusKey] = useState(0);
  const one = target.kind === "one" ? page.sessions.find((s) => s.handle === target.handle) : undefined;
  const read = () => {
    setReading("reading");
    setFailure("");
    api.me().then(
      (got) => {
        setFrozen({ generation: got.sessions_generation, now: got.now, current: got.sessions.find((s) => s.current)?.handle ?? "", rows: others(got) });
        setReading("ready");
      },
      (err) => {
        if (err instanceof AdminError && err.failure === "unauthenticated") { onClose({ type: "signedOut" }); return; }
        setReading("failed");
        setFocusKey((k) => k + 1);
      },
    );
  };
  useEffect(() => {
    const d = dialog.current;
    if (d && !d.open) d.showModal();
    heading.current?.focus();
    if (target.kind === "others") read();
  }, []);
  useEffect(() => {
    if (!focusKey) return;
    const timer = setTimeout(() => summary.current?.focus(), 120);
    return () => clearTimeout(timer);
  }, [focusKey]);
  const close = (r: EndResult) => {
    dialog.current?.close();
    onClose(r);
  };
  const go = async () => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    setFailure("");
    try {
      const op = target.kind === "others" ? await api.endOthers(frozen?.generation ?? "", (frozen?.rows ?? []).map((s) => s.handle)) : await api.endOne(target.handle);
      close({ type: "done", op });
    } catch (err) {
      const outcome = submitOutcome(err, target.kind);
      if (outcome === "signedOut") { close({ type: "signedOut" }); return; }
      if (outcome === "changed" || outcome === "gone") { close({ type: outcome }); return; }
      if (outcome === "limited") setFailure(m.endLimited(l, err instanceof AdminError ? err.retryAfter : 0));
      else if (outcome === "reauth") setFailure(c.reauthText(l, err instanceof AdminError ? err.failure : ""));
      else if (outcome === "refused") { setRefused(true); setFailure(c.reauthText(l, "refused")); }
      else { setUnknown(true); setFailure(e.unknown(l)); }
      setFocusKey((k) => k + 1);
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };
  const count = frozen?.rows.length ?? 0;
  const zero = target.kind === "others" && reading === "ready" && count === 0;
  const canSubmit = !unknown && !refused && !zero && (target.kind === "one" || (reading === "ready" && count > 0));
  return (
    <dialog ref={dialog} className="mysec-end" aria-labelledby="mysec-end-title" aria-describedby="mysec-end-lead"
      onCancel={(ev) => { ev.preventDefault(); if (!busyRef.current) close({ type: unknown ? "reload" : "cancel" }); }}>
      <h2 id="mysec-end-title" tabIndex={-1} ref={heading}>{target.kind === "one" ? m.endOneTitle(l) : e.othersTitle(l)}</h2>
      {reading === "reading" && <p className="live-meta" role="status">{e.reading(l)}</p>}
      {(failure || reading === "failed") && (
        <div className="service-summary" role="alert" tabIndex={-1} ref={summary}>
          <h2>{reading === "failed" ? m.loadFailed(l) : unknown ? e.titleUnknown(l) : e.titleNothing(l)}</h2>
          <p>{reading === "failed" ? m.loadRetry(l) : failure}</p>
        </div>
      )}
      {target.kind === "one" && one && (
        <>
          <dl className="live-facts">
            <div><dt>{m.reference(l)}</dt><dd><span className="live-mono">{ltr(one.handle)}</span></dd></div>
            <div><dt>{m.colSignedIn(l)}</dt><dd>{when(l, one.signed_in)}</dd></div>
            <div><dt>{m.colEnds(l)}</dt><dd>{when(l, one.ends)}</dd></div>
          </dl>
          <p className="mysec-end-lead" id="mysec-end-lead">{m.endOneImpact(l)}</p>
        </>
      )}
      {zero && <p className="live-empty" id="mysec-end-lead">{m.noOthers(l)}</p>}
      {target.kind === "others" && frozen && count > 0 && (
        <>
          <p className="mysec-end-lead" id="mysec-end-lead">{e.lead(l, count)}</p>
          <p className="mysec-end-scope">{e.scope(l)}</p>
          {frozen.current && <p className="live-sub">{e.thisRef(l)} <span className="live-mono">{ltr(frozen.current)}</span></p>}
          <div className="mysec-manifest-head">
            <h3>{e.manifestTitle(l, count)}</h3>
            <p className="live-meta">{e.listTime(l, whenSeconds(l, frozen.now))}</p>
          </div>
          <div className="live-table mysec-manifest">
            <table role="table">
              <caption className="visually-hidden">{e.manifestTitle(l, count)}</caption>
              <thead role="rowgroup"><tr role="row">
                <th scope="col" role="columnheader">{m.reference(l)}</th>
                <th scope="col" role="columnheader">{m.colSignedIn(l)}</th>
                <th scope="col" role="columnheader">{m.colEnds(l)}</th>
              </tr></thead>
              <tbody role="rowgroup">
                {frozen.rows.map((s) => (
                  <tr key={s.handle} role="row">
                    <td role="cell"><span className="live-mono">{ltr(s.handle)}</span></td>
                    <td role="cell" data-label={m.colSignedIn(l)}><DateTime l={l} iso={s.signed_in} /></td>
                    <td role="cell" data-label={m.colEnds(l)}><DateTime l={l} iso={s.ends} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mysec-end-note">{e.note(l)}</p>
        </>
      )}
      <div className="live-buttons">
        {reading === "failed" && <button type="button" className="live-secondary" onClick={read}>{c.tryAgain(l)}</button>}
        {canSubmit && (
          <button type="button" className="live-primary" aria-busy={busy || undefined} disabled={busy} onClick={go}>
            {busy ? e.busy(l) : target.kind === "one" ? m.endOneButton(l) : e.button(l, count)}
          </button>
        )}
        {!busy && (unknown
          ? <button type="button" className="live-secondary" onClick={() => close({ type: "reload" })}>{e.reloadClose(l)}</button>
          : <button type="button" className="live-secondary" onClick={() => close({ type: "cancel" })}>{zero ? e.close(l) : w.cancel(l)}</button>)}
      </div>
    </dialog>
  );
}

function Items({ l, row }: { l: Locale; row: SelfEndRow }) {
  const title: Record<ItemGroup, string> = { done: e.groupDone(l), pending: e.groupPending(l), failed: e.groupFailed(l), not_applicable: e.groupNotApplicable(l), outside_scope: e.groupOutside(l) };
  return (
    <>
      {groupOrder.map((g) => {
        const items = row.items.filter((i) => groupOf(i) === g);
        if (!items.length) return null;
        return (
          <div key={g} className="mysec-item-group" data-group={g}>
            <h6>{title[g]}</h6>
            <ul>
              {items.map((i) => {
                const reason = i.state === "done" ? "" : e.reason(l, i.reason);
                return (
                  <li key={i.item} className={i.state === "failed" || i.item === "C7" ? "mysec-strong" : undefined} data-item={i.item} data-state={i.state}>
                    <span>{e.item(l, i.item)}</span>
                    {reason && <span className="mysec-reason">{reason}</span>}
                  </li>
                );
              })}
            </ul>
          </div>
        );
      })}
    </>
  );
}

export { stateSentence };

function StateSentence({ l, op }: { l: Locale; op: SelfEnd }) {
  if (op.count === 0) return <div className="mysec-op-state"><p>{e.completeNone(l)}</p></div>;
  if (op.state === "cleanup_running") return <div className="mysec-op-state"><p>{e.running(l)}</p></div>;
  if (op.state === "cleanup_incomplete") {
    return <div className="mysec-op-state"><p>{e.incomplete(l, op.count)}</p><p className="mysec-strong">{e.incompleteCount(l, op.cleanup.failed)}</p></div>;
  }
  return <div className="mysec-op-state"><p>{e.complete(l, op.count)}</p><p className="mysec-strong">{e.noRecall(l)}</p></div>;
}

export type Check = { at: string; mode: "polling" | "idle" | "stopped" | "failed" };

function Operation({ l, op, newest, page, open, onToggle, check, onEndHeld, onGoPassword, onCheckAgain }: {
  l: Locale; op: SelfEnd; newest: boolean; page: SelfPage; open: boolean; onToggle: (open: boolean) => void; check: Check;
  onEndHeld: (handle: string) => void; onGoPassword: () => void; onCheckAgain: () => void;
}) {
  const present = presentHandles(page);
  const gone = goneCount(op);
  const word = op.count === 0 ? e.nothingToClean(l) : op.state === "cleanup_running" ? e.wordRunning(l) : op.state === "complete" ? e.wordComplete(l) : e.wordIncomplete(l);
  const id = "mysec-op-" + op.operation;
  return (
    <li className="mysec-op" data-operation={op.operation} data-state={op.state}>
      <details className="mysec-op-wrap" open={open} onToggle={(ev) => onToggle((ev.target as HTMLDetailsElement).open)}>
        <summary className="mysec-op-summary">
          <h4 id={id} tabIndex={-1}>{op.kind === "one" ? e.opOne(l) : e.opOthers(l)}</h4>
          <span className="live-meta"> · {whenSeconds(l, op.blocked_at)} · {word}</span>
        </summary>
        <div className="mysec-op-body">
          <p className="live-sub"><span className="live-mono">{ltr(op.operation)}</span></p>
          {op.replay && <p className="live-sub">{e.replay(l)}</p>}
          <StateSentence l={l} op={op} />
          <dl className="mysec-op-lines">
            <div>
              <dt>{e.accessBlocked(l)}</dt>
              <dd>{op.count === 0 ? e.blockedNone(l) : <>{whenSeconds(l, op.blocked_at)} · {e.sessionsCount(l, op.count)}</>}</dd>
            </div>
            <div>
              <dt>{e.cleanup(l)}</dt>
              <dd>
                <span className={op.state === "cleanup_incomplete" && op.count > 0 ? "mysec-strong" : undefined}>{word}</span>
                {op.count > 0 && (
                  <>
                    <dl className="mysec-counts">
                      <div><dt>{e.countDone(l)}</dt><dd>{op.cleanup.done}</dd></div>
                      <div><dt>{e.countCleared(l)}</dt><dd>{op.cleanup.already_ended}</dd></div>
                      <div><dt>{e.wordRunning(l)}</dt><dd>{op.cleanup.pending}</dd></div>
                      <div><dt>{e.wordIncomplete(l)}</dt><dd>{op.cleanup.failed}</dd></div>
                    </dl>
                    <p className="mysec-equation">{e.equation(l, op.count, op.cleanup.done, op.cleanup.already_ended, op.cleanup.pending, op.cleanup.failed)}</p>
                  </>
                )}
              </dd>
            </div>
          </dl>
          {heldShown(op) && (
            <section className="mysec-held" aria-labelledby={id + "-held"}>
              <h5 id={id + "-held"}>{e.heldTitle(l, op.not_included.length)}</h5>
              <p><strong>{e.heldFirst(l)}</strong> {e.heldRest(l, op.not_included.length)}</p>
              <button type="button" className="service-button-link mysec-go-password" onClick={onGoPassword}>{e.goPassword(l)}</button>
              <ul className="mysec-held-list">
                {op.not_included.map((s) => (
                  <li key={s.handle}>
                    <span className="live-mono">{ltr(s.handle)}</span>
                    <span className="live-sub">{m.signedInAt(l, when(l, s.signed_in))}</span>
                    {present.has(s.handle)
                      ? <button type="button" className="live-secondary live-revoke" data-session={s.handle} onClick={() => onEndHeld(s.handle)}>{m.end(l)}<span className="visually-hidden"> {m.reference(l)} {s.handle}</span></button>
                      : <span className="mysec-no-longer">{e.noLonger(l)}</span>}
                  </li>
                ))}
              </ul>
            </section>
          )}
          {op.kind === "others" && op.generation_changed && op.not_included.length === 0 && <p className="live-sub">{e.changedOnly(l)}</p>}
          {op.kind === "others" && gone > 0 && <p className="mysec-gone">{e.gone(l, gone)}</p>}
          <section className="mysec-op-sessions" aria-labelledby={id + "-rows"}>
            <h5 id={id + "-rows"}>{e.inRequest(l, op.sessions.length)}</h5>
            <ol className="mysec-op-list">
              {op.sessions.map((row) => {
                const outcome = rowOutcome(row);
                const text = outcome === "gone" ? e.rowGone(l) : outcome === "complete" ? e.rowComplete(l) : outcome === "running" ? e.rowRunning(l) : e.rowFailed(l);
                return (
                  <li key={row.handle} className="mysec-op-row" data-result={row.result} data-outcome={outcome}>
                    <div>
                      <span className="live-mono">{ltr(row.handle)}</span>
                      {row.signed_in && <span className="live-sub">{m.signedInAt(l, when(l, row.signed_in))}</span>}
                    </div>
                    <div>
                      <p className={outcome === "failed" ? "mysec-strong" : undefined}>{text}</p>
                      {row.result === "ended" ? (
                        <details className="mysec-items">
                          <summary><span>{e.details(l)} {ltr(row.handle)}</span></summary>
                          <Items l={l} row={row} />
                        </details>
                      ) : <p className="mysec-op-none">{e.nothingSession(l)}</p>}
                    </div>
                  </li>
                );
              })}
            </ol>
          </section>
          {newest && check.at && (
            <div className="mysec-check">
              <p className="mysec-checked">
                {check.mode === "stopped" ? e.stillRunning(l, whenSeconds(l, check.at)) : check.mode === "failed" ? e.checkFailed(l, whenSeconds(l, check.at)) : e.checkedAt(l, whenSeconds(l, check.at))}
              </p>
              {(check.mode === "stopped" || check.mode === "failed") && <button type="button" className="live-secondary mysec-check-again" onClick={onCheckAgain}>{e.checkAgain(l)}</button>}
            </div>
          )}
        </div>
      </details>
    </li>
  );
}

export function EndsRegion({ l, page, opened, onToggle, check, onEndHeld, onGoPassword, onCheckAgain }: {
  l: Locale; page: SelfPage; opened: Record<string, boolean>; onToggle: (op: string, open: boolean) => void; check: Check;
  onEndHeld: (handle: string) => void; onGoPassword: () => void; onCheckAgain: () => void;
}) {
  if (page.session_ends.length === 0) return null;
  const present = presentHandles(page);
  return (
    <section className="mysec-ends" aria-labelledby="mysec-ends-title">
      <h3 id="mysec-ends-title">{e.endsTitle(l)}</h3>
      <p className="live-meta">{e.endsMeta(l)}</p>
      <ol className="mysec-ops">
        {page.session_ends.map((op, i) => (
          <Operation key={op.operation} l={l} op={op} newest={i === 0} page={page} open={opened[op.operation] ?? openByDefault(op, i === 0, present)}
            onToggle={(open) => onToggle(op.operation, open)} check={check} onEndHeld={onEndHeld} onGoPassword={onGoPassword} onCheckAgain={onCheckAgain} />
        ))}
      </ol>
    </section>
  );
}
