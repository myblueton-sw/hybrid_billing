import { RowDiagnostics, DiagnosticDetails, useCurrentCommand, useFailureDiagnostic } from "../../../../service/CommandDiagnostics";
import { diagnosticOf, type RowDiagnostic, type CommandDiagnostic } from "../../../../service/command-diagnostics";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient } from "../../../../service/admin";
import type { RecoveryPage } from "../../../../service/recovery";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { Rich } from "./live-format";
import {
  refusalDiagnostics, clearDraft, deniedOutcome, discard, draftOf, fieldReaction, frozenValues, isProposalCommand, keepDraft, pendingFor, pendingNow, react, reasonProblem, settle, snapshotOf, stillAllowed,
  utf8Bytes, type CommandKey, type Reaction, type Selection,
} from "./live-model";

export type Announce = string | ((got: RecoveryPage | null) => string);

export type Shared = {
  l: Locale;
  page: RecoveryPage;
  api: AdminClient;
  me: string;
  selection: Selection;
  canCommand: boolean;
  onDone: (text: Announce) => void;
  onNotice: (text: string, diagnostic?: CommandDiagnostic) => void;
  onReload: () => Promise<RecoveryPage | null>;
  onSignedOut: (diagnostic?: CommandDiagnostic) => void;
};

type FormProps = Shared & {
  command: CommandKey;
  target: string;
  values: Record<string, string>;
  title: string;
  facts: ReactNode;
  impact: ReactNode;
  verb: string;
  done: (fields: Record<string, string>, state: string, snapshot: string[]) => Announce;
  onClose: (how?: "done" | "refused") => void;
  context?: Record<string, string>;
};

export function formId(command: CommandKey, target: string): string {
  return "p03-" + command.slice("recovery.".length) + "-" + target.replace(/[^A-Za-z0-9-]/g, "");
}

export function CommandForm(props: FormProps) {
  const { l, page, api, command, target, values, title, facts, impact, verb, done, onClose, onDone, onNotice, onReload, onSignedOut } = props;
  const current = useCurrentCommand(command, target);
  const { diagnostic, setDiagnostic } = useFailureDiagnostic(current);
  const id = formId(command, target);
  const [rationale, setRationale] = useState(() => frozenValues(pendingNow(command, target))?.rationale ?? draftOf(command, target).rationale ?? "");
  const [reasonError, setReasonError] = useState("");
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; alert: boolean } | null>(null);
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const editTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const sending = useRef(false);
  const pending = pendingNow(command, target);
  const shown = JSON.stringify(values);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); else if (document.activeElement === document.body) editTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);
  useEffect(() => { discard(command, target, { ...values, rationale: rationale.trim() }); }, [shown]);

  const set = (v: string) => {
    setRationale(v);
    keepDraft(command, target, { rationale: v });
    discard(command, target, { ...values, rationale: v.trim() });
    setReasonError("");
    setMessage(null);
    setDiagnostic(null);
  };

  const reasonText = (v: string) => {
    const p = reasonProblem(v);
    return p === "missing" ? k.form.rationaleMissing(l) : p === "long" ? k.form.rationaleLong(l) : "";
  };

  const review = (e: FormEvent) => {
    e.preventDefault();
    const found = reasonText(rationale);
    setReasonError(found);
    if (found) { setSummaryKey((n) => n + 1); return; }
    setStep("confirm");
  };

  const send = async () => {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setMessage(null);
    setDiagnostic(null);
    const reason = rationale.trim();
    const hadPending = !!pendingNow(command, target);
    const tried = pendingFor(command, target, { ...values, rationale: reason }, page.revision, attempt, snapshotOf(page, command), props.context ?? {});
    try {
      const out = await api.command(command, target, tried.expected, reason, values, { request: tried.request, key: tried.key });
      if (!current()) return;
      settle(command, target, "ok");
      clearDraft(command, target);
      onDone(done(out.fields, out.state, tried.snapshot));
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
      if (reaction === "field") {
        const how = fieldReaction(command, fields);
        if (how === "changed") {
          setMessage({ text: k.lostChanged(l), alert: false });
          const got = await onReload();
          if (!current()) return;
          if (got && !stillAllowed(got, command, target)) onClose("refused");
        } else if (how === "built") setMessage({ text: k.builtBug(l), alert: true });
        else {
          setReasonError(reasonProblem(rationale) === "long" ? k.form.rationaleLong(l) : k.form.checkReason(l));
          setSummaryKey((n) => n + 1);
        }
      } else if (reaction === "reason") {
        setReasonError(reasonProblem(rationale) === "long" ? k.form.rationaleLong(l) : k.form.checkReason(l));
        setSummaryKey((n) => n + 1);
      } else if (reaction === "hint") {
        await onReload();
        if (!current()) return;
        onClose("refused");
      } else if (reaction === "conflict") {
        const got = await onReload();
        if (!current()) return;
        if (got && !stillAllowed(got, command, target)) onClose("refused");
        else setMessage({ text: k.conflictAgain(l), alert: false });
      } else if (reaction === "denied") {
        const got = await onReload();
        if (!current()) return;
        if (!got) setMessage({ text: k.gone(l, isProposalCommand(command)), alert: true });
        else {
          const outcome = deniedOutcome(command, target, hadPending, got);
          onNotice(outcome === "applied_proposal" ? k.vanishedProposal(l) : k.gone(l, isProposalCommand(command)), reference.detail);
          onClose("done");
        }
      } else if (reaction === "signin") onSignedOut(reference.detail);
      else if (reaction === "wait") setMessage({ text: sc.limited(l, retryPhrase(l, err instanceof AdminError ? err.retryAfter || 60 : 60)), alert: true });
      else if (reaction === "retry") setMessage({ text: k.pendingAttempt(l), alert: true });
      else if (reaction === "renew") setMessage({ text: k.renew(l), alert: true });
      else if (reaction === "partial") {
        const text = k.partial(l, err instanceof AdminError ? err.operation : "");
        const got = await onReload();
        if (!current()) return;
        if (got && !stillAllowed(got, command, target)) {
          onNotice(text, reference.detail);
          onClose("done");
        } else setMessage({ text, alert: true });
      }
      else if (reaction === "reauth") setMessage({ text: sc.reauthText(l, failure), alert: true });
      else setMessage({ text: diagnosticCopy.unconfirmed(l), alert: true });
    } finally {
      if (current()) sending.current = false;
      if (current()) setBusy(false);
    }
  };

  const problems: Problems = reasonError ? [{ field: id + "-reason", message: reasonError }] : [];

  return (
    <div className="p03-form-wrap">
      {diagnostic && <DiagnosticDetails l={l} detail={diagnostic} />}
      {step === "edit" && <h4 ref={editTitle} tabIndex={-1} className="p03-form-title"><Rich text={verb} /></h4>}
      {pending && step === "edit" && !message && <p className="live-note-rule" role="status">{k.pendingAttempt(l)}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error" : "live-note-rule"} role={message.alert ? "alert" : "status"}>
          <p><Rich text={message.text} /></p>
        </div>
      )}
      {step === "edit" ? (
        <form className="p03-form" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          <div className="service-field live-reason">
            <label htmlFor={id + "-reason"}>{k.form.rationale(l)}</label>
            {reasonError && <p className="service-field-error" id={id + "-reason-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{reasonError}</p>}
            <textarea
              id={id + "-reason"} rows={3} value={rationale} aria-invalid={!!reasonError || undefined}
              aria-describedby={[id + "-reason-hint", reasonError ? id + "-reason-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set(e.target.value)}
            />
            <p className="service-hint" id={id + "-reason-hint"}>{k.form.rationaleHint(l, utf8Bytes(rationale.trim()))}</p>
          </div>
          <div className="live-buttons">
            <button type="submit" className="live-secondary">{k.form.review(l)}</button>
            <button type="button" className="service-button-link" onClick={() => onClose()}>{k.form.cancel(l)}</button>
          </div>
        </form>
      ) : (
        <div className="p03-confirm" aria-busy={busy || undefined}>
          <h4 ref={confirmTitle} tabIndex={-1} className="p03-confirm-title"><Rich text={title} /></h4>
          <dl className="live-facts">
            {facts}
            <div className="p03-span"><dt>{k.form.rationale(l)}</dt><dd><bdi>{rationale.trim()}</bdi></dd></div>
          </dl>
          <div className="live-impact">{impact}</div>
          {pending && <p className="p03-small p03-repeat">{k.repeat(l)}</p>}
          <div className="live-buttons">
            <button type="button" className="live-primary" disabled={busy} onClick={() => void send()}>{busy ? k.form.sending(l) : <Rich text={verb} />}</button>
            <button type="button" className="live-secondary" disabled={busy} onClick={() => setStep("edit")}>{k.form.backToEdit(l)}</button>
          </div>
        </div>
      )}
    </div>
  );
}

export type Blocker = { label: string; text: string };

export function RowNote({ l, diagnostics = [], kept, blockers, focus, extra }: { l: Locale; diagnostics?: RowDiagnostic[]; kept: string; blockers: Blocker[]; focus: number; extra?: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focus) ref.current?.focus(); }, [focus]);
  if (!kept && blockers.length === 0 && !extra) return null;
  const seen = new Set<string>();
  const list = blockers.filter((b) => {
    const id = b.label + "\u0000" + b.text;
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
  return (
    <div className="p03-row-note" ref={ref} tabIndex={-1} role={focus ? "status" : undefined}>
      {kept && <><p className="live-note-rule">{kept}</p><RowDiagnostics l={l} entries={diagnostics} /></>}
      {list.length > 0 && (
        <ul className="p03-blocked">
          {list.map((b) => <li key={b.label + b.text}>{b.label && <strong>{b.label}: </strong>}<Rich text={b.text} /></li>)}
        </ul>
      )}
      {extra}
    </div>
  );
}

export const line = (label: string, text: string): Blocker => ({ label, text });

export function useOpen<T extends string>() {
  const diagnostics = refusalDiagnostics;
  const [open, setOpen] = useState<T | null>(null);
  const last = useRef<T | null>(null);
  const [focusNote, setFocusNote] = useState(0);
  const id = useId();
  const mark = (what: T) => id + "|" + what;
  const toggle = (what: T) => {
    last.current = what;
    setOpen((cur) => (cur === what ? null : what));
  };
  const close = (how?: "done" | "refused") => {
    setOpen(null);
    if (how === "refused") requestAnimationFrame(() => setFocusNote((x) => x + 1));
    else if (!how) {
      requestAnimationFrame(() => requestAnimationFrame(() => {
        const what = last.current;
        if (!what) return;
        const found = Array.from(document.querySelectorAll<HTMLButtonElement>("[data-opener]")).find((b) => b.dataset.opener === mark(what));
        found?.focus();
      }));
    }
  };
  return { open, toggle, close, focusNote, mark, diagnostics };
}
