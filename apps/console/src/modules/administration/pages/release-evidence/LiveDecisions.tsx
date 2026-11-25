import { useEffect, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { EvidencePage, Proposal } from "../../../../service/release-evidence";
import { AdminError } from "../../../../service/admin";
import { useOlder } from "../../../../shared/KeptHistory";
import { blockerText, f, k } from "./live-copy";
import { counted, has, kindWords, moduleWords, plurals, proposalStateWords, summaryWords } from "./live-codes";
import {
  approveUsable, blockerPolicy, drop, frozenTargetOk, keptKey, ordered, pendingNow, profileById, proposalKey, proposalTarget, proposalUnexpected, refusedNow, requirementByLabel,
  requirementNumber, retiredAfter, rowId, stateAtCandidate, stateSentence, withdrawUsable,
} from "./live-model";
import { Absent, Code, Fixed, Free, State, Label, MoreLine, Stamp, Template, Unexpected, Who, href, ltr } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { RetireImpact, StateText, blockingUntested } from "./LiveRequirements";
import { ProfileFacts, ProfileImpact, profileContext } from "./LiveLoad";

type Decision = "release_evidence.approve" | "release_evidence.withdraw";

const sv = (v: unknown): string => (typeof v === "string" ? v : typeof v === "number" ? String(v) : "");

function Slot({ l, v, label = false }: { l: Locale; v: unknown; label?: boolean }) {
  const s = sv(v);
  if (!s) return <Absent>{k.unknown(l)}</Absent>;
  return label ? <Label value={s} /> : ltr(s);
}

export function SummaryText({ l, page, p, links = true }: { l: Locale; page: EvidencePage; p: Proposal; links?: boolean }) {
  const s = p.summary;
  const base = p.module + ":" + p.kind;
  let key = base;
  if (base === "release_trace:retire") key = s.replacement ? base + ":replaced" : base + ":alone";
  if (base === "release_trace:reconcile") key = s.matched === true ? base + ":matched" : s.matched === false ? base + ":unmatched" : "";
  if (!key || !has(summaryWords, key)) return <Code l={l} value={base} table={kindWords} say={k.kind} />;
  const label = sv(s.label);
  const req = links && label ? requirementByLabel(page, label) : undefined;
  const n = req ? requirementNumber(req.target) : "";
  const profile = sv(s.profile);
  const slots: Record<string, ReactNode> = {
    label: n ? <a className="p05-link" href={href(l, "requirements", { requirement: n })}><Label value={label} /></a> : <Slot l={l} v={s.label} label />,
    replacement: <Slot l={l} v={s.replacement} label />,
    reason: <Free l={l} value={sv(s.reason)} />,
    commit: <Slot l={l} v={s.commit} />,
    build: <Slot l={l} v={s.build} />,
    harness: <Slot l={l} v={s.harness} />,
    path: <Slot l={l} v={s.path} />,
    period: <Slot l={l} v={s.period} />,
    differences: typeof s.differences === "number" ? <>{counted(l, s.differences, plurals.differences)}</> : <Slot l={l} v={s.differences} />,
    profile: links && p.kind === "retire" && profile && profileById(page, profile) ? <a className="p05-link" href={href(l, "load", { profile })}>{ltr(profile)}</a> : <Slot l={l} v={s.profile} />,
    gate: <Slot l={l} v={s.gate} />,
    hardware: <Slot l={l} v={s.hardware} />,
    budget: <Slot l={l} v={s.budget} />,
  };
  return <Template text={k.summary(l, key)} slots={slots} />;
}

function approveContext(page: EvidencePage, p: Proposal): Record<string, string> {
  const base: Record<string, string> = { view: "decisions", proposal: p.number === null ? "" : String(p.number), module: p.module };
  if (p.module === "perf_gate") {
    const pr = profileById(page, sv(p.summary.profile));
    return pr ? { ...base, ...profileContext(pr), view: "decisions", known: "1" } : { ...base, profile: sv(p.summary.profile), known: "0" };
  }
  const label = sv(p.summary.label);
  const r = label ? requirementByLabel(page, label) : undefined;
  const repLabel = sv(p.summary.replacement);
  const rep = repLabel ? requirementByLabel(page, repLabel) : undefined;
  return {
    ...base, label, known: r ? "1" : "0", state: r ? stateAtCandidate(r, page.candidate) ?? "" : "", common: r?.common ? "1" : "0", blocking: r?.blocking ? "1" : "0",
    untested: blockingUntested(r, page.candidate) ? "1" : "0", replacement: repLabel, rep_known: rep ? "1" : "0", rep_state: rep ? stateAtCandidate(rep, page.candidate) ?? "" : "",
  };
}

function ApproveFacts({ l, page, p, ctx }: { l: Locale; page: EvidencePage; p: Proposal; ctx: Record<string, string> }) {
  const sample = page.requirements.origin === "sample_seed";
  return (
    <>
      <div><dt>{f.factProposal(l)}</dt><dd>{ctx.proposal ? ltr("#" + ctx.proposal) : <Absent>{k.unknown(l)}</Absent>}</dd></div>
      <div><dt>{f.factProposer(l)}</dt><dd><Who l={l} p={p.proposer} /></dd></div>
      <div className="p05-span"><dt>{f.factSummary(l)}</dt><dd><SummaryText l={l} page={page} p={p} links={false} /></dd></div>
      <div className="p05-span"><dt>{f.factProposerReason(l)}</dt><dd><Free l={l} value={p.rationale} empty={k.noReason(l)} /></dd></div>
      {ctx.module === "release_trace" && (
        ctx.known === "1" ? (
          <>
            <div className="p05-span">
              <dt>{f.factRequirement(l)}</dt>
              <dd>
                <Label value={ctx.label} />{" "}
                {ctx.common === "1" && <span className="p05-tag">{k.common(l)}</span>} {ctx.blocking === "1" && <span className="p05-tag">{k.blocking(l)}</span>}
                <span className="p05-line"><StateText l={l} state={ctx.state || null} candidate={page.candidate} sample={sample} /></span>
              </dd>
            </div>
            <div className="p05-span">
              <dt>{f.factReplacement(l)}</dt>
              <dd>
                {!ctx.replacement ? k.noReplacement(l) : ctx.rep_known === "1" ? <><Label value={ctx.replacement} /> · <StateText l={l} state={ctx.rep_state || null} candidate={page.candidate} sample={sample} /></> : <Template text={f.notInRead(l)} slots={{ label: <Label value={ctx.replacement} /> }} />}
              </dd>
            </div>
          </>
        ) : (
          <div className="p05-span"><dt>{f.factRequirement(l)}</dt><dd><Template text={f.notInRead(l)} slots={{ label: <Slot l={l} v={ctx.label} label /> }} /></dd></div>
        )
      )}
      {ctx.module === "perf_gate" && (ctx.known === "1" ? <ProfileFacts l={l} ctx={ctx} candidate={page.candidate} /> : <div className="p05-span"><dt>{f.factProfile(l)}</dt><dd><Template text={f.notInRead(l)} slots={{ label: <Slot l={l} v={ctx.profile} /> }} /></dd></div>)}
    </>
  );
}

function ApproveImpact({ l, page, ctx }: { l: Locale; page: EvidencePage; ctx: Record<string, string> }) {
  if (ctx.module === "perf_gate") return <ProfileImpact l={l} status={ctx.known === "1" && ctx.status_set === "1" ? ctx.status ?? "" : null} candidate={page.candidate} missing={ctx.known !== "1"} approve />;
  return <RetireImpact l={l} label={ctx.label ?? ""} replacement={ctx.replacement ?? ""} blockingUntested={ctx.untested === "1"} unknown={ctx.known !== "1"} approve />;
}

function StateTag({ l, p }: { l: Locale; p: Proposal }) {
  if (proposalUnexpected(p)) return <Unexpected l={l} code="proposal.state" />;
  if (p.state === "pending") return <span className="p05-tag">{k.proposalState(l, "pending")}</span>;
  return <State l={l} value={p.state} table={proposalStateWords} say={k.proposalState} locator="proposal.state" />;
}

function Row({ p, ...shared }: Shared & { p: Proposal }) {
  const { l, page, selection, canCommand } = shared;
  const target = proposalTarget(p);
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<Decision>(target);
  const pendingRow = p.state === "pending";
  const retire = p.kind === "retire";
  const n = p.number === null ? k.unknown(l) : String(p.number);
  const approvePending = target ? pendingNow("release_evidence.approve", target) : undefined;
  const withdrawPending = target ? pendingNow("release_evidence.withdraw", target) : undefined;
  const canApprove = retire && canCommand && !!target && pendingRow && (approveUsable(p) || (!!approvePending && frozenTargetOk("release_evidence.approve", target)));
  const canWithdraw = retire && canCommand && !!target && pendingRow && (withdrawUsable(p) || (!!withdrawPending && frozenTargetOk("release_evidence.withdraw", target)));
  const stale = !pendingRow && !!target && (!!approvePending || !!withdrawPending);
  useEffect(() => {
    if (!stale) return;
    drop("release_evidence.approve", target);
    drop("release_evidence.withdraw", target);
  }, [stale, target]);
  const refusedApprove = target ? refusedNow("release_evidence.approve", target) : "";
  const refusedWithdraw = target ? refusedNow("release_evidence.withdraw", target) : "";
  const kept = refusedApprove ? k.kept(l, keptKey("release_evidence.approve", refusedApprove)) : refusedWithdraw ? k.kept(l, keptKey("release_evidence.withdraw", refusedWithdraw)) : "";
  const earlier = refusedApprove === "earlier" || refusedWithdraw === "earlier";
  const blockers: Blocker[] = [];
  let state = "";
  if (pendingRow && retire && !earlier) {
    if (!target) blockers.push(line("", k.noCommand(l)));
    else {
      if (!canApprove && blockerPolicy(p.approve_blocked, { pendingRow: true }) === "explain") blockers.push(line(k.approve(l), blockerText(l, p.approve_blocked)));
      if (!canWithdraw) {
        if (blockerPolicy(p.withdraw_blocked, { pendingRow: true }) === "explain") blockers.push(line(k.withdraw(l), blockerText(l, p.withdraw_blocked)));
        else if (stateSentence(p.withdraw_blocked, { pendingRow: true })) state = k.stateSentence(l, stateSentence(p.withdraw_blocked, { pendingRow: true }));
      }
    }
  }
  const key = proposalKey(p);
  const current = selection.proposal !== "" && selection.proposal === key;
  const unexpected = proposalUnexpected(p);
  const label = p.module === "release_trace" ? sv(p.summary.label) : sv(p.summary.profile);
  return (
    <li className={"p05-proposal" + (current ? " live-selected" : "")} data-note={target || undefined} id={key ? rowId("proposal", key) : undefined} aria-current={current || undefined}>
      <div className="p05-meta">
        <p className="p05-small p05-muted"><Code l={l} value={p.module} table={moduleWords} say={k.module} /></p>
        <h4 className="p05-proposal-n">{ltr(k.proposalN(l, n))}{current && <span className="p05-tag p05-selected-tag" aria-hidden="true">{k.selected(l)}</span>}</h4>
        <p className="p05-small"><Code l={l} value={p.module + ":" + p.kind} table={kindWords} say={k.kind} /></p>
        <div className="p05-state"><StateTag l={l} p={p} /></div>
        <p className="p05-small"><Template text={k.proposedBy(l)} slots={{ who: <Who l={l} p={p.proposer} /> }} /></p>
        <p className="p05-small p05-muted"><Template text={k.createdAt(l)} slots={{ at: <Stamp l={l} at={p.created_at} /> }} /></p>
      </div>
      <div className="p05-body">
        <p className="p05-summary"><SummaryText l={l} page={page} p={p} /></p>
        {pendingRow && <p className="p05-small">{k.reasonLabel(l)}: <Free l={l} value={p.rationale} empty={k.noReason(l)} /></p>}
        {!pendingRow && <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={[]} focus={focusNote} />}
        {!pendingRow && !unexpected && (
          p.state === "store_retired" ? <p className="p05-small">{k.storeRetiredRow(l)}</p> : (
            <>
              {p.state === "applied" && <p className="p05-small"><Template text={p.module === "perf_gate" && p.kind === "budget" ? k.registrationApprovedByWho(l) : k.approvedBy(l)} slots={{ who: <Who l={l} p={p.approver ?? p.closed_by} /> }} /></p>}
              {p.state === "withdrawn" && <p className="p05-small"><Template text={k.withdrawnBy(l)} slots={{ who: <Who l={l} p={p.closed_by} /> }} /></p>}
              {p.closing_rationale && <p className="p05-small">{k.closingReason(l)}: <Free l={l} value={p.closing_rationale} /></p>}
              {p.state === "applied" && <p className="p05-small p05-muted">{k.appliedNote(l)}</p>}
            </>
          )
        )}
        {pendingRow && !retire && <p className="p05-small">{k.readOnlyKind(l)}</p>}
        {pendingRow && retire && (
          <div className="p05-act">
            <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} state={state} pending={(approvePending || withdrawPending) && !open ? f.pendingAttempt(l) : ""} />
            {(canApprove || canWithdraw) && !open && (
              <div className="live-buttons">
                {canApprove && <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("release_evidence.approve")} onClick={() => toggle("release_evidence.approve")}>{k.approve(l)}</button>}
                {canWithdraw && <button type="button" className="live-secondary" aria-expanded={false} data-opener={mark("release_evidence.withdraw")} onClick={() => toggle("release_evidence.withdraw")}>{k.withdraw(l)}</button>}
              </div>
            )}
            {open === "release_evidence.approve" && canApprove && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={{}}
                heading={f.approveHeading(l, n)} title={f.confirmApprove(l)}
                context={() => approveContext(page, p)}
                facts={(ctx) => <ApproveFacts l={l} page={page} p={p} ctx={ctx} />}
                impact={(ctx) => <ApproveImpact l={l} page={page} ctx={ctx} />}
                done={(_, st) => (st !== "applied" ? f.resultUnknown(l, st) : (got) => f.approveDone(l, n, retiredAfter(got, p) && label ? "\u2066" + label + "\u2069" : ""))}
                onClose={close}
              />
            )}
            {open === "release_evidence.withdraw" && canWithdraw && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={{}}
                heading={f.withdrawHeading(l, n)} title={f.confirmWithdraw(l)}
                context={() => ({ view: "decisions", proposal: p.number === null ? "" : String(p.number) })}
                facts={(ctx) => (
                  <>
                    <div><dt>{f.factProposal(l)}</dt><dd>{ctx.proposal ? ltr("#" + ctx.proposal) : <Absent>{k.unknown(l)}</Absent>}</dd></div>
                    <div className="p05-span"><dt>{f.factSummary(l)}</dt><dd><SummaryText l={l} page={page} p={p} links={false} /></dd></div>
                  </>
                )}
                impact={() => <p>{f.withdrawImpact(l)}</p>}
                done={(_, st) => (st === "withdrawn" ? f.withdrawDone(l, n) : f.resultUnknown(l, st))}
                onClose={close}
              />
            )}
          </div>
        )}
      </div>
    </li>
  );
}

export function DecisionsView(shared: Shared) {
  const { l, page } = shared;
  const d = page.decisions;
  const history = d.proposals.rows === null ? null : d.proposal_history;
  const older = useOlder(l, page.revision, history, d.proposals.rows ?? [], shared.api.evidenceProposals, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { void shared.onReload(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  });
  const { pending, closed } = ordered(older.rows);
  const unsure = history === null && d.proposals.more !== false;
  const cut = history !== null && history.total > closed.length;
  return (
    <>
      <p className="p05-lead">{k.decisionsLead(l)}</p>
      <dl className="p05-strip p05-strip-3">
        <div><dt>{k.supportDecisions(l)}</dt><dd><Fixed l={l} value={d.support_decisions} locator="support_decision" /></dd></div>
        <div><dt>{k.activationReview(l)}</dt><dd><Fixed l={l} value={d.activation_review} locator="activation" /></dd></div>
        <div><dt>{k.ledger(l)}</dt><dd><Fixed l={l} value={d.ledger} locator="ledger" expected="in_memory" say={k.ledgerWord} /></dd></div>
      </dl>
      {history === null && <MoreLine l={l} more={d.proposals.more} total={d.proposals.total} forms={plurals.showingNewest50} />}
      {older.head}
      {d.proposals.rows === null ? <p><Absent>{k.unknown(l)}</Absent></p> : d.proposals.rows.length === 0 ? <p>{d.proposals.more !== false ? k.noProposalsShown(l) : k.noProposals(l)}</p> : (
        <>
          <section className="p05-section-first" aria-labelledby="p05-waiting">
            <h3 id="p05-waiting">{k.waiting(l, pending.length, unsure)}</h3>
            {unsure && <p className="p05-small p05-muted">{k.olderWaiting(l)}</p>}
            {pending.length > 0 && <ol className="p05-proposals">{pending.map((p, i) => <Row key={p.target || "p" + i} p={p} {...shared} />)}</ol>}
          </section>
          <section className="p05-section" aria-labelledby="p05-closed">
            <h3 id="p05-closed">{k.closedGroup(l, closed.length, unsure || cut)}</h3>
            {older.count}
            {closed.length > 0 && <ol className="p05-proposals">{closed.map((p, i) => <Row key={p.target || "c" + i} p={p} {...shared} />)}</ol>}
            {older.foot}
          </section>
        </>
      )}
    </>
  );
}
