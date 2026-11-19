import { currentScope } from "./live-model";
import { refusalDetail, type RefusalDetail as Refusal } from "../../../../service/refusal";
import { RefusalDetail } from "../../../../service/RefusalDetail";
import { useEffect, useId, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient } from "../../../../service/admin";
import type { EvidencePage } from "../../../../service/release-evidence";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { f, k } from "./live-copy";
import { Free, ltr } from "./live-format";
import {
  cancelled, clearDraft, safeId, deniedOutcome, discard, draftOf, fieldReaction, frozenValues, goneKind, hadEarlier, keepDraft, pendingFor, pendingNow, react, reasonProblem, renew, settle, snapshotOf,
  stillAllowed, utf8Bytes, type CommandKey, type FieldReaction, type Mode, type Reaction, type Selection,
} from "./live-model";

type Problems = Array<{ field: string; message: string }>;

function Summary({ locale: l, title, problems, focusKey }: { locale: Locale; title: string; problems: Problems; focusKey: number }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focusKey) ref.current?.focus(); }, [focusKey]);
  return (
    <div className="service-summary" role="alert" tabIndex={-1} ref={ref} lang={l}>
      <h2>{title}</h2>
      <ul>
        {problems.map((p) => (
          <li key={p.field + p.message}>
            <a href={"#" + p.field} onClick={(e) => { e.preventDefault(); document.getElementById(p.field)?.focus(); }}>{p.message}</a>
          </li>
        ))}
      </ul>
    </div>
  );
}

export type Announce = string | ((got: EvidencePage | null) => string);

export type Shared = {
  l: Locale;
  page: EvidencePage;
  api: AdminClient;
  me: string;
  selection: Selection;
  canCommand: boolean;
  mode: Mode;
  onAttempt?: () => void;
  onDone: (text: Announce) => void;
  onNotice: (text: string, detail?: Refusal) => void;
  onReload: () => Promise<EvidencePage | null>;
  onSignedOut: (detail?: Refusal) => void;
};

export type Draft = Record<string, string>;
export type FieldErrors = Record<string, string>;

export type Fields = {
  keys: string[];
  saved: string[];
  render: (draft: Draft, set: (key: string, v: string) => void, errors: FieldErrors, id: string) => ReactNode;
  check: (draft: Draft) => FieldErrors;
  values: (draft: Draft) => Record<string, string>;
};

type FormProps = Shared & {
  command: CommandKey;
  target: string;
  values?: Record<string, string>;
  fields?: Fields;
  heading: ReactNode;
  title: string;
  context: (draft: Draft) => Record<string, string>;
  facts: (ctx: Record<string, string>, draft: Draft) => ReactNode;
  impact: (ctx: Record<string, string>, draft: Draft) => ReactNode;
  onField?: (how: FieldReaction) => void;
  done: (fields: Record<string, string>, state: string, snapshot: string[]) => Announce;
  onClose: (how?: "done" | "refused") => void;
};

type Message = { text: string; alert: boolean; next?: string };

export function formId(command: CommandKey, target: string): string {
  return "p05-" + command.slice("release_evidence.".length) + "-" + safeId(target);
}

function without(r: Record<string, string>, key: string): Record<string, string> {
  return Object.fromEntries(Object.entries(r).filter(([name]) => name !== key));
}

function initial(command: CommandKey, target: string, keys: string[]): Draft {
  const frozen = frozenValues(pendingNow(command, target));
  const saved = draftOf(command, target);
  const out: Draft = { rationale: frozen?.rationale ?? saved.rationale ?? "" };
  for (const key of keys) out[key] = frozen ? (frozen[key] ?? (key === "replacement" ? "none" : "")) : (saved[key] ?? "");
  return out;
}

export function ErrorDetail({ l, next }: { l: Locale; next?: string }) {
  if (!next) return null;
  return (
    <dl className="p05-error-detail">
      {next && <div><dt>{k.nextStep(l)}</dt><dd>{next}</dd></div>}
    </dl>
  );
}

export function CommandForm(props: FormProps) {
  const { l, page, api, command, target, fields, heading, title, facts, impact, onField, done, onClose, onDone, onNotice, onReload, onSignedOut } = props;
  const id = formId(command, target);
  const [draft, setDraft] = useState<Draft>(() => initial(command, target, fields?.keys ?? []));
  const [reasonError, setReasonError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<Message | null>(null);
  const [renewing, setRenewing] = useState(false);
  const [detail, setDetail] = useState<Refusal | undefined>();
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const sending = useRef(false);
  const pending = pendingNow(command, target);
  const values = fields ? fields.values(draft) : (props.values ?? {});
  const shown = JSON.stringify(values);
  const rationale = draft.rationale ?? "";
  const ctx = pending ? pending.context : props.context(draft);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);
  useEffect(() => { discard(command, target, { ...values, rationale: rationale.trim() }); }, [shown]);

  const set = (key: string, v: string) => {
    const next = { ...draft, [key]: v };
    setDraft(next);
    if (key === "rationale" || fields?.saved.includes(key)) keepDraft(command, target, { [key]: v });
    const nextValues = fields ? fields.values(next) : (props.values ?? {});
    discard(command, target, { ...nextValues, rationale: (next.rationale ?? "").trim() });
    if (key === "rationale") setReasonError("");
    else setFieldErrors((e) => without(e, key));
    setMessage(null);
    setDetail(undefined);
  };

  const reasonText = (v: string) => {
    const p = reasonProblem(v);
    return p === "missing" ? f.rationaleMissing(l) : p === "long" ? f.rationaleLong(l) : "";
  };

  const review = (e: FormEvent) => {
    e.preventDefault();
    setDetail(undefined);
    const found = reasonText(rationale);
    const errors = fields ? fields.check(draft) : {};
    setReasonError(found);
    setFieldErrors(errors);
    if (found || Object.keys(errors).length > 0) { setSummaryKey((n) => n + 1); return; }
    setStep("confirm");
  };

  const send = async () => {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setMessage(null);
    setDetail(undefined);
    const reason = rationale.trim();
    const had = hadEarlier(command, target);
    const hadPending = !!pendingNow(command, target);
    const startedScope = currentScope();
    const tried = pendingFor(command, target, { ...values, rationale: reason }, page.revision, attempt, snapshotOf(page, command), ctx);
    props.onAttempt?.();
    const body = without(frozenValues(tried) ?? {}, "rationale");
    try {
      const out = await api.command(command, target, tried.expected, reason, body, { request: tried.request, key: tried.key });
      if (startedScope !== currentScope()) return;
      settle(command, target, "ok");
      clearDraft(command, target);
      onDone(done(out.fields, out.state, tried.snapshot));
      onClose("done");
    } catch (err) {
      if (startedScope !== currentScope()) return;
      const detail = refusalDetail(err);
      setDetail(detail);
      const e = err instanceof AdminError ? err : null;
      const failure = e ? e.failure : "failed";
      const fieldNames = e ? e.fields : [];
      const reaction: Reaction = react(failure, fieldNames);
      settle(command, target, reaction, failure, had, detail);
      setStep(reaction === "conflict" || reaction === "reauth" || reaction === "retry" || reaction === "wait" ? "confirm" : "edit");
      if (reaction === "field") {
        const how = fieldReaction(command, fieldNames);
        onField?.(how);
        if (how === "built") setMessage({ text: f.builtBug(l), alert: true, next: k.nextReload(l) });
        else if (how === "rationale") {
          setReasonError(reasonProblem(rationale) === "long" ? f.rationaleLong(l) : f.checkReason(l));
          setSummaryKey((n) => n + 1);
        } else if (how === "replacement") {
          setFieldErrors({ replacement: f.replacementGone(l) });
          setSummaryKey((n) => n + 1);
          await onReload();
          if (startedScope !== currentScope()) return;
        } else {
          setFieldErrors({ reason: fields?.check(draft).reason || f.checkRetireReason(l) });
          setSummaryKey((n) => n + 1);
        }
      } else if (reaction === "reason") {
        setReasonError(reasonProblem(rationale) === "long" ? f.rationaleLong(l) : f.checkReason(l));
        setSummaryKey((n) => n + 1);
      } else if (reaction === "hint") {
        await onReload();
        if (startedScope !== currentScope()) return;
        onClose("refused");
      } else if (reaction === "conflict") {
        const got = await onReload();
        if (startedScope !== currentScope()) return;
        if (got && !stillAllowed(got, command, target)) onClose("refused");
        else setMessage({ text: f.conflictAgain(l), alert: false });
      } else if (reaction === "denied") {
        const got = await onReload();
        if (startedScope !== currentScope()) return;
        if (got) {
          const outcome = deniedOutcome(command, target, hadPending, got);
          onNotice(outcome === "applied_proposal" ? k.vanishedProposal(l) : k.gone(l, goneKind(command)), detail);
          onClose("done");
        }
      } else if (reaction === "signin") onSignedOut(detail);
      else if (reaction === "wait") {
        const when = retryPhrase(l, e ? e.retryAfter || 60 : 60);
        setMessage({ text: sc.limited(l, when) + " " + f.pendingAttempt(l), alert: true, next: k.nextWait(l, when) });
      } else if (reaction === "retry") setMessage({ text: f.pendingAttempt(l), alert: true, next: k.nextTry(l) });
      else if (reaction === "renew") {
        renew(command, target);
        setRenewing(true);
        setMessage({ text: f.renew(l), alert: true, next: k.nextStart(l) });
      } else if (reaction === "partial") {
        const text = f.partial(l, e ? "\u2066" + e.operation + "\u2069" : "");
        const got = await onReload();
        if (startedScope !== currentScope()) return;
        if (got && !stillAllowed(got, command, target)) {
          onNotice(text, detail);
          onClose("done");
        } else setMessage({ text, alert: true, next: k.nextReload(l) });
      } else if (reaction === "reauth") setMessage({ text: failure === "reauth_cancelled" ? f.reauthCancelled(l) : sc.reauthText(l, failure), alert: true, next: k.nextTry(l) });
      else setMessage({ text: f.failed(l), alert: true, next: k.nextTry(l) });
    } finally {
      if (startedScope === currentScope()) {
        sending.current = false;
        setBusy(false);
      }
    }
  };

  const cancel = () => {
    cancelled(command, target);
    onClose();
  };

  const startAgain = () => {
    setRenewing(false);
    setMessage(null);
    setDetail(undefined);
    setStep("edit");
  };

  const problems: Problems = [
    ...(fields?.keys ?? []).filter((key) => fieldErrors[key]).map((key) => ({ field: id + "-" + key, message: fieldErrors[key] })),
    ...(reasonError ? [{ field: id + "-rationale", message: reasonError }] : []),
  ];
  const n = utf8Bytes(rationale.trim());

  return (
    <div className="p05-form-wrap">
      {step === "edit" && <h3 className="p05-form-title">{heading}</h3>}
      {pending && !message && <p className="p05-pending" role="status">{f.pendingAttempt(l)}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error p05-message" : "p05-message p05-quiet-note"} role={message.alert ? "alert" : "status"}>
          <p>{message.text}</p>
          <RefusalDetail locale={l} detail={detail} />
          <ErrorDetail l={l} next={message.next} />
          {renewing && <div className="live-buttons"><button type="button" className="live-secondary" onClick={startAgain}>{f.startAgain(l)}</button></div>}
        </div>
      )}
      {!message && <RefusalDetail locale={l} detail={detail} />}
      {step === "edit" ? (
        <form className="p05-form" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          {fields?.render(draft, set, fieldErrors, id)}
          <div className="service-field live-reason">
            <label htmlFor={id + "-rationale"}>{f.rationale(l)}</label>
            {reasonError && <p className="service-field-error" id={id + "-rationale-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{reasonError}</p>}
            <textarea
              id={id + "-rationale"} rows={4} value={rationale} aria-invalid={!!reasonError || undefined}
              aria-describedby={[id + "-rationale-hint", reasonError ? id + "-rationale-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set("rationale", e.target.value)}
            />
            <p className="service-hint p05-counter" id={id + "-rationale-hint"}>{f.rationaleHint(l, n)}</p>
          </div>
          <div className="live-buttons">
            <button type="submit" className="live-primary">{f.review(l)}</button>
            <button type="button" className="live-secondary" onClick={cancel}>{f.cancel(l)}</button>
          </div>
        </form>
      ) : (
        <section className="p05-confirm" aria-busy={busy || undefined} aria-labelledby={id + "-confirm"}>
          <h3 ref={confirmTitle} id={id + "-confirm"} tabIndex={-1} className="p05-confirm-title">{title}</h3>
          {pending && <p className="p05-repeat">{f.repeat(l)}</p>}
          <dl className="live-facts">
            {facts(ctx, draft)}
            <div className="p05-span"><dt>{f.rationale(l)}</dt><dd><Free l={l} value={rationale.trim()} /></dd></div>
          </dl>
          <div className="p05-impact">{impact(ctx, draft)}</div>
          <div className="live-buttons">
            <button type="button" className="live-primary" disabled={busy} onClick={() => void send()}>{busy ? f.sending(l) : f.confirm(l)}</button>
            <button type="button" className="live-secondary" disabled={busy} onClick={cancel}>{f.cancel(l)}</button>
          </div>
        </section>
      )}
    </div>
  );
}

export type Blocker = { label: string; text: string };

export function RowNote({ kept, blockers, focus, l, detail, extra, state, pending }: { kept: string; blockers: Blocker[]; focus: number; l?: Locale; detail?: Refusal; extra?: ReactNode; state?: string; pending?: string }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => { if (focus) ref.current?.focus(); }, [focus]);
  const seen = new Set<string>();
  const list = blockers.filter((b) => {
    const key = b.label + "\u0000" + b.text;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
  if (!kept && !state && !pending && list.length === 0 && !extra) return <div className="p05-row-note" ref={ref} tabIndex={-1} />;
  return (
    <div className="p05-row-note" ref={ref} tabIndex={-1} role={focus ? "status" : undefined}>
      {pending && <p className="p05-pending">{pending}</p>}
      {kept && <p className="p05-kept">{kept}</p>}
      {kept && l && <RefusalDetail locale={l} detail={detail} />}
      {state && <p className="p05-state-note">{state}</p>}
      {list.length > 0 && (
        <ul className="p05-blocked">
          {list.map((b) => <li key={b.label + b.text}>{b.label && <strong>{b.label}: </strong>}{b.text}</li>)}
        </ul>
      )}
      {extra}
    </div>
  );
}

export const line = (label: string, text: string): Blocker => ({ label, text });

export function useOpen<T extends string>(note = "") {
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
          const box = document.querySelector<HTMLElement>(`[data-note="${note}"] .p05-row-note`);
          if (box && !box.contains(document.activeElement)) box.focus();
        })));
      }
    } else if (!how) {
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
