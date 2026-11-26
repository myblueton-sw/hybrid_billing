import { useEffect, useRef, useState, useSyncExternalStore, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import type { CredentialDetail } from "../../../../service/service-query-types";
import { FieldMessage, ValidationSummary, fieldAttributes, rationaleError, requestBounds, useFormValidation, type FieldErrors } from "./form-validation";
import { w } from "../users/live-copy";
import { c } from "../../../../service/copy";
import { v } from "./live-copy";
import { r } from "./rotate-copy";
import { CommandFailure } from "./command-failure";
import { admitsIssueDraft } from "./issue-draft";
import { issueExec } from "./issue-send";
import { Facts, bdi, usePanelHeading } from "./query-ui";
import { BoundedReason, Reauthenticate, usePresent } from "./command-ui";
import type { Recovery } from "./recovery-context";
import { rotationEnd, utcTime, wholeDays, wholeMinutes } from "./live-model";

export function RotateControl({ l, api, recovery, detail, user, refresh, show, cancel }: {
  l: Locale; api: AdminClient; recovery: Recovery; detail: CredentialDetail; user: string; refresh: () => void; show: (service: string) => void; cancel: () => void;
}) {
  const { issues } = recovery;
  useSyncExternalStore(issues.subscribe, issues.version);
  const present = usePresent();
  const heading = usePanelHeading("rotate:" + detail.row.id);
  const service = detail.row.service;
  const held = issues.get(user, service);
  const entry = held?.rotate?.credential === detail.row.id ? held : undefined;
  const other = !!held && !entry;
  const refusal = entry ? null : issues.refusal(user, service);
  const lim = detail.limits;
  const [days, setDays] = useState(String(lim.credential_days));
  const [overlap, setOverlap] = useState(String(lim.credential_overlap_minutes));
  const [reason, setReason] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const reviewTitle = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (reviewing) reviewTitle.current?.focus(); }, [reviewing]);
  const validation = useFormValidation();
  const { errors } = validation;
  const shown = { days: entry?.days ?? days, overlap: entry?.rotate?.overlap ?? overlap, reason: entry?.reason ?? reason };
  const minutes = wholeMinutes(shown.overlap, lim.secret_pickup_minutes, lim.credential_overlap_max_minutes);
  const oldEnd = minutes ? rotationEnd(detail.observed_at, detail.row.not_after, minutes) : null;
  const allowed = detail.eligibility.can_rotate;
  const edit = <T,>(set: (value: T) => void, field: "days" | "overlap" | "reason") => (value: T) => {
    if (!present.current || !issues.current() || held) return;
    set(value);
    setReviewing(false);
    validation.clear(field);
  };
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (!present.current || !issues.current() || other) return;
    if (entry && entry.phase !== "unknown") return;
    if (!entry) {
      const fields: FieldErrors = {};
      if (!wholeDays(days, lim.credential_days)) fields.days = v.errDays(l, lim.credential_days);
      if (minutes === null) fields.overlap = r.errOverlap(l, lim.secret_pickup_minutes, lim.credential_overlap_max_minutes);
      const why = rationaleError(l, reason, lim.rationale_bytes);
      if (why) fields.reason = why;
      if (Object.keys(fields).length) { setReviewing(false); validation.fail(fields); return; }
      if (!allowed) { setReviewing(false); validation.fail({}, r.gone(l, detail.row.id)); return; }
      if (!admitsIssueDraft({ target: detail.row.id, days, reason }, lim)) { setReviewing(false); validation.fail({}, requestBounds(l)); return; }
      validation.reset();
      if (!reviewing) { setReviewing(true); return; }
    }
    setReviewing(false);
    await issues.submit(user, service, { days: shown.days, reason: shown.reason, rotate: { credential: detail.row.id, overlap: String(minutes ?? shown.overlap) } },
      detail.revision, lim.replay_seconds, issueExec(api, service));
    if (present.current && issues.current()) refresh();
  };
  const time = (iso: string) => <bdi dir="ltr" className="live-nowrap">{utcTime(l, iso).replace(/[\u061c\u200e\u200f]/g, "")}</bdi>;
  if (other)
    return (
      <section className="live-step" aria-labelledby="svc-rotate-title">
        <h2 id="svc-rotate-title" tabIndex={-1} ref={heading}>{r.title(l)}</h2>
        <p className="live-impact">{v.attemptsDiscardNote(l)}</p>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={cancel}>{r.cancel(l)}</button></div>
      </section>
    );
  return (
    <form className="live-step" onSubmit={submit} noValidate aria-labelledby="svc-rotate-title">
      <h2 id="svc-rotate-title" tabIndex={-1} ref={heading}>{r.title(l)}</h2>
      <p className="live-who">{r.current(l)}: <span className="live-mono">{bdi(detail.row.id)}</span></p>
      {!entry && !reviewing && <p className="live-impact">{r.lead(l)}</p>}
      <Reauthenticate l={l} state={detail.eligibility.rotate_step_up_state} refresh={refresh} />
      {!entry && !allowed && <p>{r.gone(l, detail.row.id)}</p>}
      <ValidationSummary l={l} validation={validation} />
      {reviewing && oldEnd ? (
        <section className="live-review" aria-labelledby="svc-review-title">
          <h3 id="svc-review-title" tabIndex={-1} ref={reviewTitle}>{r.reviewTitle(l)}</h3>
          <Facts items={[
            [r.willEnd(l), <span className="live-mono">{bdi(detail.row.id)}</span>],
            [oldEnd.sooner ? r.endsAt(l) : r.endsAbout(l), <>{time(oldEnd.at)}<span className="live-to live-sub">{oldEnd.sooner ? r.oldEndsSooner(l, utcTime(l, oldEnd.at)) : r.fromConfirm(l)}</span></>],
            [r.newValidFor(l), `${shown.days} ${v.days(l)}`],
            [r.overlap(l), `${shown.overlap} ${r.minutes(l)}`],
            [w.reason(l), shown.reason],
          ]} />
          <ul className="live-review-list">
            <li>{r.once(l, lim.secret_pickup_minutes)}</li>
            <li>{r.notMovedBack(l)}</li>
            <li>{r.notKnown(l)}</li>
            <li>{r.exposed(l)}</li>
          </ul>
        </section>
      ) : (
        <fieldset disabled={!!entry}>
          <label>
            {r.newValidFor(l)}
            <input {...fieldAttributes("days", errors.days, "svc-days-hint")} inputMode="numeric" value={shown.days}
              onChange={(e) => { if (e.target.value.length <= 3) edit(setDays, "days")(e.target.value); }} />
          </label>
          <FieldMessage field="days" error={errors.days} />
          <p id="svc-days-hint">{v.daysHint(l, lim.credential_days)}</p>
          <label>
            {r.overlap(l)}
            <input {...fieldAttributes("overlap", errors.overlap, "svc-overlap-hint")} inputMode="numeric" dir="ltr" value={shown.overlap}
              onChange={(e) => { if (e.target.value.length <= 6) edit(setOverlap, "overlap")(e.target.value); }} />
          </label>
          <FieldMessage field="overlap" error={errors.overlap} />
          <p id="svc-overlap-hint">{r.overlapHint(l, lim.secret_pickup_minutes, lim.credential_overlap_max_minutes, lim.credential_overlap_minutes)}</p>
          {oldEnd && <p className="live-note" aria-live="polite">{oldEnd.sooner ? r.oldEndsSooner(l, utcTime(l, oldEnd.at)) : r.oldEnds(l, utcTime(l, oldEnd.at))}</p>}
          <BoundedReason l={l} limit={lim.rationale_bytes} value={shown.reason} onChange={edit(setReason, "reason")} error={errors.reason ?? ""} />
        </fieldset>
      )}
      {entry && (
        <p role="status">
          {entry.phase === "unknown" ? r.unknown(l, service).join(" ")
            : entry.phase === "issued" ? r.rotated(l, entry.issued!.credential, detail.row.id, service)
              : entry.phase === "sending" ? r.sending(l, service) : r.lapsed(l, service).join(" ")}
        </p>
      )}
      {(entry?.error != null || refusal != null) && (
        <CommandFailure l={l} error={entry?.error ?? refusal?.error}
          outcome={entry?.phase === "unknown" || entry?.phase === "lapsed" || refusal?.retry ? "unknown" : "refused"} />
      )}
      <div className="live-buttons">
        <button className="live-primary" type="submit" disabled={entry ? entry.phase !== "unknown" : !allowed}>
          {entry?.phase === "unknown" ? c.tryAgain(l) : reviewing ? r.title(l) : r.review(l)}
        </button>
        {reviewing && !entry && <button className="live-secondary" type="button" onClick={() => setReviewing(false)}>{r.edit(l)}</button>}
        {entry?.phase === "issued" && <button className="live-secondary" type="button" onClick={() => show(service)}>{v.showSecret(l)}</button>}
        {entry && entry.phase !== "sending" && entry.phase !== "issued" && (
          <button className="live-secondary" type="button" onClick={() => issues.discard(user, service)}>{v.discardAttempt(l)}</button>
        )}
        {!entry && <button className="service-button-link" type="button" onClick={cancel}>{r.cancel(l)}</button>}
      </div>
    </form>
  );
}
