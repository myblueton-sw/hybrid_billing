import { useOlder } from "../../../../shared/KeptHistory";
import { RowDiagnostics, DiagnosticDetails, useCurrentCommand, useFailureDiagnostic } from "../../../../service/CommandDiagnostics";
import { diagnosticOf, type RowDiagnostic, type CommandDiagnostic } from "../../../../service/command-diagnostics";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import { useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient } from "../../../../service/admin";
import type { InstallationPage, InstallProposal } from "../../../../service/installation";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import {
  refusalDiagnostics, actor, clearDraft, endpoint, commandTarget, discard, draftOf, duplicateOf, hashChanged, isAcceptance, keepDraft, ordered, pendingFor, pendingNow, planChangedField, planTarget,
  proposeState, react, reasonTooLong, refusedNow, settle, summaryParts, utf8Bytes, withdrawSentence, type CommandKey, type Reaction,
} from "./live-model";
import { Code, Known, ltr, stamp } from "./live-format";

export type Shared = {
  l: Locale;
  page: InstallationPage;
  api: AdminClient;
  me: string;
  selected: string;
  stale: boolean;
  onDone: (text: string) => void;
  onNotice: (text: string, diagnostic?: CommandDiagnostic) => void;
  onReload: () => Promise<boolean>;
  onDenied: () => void;
  onSignedOut: (diagnostic?: CommandDiagnostic) => void;
};

type FormProps = Shared & {
  command: CommandKey;
  target: string;
  values: Record<string, string>;
  title: string;
  facts: ReactNode;
  impact: string;
  verb: string;
  done: (fields: Record<string, string>, state: string) => string;
  gone: string;
  onClose: (how?: "done" | "refused") => void;
};

export function CommandForm(props: FormProps) {
  const { l, page, api, command, target, values, title, facts, impact, verb, done, gone, stale, onClose, onDone, onNotice, onReload, onSignedOut } = props;
  const held = stale ? "p01-stale" : undefined;
  const current = useCurrentCommand(command, target);
  const { diagnostic, setDiagnostic } = useFailureDiagnostic(current);
  const id = "p01-" + command.slice(13) + "-" + target.replace(/[^a-z0-9-]/g, "");
  const [rationale, setRationale] = useState(() => draftOf(command, target).rationale ?? "");
  const [error, setError] = useState("");
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; alert: boolean } | null>(null);
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const pending = pendingNow(command, target);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);

  const set = (v: string) => {
    setRationale(v);
    keepDraft(command, target, { rationale: v });
    discard(command, target, { ...values, rationale: v });
    setError("");
    setMessage(null);
    setDiagnostic(null);
  };

  const review = (e: FormEvent) => {
    e.preventDefault();
    const found = !rationale.trim() ? k.form.rationaleMissing(l) : reasonTooLong(rationale) ? k.form.rationaleLong(l) : "";
    setError(found);
    if (found) { setSummaryKey((n) => n + 1); return; }
    setStep("confirm");
  };

  const send = async () => {
    if (busy) return;
    setBusy(true);
    setMessage(null);
    setDiagnostic(null);
    const tried = pendingFor(command, target, { ...values, rationale }, page.revision, attempt);
    try {
      const out = await api.command(command, target, tried.expected, rationale, values, { request: tried.request, key: tried.key });
      if (!current()) return;
      settle(command, target, "ok");
      clearDraft(command, target);
      onDone(done(out.fields, out.state));
      onClose("done");
    } catch (err) {
      if (!current()) return;
      const reference = { command, detail: diagnosticOf(err, target, tried.request) };
      setDiagnostic(reference.detail);
      const failure = err instanceof AdminError ? err.failure : "failed";
      const fields = err instanceof AdminError ? err.fields : [];
      const reaction: Reaction = react(failure, fields);
      settle(command, target, reaction, reference);
      setStep(reaction === "conflict" ? "confirm" : "edit");
      if (reaction === "field" && planChangedField(fields)) {
        setMessage({ text: k.planChanged(l), alert: false });
        await onReload();
        if (!current()) return;
      } else if (reaction === "field" || reaction === "reason") {
        setError(reasonTooLong(rationale) ? k.form.rationaleLong(l) : k.form.checkReason(l));
        setSummaryKey((n) => n + 1);
      } else if (reaction === "hint") {
        await onReload();
        if (!current()) return;
        onClose("refused");
      } else if (reaction === "conflict") {
        setMessage({ text: k.conflictAgain(l), alert: false });
        await onReload();
        if (!current()) return;
      } else if (reaction === "denied") {
        const reloaded = await onReload();
        if (!current()) return;
        if (reloaded) { onNotice(gone, reference.detail); onClose("done"); }
        else setMessage({ text: gone, alert: true });
      } else if (reaction === "signin") onSignedOut(reference.detail);
      else if (reaction === "wait") setMessage({ text: sc.limited(l, retryPhrase(l, err instanceof AdminError ? err.retryAfter || 60 : 60)), alert: true });
      else if (reaction === "retry") setMessage({ text: k.pendingAttempt(l), alert: true });
      else if (reaction === "renew") setMessage({ text: k.renew(l), alert: true });
      else if (reaction === "partial") setMessage({ text: k.partial(l, err instanceof AdminError ? err.operation : ""), alert: true });
      else if (reaction === "reauth") setMessage({ text: sc.reauthText(l, failure), alert: true });
      else setMessage({ text: diagnosticCopy.unconfirmed(l), alert: true });
    } finally {
      if (current()) setBusy(false);
    }
  };

  const problems: Problems = error ? [{ field: id + "-reason", message: error }] : [];

  return (
    <div className="p01-form-wrap">
      {diagnostic && <DiagnosticDetails l={l} detail={diagnostic} />}
      {pending && step === "edit" && !message && <p className="live-note-rule" role="status">{k.pendingAttempt(l)}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error" : "live-note-rule"} role={message.alert ? "alert" : "status"}>
          <p>{message.text}</p>
        </div>
      )}
      {step === "edit" ? (
        <form className="p01-form" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          <div className="service-field live-reason">
            <label htmlFor={id + "-reason"}>{k.form.rationale(l)}</label>
            {error && <p className="service-field-error" id={id + "-reason-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{error}</p>}
            <textarea
              id={id + "-reason"} rows={3} value={rationale} aria-invalid={!!error || undefined}
              aria-describedby={[id + "-reason-hint", error ? id + "-reason-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set(e.target.value)}
            />
            <p className="service-hint" id={id + "-reason-hint"}>{k.form.rationaleHint(l, utf8Bytes(rationale))}</p>
          </div>
          <div className="live-buttons">
            <button type="submit" className="live-secondary" disabled={stale} aria-describedby={held}>{k.form.review(l)}</button>
            <button type="button" className="service-button-link" onClick={() => onClose()}>{k.form.cancel(l)}</button>
          </div>
        </form>
      ) : (
        <div className="p01-confirm" aria-busy={busy || undefined}>
          <h3 ref={confirmTitle} tabIndex={-1}>{title}</h3>
          <dl className="live-facts">
            {facts}
            <div><dt>{k.form.rationale(l)}</dt><dd><bdi>{rationale}</bdi></dd></div>
          </dl>
          <p className="live-impact">{impact}</p>
          <div className="live-buttons">
            <button type="button" className="live-primary" disabled={busy || stale} aria-describedby={held} onClick={() => void send()}>{busy ? k.form.sending(l) : verb}</button>
            <button type="button" className="live-secondary" disabled={busy} onClick={() => setStep("edit")}>{k.form.backToEdit(l)}</button>
          </div>
        </div>
      )}
    </div>
  );
}

export function SummaryText({ l, p }: { l: Locale; p: InstallProposal }) {
  const s = summaryParts(p);
  if (!s.known) return <Code l={l} value={p.kind} say={k.kind} />;
  const lead = k.summaryLead(l, s.kind);
  if (s.kind === "destination") {
    const [host, port, ranges] = s.values;
    return (
      <>
        {lead} <bdi dir="ltr">{endpoint(host, port, k.unknown(l))}</bdi>
        {ranges && <> (<bdi dir="ltr">{ranges}</bdi>)</>}
      </>
    );
  }
  return <>{lead} <Known l={l} value={s.values[0]} /></>;
}

function numberText(l: Locale, p: InstallProposal): string {
  return p.number === null ? k.unknown(l) : String(p.number);
}

function Facts({ l, p }: { l: Locale; p: InstallProposal }) {
  return (
    <>
      <div><dt>{k.form.proposal(l)}</dt><dd className="p01-num">{numberText(l, p)}</dd></div>
      <div><dt>{k.form.kind(l)}</dt><dd><Code l={l} value={p.kind} say={k.kind} /></dd></div>
      <div><dt>{k.form.summary(l)}</dt><dd><SummaryText l={l} p={p} /></dd></div>
      {isAcceptance(p) && <div><dt>{k.proposedHash(l)}</dt><dd className="p01-hash"><Known l={l} value={p.summary.plan_hash} /></dd></div>}
      <div><dt>{k.form.proposerReason(l)}</dt><dd>{p.rationale ? <bdi>{p.rationale}</bdi> : k.noReason(l)}</dd></div>
    </>
  );
}

function Row({ p, ...shared }: Shared & { p: InstallProposal }) {
  const { l, selected } = shared;
  const [open, setOpen] = useState<CommandKey | null>(null);
  const opener = useRef<HTMLButtonElement | null>(null);
  const note = useRef<HTMLDivElement>(null);
  const [focusNote, setFocusNote] = useState(0);
  const diagnostics = refusalDiagnostics;
  const target = commandTarget(p);
  const pendingRow = p.state === "pending";
  const n = numberText(l, p);
  const approveUsable = !!target && (p.can_approve || !!pendingNow("installation.approve", target));
  const withdrawUsable = !!target && (p.can_withdraw || !!pendingNow("installation.withdraw", target));
  const kept = !target ? "" : refusedNow("installation.approve", target) ? k.keptApprove(l) : refusedNow("installation.withdraw", target) ? k.keptWithdraw(l) : "";
  const blockers: string[] = [];
  if (pendingRow && !target) blockers.push(k.noCommand(l));
  if (pendingRow && target && !approveUsable) blockers.push(k.approveBlocked(l, p.approve_blocked));
  if (pendingRow && target && !withdrawUsable && withdrawSentence(p)) blockers.push(k.withdrawBlocked(l));
  const current = selected !== "" && selected === String(p.number);

  useEffect(() => { if (focusNote) note.current?.focus(); }, [focusNote]);

  const close = (how?: "done" | "refused") => {
    setOpen(null);
    if (how === "refused") requestAnimationFrame(() => setFocusNote((x) => x + 1));
    else if (!how) requestAnimationFrame(() => opener.current?.focus());
  };

  const closedLine = p.state === "applied"
    ? k.approvedBy(l, actor(p.approver_login || p.closed_by_login, p.approver || p.closed_by) || k.unknown(l))
    : p.state === "withdrawn"
      ? k.withdrawnBy(l, actor(p.closed_by_login, p.closed_by) || k.unknown(l))
      : k.closedBy(l, actor(p.closed_by_login, p.closed_by) || k.unknown(l));

  return (
    <li className={"p01-proposal" + (current ? " live-selected" : "")} id={p.number === null ? undefined : "p01-proposal-" + p.number} aria-current={current || undefined}>
      <div className="p01-meta">
        <p className="p01-num">{k.proposalN(l, n)}</p>
        <p className="p01-kind"><Code l={l} value={p.kind} say={k.kind} /></p>
        <p><span className="p01-tag"><Code l={l} value={p.state} say={k.state} /></span></p>
        <p className="p01-small">{k.proposedBy(l, actor(p.proposer_login, p.proposer) || k.unknown(l))}</p>
        <p className="p01-small live-quiet">{k.created(l, stamp(l, p.created_at))}</p>
      </div>
      <div className="p01-body">
        <p className="p01-summary"><SummaryText l={l} p={p} /></p>
        {pendingRow && <p className="p01-small">{k.reason(l)}: {p.rationale ? <bdi>{p.rationale}</bdi> : k.noReason(l)}</p>}
        {isAcceptance(p) && (
          <>
            <dl className="p01-hashes">
              <div><dt>{k.proposedHash(l)}</dt><dd className="p01-hash"><Known l={l} value={p.summary.plan_hash} /></dd></div>
              {p.current_hash && <div><dt>{k.currentHash(l)}</dt><dd className="p01-hash">{ltr(p.current_hash)}</dd></div>}
            </dl>
            {hashChanged(p) && <p className="p01-small">{k.hashChanged(l)}</p>}
          </>
        )}
        {!pendingRow && (
          <>
            <p className="p01-small">{closedLine}</p>
            <dl className="live-facts p01-closed">
              <div><dt>{k.closingReason(l)}</dt><dd>{p.closing_rationale ? <bdi>{p.closing_rationale}</bdi> : k.noReason(l)}</dd></div>
            </dl>
          </>
        )}
        {pendingRow && (
          <>
            {(kept || blockers.length > 0) && (
              <div className="p01-row-note" ref={note} tabIndex={-1} role="status">
                {kept && <><p className="live-note-rule">{kept}</p><RowDiagnostics l={l} entries={diagnostics(target)} /></>}
                {blockers.length > 0 && <ul className="p01-blocked">{blockers.map((b) => <li key={b}>{b}</li>)}</ul>}
              </div>
            )}
            {(approveUsable || withdrawUsable) && (
              <div className="live-buttons">
                {approveUsable && (
                  <button type="button" className="live-primary" disabled={shared.stale} aria-describedby={shared.stale ? "p01-stale" : undefined} aria-expanded={open === "installation.approve"} onClick={(e) => { opener.current = e.currentTarget; setOpen(open === "installation.approve" ? null : "installation.approve"); }}>
                    {k.approve(l)}
                  </button>
                )}
                {withdrawUsable && (
                  <button type="button" className="live-secondary" disabled={shared.stale} aria-describedby={shared.stale ? "p01-stale" : undefined} aria-expanded={open === "installation.withdraw"} onClick={(e) => { opener.current = e.currentTarget; setOpen(open === "installation.withdraw" ? null : "installation.withdraw"); }}>
                    {k.withdraw(l)}
                  </button>
                )}
              </div>
            )}
            {open && target && (
              <CommandForm
                key={open} {...shared} command={open} target={target} values={{}}
                title={open === "installation.approve" ? k.form.confirmApprove(l) : k.form.confirmWithdraw(l)}
                facts={<Facts l={l} p={p} />}
                impact={open === "installation.approve" ? k.form.approveImpact(l) : k.form.withdrawImpact(l)}
                verb={open === "installation.approve" ? k.form.approveVerb(l, n) : k.form.withdrawVerb(l, n)}
                done={(_, state) => (open === "installation.approve" ? k.approvedDone(l, n, k.state(l, state)) : k.withdrawnDone(l, n))}
                gone={k.gone(l)}
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
  const older = useOlder(l, page.revision, page.proposal_history, page.proposals, shared.api.installationProposals, (e) => {
    if (e instanceof AdminError && e.failure === "denied") { shared.onDenied(); return true; }
    if (e instanceof AdminError && e.failure === "unauthenticated") { shared.onSignedOut(); return true; }
    return false;
  });
  const { pending, closed } = ordered(older.rows);
  return (
    <section className="p01-proposals" aria-labelledby="p01-proposals-title">
      <h3 id="p01-proposals-title">{k.proposals(l)}</h3>
      {page.proposals.length === 0 && <p className="live-empty">{k.noProposals(l)}</p>}
      {older.head}
      {pending.length > 0 && (
        <>
          <h4>{k.waitingGroup(l, pending.length)}</h4>
          <ul className="p01-list">{pending.map((p, i) => <Row key={p.target || "p" + i} p={p} {...shared} />)}</ul>
        </>
      )}
      {closed.length > 0 && (
        <>
          <h4>{k.closedGroup(l, closed.length)}</h4>
          {older.count}
          <ul className="p01-list">{closed.map((p, i) => <Row key={p.target || "c" + i} p={p} {...shared} />)}</ul>
        </>
      )}
      {closed.length === 0 && older.count}
      {older.foot}
    </section>
  );
}

export function Propose(shared: Shared) {
  const { l, page, me } = shared;
  const [open, setOpen] = useState(false);
  const opener = useRef<HTMLButtonElement | null>(null);
  const note = useRef<HTMLDivElement>(null);
  const [focusNote, setFocusNote] = useState(0);
  const diagnostics = refusalDiagnostics;
  const state = proposeState(page, me);
  const p = page.preview;
  const values = { version: p.version, hash: p.hash };
  const kept = pendingNow("installation.propose_plan", planTarget);
  const frozen: Record<string, string> = kept ? (JSON.parse(kept.values) as Record<string, string>) : {};
  const sent = kept ? { version: frozen.version ?? "", hash: frozen.hash ?? "" } : values;
  const usable = state.offered || !!kept;
  const refused = refusedNow("installation.propose_plan", planTarget);

  useEffect(() => { if (focusNote) note.current?.focus(); }, [focusNote]);

  const close = (how?: "done" | "refused") => {
    setOpen(false);
    if (how === "refused") requestAnimationFrame(() => setFocusNote((x) => x + 1));
    else if (!how) requestAnimationFrame(() => opener.current?.focus());
  };

  return (
    <section className="p01-propose" aria-label={k.propose(l)}>
      {(refused || !usable || state.stale) && (
        <div className="p01-row-note" ref={note} tabIndex={-1} role="status">
          {refused && <><p className="live-note-rule">{k.keptPropose(l)}</p><RowDiagnostics l={l} entries={diagnostics(planTarget)} /></>}
          {!usable && <p className="p01-blocker">{k.proposeBlocked(l, state.blocker, p.version)}</p>}
          {usable && state.stale && <p className="p01-blocker">{k.staleOwn(l)}</p>}
        </div>
      )}
      {usable && (
        <div className="live-buttons">
          <button type="button" className="live-secondary" disabled={shared.stale} aria-describedby={shared.stale ? "p01-stale" : undefined} aria-expanded={open} onClick={(e) => { opener.current = e.currentTarget; setOpen(!open); }}>
            {k.propose(l)}
          </button>
        </div>
      )}
      {open && usable && (
        <CommandForm
          {...shared} command="installation.propose_plan" target={planTarget} values={sent}
          title={k.form.confirmPropose(l)}
          facts={
            <>
              <div><dt>{k.version(l)}</dt><dd><Known l={l} value={sent.version} /></dd></div>
              <div><dt>{k.hash(l)}</dt><dd className="p01-hash"><Known l={l} value={sent.hash} /></dd></div>
            </>
          }
          impact={k.form.proposeImpact(l)}
          verb={k.form.proposeVerb(l, sent.version)}
          done={(fields) => (duplicateOf(fields.existing) ? k.duplicate(l, fields.proposal || k.unknown(l)) : k.proposedDone(l, fields.proposal || k.unknown(l), sent.version))}
          gone={k.goneProposal(l)}
          onClose={close}
        />
      )}
    </section>
  );
}
