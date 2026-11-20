import { useEffect, useState, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { Proposal, RetentionPage } from "../../../../service/retention";
import { counted, plurals, summaryWords, words } from "./live-codes";
import { k } from "./live-copy";
import {
  actor, approveNarrow, approveValuesFor, blockerPolicy, clearReviewed, currentContext, currentFrom, currentOf, drop, frozenTargetOk, frozenValues, holdOf, impactValid, impactValues, keepReviewed,
  keptKey, ordered, pendingNow, policyOf, proposalTarget, proposalsOfKind, refusedNow, reviewedChanged, reviewedOf, stateSentence, type Current,
} from "./live-model";
import { Code, Free, HoldFacts, Rich, Stamp, Template, Who, fsi, holdContext, href, ltr, stamp } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";

const num = (l: Locale, n: number | null): ReactNode => (n === null ? k.unknown(l) : <bdi dir="ltr">{n}</bdi>);

export function SummaryText({ l, p }: { l: Locale; p: Proposal }) {
  if (p.kind === "policy") {
    const s = p.policy;
    if (!s) return <>{k.unknown(l)}</>;
    const total = s.period_days !== null && s.grace_days !== null ? s.period_days + s.grace_days : null;
    return <Template text={words(l, summaryWords, "policy")} slots={{ class: <Code l={l} value={s.class} say={k.className} />, p: num(l, s.period_days), g: num(l, s.grace_days), t: num(l, total) }} />;
  }
  if (p.kind === "plan") {
    const s = p.plan;
    if (!s) return <>{k.unknown(l)}</>;
    return (
      <Template
        text={words(l, summaryWords, "plan")}
        slots={{ n: num(l, s.artifact_count), class: <Code l={l} value={s.class} say={k.className} />, c: num(l, s.chunk_size), s: num(l, s.lease_seconds), reason: <Free l={l} value={s.reason} /> }}
      />
    );
  }
  if (p.kind === "release") {
    const s = p.release;
    if (!s) return <>{k.unknown(l)}</>;
    return <Template text={words(l, summaryWords, "release")} slots={{ hold: s.hold ? ltr(s.hold) : k.unknown(l), reason: <Free l={l} value={s.reason} /> }} />;
  }
  return <Code l={l} value={p.kind} say={k.proposalKind} />;
}

export function CurrentText({ l, c }: { l: Locale; c: Current }) {
  if (c.kind === "none") return <>{k.noPolicyShort(l)}</>;
  if (c.kind === "unknown") return <>{k.unknown(l)}</>;
  return <>{ltr(`${c.period} + ${c.grace} = ${c.period + c.grace}`)} {k.days(l)}</>;
}

function ImpactNowText({ l, p }: { l: Locale; p: Proposal }) {
  if (p.state !== "pending" || p.impact_required !== true) return null;
  const i = p.impact_now;
  if (!impactValid(i)) return <p className="p04-small">{k.narrow(l, "impact_unreadable")}</p>;
  return (
    <>
      <p className="p04-impact-now"><Rich text={counted(l, i.count, plurals.impactNow, { w: String(i.workspace_count), at: fsi(stamp(l, i.as_of)) })} /></p>
      <dl className="p04-digest"><div><dt>{k.impactDigest(l)}</dt><dd>{ltr(i.digest, true)}</dd></div></dl>
    </>
  );
}

function AtProposal({ l, p }: { l: Locale; p: Proposal }) {
  const a = p.policy?.impact_at_proposal;
  if (!a) return null;
  return <p className="p04-small">{a.count === null ? k.unknown(l) : counted(l, a.count, plurals.impactAtProposal, { at: fsi(stamp(l, a.as_of)) })}</p>;
}

function impactContext(p: Proposal): Record<string, string> {
  const i = p.impact_now;
  if (p.impact_required !== true || !impactValid(i)) return { shortening: p.impact_required === true ? "unknown" : "no" };
  return { shortening: "yes", as_of: i.as_of, count: String(i.count), workspace_count: String(i.workspace_count), digest: i.digest };
}

function ApproveFacts({ l, p, ctx }: { l: Locale; p: Proposal; ctx: Record<string, string> }) {
  return (
    <>
      <div><dt>{k.factProposal(l)}</dt><dd className="p04-num">{ctx.proposal ? ltr("#" + ctx.proposal) : k.unknown(l)}</dd></div>
      <div><dt>{k.factProposedBy(l)}</dt><dd><Who l={l} p={p.proposer} /></dd></div>
      <div className="p04-span"><dt>{k.factSummary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
      <div className="p04-span"><dt>{k.factProposerReason(l)}</dt><dd><Free l={l} value={p.rationale} empty={k.noReason(l)} /></dd></div>
      {p.kind === "policy" && (
        <>
          <div className="p04-span"><dt>{k.factCurrentPolicy(l)}</dt><dd><CurrentText l={l} c={currentFrom(ctx)} /></dd></div>
          {ctx.shortening === "yes" && (
            <>
              <div><dt>{k.factBecomeDue(l)}</dt><dd>{ltr(ctx.count)}</dd></div>
              <div><dt>{k.factInWorkspace(l)}</dt><dd>{ltr(ctx.workspace_count)}</dd></div>
              <div><dt>{k.factComputed(l)}</dt><dd><Stamp l={l} at={ctx.as_of} /></dd></div>
              <div className="p04-span"><dt>{k.impactDigest(l)}</dt><dd>{ltr(ctx.digest, true)}</dd></div>
            </>
          )}
        </>
      )}
      {p.kind === "release" && <HoldFacts l={l} ctx={ctx} />}
    </>
  );
}

function ApproveImpact({ l, p, ctx }: { l: Locale; p: Proposal; ctx: Record<string, string> }) {
  if (p.kind === "release") return <p><Rich text={k.releaseApproveImpact(l, ctx.hold || p.release?.hold || "")} /></p>;
  if (ctx.shortening === "yes") {
    return (
      <p>
        <strong>{k.reviewStrong(l)}</strong> <Rich text={counted(l, Number(ctx.count), plurals.impactReview, { w: ctx.workspace_count, at: fsi(stamp(l, ctx.as_of)) })} />
      </p>
    );
  }
  return <p>{k.notShortening(l)}</p>;
}

type Decision = "retention.approve" | "retention.withdraw";

function Row({ p, ...shared }: Shared & { p: Proposal }) {
  const { l, page, selection, canCommand } = shared;
  const target = proposalTarget(p);
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<Decision>(target);
  const pendingRow = p.state === "pending";
  const plan = p.kind === "plan";
  const n = p.number === null ? k.unknown(l) : String(p.number);
  const narrow = approveNarrow(p);
  const approvePending = target ? pendingNow("retention.approve", target) : undefined;
  const withdrawPending = target ? pendingNow("retention.withdraw", target) : undefined;
  const approveUsable = !plan && canCommand && !!target && pendingRow && ((p.can_approve && narrow === "") || (!!approvePending && frozenTargetOk("retention.approve", target)));
  const withdrawUsable = !plan && canCommand && !!target && pendingRow && (p.can_withdraw || (!!withdrawPending && frozenTargetOk("retention.withdraw", target)));
  const stale = !pendingRow && !!target && (!!approvePending || !!withdrawPending);
  const landed = (!!approvePending && p.state === "applied") || (!!withdrawPending && p.state === "withdrawn");
  const [applied, setApplied] = useState(false);
  useEffect(() => {
    if (!stale) return;
    drop("retention.approve", target);
    drop("retention.withdraw", target);
    if (landed) setApplied(true);
  }, [stale, target, landed]);
  const refusedApprove = target ? refusedNow("retention.approve", target) : "";
  const refusedWithdraw = target ? refusedNow("retention.withdraw", target) : "";
  const kept = refusedApprove ? k.kept(l, keptKey("retention.approve", refusedApprove)) : refusedWithdraw ? k.kept(l, keptKey("retention.withdraw", refusedWithdraw)) : "";
  const blockers: Blocker[] = [];
  let state = "";
  if (!plan && pendingRow && !target) blockers.push(line("", k.narrow(l, "no_command")));
  if (!plan && pendingRow && target && !approveUsable) {
    if (!p.can_approve && blockerPolicy(p.approve_blocked, "approve", { pendingRow: true }) === "explain") {
      blockers.push(line(k.approve(l), p.approve_blocked === "placer" ? k.placer(l, "approve") : k.blocker(l, p.approve_blocked)));
    } else if (p.can_approve && canCommand && narrow) blockers.push(line(k.approve(l), k.narrow(l, narrow)));
  }
  if (!plan && pendingRow && target && !withdrawUsable && !p.can_withdraw) {
    if (blockerPolicy(p.withdraw_blocked, "withdraw", { pendingRow: true }) === "explain") blockers.push(line(k.withdraw(l), k.blocker(l, p.withdraw_blocked)));
    else if (stateSentence(p.withdraw_blocked, "withdraw")) state = k.stateSentence(l, stateSentence(p.withdraw_blocked, "withdraw"));
  }
  const current = selection.proposal !== "" && selection.proposal === String(p.number);
  const live = (): Record<string, string> => {
    const base: Record<string, string> = { view: p.kind === "release" ? "holds" : "policies", proposal: p.number === null ? "" : String(p.number) };
    if (p.kind === "policy") return { ...base, ...currentContext(currentOf(policyOf(page, p.policy?.class ?? ""))), ...impactContext(p) };
    if (p.kind === "release") return { ...base, ...holdContext(l, holdOf(page, p.release?.hold ?? "")) };
    return base;
  };
  const approveValues = approveValuesFor(p, approvePending);
  const reviewed = target ? reviewedOf(target) : null;
  const liveImpact = impactContext(p);
  const changed = p.kind === "policy" && !approvePending && liveImpact.shortening === "yes" && reviewedChanged(reviewed, liveImpact.count, liveImpact.digest);
  const closedLine = p.state === "applied"
    ? k.approvedBy(l, actor(p.approver) || actor(p.closed_by) || k.unknown(l))
    : p.state === "withdrawn"
      ? k.withdrawnBy(l, actor(p.closed_by) || k.unknown(l))
      : "";
  const heldHold = p.release?.hold ?? "";
  const releasedAfter = (got: RetentionPage | null) => (got && holdOf(got, heldHold)?.released === true ? heldHold : "");

  return (
    <li className={"p04-proposal" + (current ? " live-selected" : "")} data-note={target || undefined} id={p.number === null ? undefined : "p04-proposal-" + p.number} aria-current={current || undefined}>
      <div className="p04-meta">
        <p className="p04-num p04-strong">{k.proposalN(l, n)}</p>
        <p><span className="p04-tag"><Code l={l} value={p.state} say={k.proposalState} /></span></p>
        <p className="p04-small"><Rich text={k.proposedBy(l, actor(p.proposer) || k.unknown(l))} /></p>
        <p className="p04-small live-quiet">{k.created(l, stamp(l, p.created_at))}</p>
      </div>
      <div className="p04-body">
        <p className="p04-summary"><SummaryText l={l} p={p} /></p>
        <AtProposal l={l} p={p} />
        <ImpactNowText l={l} p={p} />
        {pendingRow && <p className="p04-small">{k.reasonLabel(l)}: <Free l={l} value={p.rationale} empty={k.noReason(l)} /></p>}
        {pendingRow && p.kind === "release" && p.release?.hold && holdOf(page, p.release.hold) && (
          <p className="p04-small"><a className="p04-link" href={href(l, "holds", { hold: p.release.hold })}>{ltr(p.release.hold)}</a></p>
        )}
        {!pendingRow && (
          <RowNote l={l} diagnostics={diagnostics(target)}
            kept="" blockers={[]} focus={focusNote}
            extra={
              <>
                {applied && <p className="live-note-rule" role="status">{k.mayApplied(l)}</p>}
                {p.state === "retired" ? (
                  <p className="p04-small">{k.retiredRow(l)}</p>
                ) : (
                  <>
                    {closedLine && <p className="p04-small"><Rich text={closedLine} /></p>}
                    {p.closing_rationale && (
                      <dl className="live-facts p04-closed">
                        <div><dt>{k.closingReason(l)}</dt><dd><Free l={l} value={p.closing_rationale} /></dd></div>
                      </dl>
                    )}
                  </>
                )}
              </>
            }
          />
        )}
        {pendingRow && !plan && (
          <>
            <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} state={state} />
            {(approveUsable || withdrawUsable) && !open && (
              <div className="live-buttons">
                {approveUsable && (
                  <button type="button" className="live-primary" aria-expanded={false} data-opener={mark("retention.approve")} onClick={() => toggle("retention.approve")}>{k.approve(l)}</button>
                )}
                {withdrawUsable && (
                  <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("retention.withdraw")} onClick={() => toggle("retention.withdraw")}>{k.withdraw(l)}</button>
                )}
              </div>
            )}
            {open === "retention.approve" && target && approveUsable && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={approveValues}
                heading={k.approveHeading(l, n)} title={k.confirmApprove(l)} verb={k.approveVerb(l, n)}
                context={live}
                warn={changed && reviewed ? <p className="p04-warn">{k.impactChanged(l, `${reviewed.count} → ${liveImpact.count}`)}</p> : undefined}
                facts={(ctx) => <ApproveFacts l={l} p={p} ctx={ctx} />}
                impact={(ctx) => <ApproveImpact l={l} p={p} ctx={ctx} />}
                hooks={{
                  review: () => { if (liveImpact.shortening === "yes" && !reviewed) keepReviewed(target, liveImpact.count, liveImpact.digest); },
                  send: (v) => { if (v.impact_count !== undefined && v.impact_digest !== undefined) keepReviewed(target, v.impact_count, v.impact_digest); },
                  cancel: () => clearReviewed(target),
                  settled: (r) => { if (r !== "conflict" && r !== "hint" && r !== "retry") clearReviewed(target); },
                }}
                done={(_, st) =>
                  st !== "applied" ? k.resultUnknown(l, st) : p.kind === "release" ? (got) => k.releaseApplied(l, n, releasedAfter(got)) : k.policyApplied(l, n)
                }
                onClose={close}
              />
            )}
            {open === "retention.withdraw" && target && withdrawUsable && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={{}}
                heading={k.withdrawHeading(l, n)} title={k.confirmWithdraw(l)} verb={k.withdrawVerb(l, n)}
                context={live}
                facts={(ctx) => (
                  <>
                    <div><dt>{k.factProposal(l)}</dt><dd className="p04-num">{ctx.proposal ? ltr("#" + ctx.proposal) : k.unknown(l)}</dd></div>
                    <div className="p04-span"><dt>{k.factSummary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
                  </>
                )}
                impact={() => <p>{k.withdrawImpact(l)}</p>}
                done={(_, st) => (st === "withdrawn" ? k.withdrawnDone(l, n) : k.resultUnknown(l, st))}
                onClose={close}
              />
            )}
          </>
        )}
      </div>
    </li>
  );
}

export function Proposals({ kind, ...shared }: Shared & { kind: "policy" | "release" | "plan" }) {
  const { l, page } = shared;
  const list = proposalsOfKind(page, kind);
  const { pending, closed } = ordered(list ?? []);
  const title = kind === "policy" ? k.policyProposalsTitle(l) : kind === "release" ? k.releaseProposalsTitle(l) : k.planProposalsTitle(l);
  const head = "p04-proposals-" + kind;
  return (
    <section className="p04-section p04-proposals" aria-labelledby={head}>
      <h3 id={head}>{title}</h3>
      {kind === "plan" && <p className="p04-small">{k.planReadOnly(l)}</p>}
      {page.proposals.more === true && <p className="p04-small live-quiet">{k.newestProposals(l)}</p>}
      {page.proposals.more === null && <p className="p04-small live-quiet">{k.maybeMore(l)}</p>}
      {list === null && <p className="live-empty">{k.unknown(l)}</p>}
      {list !== null && list.length === 0 && <p className="live-empty">{k.noProposals(l, kind)}</p>}
      {pending.length > 0 && (
        <>
          <h4>{k.waitingGroup(l, pending.length)}</h4>
          <ul className="p04-list">{pending.map((p, i) => <Row key={p.target || "p" + i} p={p} {...shared} />)}</ul>
        </>
      )}
      {closed.length > 0 && (
        <>
          <h4>{k.closedGroup(l, closed.length)}</h4>
          <ul className="p04-list">{closed.map((p, i) => <Row key={p.target || "c" + i} p={p} {...shared} />)}</ul>
          <p className="p04-small live-quiet">{k.appliedNote(l)}</p>
        </>
      )}
    </section>
  );
}
