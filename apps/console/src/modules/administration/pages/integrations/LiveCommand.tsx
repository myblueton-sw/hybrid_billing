import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient } from "../../../../service/admin";
import type { IntegrationsPage } from "../../../../service/integrations";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import {
  clearDraft, deniedOutcome, discard, draftOf, fieldReaction, frozenValues, groupOf, keepDraft, pendingFor, pendingNow, react, reasonProblem, settle, snapshotOf, utf8Bytes,
  type CommandKey, type Reaction, type Selection,
} from "./live-model";

export type Shared = {
  l: Locale;
  page: IntegrationsPage;
  api: AdminClient;
  me: string;
  selection: Selection;
  canCommand: boolean;
  onDone: (text: string) => void;
  onNotice: (text: string) => void;
  onReload: () => Promise<IntegrationsPage | null>;
  onSignedOut: () => void;
};

export type ChoiceField = "destination" | "types";

type FormProps = Shared & {
  command: CommandKey;
  target: string;
  values: Record<string, string>;
  title: string;
  facts: ReactNode;
  impact: string;
  verb: string;
  done: (fields: Record<string, string>, state: string, snapshot: string[]) => string;
  onClose: (how?: "done" | "refused") => void;
  choices?: ReactNode;
  choiceProblems?: () => Problems;
  fieldIds?: Partial<Record<ChoiceField, string>>;
  context?: Record<string, string>;
  untitled?: boolean;
};

export function formId(command: CommandKey, target: string): string {
  return "p02-" + command.slice("integrations.".length) + "-" + target.replace(/[^A-Za-z0-9-]/g, "");
}

export function CommandForm(props: FormProps) {
  const { l, page, api, command, target, values, title, facts, impact, verb, done, onClose, onDone, onNotice, onReload, onSignedOut } = props;
  const id = formId(command, target);
  const [rationale, setRationale] = useState(() => frozenValues(pendingNow(command, target))?.rationale ?? draftOf(command, target).rationale ?? "");
  const [reasonError, setReasonError] = useState("");
  const [choiceErrors, setChoiceErrors] = useState<Problems>([]);
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; alert: boolean } | null>(null);
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const pending = pendingNow(command, target);
  const shown = JSON.stringify(values);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);
  useEffect(() => {
    discard(command, target, { ...values, rationale: rationale.trim() });
    setChoiceErrors([]);
  }, [shown]);

  const set = (v: string) => {
    setRationale(v);
    keepDraft(command, target, { rationale: v });
    discard(command, target, { ...values, rationale: v.trim() });
    setReasonError("");
    setMessage(null);
  };

  const reasonText = (v: string) => {
    const p = reasonProblem(v);
    return p === "missing" ? k.form.rationaleMissing(l) : p === "long" ? k.form.rationaleLong(l) : "";
  };

  const review = (e: FormEvent) => {
    e.preventDefault();
    const found = reasonText(rationale);
    const choices = props.choiceProblems ? props.choiceProblems() : [];
    setReasonError(found);
    setChoiceErrors(choices);
    if (found || choices.length > 0) { setSummaryKey((n) => n + 1); return; }
    setStep("confirm");
  };

  const send = async () => {
    if (busy) return;
    setBusy(true);
    setMessage(null);
    const reason = rationale.trim();
    const hadPending = !!pendingNow(command, target);
    const tried = pendingFor(command, target, { ...values, rationale: reason }, page.revision, attempt, snapshotOf(page, command), props.context ?? {});
    try {
      const out = await api.command(command, target, tried.expected, reason, values, { request: tried.request, key: tried.key });
      settle(command, target, "ok");
      clearDraft(command, target);
      onDone(done(out.fields, out.state, tried.snapshot));
      onClose("done");
    } catch (err) {
      const failure = err instanceof AdminError ? err.failure : "failed";
      const fields = err instanceof AdminError ? err.fields : [];
      const reaction: Reaction = react(failure, fields);
      settle(command, target, reaction);
      setStep(reaction === "conflict" ? "confirm" : "edit");
      if (reaction === "field") {
        const how = fieldReaction(command, fields);
        if (how === "changed") {
          setMessage({ text: k.versionsChanged(l), alert: false });
          await onReload();
        } else if (how === "kind") setMessage({ text: k.kindBug(l), alert: true });
        else if (how === "built") setMessage({ text: k.builtBug(l), alert: true });
        else if (how === "choice") {
          const ids = props.fieldIds ?? {};
          const list: Problems = [];
          if (fields.includes("destination")) list.push({ field: ids.destination ?? id + "-reason", message: k.destinationChanged(l) });
          if (fields.includes("types")) list.push({ field: ids.types ?? id + "-reason", message: k.typesChanged(l) });
          if (fields.includes("rationale")) setReasonError(k.form.checkReason(l));
          setChoiceErrors(list);
          setSummaryKey((n) => n + 1);
          await onReload();
        } else {
          setReasonError(reasonProblem(rationale) === "long" ? k.form.rationaleLong(l) : k.form.checkReason(l));
          setSummaryKey((n) => n + 1);
        }
      } else if (reaction === "reason") {
        setReasonError(reasonProblem(rationale) === "long" ? k.form.rationaleLong(l) : k.form.checkReason(l));
        setSummaryKey((n) => n + 1);
      } else if (reaction === "hint") {
        await onReload();
        onClose("refused");
      } else if (reaction === "conflict") {
        setMessage({ text: k.conflictAgain(l), alert: false });
        await onReload();
      } else if (reaction === "denied") {
        const got = await onReload();
        if (!got) setMessage({ text: k.gone(l, groupOf(command)), alert: true });
        else {
          const outcome = deniedOutcome(command, target, hadPending, got);
          const event = tried.context.event ?? "";
          onNotice(
            outcome === "applied_letter"
              ? k.vanishedLetter(l) + (command === "integrations.letter_approve_redeliver" ? " " + k.vanishedHint(l, event) : "")
              : outcome === "applied_proposal" ? k.vanishedProposal(l) : k.gone(l, groupOf(command)),
          );
          onClose("done");
        }
      } else if (reaction === "signin") onSignedOut();
      else if (reaction === "wait") setMessage({ text: sc.limited(l, retryPhrase(l, err instanceof AdminError ? err.retryAfter || 60 : 60)), alert: true });
      else if (reaction === "retry") setMessage({ text: k.pendingAttempt(l), alert: true });
      else if (reaction === "renew") setMessage({ text: k.renew(l), alert: true });
      else if (reaction === "partial") setMessage({ text: k.partial(l, err instanceof AdminError ? err.operation : ""), alert: true });
      else if (reaction === "reauth") setMessage({ text: sc.reauthText(l, failure), alert: true });
      else setMessage({ text: sc.failed(l), alert: true });
    } finally {
      setBusy(false);
    }
  };

  const problems: Problems = [...choiceErrors, ...(reasonError ? [{ field: id + "-reason", message: reasonError }] : [])];

  return (
    <div className="p02-form-wrap">
      {!props.untitled && <h4 className="p02-form-title">{verb}</h4>}
      {pending && step === "edit" && !message && <p className="live-note-rule" role="status">{k.pendingAttempt(l)}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error" : "live-note-rule"} role={message.alert ? "alert" : "status"}>
          <p>{message.text}</p>
        </div>
      )}
      {step === "edit" ? (
        <form className="p02-form" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          {props.choices}
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
        <div className="p02-confirm" aria-busy={busy || undefined}>
          <h3 ref={confirmTitle} tabIndex={-1}>{title}</h3>
          <dl className="live-facts">
            {facts}
            <div><dt>{k.form.rationale(l)}</dt><dd><bdi>{rationale.trim()}</bdi></dd></div>
          </dl>
          <p className="live-impact">{impact}</p>
          <div className="live-buttons">
            <button type="button" className="live-primary" disabled={busy} onClick={() => void send()}>{busy ? k.form.sending(l) : verb}</button>
            <button type="button" className="live-secondary" disabled={busy} onClick={() => setStep("edit")}>{k.form.backToEdit(l)}</button>
          </div>
        </div>
      )}
    </div>
  );
}

export type Blocker = { label: string; text: string };

export function RowNote({ kept, blockers, focus }: { kept: ReactNode; blockers: Blocker[]; focus: number }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focus) ref.current?.focus(); }, [focus]);
  if (!kept && blockers.length === 0) return null;
  const seen = new Set<string>();
  const list = blockers.filter((b) => {
    const id = b.label + "\u0000" + b.text;
    if (seen.has(id)) return false;
    seen.add(id);
    return true;
  });
  return (
    <div className="p02-row-note" ref={ref} tabIndex={-1} role={focus ? "status" : undefined}>
      {kept && <p className="live-note-rule">{kept}</p>}
      {list.length > 0 && (
        <ul className="p02-blocked">
          {list.map((b) => <li key={b.label + b.text}>{b.label && <strong>{b.label}: </strong>}{b.text}</li>)}
        </ul>
      )}
    </div>
  );
}

export function blockerLines(l: Locale, code: string, action: string, label = ""): Blocker[] {
  const extra = k.blockerExtra(l, code);
  const main = { label, text: k.blocker(l, code, action) };
  return extra ? [main, { label, text: extra }] : [main];
}

export const plain = (text: string): Blocker => ({ label: "", text });

export function useOpen<T extends string>() {
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
  return { open, toggle, close, focusNote, mark };
}
