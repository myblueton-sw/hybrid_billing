import type { Locale } from "../../../../shared/types";
import type { IntegrationsPage, PluginProposal, PluginVersion } from "../../../../service/integrations";
import { summaryWords, words } from "./live-codes";
import { k } from "./live-copy";
import { actor, blockerPolicy, drop, installOf, ordered, pendingNow, proposalTarget, refusedByList, refusedLicense, refusedNow, shownLimit, summaryOf, versionOf, type CommandKey } from "./live-model";
import { Code, Known, Prose, fsi, ltr, stamp } from "./live-format";
import { useEffect, useState } from "react";
import { CommandForm, RowNote, blockerLines, plain, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { lostRight } from "./live-rights";
import { r } from "./rights-copy";

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

export function Marked({ text }: { text: string }) {
  const pieces = text.split(/\u2068([^\u2069]*)\u2069/);
  return <>{pieces.map((piece, n) => (n % 2 === 1 ? <span key={n}>{ltr(piece)}</span> : <span key={n}>{piece}</span>))}</>;
}

export type RefusalContext = "propose" | "enable" | "approve";

export function keptLicenseText(l: Locale, page: IntegrationsPage, plugin: string, version: string, context: RefusalContext): string {
  const v = versionOf(page, plugin, version);
  const why = refusedByList(v) ? k.keptLicense(l, version, v?.license || k.unknown(l)) : k.keptLicenseUnknown(l, version);
  return k.kept(l, "plugin") + " " + why + " " + k.keptLicenseNext(l, context);
}

export function LicenseLine({ l, v, label }: { l: Locale; v: PluginVersion | undefined; label?: string }) {
  if (!v) return <span className="p02-license-line">{k.licenseUnknown(l)}</span>;
  return (
    <>
      <span className="p02-license-line">{label ?? k.licenseLabel(l)}: <Known l={l} value={v.license} /></span>
      {v.license_state === "not_permitted" ? (
        <span className="p02-license-line"><strong className="p02-license-flag">{k.licenseState(l, v.license_state)}</strong> {k.licenseConsequence(l)}</span>
      ) : (
        <span className="p02-license-line">{k.licenseState(l, v.license_state)}</span>
      )}
    </>
  );
}

const versioned = (kind: string) => kind === "install" || kind === "upgrade" || kind === "rollback" || kind === "enable";

function targetVersion(page: IntegrationsPage, p: PluginProposal): string {
  return p.kind === "enable" ? installOf(page, p.summary.plugin)?.version ?? "" : p.summary.version;
}

function TargetLicense({ l, page, p }: { l: Locale; page: IntegrationsPage; p: PluginProposal }) {
  const version = targetVersion(page, p);
  return <LicenseLine l={l} v={versionOf(page, p.summary.plugin, version)} label={k.targetLicense(l, version || k.unknown(l))} />;
}

function numberText(l: Locale, p: PluginProposal): string {
  return p.number === null ? k.unknown(l) : String(p.number);
}

function Facts({ l, p, page }: { l: Locale; p: PluginProposal; page: IntegrationsPage }) {
  return (
    <>
      <div><dt>{k.factProposal(l)}</dt><dd className="p02-num">{numberText(l, p)}</dd></div>
      <div><dt>{k.factKind(l)}</dt><dd><Code l={l} value={p.kind} say={k.proposalKind} /></dd></div>
      <div><dt>{k.factSummary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
      {versioned(p.kind) && <div><dt>{k.licenseLabel(l)}</dt><dd><TargetLicense l={l} page={page} p={p} /></dd></div>}
      <div><dt>{k.factProposerReason(l)}</dt><dd><Prose l={l} value={p.rationale} empty={k.noReason(l)} /></dd></div>
    </>
  );
}

type Decision = "integrations.plugin_approve" | "integrations.plugin_withdraw";

function Row({ p, ...shared }: Shared & { p: PluginProposal }) {
  const { l, selection, canCommand, page } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<Decision>();
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
  const licensed = refusedLicense(page, p.summary.plugin, p.kind, p.summary.version);
  const approveKept = licensed ? <Marked text={keptLicenseText(l, page, p.summary.plugin, licensed, "approve")} /> : k.kept(l, "proposal");
  const lost = target ? lostRight("integrations.plugin_approve", target) : "";
  const kept = !target ? "" : lost ? r.noLongerHeld(l, lost) : refusedNow("integrations.plugin_approve", target) ? approveKept : refusedNow("integrations.plugin_withdraw", target) ? k.keptWithdraw(l) : "";
  const blockers: Blocker[] = [];
  if (pendingRow && !target) blockers.push(plain(k.noCommand(l)));
  if (pendingRow && target && !approveUsable && !p.can_approve && blockerPolicy(p.approve_blocked, true) === "explain") blockers.push(...blockerLines(l, p.approve_blocked, "plugin_approve", k.approve(l)));
  if (pendingRow && target && !withdrawUsable && !p.can_withdraw && blockerPolicy(p.withdraw_blocked, true) === "explain") blockers.push(...blockerLines(l, p.withdraw_blocked, "plugin_withdraw", k.withdraw(l)));
  const current = selection.proposal !== "" && selection.proposal === String(p.number);
  const lostClosure = p.state === "closed" && p.closing_reason === "requester_lost_right";
  const closedLine = lostClosure
    ? r.closedFound(l, actor(p.closure_trigger_login, p.closure_trigger) || k.unknown(l), p.closed_request || k.unknown(l))
    : p.state === "applied"
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
        {versioned(p.kind) && <p className="p02-small p02-license"><TargetLicense l={l} page={page} p={p} /></p>}
        {pendingRow && <p className="p02-small">{k.reason(l)}: <Prose l={l} value={p.rationale} empty={k.noReason(l)} /></p>}
        {!pendingRow && (
          <>
            {applied && <p className="live-note-rule" role="status">{k.mayApplied(l)}</p>}
            {!lostClosure && <p className="p02-small">{closedLine}</p>}
            <dl className="live-facts p02-closed">
              <div><dt>{k.closingReason(l)}</dt><dd>{lostClosure ? <Marked text={closedLine} /> : <Prose l={l} value={p.closing_rationale} empty={k.noReason(l)} />}</dd></div>
            </dl>
          </>
        )}
        {pendingRow && (
          <>
            <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} />
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
                facts={<Facts l={l} p={p} page={page} />}
                impact={open === "integrations.plugin_approve" ? k.approveImpact(l) : k.withdrawImpact(l)}
                verb={open === "integrations.plugin_approve" ? k.approveVerb(l, n) : k.withdrawVerb(l, n)}
                done={(_, state) => (open === "integrations.plugin_approve" ? k.approvedDone(l, n, k.result(l, state)) : k.withdrawnDone(l, n))}
                object={r.proposalObject(l, n)}
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
