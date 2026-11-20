import { RowDiagnostics, DiagnosticDetails, useCurrentCommand, useFailureDiagnostic } from "../../../../service/CommandDiagnostics";
import { diagnosticOf, type RowDiagnostic, type CommandDiagnostic } from "../../../../service/command-diagnostics";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient } from "../../../../service/admin";
import type { RetentionPage } from "../../../../service/retention";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { Free, Rich } from "./live-format";
import {
  refusalDiagnostics, clearDraft, deniedOutcome, discard, draftOf, drop, fieldReaction, frozenValues, goneKind, keepDraft, pendingFor, pendingNow, react, reasonProblem, settle, snapshotOf, stillAllowed,
  utf8Bytes, type CommandKey, type FieldReaction, type Reaction, type Selection,
} from "./live-model";

export type Announce = string | ((got: RetentionPage | null) => string);

export type Shared = {
  l: Locale;
  page: RetentionPage;
  api: AdminClient;
  me: string;
  workspace: string;
  selection: Selection;
  canCommand: boolean;
  onDone: (text: Announce) => void;
  onNotice: (text: string, diagnostic?: CommandDiagnostic) => void;
  onReload: () => Promise<RetentionPage | null>;
  onSignedOut: (diagnostic?: CommandDiagnostic) => void;
};

export type Draft = Record<string, string>;
export type FieldErrors = Record<string, string>;

export type Fields = {
  keys: string[];
  saved: string[];
  render: (draft: Draft, set: (key: string, v: string) => void, errors: FieldErrors, id: string) => ReactNode;
  check: (draft: Draft) => FieldErrors;
  values: (draft: Draft) => Record<string, string>;
  defaults?: Draft;
  ready?: boolean;
};

export type Hooks = {
  review?: () => void;
  send?: (values: Record<string, string>) => void;
  cancel?: () => void;
  settled?: (r: Reaction | "ok", failure: string) => void;
  field?: (how: FieldReaction) => void;
};

type FormProps = Shared & {
  command: CommandKey;
  target: string;
  values?: Record<string, string>;
  fields?: Fields;
  heading: string;
  title: string;
  verb: string;
  context: (draft: Draft) => Record<string, string>;
  facts: (ctx: Record<string, string>, draft: Draft) => ReactNode;
  impact: (ctx: Record<string, string>, draft: Draft) => ReactNode;
  warn?: ReactNode;
  editNote?: (pending: boolean) => ReactNode;
  stale?: (frozen: Record<string, string>) => ReactNode;
  hooks?: Hooks;
  done: (fields: Record<string, string>, state: string, snapshot: string[]) => Announce;
  onClose: (how?: "done" | "refused") => void;
};

export function formId(command: CommandKey, target: string): string {
  return "p04-" + command.slice("retention.".length) + "-" + target.replace(/[^A-Za-z0-9-]/g, "");
}

function without(r: Record<string, string>, key: string): Record<string, string> {
  return Object.fromEntries(Object.entries(r).filter(([name]) => name !== key));
}

function initial(command: CommandKey, target: string, keys: string[], defaults: Draft = {}): Draft {
  const frozen = frozenValues(pendingNow(command, target));
  const saved = draftOf(command, target);
  const out: Draft = { rationale: frozen?.rationale ?? saved.rationale ?? "" };
  for (const key of keys) out[key] = frozen ? (frozen[key] ?? "") : (saved[key] ?? defaults[key] ?? "");
  return out;
}

export function CommandForm(props: FormProps) {
  const { l, page, api, command, target, fields, heading, title, verb, facts, impact, warn, editNote, stale, hooks, done, onClose, onDone, onNotice, onReload, onSignedOut } = props;
  const current = useCurrentCommand(command, target);
  const { diagnostic, setDiagnostic } = useFailureDiagnostic(current);
  const id = formId(command, target);
  const [draft, setDraft] = useState<Draft>(() => initial(command, target, fields?.keys ?? [], fields?.defaults));
  const [reasonError, setReasonError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [again, setAgain] = useState(false);
  const [message, setMessage] = useState<{ text: string; alert: boolean } | null>(null);
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const staleRef = useRef<HTMLDivElement>(null);
  const sending = useRef(false);
  const pending = pendingNow(command, target);
  const values = fields ? fields.values(draft) : (props.values ?? {});
  const shown = JSON.stringify(values);
  const rationale = draft.rationale ?? "";
  const ctx = pending ? pending.context : props.context(draft);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);
  useEffect(() => { if (pending?.stale) staleRef.current?.focus(); }, [pending?.stale]);
  useEffect(() => { if (!pending?.stale) discard(command, target, { ...values, rationale: rationale.trim() }); }, [shown]);

  const set = (key: string, v: string) => {
    const next = { ...draft, [key]: v };
    setDraft(next);
    if (key === "rationale" || fields?.saved.includes(key)) keepDraft(command, target, { [key]: v });
    const nextValues = fields ? fields.values(next) : (props.values ?? {});
    if (!pending?.stale) discard(command, target, { ...nextValues, rationale: (next.rationale ?? "").trim() });
    if (key === "rationale") setReasonError("");
    else setFieldErrors((e) => without(e, key));
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
    const errors = fields ? fields.check(draft) : {};
    setReasonError(found);
    setFieldErrors(errors);
    if (found || Object.keys(errors).length > 0) { setSummaryKey((n) => n + 1); return; }
    hooks?.review?.();
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
    const tried = pendingFor(command, target, { ...values, rationale: reason }, page.revision, attempt, snapshotOf(page, command), ctx);
    const body = without(frozenValues(tried) ?? {}, "rationale");
    hooks?.send?.(body);
    try {
      const out = await api.command(command, target, tried.expected, reason, body, { request: tried.request, key: tried.key });
      if (!current()) return;
      settle(command, target, "ok");
      clearDraft(command, target);
      hooks?.settled?.("ok", "");
      onDone(done(out.fields, out.state, tried.snapshot));
      onClose("done");
    } catch (err) {
      if (!current()) return;
      const reference = { command, detail: diagnosticOf(err, target, tried.request) };
      setDiagnostic(reference.detail);
      const failure = err instanceof AdminError ? err.failure : "failed";
      const fieldNames = err instanceof AdminError ? err.fields : [];
      const reaction: Reaction = react(failure, fieldNames);
      settle(command, target, reaction, failure, reference);
      hooks?.settled?.(reaction, failure);
      setStep(reaction === "conflict" ? "confirm" : "edit");
      if (reaction === "field") {
        const how = fieldReaction(command, fieldNames);
        hooks?.field?.(how);
        if (how === "impact") {
          setMessage({ text: k.impactNotSent(l), alert: true });
          await onReload();
          if (!current()) return;
        } else if (how === "built") setMessage({ text: k.builtBug(l), alert: true });
        else if (how === "reason") {
          setReasonError(reasonProblem(rationale) === "long" ? k.form.rationaleLong(l) : k.form.checkReason(l));
          setSummaryKey((n) => n + 1);
        } else {
          const key = how === "text" ? "reason" : how;
          const text = how === "owner" ? k.form.ownerGone(l) : how === "period" ? k.form.periodRange(l) : how === "grace" ? k.form.graceRange(l) : k.form.textInvalid(l);
          setFieldErrors({ [key]: text });
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
        if (!got) setMessage({ text: k.gone(l, goneKind(command, target)), alert: true });
        else {
          const outcome = deniedOutcome(command, target, hadPending, got);
          onNotice(outcome === "applied_proposal" ? k.vanishedProposal(l) : k.gone(l, goneKind(command, target)), reference.detail);
          onClose("done");
        }
      } else if (reaction === "signin") onSignedOut(reference.detail);
      else if (reaction === "wait") setMessage({ text: sc.limited(l, retryPhrase(l, err instanceof AdminError ? err.retryAfter || 60 : 60)), alert: true });
      else if (reaction === "retry") setMessage({ text: k.pendingAttempt(l), alert: true });
      else if (reaction === "renew") {
        if (pendingNow(command, target)?.stale) {
          await onReload();
          if (!current()) return;
        } else setMessage({ text: k.renew(l), alert: true });
      } else if (reaction === "partial") {
        const text = k.partial(l, err instanceof AdminError ? err.operation : "");
        const got = await onReload();
        if (!current()) return;
        if (got && !stillAllowed(got, command, target)) {
          onNotice(text, reference.detail);
          onClose("done");
        } else setMessage({ text, alert: true });
      } else if (reaction === "reauth") setMessage({ text: sc.reauthText(l, failure), alert: true });
      else setMessage({ text: diagnosticCopy.unconfirmed(l), alert: true });
    } finally {
      if (current()) sending.current = false;
      if (current()) setBusy(false);
    }
  };

  const cancel = () => {
    hooks?.cancel?.();
    onClose();
  };

  const another = () => {
    drop(command, target);
    setAgain(true);
    setMessage(null);
    setDiagnostic(null);
    setStep("edit");
  };

  const problems: Problems = [
    ...(fields?.keys ?? []).filter((key) => fieldErrors[key]).map((key) => ({ field: id + "-" + key, message: fieldErrors[key] })),
    ...(reasonError ? [{ field: id + "-rationale", message: reasonError }] : []),
  ];
  const frozen = frozenValues(pending);

  if (pending?.stale && frozen) {
    return (
      <div className="p04-form-wrap">
        {diagnostic && <DiagnosticDetails l={l} detail={diagnostic} />}
        <div className="live-note-rule p04-stale" role="status" tabIndex={-1} ref={staleRef}>
          <p>{k.pendingAttempt(l)}</p>
          {stale?.(frozen)}
        </div>
        <div className="live-buttons">
          <button type="button" className="live-secondary" onClick={another}>{k.placeAnother(l)}</button>
          <button type="button" className="service-button-link" onClick={cancel}>{k.form.cancel(l)}</button>
        </div>
      </div>
    );
  }

  return (
    <div className="p04-form-wrap">
      {diagnostic && <DiagnosticDetails l={l} detail={diagnostic} />}
      {step === "edit" && <h4 className="p04-form-title"><Rich text={heading} /></h4>}
      {pending && step === "edit" && !message && <p className="live-note-rule" role="status">{k.pendingAttempt(l)}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error" : "live-note-rule"} role={message.alert ? "alert" : "status"}>
          <p><Rich text={message.text} /></p>
        </div>
      )}
      {step === "edit" ? (
        <form className="p04-form" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          {editNote?.(!!pending || again)}
          {fields?.render(draft, set, fieldErrors, id)}
          <div className="service-field live-reason">
            <label htmlFor={id + "-rationale"}>{k.form.rationale(l)}</label>
            {reasonError && <p className="service-field-error" id={id + "-rationale-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{reasonError}</p>}
            <textarea
              id={id + "-rationale"} rows={3} value={rationale} aria-invalid={!!reasonError || undefined}
              aria-describedby={[id + "-rationale-hint", reasonError ? id + "-rationale-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("rationale", e.target.value)}
            />
            <p className="service-hint" id={id + "-rationale-hint"}>{k.form.rationaleHint(l, utf8Bytes(rationale.trim()))}</p>
          </div>
          <div className="live-buttons">
            <button type="submit" className="live-secondary" disabled={fields?.ready === false}>{k.form.review(l)}</button>
            <button type="button" className="service-button-link" onClick={cancel}>{k.form.cancel(l)}</button>
          </div>
        </form>
      ) : (
        <div className="p04-confirm" aria-busy={busy || undefined}>
          <h4 ref={confirmTitle} tabIndex={-1} className="p04-confirm-title"><Rich text={title} /></h4>
          {warn}
          <dl className="live-facts">
            {facts(ctx, draft)}
            <div className="p04-span"><dt>{k.form.rationale(l)}</dt><dd><Free l={l} value={rationale.trim()} /></dd></div>
          </dl>
          <div className="live-impact">{impact(ctx, draft)}</div>
          {pending && <p className="p04-small p04-repeat">{k.repeat(l)}</p>}
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

export function RowNote({ l, diagnostics = [], kept, blockers, focus, extra, state }: { l: Locale; diagnostics?: RowDiagnostic[]; kept: string; blockers: Blocker[]; focus: number; extra?: ReactNode; state?: string }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focus) ref.current?.focus(); }, [focus]);
  const seen = new Set<string>();
  const list = blockers.filter((b) => {
    const key = b.label + "\u0000" + b.text;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  return (
    <div className="p04-row-note" ref={ref} tabIndex={-1} role={focus ? "status" : undefined}>
      {kept && <><p className="live-note-rule">{kept}</p><RowDiagnostics l={l} entries={diagnostics} /></>}
      {state && <p className="p04-state-note">{state}</p>}
      {list.length > 0 && (
        <ul className="p04-blocked">
          {list.map((b) => <li key={b.label + b.text}>{b.label && <strong>{b.label}: </strong>}<Rich text={b.text} /></li>)}
        </ul>
      )}
      {extra}
    </div>
  );
}

export const line = (label: string, text: string): Blocker => ({ label, text });

export function useOpen<T extends string>(note = "") {
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
    if (how === "refused") {
      requestAnimationFrame(() => setFocusNote((x) => x + 1));
      if (note) {
        requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => {
          const box = document.querySelector<HTMLElement>(`[data-note="${note}"] .p04-row-note`);
          if (box && !box.contains(document.activeElement)) box.focus();
        })));
      }
    }
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

export type { Problems };
