import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type CommandId } from "../../../../service/admin";
import type { ContractActionKey, ContractDetail, ContractEvent } from "../../../../service/contracts";
import type { Session } from "../../../../service/api";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import {
  actionOf, bodyFor, cancelReasons, clearDraft, commandFor, controlFor, coversEnd, dayProblem, discard, draftOf, gapReasons, idProblem, keepDraft, lifecycleBounds,
  lifecycles, maxResolves, offered, openGaps, pendingFor, pendingNow, react, reasonTooLong, reopenReasons, settle, settleBounds, utf8Bytes, type Reaction,
} from "./live-model";
import { day, ltr } from "./live-format";

export type Shared = {
  l: Locale;
  detail: ContractDetail;
  api: AdminClient;
  session: Session;
  onDone: (text: string) => void;
  onReload: () => Promise<void>;
  onDenied: () => void;
  onSignedOut: () => void;
};

const fieldIds: Record<string, string> = {
  from: "o03-f-from",
  cancel: "o03-f-cancel",
  reference: "o03-f-reference",
  covered_until: "o03-f-covered",
  resolves: "o03-f-resolves",
  follow_up: "o03-f-follow",
  as_of: "o03-f-asof",
  rationale: "o03-f-rationale",
};

function commandOf(key: ContractActionKey): CommandId {
  return commandFor(key) as CommandId;
}

function slotOf(key: ContractActionKey, event?: ContractEvent): string {
  return event ? key + ":" + event.id : key;
}

export function ActionBar({ keys, title, ...shared }: Shared & { keys: ContractActionKey[]; title: string }) {
  const { l, detail } = shared;
  const [open, setOpen] = useState<ContractActionKey | null>(null);
  const opener = useRef<HTMLButtonElement | null>(null);
  const contract = detail.contract.id;
  const trusted = !!detail.revision && !!contract;
  const usable = (key: ContractActionKey) => trusted && (offered(actionOf(detail, key), lifecycles.includes(key)) || !!pendingNow(slotOf(key), contract));
  const blocked = keys.filter((key) => !usable(key));
  return (
    <section className="live-group o03-actions" aria-labelledby={"o03-actions-" + keys[0]}>
      <h3 id={"o03-actions-" + keys[0]}>{title}</h3>
      <div className="live-buttons">
        {keys.filter(usable).map((key) => (
          <button
            key={key} type="button" className="live-secondary" aria-expanded={open === key}
            onClick={(e) => { opener.current = e.currentTarget; setOpen(open === key ? null : key); }}
          >
            {k.command(l, key)}
          </button>
        ))}
      </div>
      {blocked.length > 0 && (
        <ul className="o03-blocked">
          {blocked.map((key) => <li key={key}>{k.blocked(l, key, actionOf(detail, key).blocked)}</li>)}
        </ul>
      )}
      {open && (
        <CommandForm
          key={open} commandKey={open} {...shared}
          onClose={(done) => { setOpen(null); if (!done) requestAnimationFrame(() => opener.current?.focus()); }}
        />
      )}
    </section>
  );
}

type Values = Record<string, string>;

function initial(key: ContractActionKey, detail: ContractDetail): Values {
  const a = actionOf(detail, key);
  if (lifecycles.includes(key)) return { from: a.earliest || detail.today, rationale: "" };
  if (key === "contract.evidence.secured") return { reference: "", covered_until: detail.today, resolves: "", rationale: "" };
  if (key === "contract.evidence.missing") return { reason: "", follow_up: "", rationale: "" };
  if (key === "contract.settle") return { as_of: detail.today, rationale: "" };
  if (key === "contract.reopen") return { reason: "", rationale: "" };
  return { cancel: "", rationale: "" };
}

export function CommandForm({ commandKey: key, event, onClose, ...shared }: Shared & { commandKey: ContractActionKey; event?: ContractEvent; onClose: (done?: boolean) => void }) {
  const { l, detail, api, onDone, onReload, onDenied, onSignedOut } = shared;
  const contract = detail.contract.id;
  const slot = slotOf(key, event);
  const [values, setValues] = useState<Values>(() => ({ ...initial(key, detail), ...draftOf(slot, contract) }));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [summaryKey, setSummaryKey] = useState(0);
  const [step, setStep] = useState<"edit" | "confirm">("edit");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ text: string; alert: boolean; hint?: boolean } | null>(null);
  const confirmTitle = useRef<HTMLHeadingElement>(null);
  const messageRef = useRef<HTMLDivElement>(null);
  const isLifecycle = lifecycles.includes(key);
  const end = detail.contract.end;
  const pending = pendingNow(slot, contract);

  useEffect(() => { if (step === "confirm") confirmTitle.current?.focus(); }, [step]);
  useEffect(() => { if (message) messageRef.current?.focus(); }, [message]);

  const set = (field: string, v: string) => {
    const next = { ...values, [field]: v };
    setValues(next);
    keepDraft(slot, contract, next);
    discard(slot, contract, { ...bodyFor(key, next, event), rationale: next.rationale });
    setErrors((e) => { const n = { ...e }; delete n[field]; return n; });
    setMessage(null);
  };

  const validate = (): Record<string, string> => {
    const e: Record<string, string> = {};
    const dateError = (field: string, min: string, max: string) => {
      const p = dayProblem(values[field] ?? "", min, max);
      if (p === "format") e[field] = k.form.dayFormat(l);
      else if (p === "min") e[field] = k.form.dayMin(l, day(l, min));
      else if (p === "max") e[field] = k.form.dayMax(l, day(l, max));
    };
    if (isLifecycle) {
      const b = lifecycleBounds(detail.today, actionOf(detail, key).earliest);
      dateError("from", b.min, b.max);
    }
    if (key === "contract.withdraw" && event?.cancel_required && !cancelReasons.includes(values.cancel)) e.cancel = k.form.cancelMissing(l);
    if (key === "contract.evidence.secured") {
      if (idProblem(values.reference)) e.reference = k.form.referenceBad(l);
      dateError("covered_until", "", detail.today);
    }
    if (key === "contract.evidence.missing") {
      if (!gapReasons.includes(values.reason)) e.reason = k.form.chooseReason(l);
      if (idProblem(values.follow_up)) e.follow_up = k.form.referenceBad(l);
    }
    if (key === "contract.settle") {
      const b = settleBounds(detail.today, end);
      dateError("as_of", b.min, b.max);
    }
    if (key === "contract.reopen" && !reopenReasons.includes(values.reason)) e.reason = k.form.chooseReason(l);
    if (!values.rationale.trim()) e.rationale = k.form.rationaleMissing(l);
    else if (reasonTooLong(values.rationale)) e.rationale = k.form.rationaleLong(l);
    return e;
  };

  const body = (): Values => bodyFor(key, values, event);

  const review = (e: FormEvent) => {
    e.preventDefault();
    const found = validate();
    setErrors(found);
    if (Object.keys(found).length) { setSummaryKey((n) => n + 1); return; }
    setStep("confirm");
  };


  const send = async () => {
    if (busy) return;
    setBusy(true);
    setMessage(null);
    const sent = body();
    const tried = pendingFor(slot, contract, { ...sent, rationale: values.rationale }, detail.revision, attempt);
    try {
      const out = await api.command(commandOf(key), contract, tried.expected, values.rationale, sent, { request: tried.request, key: tried.key });
      settle(slot, contract, "ok");
      clearDraft(slot, contract);
      const when = sent.from || sent.as_of || "";
      onDone(k.done(l, k.command(l, key), when ? day(l, when) : "", k.settlement(l, out.state)));
      onClose(true);
    } catch (err) {
      const failure = err instanceof AdminError ? err.failure : "failed";
      const fields = err instanceof AdminError ? err.fields : [];
      const reaction: Reaction = react(failure, fields);
      settle(slot, contract, reaction);
      setStep(reaction === "conflict" ? "confirm" : "edit");
      if (reaction === "field") {
        const f = fields[0];
        if (f === "event") { setMessage({ text: k.form.eventGone(l), alert: true }); void onReload(); }
        else if (f === "resolves") { setErrors({ resolves: k.form.gapGone(l) }); setSummaryKey((n) => n + 1); void onReload(); }
        else { setErrors({ [controlFor(f)]: serverFieldText(l, controlFor(f)) }); setSummaryKey((n) => n + 1); }
      } else if (reaction === "reason") {
        setErrors({ rationale: k.form.rationaleLong(l) });
        setSummaryKey((n) => n + 1);
      } else if (reaction === "hint") {
        setMessage({ text: k.form.stillBlocked(l), alert: false, hint: true });
        await onReload();
      } else if (reaction === "conflict") {
        setMessage({ text: k.form.conflict(l), alert: false });
        await onReload();
      } else if (reaction === "denied") onDenied();
      else if (reaction === "signin") onSignedOut();
      else if (reaction === "wait") setMessage({ text: sc.limited(l, retryPhrase(l, err instanceof AdminError ? err.retryAfter || 60 : 60)), alert: true });
      else if (reaction === "retry") setMessage({ text: k.form.pending(l, k.command(l, key)), alert: true });
      else if (reaction === "renew") setMessage({ text: k.form.renew(l), alert: true });
      else if (reaction === "partial") setMessage({ text: k.form.partial(l, err instanceof AdminError ? err.operation : ""), alert: true });
      else setMessage({ text: sc.failed(l), alert: true });
    } finally {
      setBusy(false);
    }
  };

  const problems: Problems = Object.entries(errors).map(([f, m]) => ({ field: f === "reason" ? (key === "contract.reopen" ? "o03-f-reopen" : "o03-f-gap") : fieldIds[f] ?? "o03-f-rationale", message: m }));
  const err = (f: string) => errors[f] ? <p className="service-field-error" id={(fieldIds[f] ?? f) + "-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors[f]}</p> : null;
  const described = (f: string, hint = "") => [hint, errors[f] ? (fieldIds[f] ?? f) + "-error" : ""].filter(Boolean).join(" ") || undefined;
  const effective = values.from || values.as_of || "";

  return (
    <div className="o03-form-wrap">
      {pending && step === "edit" && !message && <p className="live-note-rule" role="status">{k.form.pending(l, k.command(l, key))}</p>}
      {message && (
        <div ref={messageRef} tabIndex={-1} className={message.alert ? "live-error" : "live-note-rule"} role={message.alert ? "alert" : "status"}>
          <p>{message.hint && !offered(actionOf(detail, key), isLifecycle) && key !== "contract.withdraw" ? k.blocked(l, key, actionOf(detail, key).blocked) : message.text}</p>
        </div>
      )}
      {step === "edit" ? (
        <form className="o03-form live-step" noValidate onSubmit={review} aria-busy={busy || undefined}>
          {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
          {isLifecycle && (
            <div className="service-field">
              <label htmlFor="o03-f-from">{k.form.from(l)}</label>
              <p className="service-hint" id="o03-f-from-hint">{k.form.fromHint(l, day(l, lifecycleBounds(detail.today, actionOf(detail, key).earliest).min))}</p>
              {err("from")}
              <input id="o03-f-from" type="date" className="o03-date" value={values.from} min={lifecycleBounds(detail.today, actionOf(detail, key).earliest).min}
                max={lifecycleBounds(detail.today, actionOf(detail, key).earliest).max} aria-invalid={!!errors.from || undefined}
                aria-describedby={described("from", "o03-f-from-hint")} onChange={(e) => set("from", e.target.value)} />
            </div>
          )}
          {key === "contract.end" && <p className="live-impact">{k.form.endImpact(l)}</p>}
          {key === "contract.withdraw" && event?.cancel_required && (
            <fieldset className="live-roles" id="o03-f-cancel" tabIndex={-1} aria-describedby={described("cancel", "o03-f-cancel-hint")}>
              <legend>{k.form.cancelLegend(l)}</legend>
              <p className="service-hint" id="o03-f-cancel-hint">{k.form.cancelHint(l)}</p>
              {err("cancel")}
              {cancelReasons.map((r) => (
                <label key={r}><input type="radio" name="o03-cancel" value={r} checked={values.cancel === r} onChange={() => set("cancel", r)} />{k.cancel(l, r)}</label>
              ))}
            </fieldset>
          )}
          {key === "contract.withdraw" && detail.contract.settlement === "pending_final" && <p className="live-note">{k.form.withdrawKeepsGaps(l)}</p>}
          {key === "contract.evidence.secured" && (
            <>
              <div className="service-field">
                <label htmlFor="o03-f-reference">{k.form.reference(l)}</label>
                <p className="service-hint" id="o03-f-reference-hint">{k.form.referenceHint(l)}</p>
                {err("reference")}
                <input id="o03-f-reference" className="o03-id" dir="ltr" maxLength={128} value={values.reference} aria-invalid={!!errors.reference || undefined}
                  aria-describedby={described("reference", "o03-f-reference-hint")} onChange={(e) => set("reference", e.target.value)} />
              </div>
              <div className="service-field">
                <label htmlFor="o03-f-covered">{k.form.covered(l)}</label>
                {end && !coversEnd(values.covered_until, end) && <p className="service-hint" id="o03-f-covered-hint">{k.form.coverEnd(l, day(l, end))}</p>}
                {err("covered_until")}
                <input id="o03-f-covered" type="date" className="o03-date" value={values.covered_until} max={detail.today} aria-invalid={!!errors.covered_until || undefined}
                  aria-describedby={described("covered_until", end && !coversEnd(values.covered_until, end) ? "o03-f-covered-hint" : "")} onChange={(e) => set("covered_until", e.target.value)} />
              </div>
              {openGaps(detail.evidence).length > 0 && (
                <fieldset className="live-roles" id="o03-f-resolves" tabIndex={-1}>
                  <legend>{k.form.resolvesLegend(l)}</legend>
                  {err("resolves")}
                  <p className="service-hint">{k.form.resolvesMax(l, maxResolves)}</p>
                  {openGaps(detail.evidence).map((g) => {
                    const chosen = values.resolves ? values.resolves.split(",") : [];
                    return (
                      <label key={g.write} className="live-check">
                        <input type="checkbox" checked={chosen.includes(g.write)} disabled={!chosen.includes(g.write) && chosen.length >= maxResolves}
                          onChange={(e) => set("resolves", (e.target.checked ? [...chosen, g.write] : chosen.filter((x) => x !== g.write)).join(","))} />
                        {k.gap(l, g.reason)} {ltr(g.write)}
                      </label>
                    );
                  })}
                </fieldset>
              )}
              <p className="live-note">{k.form.owner(l, shared.session.user)}</p>
            </>
          )}
          {key === "contract.evidence.missing" && (
            <>
              <div className="service-field">
                <label htmlFor="o03-f-gap">{k.form.gapReason(l)}</label>
                {err("reason")}
                <select id="o03-f-gap" className="live-select" value={values.reason} aria-invalid={!!errors.reason || undefined} aria-describedby={described("reason")}
                  onChange={(e) => set("reason", e.target.value)}>
                  <option value="">{k.form.chooseReason(l)}</option>
                  {gapReasons.map((r) => <option key={r} value={r}>{k.gap(l, r)}</option>)}
                </select>
                {values.reason === "access_withdrawn_emergency" && <p className="service-hint">{k.form.emergency(l)}</p>}
              </div>
              <div className="service-field">
                <label htmlFor="o03-f-follow">{k.form.followUp(l)}</label>
                <p className="service-hint" id="o03-f-follow-hint">{k.form.referenceHint(l)}</p>
                {err("follow_up")}
                <input id="o03-f-follow" className="o03-id" dir="ltr" maxLength={128} value={values.follow_up} aria-invalid={!!errors.follow_up || undefined}
                  aria-describedby={described("follow_up", "o03-f-follow-hint")} onChange={(e) => set("follow_up", e.target.value)} />
              </div>
              <p className="live-note">{k.form.owner(l, shared.session.user)}</p>
            </>
          )}
          {key === "contract.settle" && (
            <div className="service-field">
              <label htmlFor="o03-f-asof">{k.form.asOf(l)}</label>
              {err("as_of")}
              <input id="o03-f-asof" type="date" className="o03-date" value={values.as_of} min={settleBounds(detail.today, end).min || undefined} max={detail.today}
                aria-invalid={!!errors.as_of || undefined} aria-describedby={described("as_of")} onChange={(e) => set("as_of", e.target.value)} />
            </div>
          )}
          {key === "contract.reopen" && (
            <fieldset className="live-roles" id="o03-f-reopen" tabIndex={-1}>
              <legend>{k.form.reopenCause(l)}</legend>
              {err("reason")}
              {reopenReasons.map((r) => (
                <label key={r}><input type="radio" name="o03-reopen" value={r} checked={values.reason === r} onChange={() => set("reason", r)} />{k.reopenCause(l, r)}</label>
              ))}
            </fieldset>
          )}
          <div className="service-field live-reason">
            <label htmlFor="o03-f-rationale">{k.form.rationale(l)}</label>
            <p className="service-hint" id="o03-f-rationale-hint">{k.form.rationaleHint(l, utf8Bytes(values.rationale))}</p>
            {err("rationale")}
            <textarea id="o03-f-rationale" rows={3} value={values.rationale} aria-invalid={!!errors.rationale || undefined}
              aria-describedby={described("rationale", "o03-f-rationale-hint")} onChange={(e) => set("rationale", e.target.value)} />
          </div>
          <div className="live-buttons">
            <button type="submit" className="live-secondary">{k.form.review(l)}</button>
            <button type="button" className="service-button-link" onClick={() => onClose()}>{k.form.cancel(l)}</button>
          </div>
        </form>
      ) : (
        <div className="o03-confirm" aria-busy={busy || undefined}>
          <h3 id="o03-confirm-title" ref={confirmTitle} tabIndex={-1}>{k.form.confirmTitle(l)}</h3>
          <dl className="live-facts">
            <div><dt>{k.colContract(l)}</dt><dd><bdi>{detail.contract.name || contract}</bdi> {ltr(contract)}</dd></div>
            <div><dt>{k.form.action(l)}</dt><dd>{k.command(l, key)}{event ? " — " + k.event(l, event.kind) + " " + day(l, event.from) : ""}</dd></div>
            {effective && <div><dt>{k.form.effective(l)}</dt><dd>{key === "contract.settle" ? k.asOf(l, day(l, effective)) : k.form.effectiveDay(l, day(l, effective))}</dd></div>}
            {key === "contract.evidence.secured" && <div><dt>{k.form.reference(l)}</dt><dd>{ltr(values.reference)} · {k.form.covered(l)} {day(l, values.covered_until)}</dd></div>}
            {key === "contract.evidence.missing" && <div><dt>{k.form.gapReason(l)}</dt><dd>{k.gap(l, values.reason)} · {ltr(values.follow_up)}</dd></div>}
            {key === "contract.reopen" && <div><dt>{k.form.reopenCause(l)}</dt><dd>{k.reopenCause(l, values.reason)}</dd></div>}
            {key === "contract.withdraw" && values.cancel && <div><dt>{k.form.cancelLegend(l)}</dt><dd>{k.cancel(l, values.cancel)}</dd></div>}
            <div><dt>{k.form.rationale(l)}</dt><dd><bdi>{values.rationale}</bdi></dd></div>
          </dl>
          {key === "contract.end" && <p className="live-impact">{k.form.endDay(l, day(l, values.from))}</p>}
          {key === "contract.withdraw" && detail.contract.settlement === "pending_final" && (
            <p className="live-impact">{event?.cancel_required ? k.form.withdrawReopens(l) + " " : ""}{k.form.withdrawKeepsGaps(l)}</p>
          )}
          {key === "contract.settle" && <p className="live-impact">{k.form.settleImpact(l)}</p>}
          <div className="live-buttons">
            <button type="button" className="live-primary" disabled={busy} onClick={() => void send()}>{busy ? k.form.sending(l) : k.confirmVerb(l, k.command(l, key))}</button>
            <button type="button" className="live-secondary" disabled={busy} onClick={() => setStep("edit")}>{k.form.backToEdit(l)}</button>
          </div>
        </div>
      )}
    </div>
  );
}

function serverFieldText(l: Locale, field: string): string {
  if (field === "from" || field === "as_of" || field === "covered_until") return k.form.dayFormat(l);
  if (field === "reason") return k.form.chooseReason(l);
  if (field === "cancel") return k.form.cancelMissing(l);
  if (field === "reference" || field === "follow_up") return k.form.referenceBad(l);
  if (field === "resolves") return k.form.gapGone(l);
  return k.form.checkValue(l);
}
