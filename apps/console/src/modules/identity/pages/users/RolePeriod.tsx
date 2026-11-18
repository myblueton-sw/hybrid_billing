import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type Attempt, type ClockState, type SettingRow, type UserRow } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { Summary } from "../../../../service/Service";
import { p } from "./period-copy";
import { bounds, dateParts, direction, endProblem, endFields, initialEnd, localInput, remaining, setting, utcOf, zoneOf, type EndValue } from "./period-model";

export { endFields, initialEnd, type EndValue };
import { b } from "./bulk-copy";
import { failureText as failure } from "./live-model";
import { w } from "./live-copy";
import { FailureBox, Reason, reasonProblem, settles } from "./Live";
import { At, fill, Stamp } from "./Stamp";

export function Emph({ l, text, iso = "" }: { l: Locale; text: string; iso?: string }) {
  const [first, ...rest] = text.split(l === "ar" ? /(?<=[.؟])\s/ : /(?<=[.?])\s/);
  const body = (s: string) => (iso && s.includes("{at}") ? <At l={l} template={s} iso={iso} /> : s);
  return <><strong>{body(first)}</strong>{rest.length > 0 && <> {body(rest.join(" "))}</>}</>;
}

export function PeriodCell({ l, u, serverTime }: { l: Locale; u: UserRow; serverTime: string }) {
  const m = u.membership;
  if (!m) return <><span aria-hidden="true">—</span><span className="visually-hidden">{p.noPeriod(l)}</span></>;
  const unconfirmed = m.ledger === "unconfirmed" && <span className="live-sub live-strong">{p.unconfirmed(l)}</span>;
  if (m.role_period === "ended" || m.suspended_for === "role_period_ended") {
    return <><span className="live-strong m01-line">{p.ended(l)}</span>{m.role_until && <span className="live-sub m01-dated">{p.endedL(l)} <Stamp l={l} iso={m.role_until} zone={false} /></span>}{unconfirmed}</>;
  }
  if (m.role_period === "open") return <><span className="m01-line">{p.noEnd(l)}</span>{unconfirmed}</>;
  if (m.role_period === "ends_soon") {
    const left = remaining(m.role_until, serverTime);
    return (
      <>
        <span className="live-strong m01-line">{left.today ? <At l={l} template={p.endsToday(l)} iso={m.role_until} date={false} zone={false} /> : p.endsIn(l, left.days)}</span>
        <span className="live-sub m01-dated"><At l={l} template={p.until(l)} iso={m.role_until} /></span>
        {unconfirmed}
      </>
    );
  }
  return (
    <>
      <span className="m01-line"><At l={l} template={p.until(l)} iso={m.role_until} time={false} zone={false} /></span>
      <span className="live-sub"><Stamp l={l} iso={m.role_until} date={false} /></span>
      {unconfirmed}
    </>
  );
}

export function PeriodFact({ l, u, serverTime, onJob }: { l: Locale; u: UserRow; serverTime: string; onJob: (id: string) => void }) {
  const m = u.membership;
  return (
    <>
      {m && (
        <div>
          <dt>{p.rolePeriod(l)}</dt>
          <dd>
            {m.role_from && <span className="m01-line m01-dated">{p.fromL(l)} <Stamp l={l} iso={m.role_from} /></span>}
            <PeriodCell l={l} u={u} serverTime={serverTime} />
          </dd>
        </div>
      )}
      {u.revocation && (
        <div>
          <dt>{b.latestRevocation(l)}</dt>
          <dd>
            <button type="button" className="service-button-link m01-link" onClick={() => onJob(u.revocation!.bulk)}>{b.bulkN(l, u.revocation.bulk.replace("bulk-revocation-", ""))}</button>
          </dd>
        </div>
      )}
    </>
  );
}

export function RoleEnds({ l, id, value, onChange, own, clock, settings, error }: {
  l: Locale; id: string; value: EndValue; onChange: (v: EndValue) => void; own: string; clock: ClockState; settings: SettingRow[]; error: string;
}) {
  const bnd = bounds(clock.server_time, setting(settings, "role_period_max_days", 366), own);
  const utc = utcOf(value.local);
  return (
    <>
      <fieldset className="live-roles m01-end">
        <legend>{p.roleEnds(l)}</legend>
        <label>
          <input type="radio" name={id + "-end"} checked={value.kind === "at"} onChange={() => onChange({ ...value, kind: "at" })} />
          {p.endDate(l)}
        </label>
        <label>
          <input type="radio" name={id + "-end"} checked={value.kind === "none"} disabled={!!own} aria-describedby={own ? id + "-why" : undefined}
            onChange={() => onChange({ ...value, kind: "none" })} />
          {p.noEnd(l)}
        </label>
        {own && <p className="m01-why" id={id + "-why"}><At l={l} template={p.ownLimit(l)} iso={own} time={false} zone={false} /></p>}
      </fieldset>
      {value.kind === "at" && (
        <div className="service-field m01-when-field">
          <label htmlFor={id}>{p.endDateTime(l)}</label>
          <p className="service-hint" id={id + "-hint"}>{fill(p.zone(l, "{at}"), <bdi dir="ltr">{zoneOf(new Date())}</bdi>)}</p>
          {error && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{c.errorPrefix(l)} </span>{error}</p>}
          <input id={id} type="datetime-local" step={60} dir="ltr" value={value.local} min={bnd.min ? localInput(bnd.min) : undefined} max={bnd.max ? localInput(bnd.max) : undefined}
            aria-invalid={error ? true : undefined} aria-describedby={[id + "-hint", error ? id + "-error" : ""].filter(Boolean).join(" ")}
            onChange={(e) => onChange({ ...value, local: e.target.value })} />
          <p className="m01-utc">{p.sentAs(l)} <bdi dir="ltr" className="live-mono">{utc || "—"}</bdi></p>
          <dl className="m01-limits">
            <div><dt>{p.serverNow(l)}</dt><dd><Stamp l={l} iso={clock.server_time} /></dd></div>
            {bnd.max && <div><dt>{p.latest(l)}</dt><dd><Stamp l={l} iso={bnd.max} /></dd></div>}
            {own && <div><dt>{p.own(l)}</dt><dd><Stamp l={l} iso={own} /></dd></div>}
          </dl>
        </div>
      )}
    </>
  );
}

export function endError(l: Locale, v: EndValue, clock: ClockState, settings: SettingRow[], own: string): string {
  if (v.kind === "none") return "";
  const problem = endProblem(utcOf(v.local), bounds(clock.server_time, setting(settings, "role_period_max_days", 366), own));
  return problem === "missing" ? p.enterEnd(l) : problem ? p.capRefused(l) : "";
}

export function Settings({ l, settings }: { l: Locale; settings: SettingRow[] }) {
  const cols = b.settingCols(l);
  return (
    <details className="m01-settings">
      <summary>{p.settings(l)}</summary>
      <div className="security-table-scroll">
        <table>
          <thead><tr>{cols.map((h) => <th key={h} scope="col">{h}</th>)}</tr></thead>
          <tbody>
            {settings.map((s) => (
              <tr key={s.name}>
                <td data-label={cols[0]}>{b.settingName(l, s.name)}</td>
                <td data-label={cols[1]} className="m01-num"><bdi dir="ltr">{s.value}</bdi> {s.unit}</td>
                <td data-label={cols[2]} className="m01-num"><bdi dir="ltr">{s.default}</bdi></td>
                <td data-label={cols[3]} className="m01-num"><bdi dir="ltr">{s.min}–{s.max}</bdi></td>
                <td data-label={cols[4]}>{b.source(l, s.source)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  );
}

export function PeriodForm({ l, user, own, clock, settings, api, revision, onDone, onCancel, onReload }: {
  l: Locale; user: UserRow; own: string; clock: ClockState; settings: SettingRow[]; api: AdminClient; revision: string;
  onDone: (notice: string[]) => void; onCancel: () => void; onReload: () => void;
}) {
  const m = user.membership!;
  const [end, setEnd] = useState<EndValue>(() => initialEnd(m.role_period === "ended" ? "" : m.role_until, m.role_period === "ended" ? own : ""));
  const [reason, setReason] = useState("");
  const [problem, setProblem] = useState({ end: "", reason: "" });
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { tried.current = attempt(); }, [revision]);
  const { values, utc } = endFields(end);
  const dir = direction(m, utc);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found = { end: endError(l, end, clock, settings, own), reason: reasonProblem(l, reason) };
    setProblem(found);
    setError(null);
    if (found.end || found.reason) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      const out = await api.command("identity.set_role_period", user.user_id, revision, reason, { end: end.kind, ...values }, tried.current);
      const durable = out.fields.durable;
      onDone(durable === "no" ? p.durableNo(l) : durable === "unknown" ? p.durableUnknown(l) : out.fields.ledger === "unconfirmed" ? [p.unconfirmedNotice(l)] : [p.done(l)]);
    } catch (err) {
      setError(err);
      if (err instanceof AdminError && err.failure === "invalid") setProblem({ end: p.capRefused(l), reason: "" });
      if (err instanceof AdminError && err.failure === "stale") tried.current = attempt();
      if (settles(err)) onReload();
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const problems = [problem.end && { field: "m01-period-end", message: problem.end }, problem.reason && { field: "live-reason", message: problem.reason }].filter(Boolean) as { field: string; message: string }[];
  const describe = (lc: Locale, e: unknown) => {
    if (e instanceof AdminError && e.failure === "stage") return { title: w.saveFailed(lc), body: m.role_period === "ended" ? p.ended_refused(lc) : p.unchanged(lc) };
    if (e instanceof AdminError && e.failure === "refused") return { title: w.refusedTitle(lc), body: b.blocked(lc) };
    return failure(lc, e);
  };
  return (
    <form className="live-step m01-period-form" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{p.change(l)}</h2>
      <p className="live-who"><bdi dir="ltr">{user.login || user.user_id}</bdi></p>
      {(clock.clock_behind || clock.clock_ahead) && <p className="m01-clock">{p.clockDiffers(l)}</p>}
      {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
      {error !== null && !(error instanceof AdminError && error.failure === "invalid") && <FailureBox l={l} error={error} focusKey={focusKey} describe={describe} />}
      <div className="m01-current">{p.current(l)} <PeriodCell l={l} u={user} serverTime={clock.server_time} /></div>
      <RoleEnds l={l} id="m01-period-end" value={end} onChange={setEnd} own={own} clock={clock} settings={settings} error={problem.end} />
      {dir !== "same" && (
        <p className="live-impact m01-period-impact">{dir === "shorter" ? <Emph text={p.shorter(l)} iso={utc} l={l} /> : <Emph text={p.longer(l)} l={l} />}</p>
      )}
      <Settings l={l} settings={settings} />
      <Reason l={l} value={reason} onChange={setReason} error={problem.reason} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{busy ? w.saving(l) : p.change(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}
