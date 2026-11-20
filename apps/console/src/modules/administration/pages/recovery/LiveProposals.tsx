import { useEffect, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { RecoveryProposal } from "../../../../service/recovery";
import { summaryWords, words } from "./live-codes";
import { k } from "./live-copy";
import {
  actor, approveNarrow, basisContext, basisFrom, basisOf, blockerPolicy, drop, frozenTargetOk, isolationOf, ordered, pendingNow, proposalInstallation, proposalTarget, refusedNow, resumeOf,
  shownLimit, summaryOf, type Basis,
} from "./live-model";
import { Code, InstallNow, ModeImpact, Prose, Template, fsi, href, ltr, stamp } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";

export function SummaryText({ l, p }: { l: Locale; p: RecoveryProposal }) {
  const s = summaryOf(p);
  if (!s.known) return <Code l={l} value={p.kind} say={k.proposalKind} />;
  const slots: Record<string, ReactNode> = {};
  for (const [name, part] of Object.entries(s.parts)) {
    if ("id" in part) slots[name] = part.id ? ltr(part.id, name.endsWith("digest")) : k.unknown(l);
    else if ("num" in part) slots[name] = part.num === null ? k.unknown(l) : <bdi dir="ltr">{part.num}</bdi>;
    else if ("date" in part) slots[name] = fsi(stamp(l, part.date));
    else if ("skew" in part) slots[name] = part.skew === null ? k.unknown(l) : k.skew(l, part.skew);
    else slots[name] = part.writerKind ? <Code l={l} value={part.writerKind} say={k.writerKind} /> : k.unknown(l);
  }
  return <Template text={words(l, summaryWords, s.kind)} slots={slots} />;
}

function numberText(l: Locale, p: RecoveryProposal): string {
  return p.number === null ? k.unknown(l) : String(p.number);
}

function Facts({ l, p, b, approve }: { l: Locale; p: RecoveryProposal; b: Basis; approve: boolean }) {
  return (
    <>
      <div><dt>{k.factProposal(l)}</dt><dd className="p03-num">{numberText(l, p)}</dd></div>
      <div><dt>{k.factKind(l)}</dt><dd><Code l={l} value={p.kind} say={k.proposalKind} /></dd></div>
      <div className="p03-span"><dt>{k.factSummary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
      {approve && (
        <>
          <div><dt>{k.factProposedBy(l)}</dt><dd>{actor(p.proposer_login, p.proposer) ? <bdi dir="ltr">{actor(p.proposer_login, p.proposer)}</bdi> : k.unknown(l)}</dd></div>
          <div><dt>{k.factProposerReason(l)}</dt><dd><Prose l={l} value={p.rationale} empty={k.noReason(l)} /></dd></div>
          <div className="p03-span"><dt>{k.factInstallState(l)}</dt><dd><InstallNow l={l} b={b} /></dd></div>
        </>
      )}
    </>
  );
}

type Decision = "recovery.approve" | "recovery.withdraw";

function Row({ p, ...shared }: Shared & { p: RecoveryProposal }) {
  const { l, page, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<Decision>();
  const target = proposalTarget(p);
  const pendingRow = p.state === "pending";
  const n = numberText(l, p);
  const id = proposalInstallation(p);
  const narrow = approveNarrow(page, p);
  const approvePending = target ? pendingNow("recovery.approve", target) : undefined;
  const withdrawPending = target ? pendingNow("recovery.withdraw", target) : undefined;
  const approveUsable = canCommand && !!target && pendingRow && ((p.can_approve && narrow === "") || (!!approvePending && frozenTargetOk("recovery.approve", target)));
  const withdrawUsable = canCommand && !!target && pendingRow && (p.can_withdraw || (!!withdrawPending && frozenTargetOk("recovery.withdraw", target)));
  const stale = !pendingRow && !!target && (!!approvePending || !!withdrawPending);
  const landed = (!!approvePending && p.state === "applied") || (!!withdrawPending && p.state === "withdrawn");
  const [applied, setApplied] = useState(false);
  useEffect(() => {
    if (!stale) return;
    drop("recovery.approve", target);
    drop("recovery.withdraw", target);
    if (landed) setApplied(true);
  }, [stale, target, landed]);
  const kept = !target ? "" : refusedNow("recovery.approve", target) ? k.kept(l, "recovery.approve") : refusedNow("recovery.withdraw", target) ? k.kept(l, "recovery.withdraw") : "";
  const blockers: Blocker[] = [];
  if (pendingRow && !target) blockers.push(line("", k.narrow(l, "no_command")));
  if (pendingRow && target && !approveUsable) {
    if (!p.can_approve && blockerPolicy(p.approve_blocked, "approve", { pendingRow: true }) === "explain") blockers.push(line(k.approve(l), k.blocker(l, p.approve_blocked)));
    else if (p.can_approve && canCommand && narrow) blockers.push(line(k.approve(l), k.narrow(l, narrow)));
  }
  if (pendingRow && target && !withdrawUsable && !p.can_withdraw && blockerPolicy(p.withdraw_blocked, "withdraw", { pendingRow: true }) === "explain") {
    blockers.push(line(k.withdraw(l), k.blocker(l, p.withdraw_blocked)));
  }
  const current = selection.proposal !== "" && selection.proposal === String(p.number);
  const live = basisOf(page, id);
  const b = open === "recovery.approve" && approvePending ? basisFrom(approvePending.context) : open === "recovery.withdraw" && withdrawPending ? basisFrom(withdrawPending.context) : live;
  const closedLine = p.state === "applied"
    ? k.approvedBy(l, actor(p.approver_login || p.closed_by_login, p.approver || p.closed_by) || k.unknown(l))
    : p.state === "withdrawn"
      ? k.withdrawnBy(l, actor(p.closed_by_login, p.closed_by) || k.unknown(l))
      : "";
  const servingAfter = (got: typeof page | null) => (got && isolationOf(got, id)?.state === "serving" ? id : "");

  return (
    <li className={"p03-proposal" + (current ? " live-selected" : "")} id={p.number === null ? undefined : "p03-proposal-" + p.number} aria-current={current || undefined}>
      <div className="p03-meta">
        <p className="p03-num p03-strong">{k.proposalN(l, n)}</p>
        <p className="p03-muted"><Code l={l} value={p.kind} say={k.proposalKind} /></p>
        <p><span className="p03-tag"><Code l={l} value={p.state} say={k.proposalState} /></span></p>
        <p className="p03-small">{k.proposedBy(l, actor(p.proposer_login, p.proposer) || k.unknown(l))}</p>
        <p className="p03-small live-quiet">{k.created(l, stamp(l, p.created_at))}</p>
      </div>
      <div className="p03-body">
        <p className="p03-summary"><SummaryText l={l} p={p} /></p>
        {pendingRow && <p className="p03-small">{k.reason(l)}: <Prose l={l} value={p.rationale} empty={k.noReason(l)} /></p>}
        {pendingRow && id && resumeOf(page, id) && (
          <p className="p03-small">{k.installLink(l)} <a className="p03-link" href={href(l, "resume", { installation: id })}>{ltr(id)}</a></p>
        )}
        {!pendingRow && (
          <>
            {applied && <p className="live-note-rule" role="status">{k.mayApplied(l)}</p>}
            {p.state === "retired" ? (
              <p className="p03-small">{k.retiredRow(l)}</p>
            ) : (
              <>
                {closedLine && <p className="p03-small">{closedLine}</p>}
                {p.closing_rationale && (
                  <dl className="live-facts p03-closed">
                    <div><dt>{k.closingReason(l)}</dt><dd><Prose l={l} value={p.closing_rationale} /></dd></div>
                  </dl>
                )}
              </>
            )}
          </>
        )}
        {pendingRow && (
          <>
            <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} />
            {(approveUsable || withdrawUsable) && !open && (
              <div className="live-buttons">
                {approveUsable && (
                  <button type="button" className="live-primary" aria-expanded={false} data-opener={mark("recovery.approve")} onClick={() => toggle("recovery.approve")}>{k.approve(l)}</button>
                )}
                {withdrawUsable && (
                  <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("recovery.withdraw")} onClick={() => toggle("recovery.withdraw")}>{k.withdraw(l)}</button>
                )}
              </div>
            )}
            {open && target && (open === "recovery.approve" ? approveUsable : withdrawUsable) && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={{}}
                title={open === "recovery.approve" ? k.confirmApprove(l) : k.confirmWithdraw(l)}
                facts={<Facts l={l} p={p} b={b} approve={open === "recovery.approve"} />}
                impact={open === "recovery.approve" ? <ModeImpact l={l} b={b} lead={false} /> : <p>{k.withdrawImpact(l)}</p>}
                verb={open === "recovery.approve" ? k.approveVerb(l, n) : k.withdrawVerb(l, n)}
                context={basisContext(b, { proposal: n })}
                done={(_, state) =>
                  open === "recovery.approve"
                    ? state === "applied" ? (got) => k.approvedDone(l, n, servingAfter(got)) : k.resultUnknown(l, state)
                    : state === "withdrawn" ? k.withdrawnDone(l, n) : k.resultUnknown(l, state)
                }
                onClose={close}
              />
            )}
          </>
        )}
      </div>
    </li>
  );
}

export function Proposals(shared: Shared) {
  const { l, page } = shared;
  const list = page.proposals;
  const { pending, closed } = ordered(list ?? []);
  return (
    <section className="p03-section p03-proposals" aria-labelledby="p03-proposals-title">
      <h3 id="p03-proposals-title">{k.proposalsTitle(l)}</h3>
      {list === null && <p className="live-empty">{k.unknown(l)}</p>}
      {list !== null && list.length === 0 && <p className="live-empty">{k.noProposals(l)}</p>}
      {list !== null && list.length >= shownLimit && <p className="p03-small live-quiet">{k.newest(l)}</p>}
      {pending.length > 0 && (
        <>
          <h4>{k.waitingGroup(l, pending.length)}</h4>
          <ul className="p03-list">{pending.map((p, i) => <Row key={p.target || "p" + i} p={p} {...shared} />)}</ul>
        </>
      )}
      {closed.length > 0 && (
        <>
          <h4>{k.closedGroup(l, closed.length)}</h4>
          <ul className="p03-list">{closed.map((p, i) => <Row key={p.target || "c" + i} p={p} {...shared} />)}</ul>
          <p className="p03-small live-quiet">{k.appliedNote(l)}</p>
        </>
      )}
    </section>
  );
}
