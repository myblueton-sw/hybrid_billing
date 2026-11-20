import { useEffect, useRef, useState, type ReactNode } from "react";
import { ErrorDiagnostics } from "../../../../service/CommandDiagnostics";
import { AdminError } from "../../../../service/admin";
import { useAdminClient, useService } from "../../../../service/context";
import type { Locale } from "../../../../shared/types";
import { allowed, freshAttempt, reasonValid, reviewOf, uncertain, type Decision, type Intent, type Review } from "./model";
import { copyFor, type CopyKey } from "./copy";
import "./style.css";

type Props = { locale: Locale; objectId?: string; onDirty: (dirty: boolean) => void };
export default function InputReview(props: Props) {
  const service = useService();
  return <ReviewScreen key={`${props.objectId}:${service?.session.workspace}:${service?.session.user}:${service?.session.csrf}`} {...props} />;
}
function ReviewScreen({ locale, objectId, onDirty }: Props) {
  const client = useAdminClient(), t = copyFor(locale), session = useService()?.session;
  const [review, setReview] = useState<Review | null>(null), [reading, setReading] = useState(false);
  const [readError, setReadError] = useState<unknown>(null), [reason, setReason] = useState("");
  const [intent, setIntent] = useState<Intent | null>(null), [sending, setSending] = useState(false);
  const [notice, setNotice] = useState<"unknown" | "receipt" | "refused" | null>(null);
  const [failure, setFailure] = useState<unknown>(null);
  const [diagnostic, setDiagnostic] = useState<{ request: string; operation: string } | null>(null);
  const epoch = useRef(0), mounted = useRef(false), busy = useRef(false);
  const refresh = useRef<HTMLButtonElement>(null);
  const dialog = useRef<HTMLDialogElement>(null), cancel = useRef<HTMLButtonElement>(null);
  const trigger = useRef<HTMLButtonElement | null>(null);
  async function read() {
    if (!objectId || busy.current) return;
    const token = ++epoch.current;
    setReview(null); setReadError(null); setReading(true); setIntent(null);
    try {
      const out = await client.command("input.inspect", objectId, "", "", {}, { request: freshAttempt().request, key: "" });
      const next = reviewOf(out, objectId, session?.workspace ?? "");
      if (mounted.current && token === epoch.current) setReview(next);
    } catch (error) {
      if (mounted.current && token === epoch.current) {
        setReadError(error);
        if (error instanceof AdminError && ["unauthenticated", "denied"].includes(error.failure)) setReason("");
      }
    } finally { if (mounted.current && token === epoch.current) setReading(false); }
  }
  useEffect(() => {
    mounted.current = true;
    setFailure(null); setReason(""); setNotice(null); setDiagnostic(null); setReview(null); setIntent(null);
    void read();
    return () => { mounted.current = false; epoch.current++; };
  }, [objectId]);
  useEffect(() => { onDirty(sending || !!intent || !!reason); return () => onDirty(false); }, [sending, intent, reason, onDirty]);
  useEffect(() => {
    if (intent && !dialog.current?.open) { dialog.current?.showModal(); cancel.current?.focus(); }
    if (!intent && dialog.current?.open) dialog.current.close();
  }, [intent]);
  function close() { if (!busy.current) { setIntent(null); trigger.current?.focus(); } }
  function open(decision: Decision, button: HTMLButtonElement) {
    if (!review || reading || busy.current || !reasonValid(reason) || !allowed(review, decision)) return;
    trigger.current = button;
    setIntent({ decision, reason, review, attempt: freshAttempt() });
  }
  async function submit() {
    if (!intent || !objectId || busy.current) return;
    const selected = intent, token = ++epoch.current;
    busy.current = true; setSending(true); setReview(null); setReadError(null);
    try {
      const out = await client.command(`input.${selected.decision}`, objectId, selected.review.revision, selected.reason,
        { digest: selected.review.fields.digest, scope_revision: selected.review.fields.scope_revision }, selected.attempt);
      if (out.status !== "completed" || out.fields.receipt !== "historical_decision" || out.fields.review !== objectId ||
        out.fields.digest !== selected.review.fields.digest || out.fields.decision_reason !== selected.reason || out.fields.decided_by !== session?.user ||
        out.fields.recorded_scope_revision !== String(Number(selected.review.fields.scope_revision) + (selected.decision === "confirm_publish" ? 1 : 0)) ||
        out.fields.recorded_state !== (selected.decision === "confirm_publish" ? "eligible" : "rejected")) throw new AdminError("unavailable");
      if (mounted.current && token === epoch.current) { setFailure(null); setNotice("receipt"); setReason(""); setDiagnostic(null); }
    } catch (error) {
      if (mounted.current && token === epoch.current) {
        setFailure(error); setNotice(uncertain(error, notice === "unknown") ? "unknown" : "refused");
        setDiagnostic(previous => notice === "unknown" && previous ? previous : { request: selected.attempt.request, operation: error instanceof AdminError && /^[A-Za-z0-9._:-]{1,128}$/.test(error.operation) ? error.operation : "" });
        if (error instanceof AdminError && error.failure === "unauthenticated") { setReason(""); setDiagnostic(null); }
      }
    } finally {
      busy.current = false;
      if (mounted.current && token === epoch.current) { setSending(false); setIntent(null); requestAnimationFrame(() => refresh.current?.focus()); }
    }
  }
  const f = review?.fields;
  const rows = (fields: Record<string, string>, keys: CopyKey[]) => keys.map(k => <div key={k}><dt>{t(k)}</dt><dd>{k === "scope_revision" && fields[k] === "0" ? t("none") : k === "decision_reason" ? <span dir="auto">{fields[k]}</span> : <bdi dir="ltr">{fields[k]}</bdi>}</dd></div>);
  const section = (id: string, title: CopyKey, content: ReactNode) => <section data-review-item={`S06:input:${id}`}><h3>{t(title)}</h3>{content}</section>;
  const guidance: CopyKey = failure instanceof AdminError ?
    failure.failure === "unauthenticated" ? "signIn" : ["reauth", "reauth_cancelled"].includes(failure.failure) ? "reauth" :
    failure.failure === "denied" ? "noPermission" : ["conflict", "evidence_stale", "approval_stale"].includes(failure.failure) ? "stale" : "reloadRequired" : "reloadRequired";
  const terminal = f && ["eligible", "rejected"].includes(f.recorded_state);
  const unavailable = f && (f.confirmation_permission !== "allowed" ? "noPermission" : f.current_evidence !== "matched" ? "evidenceNeeded" : f.confirmation_hold === "blocked" ? "hold" : null);
  return <div className="input-review" dir={locale === "ar" ? "rtl" : "ltr"}>
    <header className="ir-heading"><div><h2>{t("title")}</h2><span className="ir-tag">{t("sample")}</span></div><button ref={refresh} type="button" disabled={!objectId || reading || sending || !!intent} onClick={() => void read()}>{t("refresh")}</button></header>
    <p>{t("limits")}</p>
    <div data-review-item="S06:input:async" aria-live="polite" aria-atomic="true">
      {!objectId && <p>{t("empty")}</p>}{reading && <p role="status">{t("loading")}</p>}
      {!!readError && <div className="ir-notice" role="alert"><p>{t(readError instanceof AdminError && readError.failure === "denied" ? "denied" : "failed")}</p><ErrorDiagnostics l={locale} error={readError} /></div>}
      {notice && <div className="ir-notice"><p>{t(notice)}</p>{!!failure && <><p>{t(guidance)}</p><ErrorDiagnostics l={locale} error={failure} /></>}{diagnostic && <div>{notice === "unknown" && <p>{t("earlier")}</p>}<dl>{rows(diagnostic, ["request"])}{diagnostic.operation && rows(diagnostic, ["operation"])}</dl></div>}</div>}
    </div>
    {f && review && <><dl className="ir-status" data-review-item="S06:input:status">
      {([["eligibility", f.current_eligibility], ["evidence", f.current_evidence], ["state", f.recorded_state]] as [CopyKey, CopyKey][]).map(([label, value]) => <div key={label}><dt>{t(label)}</dt><dd>{t(value)}</dd></div>)}
    </dl><div className="ir-grid"><div className="ir-evidence">
      {section("scope", "scope", <dl>{rows(f, ["workspace", "provider", "account", "period"])}<div><dt>{t("scope_revision")}</dt><dd>{f.scope_revision === "0" ? t("none") : <bdi dir="ltr">{f.scope_revision}</bdi>}</dd></div></dl>)}
      {section("document", "document", <><dl>{rows(f, ["document", "document_revision"])}<div><dt>{t("document_total")}</dt><dd className="ir-money"><bdi dir="ltr">{f.document_currency} {f.document_total}</bdi></dd></div>{rows(f, ["document_digest"])}</dl><p>{t("fixture")}</p></>)}
      {section("comparison", "comparison", <dl>{rows(f, ["method"])}<div><dt>{t("recorded_verdict")}</dt><dd>{t(f.recorded_verdict === "match" ? "match" : "difference")}</dd></div>{rows(f, ["compared_at", "digest"])}</dl>)}
      {terminal && section("decision", "decision", <><p>{t("historical")}</p><dl>{rows(f, ["decided_by", "decided_at", "decision_reason", "recorded_scope_revision"])}</dl></>)}
    </div><section className="ir-actions" data-review-item="S06:input:actions"><h3>{t("actions")}</h3>
      {terminal ? <p>{t("terminal")}</p> : <>{unavailable && <p>{t(unavailable)}</p>}
        <label htmlFor="ir-reason">{t("reason")}</label><input dir="auto" id="ir-reason" value={reason} onChange={e => setReason(e.target.value)} aria-describedby={`ir-reason-help${reason && !reasonValid(reason) ? " ir-reason-error" : ""}`} aria-invalid={!!reason && !reasonValid(reason)} autoComplete="off" disabled={sending || !!intent || !allowed(review, "confirm_publish") && !allowed(review, "reject")} />
        {!!reason && !reasonValid(reason) && <p id="ir-reason-error" role="alert">{t("reasonInvalid")}</p>}
        <p id="ir-reason-help">{t("reasonHelp")}</p><div className="ir-buttons">
          <button type="button" disabled={!reasonValid(reason) || !allowed(review, "confirm_publish") || !!intent} onClick={e => open("confirm_publish", e.currentTarget)}>{t("confirm")}</button>
          <button type="button" disabled={!reasonValid(reason) || !allowed(review, "reject") || !!intent} onClick={e => open("reject", e.currentTarget)}>{t("reject")}</button>
        </div></>}
    </section></div></>}
    <dialog ref={dialog} className="ir-dialog" aria-labelledby="ir-dialog-title" onCancel={e => { e.preventDefault(); close(); }}>
      {intent && <><h2 id="ir-dialog-title">{t(intent.decision === "confirm_publish" ? "confirm" : "reject")}</h2><p>{t("readback")}</p><p>{t(intent.decision === "confirm_publish" ? "confirmEffect" : "rejectEffect")}</p>
        {notice === "unknown" && <p>{t("unknown")}</p>}
        <dl>{rows(intent.review.fields, ["review", "workspace", "provider", "account", "period", "document", "digest", "scope_revision"])}
          <div><dt>{t("document_total")}</dt><dd><bdi dir="ltr">{intent.review.fields.document_currency} {intent.review.fields.document_total}</bdi></dd></div>
          <div><dt>{t("reason")}</dt><dd dir="auto">{intent.reason}</dd></div></dl>
        <p role="status">{sending ? t("pending") : ""}</p><div className="ir-buttons"><button ref={cancel} type="button" disabled={sending} onClick={close}>{t("cancel")}</button>
          <button type="button" disabled={sending} onClick={() => void submit()}>{t(intent.decision === "confirm_publish" ? "confirm" : "reject")}</button></div></>}
    </dialog>
  </div>;
}
