import type { Locale } from "../../../../shared/types";
import type { PluginProposal } from "../../../../service/integrations";
import { summaryWords, words } from "./live-codes";
import { k } from "./live-copy";
import { actor, blockerPolicy, drop, ordered, pendingNow, proposalTarget, refusedNow, shownLimit, summaryOf, type CommandKey } from "./live-model";
import { Code, Prose, fsi, ltr, stamp } from "./live-format";
import { useEffect, useState } from "react";
import { CommandForm, RowNote, blockerLines, plain, useOpen, type Blocker, type Shared } from "./LiveCommand";

export function SummaryText({ l, p }: { l: Locale; p: PluginProposal }) {
  const s = summaryOf(p);
  if (!s.known) return <Code l={l} value={p.kind} say={k.proposalKind} />;
  const template = words(l, summaryWords, s.kind);
  const pieces = template.split(/(\{[a-z]+\})/);
  return (
    <>
      {pieces.map((piece, n) => {
        const m = /^\{([a-z]+)\}$/.exec(piece);
        if (!m) return <span key={n}>{piece}</span>;
        const part = s.parts[m[1]];
        if (!part) return <span key={n}>{k.unknown(l)}</span>;
        if ("id" in part) return <span key={n}>{part.id ? ltr(part.id) : k.unknown(l)}</span>;
        if ("name" in part) return <span key={n}>{part.name ? <bdi>{part.name}</bdi> : k.unknown(l)}</span>;
        if ("date" in part) return <span key={n}>{fsi(stamp(l, part.date))}</span>;
        return <span key={n}>{part.text}</span>;
      })}
    </>
  );
}

function numberText(l: Locale, p: PluginProposal): string {
  return p.number === null ? k.unknown(l) : String(p.number);
}

function Facts({ l, p }: { l: Locale; p: PluginProposal }) {
  return (
    <>
      <div><dt>{k.factProposal(l)}</dt><dd className="p02-num">{numberText(l, p)}</dd></div>
      <div><dt>{k.factKind(l)}</dt><dd><Code l={l} value={p.kind} say={k.proposalKind} /></dd></div>
      <div><dt>{k.factSummary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
      <div><dt>{k.factProposerReason(l)}</dt><dd><Prose l={l} value={p.rationale} empty={k.noReason(l)} /></dd></div>
    </>
  );
}

type Decision = "integrations.plugin_approve" | "integrations.plugin_withdraw";

function Row({ p, ...shared }: Shared & { p: PluginProposal }) {
  const { l, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<Decision>();
  const target = proposalTarget(p);
  const pendingRow = p.state === "pending";
  const n = numberText(l, p);
  const approveUsable = canCommand && !!target && pendingRow && (p.can_approve || !!pendingNow("integrations.plugin_approve", target));
  const withdrawUsable = canCommand && !!target && pendingRow && (p.can_withdraw || !!pendingNow("integrations.plugin_withdraw", target));
  const stale = !pendingRow && !!target && (!!pendingNow("integrations.plugin_approve", target) || !!pendingNow("integrations.plugin_withdraw", target));
  const [applied, setApplied] = useState(false);
  useEffect(() => {
    if (!stale) return;
    drop("integrations.plugin_approve", target);
    drop("integrations.plugin_withdraw", target);
    setApplied(true);
  }, [stale, target]);
  const kept = !target ? "" : refusedNow("integrations.plugin_approve", target) ? k.kept(l, "proposal") : refusedNow("integrations.plugin_withdraw", target) ? k.keptWithdraw(l) : "";
  const blockers: Blocker[] = [];
  if (pendingRow && !target) blockers.push(plain(k.noCommand(l)));
  if (pendingRow && target && !approveUsable && !p.can_approve && blockerPolicy(p.approve_blocked, true) === "explain") blockers.push(...blockerLines(l, p.approve_blocked, "plugin_approve", k.approve(l)));
  if (pendingRow && target && !withdrawUsable && !p.can_withdraw && blockerPolicy(p.withdraw_blocked, true) === "explain") blockers.push(...blockerLines(l, p.withdraw_blocked, "plugin_withdraw", k.withdraw(l)));
  const current = selection.proposal !== "" && selection.proposal === String(p.number);
  const closedLine = p.state === "applied"
    ? k.approvedBy(l, actor(p.approver_login || p.closed_by_login, p.approver || p.closed_by) || k.unknown(l))
    : p.state === "withdrawn"
      ? k.withdrawnBy(l, actor(p.closed_by_login, p.closed_by) || k.unknown(l))
      : k.closedBy(l, actor(p.closed_by_login, p.closed_by) || k.unknown(l));

  return (
    <li className={"p02-proposal" + (current ? " live-selected" : "")} id={p.number === null ? undefined : "p02-proposal-" + p.number} aria-current={current || undefined}>
      <div className="p02-meta">
        <p className="p02-num p02-strong">{k.proposalN(l, n)}</p>
        <p className="p02-muted"><Code l={l} value={p.kind} say={k.proposalKind} /></p>
        <p><span className="p02-tag"><Code l={l} value={p.state} say={k.proposalState} /></span></p>
        <p className="p02-small">{k.proposedBy(l, actor(p.proposer_login, p.proposer) || k.unknown(l))}</p>
        <p className="p02-small live-quiet">{k.created(l, stamp(l, p.created_at))}</p>
      </div>
      <div className="p02-body">
        <p className="p02-summary"><SummaryText l={l} p={p} /></p>
        {pendingRow && <p className="p02-small">{k.reason(l)}: <Prose l={l} value={p.rationale} empty={k.noReason(l)} /></p>}
        {!pendingRow && (
          <>
            {applied && <p className="live-note-rule" role="status">{k.mayApplied(l)}</p>}
            <p className="p02-small">{closedLine}</p>
            <dl className="live-facts p02-closed">
              <div><dt>{k.closingReason(l)}</dt><dd><Prose l={l} value={p.closing_rationale} empty={k.noReason(l)} /></dd></div>
            </dl>
          </>
        )}
        {pendingRow && (
          <>
            <RowNote kept={kept} blockers={blockers} focus={focusNote} />
            {(approveUsable || withdrawUsable) && !open && (
              <div className="live-buttons">
                {approveUsable && (!open || open === "integrations.plugin_approve") && (
                  <button type="button" className="live-primary" aria-expanded={open === "integrations.plugin_approve"} data-opener={mark("integrations.plugin_approve")} onClick={() => toggle("integrations.plugin_approve")}>
                    {k.approve(l)}
                  </button>
                )}
                {withdrawUsable && (!open || open === "integrations.plugin_withdraw") && (
                  <button type="button" className="live-secondary" aria-expanded={open === "integrations.plugin_withdraw"} data-opener={mark("integrations.plugin_withdraw")} onClick={() => toggle("integrations.plugin_withdraw")}>
                    {k.withdraw(l)}
                  </button>
                )}
              </div>
            )}
            {open && target && (
              <CommandForm
                key={open} {...shared} command={open as CommandKey} target={target} values={{}}
                title={open === "integrations.plugin_approve" ? k.confirmApprove(l) : k.confirmWithdraw(l)}
                facts={<Facts l={l} p={p} />}
                impact={open === "integrations.plugin_approve" ? k.approveImpact(l) : k.withdrawImpact(l)}
                verb={open === "integrations.plugin_approve" ? k.approveVerb(l, n) : k.withdrawVerb(l, n)}
                done={(_, state) => (open === "integrations.plugin_approve" ? k.approvedDone(l, n, k.result(l, state)) : k.withdrawnDone(l, n))}
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
  const list = page.modules.proposals;
  const { pending, closed } = ordered(list ?? []);
  return (
    <section className="p02-section p02-proposals" aria-labelledby="p02-proposals-title">
      <h3 id="p02-proposals-title">{k.proposalsTitle(l)}</h3>
      {list === null && <p className="live-empty">{k.unknown(l)}</p>}
      {list !== null && list.length === 0 && <p className="live-empty">{k.noProposals(l)}</p>}
      {list !== null && list.length >= shownLimit && <p className="p02-small live-quiet">{k.newest(l, shownLimit)}</p>}
      {pending.length > 0 && (
        <>
          <h4>{k.waitingGroup(l, pending.length)}</h4>
          <ul className="p02-list">{pending.map((p, i) => <Row key={p.target || "p" + i} p={p} {...shared} />)}</ul>
        </>
      )}
      {closed.length > 0 && (
        <>
          <h4>{k.closedGroup(l, closed.length)}</h4>
          <ul className="p02-list">{closed.map((p, i) => <Row key={p.target || "c" + i} p={p} {...shared} />)}</ul>
        </>
      )}
    </section>
  );
}
