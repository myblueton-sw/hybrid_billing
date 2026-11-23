import { useCallback, useEffect, useRef, useState, useSyncExternalStore, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, pickupGone, sequence, type AdminClient, type CommandId, type CredentialRow, type ServiceRow, type ServicesPage } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { utf8Length } from "../../../../service/helpers";
import { Summary, type Problems } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles, useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { failureText } from "../users/live-model";
import { when } from "../authentication/live-model";
import { v } from "./live-copy";
import { attemptState, credentials, credentialState, formVisible, initialService, keepsHold, issueBlock, liveCount, neverCollected, outsideWindow, panelShown, pickupLabel, pinIssue, prefix, returnFocus, serviceState, settleFailure, split, takenByMe, validName, wholeDays } from "./live-model";
import { issueAttempts, type Entry } from "./issue-attempts";
import { guardUnload } from "../../../../shared/unload-intent";
import { HeldBlock } from "./Held";
import { flushSync } from "react-dom";
import "../users/live.css";

type Kind = "create" | "state" | "issue" | "revoke";
type Action = "suspend" | "reinstate" | "retire";
type Mode =
  | { kind: "form" }
  | { kind: "detail"; id: string }
  | { kind: "step"; id: string; action: Action }
  | { kind: "reveal"; ref: string; service: string; credential: string; until: string }
  | { kind: "revoke"; id: string; service: string; reason: string; from: string };

function describeFor(kind: Kind, max = 0, retire = false) {
  return (l: Locale, e: unknown) => {
    if (e instanceof AdminError) {
      switch (e.failure) {
        case "refused":
          if (kind === "create") return { title: v.refusedTitle(l), body: v.capReached(l, max) };
          break;
        case "stage":
          if (kind === "issue") return { title: v.refusedTitle(l), body: v.blockedRetired(l) };
          return { title: v.refusedTitle(l), body: kind === "revoke" ? v.alreadyRevoked(l) : v.stateChanged(l) };
        case "denied":
          if (kind !== "create") return { title: v.refusedTitle(l), body: v.gone(l) };
          return { title: v.deniedTitle(l), body: v.denied(l) };
        case "conflict":
        case "stale":
          return { title: w.saveFailed(l), body: kind === "issue" ? `${v.conflict(l)} ${v.issueRetry(l)}` : v.conflict(l) };
        case "partial":
          if (kind === "state" && retire) return { title: v.partialTitle(l), body: v.partialBody(l) };
          if (kind === "issue") return { title: w.saveFailed(l), body: `${v.unavailable(l)} ${v.issueRetry(l)}` };
          break;
        case "busy":
        case "unavailable":
          return { title: w.saveFailed(l), body: kind === "issue" ? `${v.unavailable(l)} ${v.issueRetry(l)}` : v.unavailable(l) };
      }
    }
    return failureText(l, e);
  };
}

type Held = "idle" | "sending" | "unknown" | "lapsed";

const elapsedSince = (first: { mono: number; wall: number }) => Math.max(performance.now() - first.mono, Date.now() - first.wall);

function useCommand(api: AdminClient, revision: string, onReload: () => void, window = -1) {
  const hold = window >= 0;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [focusKey, setFocusKey] = useState(0);
  const [track] = useState(() => attemptState(attempt));
  const [phase, setPhase] = useState<Held>("idle");
  const [retried, setRetried] = useState(false);
  const first = useRef({ mono: 0, wall: 0 });
  useEffect(() => track.revised(), [track, revision]);
  const renew = () => { track.renew(); };
  const run = async (id: CommandId, target: string, reason: string, values: Record<string, string>) => {
    const retry = hold && phase === "unknown";
    if (retry && outsideWindow(elapsedSince(first.current), window)) {
      setPhase("lapsed");
      setFocusKey((k) => k + 1);
      return null;
    }
    setBusy(true);
    setError(null);
    setRetried(retry);
    if (hold) setPhase("sending");
    const sent = track.send(revision);
    if (!retry) first.current = { mono: performance.now(), wall: Date.now() };
    try {
      const done = await api.command(id, target, sent.expected, reason, values, sent.attempt);
      track.succeeded();
      setPhase("idle");
      return done;
    } catch (err) {
      setError(err);
      const next = settleFailure(track, err, hold, sent.expected, hold && outsideWindow(elapsedSince(first.current), window, true));
      if (hold) setPhase(next === "unknown" || next === "lapsed" ? next : "idle");
      if (next !== "kept" && (hold || settles(err) || (err instanceof AdminError && keepsHold(err.failure, false)))) onReload();
      setFocusKey((k) => k + 1);
      return null;
    } finally {
      track.idle();
      setBusy(false);
    }
  };
  const discard = () => {
    if (!track.renew()) return false;
    setPhase("idle");
    setError(null);
    setRetried(false);
    return true;
  };
  return { busy, error, focusKey, setFocusKey, renew, run, setError, phase, retried, discard, request: track.request };
}

export const issues = issueAttempts(attempt, { mono: () => performance.now(), wall: () => Date.now() }, typeof window === "undefined" ? undefined : window);

const afterView = { focus: "", view: "" };

function useIssues() {
  return useSyncExternalStore(issues.subscribe, issues.version);
}


const describedBy = (...ids: string[]) => ids.filter(Boolean).join(" ") || undefined;

function LockableReason({ l, value, onChange, error, locked, extra }: { l: Locale; value: string; onChange: (v: string) => void; error: string; locked: boolean; extra?: string }) {
  if (!locked) return <Reason l={l} value={value} onChange={onChange} error={error} />;
  const n = utf8Length(value);
  return (
    <div className="service-field live-reason">
      <label htmlFor="live-reason">{w.reason(l)}</label>
      <p className="service-hint" id="live-reason-hint">{w.reasonHint(l)}</p>
      <textarea id="live-reason" rows={3} value={value} readOnly aria-describedby={describedBy("live-reason-hint", extra ?? "")} />
      <p className="live-counter" aria-live="off">{w.bytes(l, n)}</p>
    </div>
  );
}

function Facts({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="live-facts">
      {items.map(([k, value]) => <div key={k}><dt>{k}</dt><dd>{value}</dd></div>)}
    </dl>
  );
}

const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function CreateForm({ l, api, page, user, stepping, onDone, onReload, onPin }: {
  l: Locale; api: AdminClient; page: ServicesPage; user: string; stepping: boolean; onDone: (id: string) => void; onReload: () => void; onPin: () => void;
}) {
  const [name, setName] = useState("");
  const [owner, setOwner] = useState("");
  const [reason, setReason] = useState("");
  const [problems, setProblems] = useState<Problems>([]);
  const cmd = useCommand(api, page.revision, onReload, page.limits.replay_seconds);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  const locked = cmd.phase === "sending" || cmd.phase === "unknown";
  const edit = (f: (x: string) => void) => (x: string) => { if (locked) return; cmd.renew(); f(x); };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (cmd.phase === "sending" || cmd.phase === "lapsed") return;
    if (cmd.phase !== "unknown") {
      const found: Problems = [];
      if (!validName(name)) found.push({ field: "svc-name", message: v.errName(l) });
      if (!owner) found.push({ field: "svc-owner", message: v.errOwner(l) });
      const r = reasonProblem(l, reason);
      if (r) found.push({ field: "live-reason", message: r });
      setProblems(found);
      cmd.setError(null);
      if (found.length) { cmd.setFocusKey((k) => k + 1); return; }
    }
    onPin();
    const done = await cmd.run("service.create", "services", reason, { name, owner });
    if (done) onDone(prefix + name);
  };
  const startOver = () => { if (cmd.discard()) document.getElementById("svc-name")?.focus(); };
  const failure = cmd.phase === "idle" && cmd.error instanceof AdminError && cmd.error.failure === "invalid" ? cmd.error.fields : [];
  const field = (id: string) => {
    const local = problems.find((x) => x.field === id)?.message;
    if (local) return local;
    if (id === "svc-name" && failure.includes("name")) return takenByMe(page, name, user) ? v.errNameTakenMine(l, prefix + name) : v.errNameTaken(l);
    if (id === "svc-owner" && failure.includes("owner")) return v.errOwner(l);
    return "";
  };
  const shownError = failure.length || cmd.phase !== "idle" ? null : cmd.error;
  const ownerLogin = page.owners.find((o) => o.user_id === owner)?.login || owner;
  const held = locked ? "svc-create-held-lead" : "";
  const note = cmd.phase === "unknown" ? "svc-create-locked-note" : "";
  const describe = cmd.retried
    ? (x: Locale, err: unknown) => (err instanceof AdminError && err.failure !== "invalid" ? { title: v.refusedTitle(x), body: v.createRetryRefused(x, prefix + name) } : describeFor("create", page.limits.services)(x, err))
    : describeFor("create", page.limits.services);
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{v.createTitle(l)}</h2>
      <p className="live-impact">{v.createLead(l)}</p>
      <HeldBlock l={l} id="svc-create" error={cmd.error} request={cmd.request()} phase={cmd.phase} lead={v.createHeldLead(l)} focusKey={cmd.phase === "unknown" || cmd.phase === "lapsed" ? cmd.focusKey : 0}
        lapsed={<p>{v.createLapsed(l, prefix + name)}</p>} />
      {(problems.length > 0 || failure.length > 0) && (
        <Summary locale={l} title={c.check(l)} focusKey={cmd.focusKey}
          problems={problems.length ? problems : (["svc-name", "svc-owner"].filter((id) => field(id)).map((id) => ({ field: id, message: field(id) })))} />
      )}
      {shownError !== null && <FailureBox l={l} error={shownError} focusKey={cmd.focusKey} describe={describe} />}
      {cmd.phase === "lapsed" ? (
        <Facts items={[[v.name(l), <span className="live-mono">{ltr(prefix + name)}</span>], [v.owner(l), ltr(ownerLogin)], [w.reason(l), reason || "—"]]} />
      ) : (
        <>
          <div className="service-field">
            <label htmlFor="svc-name">{v.name(l)}</label>
            <p className="service-hint" id="svc-name-hint">{v.nameHint(l)}</p>
            {field("svc-name") && <p className="service-field-error" id="svc-name-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("svc-name")}</p>}
            <span className="live-unit live-prefixed" dir="ltr">
              <span className="live-mono" aria-hidden="true">{prefix}</span>
              <input id="svc-name" className="live-mono" autoComplete="off" spellCheck={false} value={name} readOnly={locked} aria-invalid={field("svc-name") ? true : undefined}
                aria-describedby={describedBy("svc-name-hint", field("svc-name") ? "svc-name-error" : "", held, note)} onChange={(e) => edit(setName)(e.target.value)} />
            </span>
            <p className="live-note">{v.idPreview(l)} <bdi dir="ltr" className="live-mono">{prefix + (name || "…")}</bdi></p>
          </div>
          {page.owners.length === 0 && !locked ? <p className="live-note">{v.noOwners(l)}</p> : (
            <div className="service-field">
              <label htmlFor="svc-owner">{v.owner(l)}</label>
              <p className="service-hint" id="svc-owner-hint">{v.ownerHint(l)}</p>
              {field("svc-owner") && <p className="service-field-error" id="svc-owner-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("svc-owner")}</p>}
              {locked ? (
                <input id="svc-owner" className="live-select" dir="ltr" readOnly value={ownerLogin} aria-describedby={describedBy("svc-owner-hint", held, note)} />
              ) : (
                <select id="svc-owner" className="live-select" value={owner} aria-invalid={field("svc-owner") ? true : undefined}
                  aria-describedby={describedBy("svc-owner-hint", field("svc-owner") ? "svc-owner-error" : "")} onChange={(e) => edit(setOwner)(e.target.value)}>
                  <option value="">{v.chooseOwner(l)}</option>
                  {page.owners.map((o) => <option key={o.user_id} value={o.user_id}>{o.login || o.user_id}</option>)}
                </select>
              )}
            </div>
          )}
          <Facts items={[[v.credentialsFact(l), v.credAfter(l)]]} />
          <LockableReason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason")} locked={locked} extra={describedBy(held, note)} />
        </>
      )}
      <div className="live-buttons">
        {cmd.phase !== "lapsed" && (
          <button type="submit" className="live-primary" aria-busy={cmd.busy || undefined} disabled={cmd.busy || (page.owners.length === 0 && !locked)}>
            {cmd.busy ? w.saving(l) : cmd.phase === "unknown" ? c.tryAgain(l) : v.createTitle(l)}
          </button>
        )}
        {(cmd.phase === "unknown" || cmd.phase === "lapsed" || (cmd.phase === "sending" && cmd.retried)) && (
          <button type="button" className="live-secondary" disabled={cmd.phase === "sending"} onClick={startOver}>{v.startOver(l)}</button>
        )}
      </div>
    </form>
  );
}

function Detail({ l, s, max, onAction, onCredentials, onClose }: {
  l: Locale; s: ServiceRow; max: number; onAction: (a: Action) => void; onCredentials: () => void; onClose: () => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), [s.id]);
  const actions: Action[] = s.state === "active" ? ["suspend", "retire"] : s.state === "suspended" ? ["reinstate", "retire"] : [];
  const title = { suspend: v.suspendTitle, reinstate: v.reinstateTitle, retire: v.retireTitle };
  return (
    <section className="live-step" aria-labelledby="svc-detail-title">
      <h2 id="svc-detail-title" tabIndex={-1} ref={heading} className="live-mono">{ltr(s.id)}</h2>
      <Facts items={[
        [v.colStatus(l), serviceState(l, s.state)],
        [v.colOwner(l), ltr(s.owner_login || s.owner)],
        [v.creator(l), ltr(s.creator_login || s.creator)],
        [v.credentialsFact(l), v.credSummary(l, liveCount(s), max)],
      ]} />
      {actions.length === 0 ? <p className="live-note">{v.retiredNote(l)}</p> : (
        <div className="live-buttons live-column">
          {actions.map((a) => <button key={a} type="button" className="live-secondary" onClick={() => onAction(a)}>{title[a](l)}</button>)}
        </div>
      )}
      <div className="live-buttons">
        <button type="button" className="service-button-link" onClick={onCredentials}>{v.manageCreds(l)}</button>
        <button type="button" className="live-secondary" onClick={onClose}>{v.close(l)}</button>
      </div>
    </section>
  );
}

function StateStep({ l, api, page, s, action, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; page: ServicesPage; s: ServiceRow; action: Action; onDone: (message: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const [reason, setReason] = useState("");
  const [problem, setProblem] = useState("");
  const cmd = useCommand(api, page.revision, onReload);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  const copy = {
    suspend: [v.suspendTitle, v.suspendImpact, v.doneSuspended],
    reinstate: [v.reinstateTitle, v.reinstateImpact, v.doneReinstated],
    retire: [v.retireTitle, v.retireImpact, v.doneRetired],
  } as const;
  const [title, impact, done] = copy[action];
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { cmd.setError(null); cmd.setFocusKey((k) => k + 1); return; }
    if (await cmd.run(("service." + action) as CommandId, s.id, reason, {})) onDone(done(l, s.id));
  };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{title(l)}</h2>
      <p className="live-who live-mono">{ltr(s.id)}</p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={cmd.focusKey} />}
      {cmd.error !== null && <FailureBox l={l} error={cmd.error} focusKey={cmd.focusKey} describe={describeFor("state", 0, action === "retire")} />}
      <Facts items={[[v.colStatus(l), serviceState(l, s.state)], [v.colOwner(l), ltr(s.owner_login || s.owner)]]} />
      <p className="live-impact">{impact(l)}</p>
      {action === "reinstate" && <p className="live-impact"><strong>{v.reinstateWarn(l)}</strong></p>}
      <Reason l={l} value={reason} onChange={(x) => { cmd.renew(); setReason(x); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={cmd.busy || undefined} disabled={cmd.busy}>{cmd.busy ? w.saving(l) : title(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function IssueForm({ l, api, page, s, user, stepping, onReload, onAccounts, onPin, onShowSecret }: {
  l: Locale; api: AdminClient; page: ServicesPage; s: ServiceRow; user: string; stepping: boolean; onReload: () => void; onAccounts: () => void; onPin: () => void; onShowSecret: () => void;
}) {
  useIssues();
  const max = page.limits.credential_days;
  const entry = issues.get(user, s.id);
  const refusal = issues.refusal(user, s.id);
  const [days, setDays] = useState(entry?.days ?? String(max));
  const [reason, setReason] = useState(entry?.reason ?? "");
  const [problems, setProblems] = useState<Problems>([]);
  const [focusKey, setFocusKey] = useState(0);
  const [blockKey, setBlockKey] = useState(0);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  const phase: Held | "issued" = entry?.phase ?? "idle";
  const seen = useRef(phase);
  useEffect(() => {
    if (seen.current !== phase && (phase === "unknown" || phase === "lapsed")) setBlockKey((k) => k + 1);
    seen.current = phase;
  }, [phase]);
  const locked = phase === "sending" || phase === "unknown" || phase === "issued";
  const shownDays = entry ? entry.days : days;
  const shownReason = entry ? entry.reason : reason;
  const block = entry ? "" : issueBlock(s, page.limits.credentials_per_service);
  const n = wholeDays(shownDays, max);
  const ends = n ? when(l, new Date(Date.parse(page.now) + n * 86400000).toISOString()) : "";
  const send = async (values: { days: string; reason: string }) => {
    onPin();
    const sent = await issues.submit(user, s.id, values, page.revision, page.limits.replay_seconds,
      (tried, expected) => api.command("service.issue_credential", s.id, expected, values.reason, { days: values.days }, tried));
    if (sent.kind === "removed") setFocusKey((k) => k + 1);
    if (sent.kind !== "issued" && sent.kind !== "refused") onReload();
  };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (phase === "sending" || phase === "issued" || phase === "lapsed") return;
    if (entry && phase === "unknown") return send({ days: entry.days, reason: entry.reason });
    const found: Problems = [];
    if (n === null) found.push({ field: "svc-days", message: v.errDays(l, max) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    issues.clearRefusal(user, s.id);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    return send({ days: String(n), reason });
  };
  const startOver = () => {
    if (!entry) return;
    const kept = { days: entry.days, reason: entry.reason };
    if (!issues.discard(user, s.id)) return;
    setDays(kept.days);
    setReason(kept.reason);
    document.getElementById("svc-days")?.focus();
  };
  const field = (id: string) => problems.find((x) => x.field === id)?.message ?? "";
  const error = refusal?.error ?? null;
  const daysError = field("svc-days") || (error instanceof AdminError && error.failure === "invalid" && error.fields.includes("days") ? v.errDays(l, max) : "");
  const refused = error instanceof AdminError && error.failure === "refused";
  const describe = refusal?.retry ? () => ({ title: v.refusedTitle(l), body: v.retryRefused(l) }) : describeFor("issue");
  const held = locked ? (phase === "issued" ? "svc-issued-lead" : "svc-held-lead") : "";
  const note = phase === "unknown" ? "svc-locked-note" : "";
  const edit = (f: (x: string) => void) => (x: string) => { if (locked) return; issues.clearRefusal(user, s.id); f(x); };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 id="svc-issue-title" tabIndex={-1} ref={heading}>{v.issueTitle(l)}</h2>
      <p className="live-who live-mono">{ltr(s.id)}</p>
      {(block !== "" || refused) && error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={describe} />}
      {block === "retired" ? <p className="live-impact">{v.blockedRetired(l)}</p>
        : block === "suspended" || (refused && s.state === "suspended") ? (
          <>
            <p className="live-impact">{v.blockedSuspended(l, s.id)}</p>
            <div className="live-buttons"><button type="button" className="live-secondary" onClick={onAccounts}>{v.goAccounts(l)}</button></div>
          </>
        ) : block === "limit" || refused ? <p className="live-impact">{v.blockedLimit(l, s.id, page.limits.credentials_per_service)}</p> : (
          <>
            {phase === "idle" && <p className="live-impact">{v.issueLead(l)}</p>}
            <HeldBlock l={l} id="svc" error={entry?.error} request={entry?.track.request()} phase={phase === "issued" ? "idle" : phase} lead={v.heldLead(l)} focusKey={blockKey}
              lapsed={<p>{v.attemptLapsed(l, s.id).join(" ")}</p>} />
            {phase === "issued" && entry?.issued && (
              <div className="live-held">
                <p id="svc-issued-lead">{v.attemptIssued(l, entry.issued.credential, s.id)}</p>
                <div className="live-buttons"><button type="button" className="live-primary" onClick={onShowSecret}>{v.showSecret(l)}</button></div>
              </div>
            )}
            {(problems.length > 0 || daysError) && <Summary locale={l} title={c.check(l)} focusKey={focusKey} problems={problems.length ? problems : [{ field: "svc-days", message: daysError }]} />}
            {error !== null && !daysError && <FailureBox l={l} error={error} focusKey={focusKey} describe={describe} />}
            {phase === "lapsed" ? (
              <Facts items={[[v.validFor(l), `${shownDays} ${v.days(l)}`], [w.reason(l), shownReason || "—"]]} />
            ) : (
              <>
                <div className="service-field live-minutes">
                  <label htmlFor="svc-days">{v.validFor(l)}</label>
                  <p className="service-hint" id="svc-days-hint">{v.daysHint(l, max)}</p>
                  {daysError && <p className="service-field-error" id="svc-days-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{daysError}</p>}
                  <span className="live-unit">
                    <input id="svc-days" inputMode="numeric" dir="ltr" value={shownDays} readOnly={locked} aria-invalid={daysError ? true : undefined}
                      aria-describedby={describedBy("svc-days-hint", daysError ? "svc-days-error" : "", held, note)} onChange={(e) => edit(setDays)(e.target.value)} />
                    <span>{v.days(l)}</span>
                  </span>
                  {ends && <p className="live-note" aria-live="polite">{v.endsOn(l, ends)}</p>}
                </div>
                <LockableReason l={l} value={shownReason} onChange={edit(setReason)} error={field("live-reason")} locked={locked} extra={describedBy(held, note)} />
              </>
            )}
            <div className="live-buttons">
              {phase !== "lapsed" && (
                <button type="submit" className="live-primary" aria-busy={phase === "sending" || undefined} disabled={phase === "sending" || phase === "issued"}>
                  {phase === "sending" ? w.saving(l) : phase === "unknown" ? c.tryAgain(l) : v.issueTitle(l)}
                </button>
              )}
              {(phase === "unknown" || phase === "lapsed" || (phase === "sending" && entry?.retry)) && (
                <button type="button" className="live-secondary" disabled={phase === "sending"} onClick={startOver}>{v.startOver(l)}</button>
              )}
            </div>
          </>
        )}
    </form>
  );
}

function Reveal({ l, api, service, refId, credential, until, onDone, onRevoke, onLeave }: {
  l: Locale; api: AdminClient; service: string; refId: string; credential: string; until: string;
  onDone: () => void; onRevoke: () => void; onLeave: () => void;
}) {
  const [secret, setSecret] = useState("");
  const [state, setState] = useState<"loading" | "shown" | "retry" | "gone">("loading");
  const [ack, setAck] = useState(false);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const retried = useRef(false);
  const asked = useRef(false);
  const heading = useRef<HTMLHeadingElement>(null);
  const fetchSecret = useCallback(() => {
    setState("loading");
    api.secret(refId).then(
      (got) => {
        if (got.credential !== credential) { setState("gone"); return; }
        setSecret(got.secret);
        setState("shown");
      },
      (e) => setState(pickupGone(e) ? "gone" : "retry"),

    );
  }, [api, refId, credential]);
  useEffect(() => {
    if (asked.current) return;
    asked.current = true;
    if (refId) fetchSecret();
    else setState("gone");
  }, [fetchSecret, refId]);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => { if (retried.current && state === "shown") heading.current?.focus(); }, [state]);
  useEffect(() => {
    if (state !== "shown") return;
    const warn = guardUnload(() => true);
    addEventListener("beforeunload", warn);
    return () => removeEventListener("beforeunload", warn);
  }, [state]);
  useEffect(() => {
    const drop = () => flushSync(() => { setSecret(""); setState("gone"); });
    addEventListener("pagehide", drop);
    return () => removeEventListener("pagehide", drop);
  }, []);
  const finish = () => {
    if (state === "shown" && !ack) {
      setProblem(v.ackError(l));
      setFocusKey((k) => k + 1);
      return;
    }
    setSecret("");
    onDone();
  };
  return (
    <section className="live-step" aria-labelledby="svc-reveal-title">
      <p className="service-status">{v.doneIssued(l, credential)}</p>
      <h2 id="svc-reveal-title" tabIndex={-1} ref={heading}>{v.revealTitle(l)}</h2>
      <Facts items={[[v.colService(l), ltr(service)], [v.colCred(l), <span className="live-mono">{ltr(credential)}</span>], [v.colValid(l), v.until(l, when(l, until))]]} />
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "svc-ack", message: problem }]} focusKey={focusKey} />}
      {problem && <p className="visually-hidden" id="svc-ack-error">{problem}</p>}
      {state === "loading" && <p role="status">{c.loading(l)}</p>}
      {state === "shown" && (
        <>
          <div className="service-field">
            <label htmlFor="svc-secret">{v.secretLabel(l)}</label>
            <p className="service-hint" id="svc-secret-hint">{v.secretHint(l)}</p>
            <textarea id="svc-secret" className="live-link live-secret" readOnly rows={2} dir="ltr" spellCheck={false} autoComplete="off" autoCapitalize="off"
              autoCorrect="off" translate="no" aria-describedby="svc-secret-hint svc-secret-once" value={secret} />
          </div>
          <p className="live-impact" id="svc-secret-once"><strong>{v.shownOnce(l)}</strong></p>
          <p className="live-impact">{v.prelaunchLead(l)} {v.prelaunchNote(l)}</p>
          <label className="live-check">
            <input id="svc-ack" type="checkbox" checked={ack} aria-invalid={problem ? true : undefined} aria-describedby={problem ? "svc-ack-error" : undefined} onChange={(e) => { setAck(e.target.checked); setProblem(""); }} />
            {v.ackLabel(l)}
          </label>
        </>
      )}
      {state === "retry" && (
        <div className="service-summary" role="alert">
          <p>{v.secretRetry(l)}</p>
          <button type="button" className="live-secondary" onClick={() => { retried.current = true; fetchSecret(); }}>{v.showSecret(l)}</button>
        </div>
      )}
      {state === "gone" && <p className="live-impact">{v.secretGone(l)}</p>}
      <div className="live-buttons">
        {state === "shown" ? <button type="button" className="live-primary" onClick={finish}>{v.done(l)}</button>
          : <button type="button" className="live-secondary" onClick={onLeave}>{v.close(l)}</button>}
        <button type="button" className={state === "gone" ? "live-primary" : "live-secondary"} onClick={() => { setSecret(""); onRevoke(); }}>{state === "gone" ? v.revokeTitle(l) : v.notStored(l)}</button>
      </div>
    </section>
  );
}

function RevokeStep({ l, api, page, id, service, initial, onDone, onCancel, onReload }: {
  l: Locale; api: AdminClient; page: ServicesPage; id: string; service: string; initial: string; onDone: (message: string) => void; onCancel: () => void; onReload: () => void;
}) {
  const [reason, setReason] = useState(initial);
  const [problem, setProblem] = useState("");
  const cmd = useCommand(api, page.revision, onReload);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  const row = page.services.find((s) => s.id === service)?.credentials.find((x) => x.id === id);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const r = reasonProblem(l, reason);
    setProblem(r);
    if (r) { cmd.setError(null); cmd.setFocusKey((k) => k + 1); return; }
    if (await cmd.run("service.revoke_credential", id, reason, {})) onDone(v.doneRevoked(l, id));
  };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{v.revokeTitle(l)}</h2>
      <p className="live-who live-mono">{ltr(id)}</p>
      {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={cmd.focusKey} />}
      {cmd.error !== null && <FailureBox l={l} error={cmd.error} focusKey={cmd.focusKey} describe={describeFor("revoke")} />}
      <Facts items={[[v.colService(l), ltr(service)], ...(row ? [[v.colStatus(l), credentialState(l, row).filter(Boolean).join(" · ")] as [string, ReactNode]] : [])]} />
      <p className="live-impact">{v.revokeImpact(l, id)}</p>
      <Reason l={l} value={reason} onChange={(x) => { cmd.renew(); setReason(x); }} error={problem} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={cmd.busy || undefined} disabled={cmd.busy}>{cmd.busy ? w.saving(l) : v.revokeTitle(l)}</button>
        <button type="button" className="live-secondary" onClick={onCancel}>{w.cancel(l)}</button>
      </div>
    </form>
  );
}

function ServiceTable({ l, rows, max, selected, onManage }: { l: Locale; rows: ServiceRow[]; max: number; selected: string; onManage?: (s: ServiceRow) => void }) {
  return (
    <div className="security-table-scroll live-table live-access">
      <table role="table">
        <colgroup><col className="live-col-a" /><col className="live-col-b" /><col className="live-col-c" />{onManage && <col className="live-col-d" />}</colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{v.colService(l)}</th>
          <th scope="col" role="columnheader">{v.colOwner(l)}</th>
          <th scope="col" role="columnheader">{v.colStatus(l)}</th>
          {onManage && <th scope="col" role="columnheader"><span className="visually-hidden">{v.colActions(l)}</span></th>}
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((s) => (
            <tr key={s.id} role="row" className={selected === s.id ? "live-selected" : undefined}>
              <td role="cell" className="live-user live-mono">{ltr(s.id)}</td>
              <td role="cell" data-label={v.colOwner(l)}>{ltr(s.owner_login || s.owner)}<span className="live-sub">{v.createdBy(l, "")}{ltr(s.creator_login || s.creator)}</span></td>
              <td role="cell" data-label={v.colStatus(l)} className={s.state !== "active" ? "live-strong" : ""}>{serviceState(l, s.state)}
                <span className="live-sub">{s.state === "suspended" ? v.suspendedSub(l) : v.credSummary(l, liveCount(s), max)}</span></td>
              {onManage && (
                <td role="cell"><button type="button" className="live-secondary live-revoke" data-service={s.id} onClick={() => onManage(s)}>{v.manage(l)}<span className="visually-hidden"> {s.id}</span></button></td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PickupTime({ l, iso }: { l: Locale; iso: string }) {
  const date = new Date(iso);
  const locale = { en: "en-US", ko: "ko-KR", ar: "ar" }[l];
  return <bdi dir="ltr"><bdi dir="ltr" className="live-nowrap">{new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(date).replace(/[\u061c\u200e\u200f]/g, "")}</bdi>{" "}<bdi dir="ltr" className="live-nowrap">{new Intl.DateTimeFormat(locale, { timeStyle: "short" }).format(date).replace(/[\u061c\u200e\u200f]/g, "")}</bdi></bdi>;
}

function CredentialTable({ l, rows, selected, onRevoke, retired = false }: { l: Locale; rows: CredentialRow[]; selected: string; onRevoke?: (c: CredentialRow) => void; retired?: boolean }) {
  return (
    <div className="security-table-scroll live-table live-access live-creds">
      <table role="table">
        <colgroup><col className="live-col-a" /><col className="live-col-b" /><col className="live-col-c" />{onRevoke && <col className="live-col-d" />}</colgroup>
        <thead role="rowgroup"><tr role="row">
          <th scope="col" role="columnheader">{v.colCred(l)}</th>
          <th scope="col" role="columnheader">{v.colValid(l)}</th>
          <th scope="col" role="columnheader">{v.colStatus(l)}</th>
          {onRevoke && <th scope="col" role="columnheader"><span className="visually-hidden">{v.colActions(l)}</span></th>}
        </tr></thead>
        <tbody role="rowgroup">
          {rows.map((x) => {
            const [state, sub] = credentialState(l, x, retired);
            const pickup = onRevoke ? pickupLabel(l, x) : null;
            return (
              <tr key={x.id} role="row" className={selected === x.id ? "live-selected" : undefined}>
                <td role="cell" className="live-user live-mono">{ltr(x.id)}</td>
                <td role="cell" data-label={v.colValid(l)}><span className="live-to">{v.from(l, when(l, x.not_before))}</span><span className="live-to">{v.until(l, when(l, x.not_after))}</span></td>
                <td role="cell" data-label={v.colStatus(l)} className={x.state !== "active" || !x.usable ? "live-strong" : ""}>{state}{sub && <span className="live-sub">{sub}</span>}
                  {pickup && <span className={pickup.never ? "live-sub live-pickup-never" : "live-sub live-pickup"}>{pickup.text}{pickup.time && <> <PickupTime l={l} iso={x.secret_pickup_until!} /></>}</span>}</td>
                {onRevoke && (
                  <td role="cell"><button type="button" className="live-secondary live-revoke" data-credential={x.id} onClick={() => onRevoke(x)}>{v.revoke(l)}<span className="visually-hidden"> {x.id}</span></button></td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function Unavailable({ title, body, action }: { title: string; body: string; action: { label: string; go: () => void } }) {
  return (
    <div className="live-users">
      <section className="live-unavailable" aria-labelledby="svc-unavailable">
        <h2 id="svc-unavailable">{title}</h2>
        <p>{body}</p>
        <button type="button" className="live-secondary" onClick={action.go}>{action.label}</button>
      </section>
    </div>
  );
}

function AttemptsRegion({ l, entries, onOpen, onDiscard, onShow, onRevoke }: {
  l: Locale; entries: Entry[]; onOpen: (service: string) => void; onDiscard: (service: string) => void; onShow: (service: string) => void; onRevoke: (e: Entry) => void;
}) {
  const discardable = entries.some((e) => e.phase === "unknown" || e.phase === "lapsed");
  return (
    <section className="live-attempts" aria-labelledby="svc-attempts-title">
      <h3 id="svc-attempts-title" tabIndex={-1}>{v.attemptsTitle(l)}</h3>
      {discardable && <p>{v.attemptsDiscardNote(l)}</p>}
      <ul>
        {entries.map((e) => {
          const [head, rest] = e.phase === "lapsed" ? v.attemptLapsed(l, e.service) : v.attemptUnknown(l, e.service);
          return (
            <li key={e.service} data-attempt={e.service}>
              {e.phase === "sending" && <span className="live-row-text" aria-busy="true">{v.attemptSending(l, e.service)}</span>}
              {(e.phase === "unknown" || e.phase === "lapsed") && <span className="live-row-text"><strong>{head}</strong> {rest}</span>}
              {e.phase === "issued" && e.issued && <span className="live-row-text">{v.attemptIssued(l, e.issued.credential, e.service)}</span>}
              {e.phase !== "sending" && (
                <span className="live-row-buttons">
                  {e.phase === "unknown" && <button type="button" className="live-secondary" onClick={() => onOpen(e.service)}>{v.openIssue(l)}<span className="visually-hidden"> {e.service}</span></button>}
                  {e.phase === "issued" && <button type="button" className="live-primary" onClick={() => onShow(e.service)}>{v.showSecret(l)}<span className="visually-hidden"> {e.service}</span></button>}
                  {e.phase === "issued" && e.issued && (
                    <button type="button" className="live-secondary" data-attempt-revoke={e.service} onClick={() => onRevoke(e)}>{v.revoke(l)}<span className="visually-hidden"> {e.issued.credential}</span></button>
                  )}
                  {(e.phase === "unknown" || e.phase === "lapsed") && <button type="button" className="live-secondary" onClick={() => onDiscard(e.service)}>{v.discardAttempt(l)}<span className="visually-hidden"> {e.service}</span></button>}
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}

function NeverRow({ l, service, c, onRevoke }: { l: Locale; service: string; c: CredentialRow; onRevoke: (service: string, id: string) => void }) {
  return (
    <li data-never={c.id}>
      <span className="live-row-text">
        <strong><span className="live-mono">{ltr(c.id)}</span> · <span className="live-mono">{ltr(service)}</span></strong>
        <span className="live-row-sub">{v.until(l, when(l, c.not_after))}</span>
      </span>
      <span className="live-row-buttons">
        <button type="button" className="live-secondary" data-never-revoke={c.id} onClick={() => onRevoke(service, c.id)}>{v.revoke(l)}<span className="visually-hidden"> {c.id}</span></button>
      </span>
    </li>
  );
}

function NeverRegion({ l, rows, onRevoke }: { l: Locale; rows: { service: string; credential: CredentialRow }[]; onRevoke: (service: string, id: string) => void }) {
  const first = rows.slice(0, 5);
  const more = rows.slice(5);
  return (
    <section className="live-never" aria-labelledby="svc-never-title">
      <h3 id="svc-never-title" tabIndex={-1}>{v.neverTitle(l)}</h3>
      <p>{v.neverLead(l)}</p>
      <ul>{first.map((r) => <NeverRow key={r.credential.id} l={l} service={r.service} c={r.credential} onRevoke={onRevoke} />)}</ul>
      {more.length > 0 && (
        <details className="live-ended">
          <summary>{v.neverMore(l, more.length)}</summary>
          <ul>{more.map((r) => <NeverRow key={r.credential.id} l={l} service={r.service} c={r.credential} onRevoke={onRevoke} />)}</ul>
        </details>
      )}
    </section>
  );
}

export default function LiveServices({ locale: l, view, session, onSignOut, onView }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
}) {
  useIssues();
  const user = session.user;
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<ServicesPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode>({ kind: "form" });
  const [chosen, setChosen] = useState("");
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  const [open, setOpen] = useState(false);

  const unstored = issues.marks(user);
  const attempts = issues.list(user);
  const credentialsView = view === "credentials";
  useEffect(() => {
    if (afterView.focus && afterView.view === view) { setFocusTo(afterView.focus); afterView.focus = ""; }
  }, [view]);
  useEffect(() => {
    if (!focusTo || !page) return;
    for (const sel of returnFocus(focusTo, credentialsView)) {
      const el = document.querySelector<HTMLElement>(sel);
      if (el) { el.focus(); break; }
    }
    setFocusTo("");
  }, [focusTo, credentialsView, page]);
  const tell = (text: string) => { setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const say = (text: string) => { setNotice(text); tell(text); };
  const load = useCallback(() => {
    const n = order.current.next();
    const seq = issues.readStarted();
    api.services().then(
      (got) => { issues.reconcile(user, got, seq); if (order.current.latest(n)) { setPage(got); setStatus("ready"); setError(null); } },
      (e) => { if (order.current.latest(n)) { if (e instanceof AdminError && e.failure === "invalid") setPage(null); setError(e); setStatus("error"); } },
    );
  }, [api, user]);
  useEffect(() => { if (view === "accounts" || view === "credentials") load(); }, [load, view]);
  const viewRef = useRef(view);
  useEffect(() => {
    if (viewRef.current !== view) {
      setMode({ kind: "form" });
      setNotice("");
      setOpen(false);
    }
    viewRef.current = view;
  }, [view]);
  const currentId = page ? initialService(page, chosen) : "";
  const visibleFor = (service: string) => formVisible({ view, mode: mode.kind, current: currentId, service, wide, open, visibility: typeof document === "undefined" ? "hidden" : document.visibilityState });
  const showSecret = (service: string) => {
    const got = issues.consume(user, service);
    if (!got) return;
    issues.mark(user, got.credential);
    setNotice("");
    setChosen(service);
    load();
    setMode({ kind: "reveal", ref: got.ref, service, credential: got.credential, until: got.until });
  };
  const phases = useRef(new Map<string, string>(attempts.map((e) => [e.service, e.phase])));
  useEffect(() => {
    const seen = new Map<string, string>();
    for (const e of attempts) {
      seen.set(e.service, e.phase);
      if (phases.current.get(e.service) === e.phase) continue;
      const visible = visibleFor(e.service);
      if (e.phase === "issued" && visible) { showSecret(e.service); continue; }
      if (visible) continue;
      if (e.phase === "unknown") tell(v.attemptUnknown(l, e.service).join(" "));
      if (e.phase === "lapsed") tell(v.attemptLapsed(l, e.service).join(" "));
      if (e.phase === "issued" && e.issued) tell(v.attemptIssued(l, e.issued.credential, e.service));
    }
    phases.current = seen;
  });
  const toAccounts = { label: v.goAccounts(l), go: () => onView("accounts") };
  if (view === "usage") return <Unavailable title={v.usageTitle(l)} body={v.usageBody(l)} action={toAccounts} />;
  if (view === "transfer") return <Unavailable title={v.transferTitle(l)} body={v.transferBody(l)} action={toAccounts} />;
  if (status === "error" && error instanceof AdminError && error.failure === "denied") {
    return <div className="live-users"><section className="live-unavailable" aria-labelledby="svc-denied"><h2 id="svc-denied">{v.deniedTitle(l)}</h2><p>{v.denied(l)}</p></section></div>;
  }
  const loadError = status === "error" && (
    <section className="live-error" role="alert">
      <h2>{v.loadFailed(l)}</h2>
      <p>{describeFor("create")(l, error).body}</p>
      {error instanceof AdminError && error.failure === "unauthenticated"
        ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
        : <button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button>}
    </section>
  );
  const max = page?.limits.credentials_per_service ?? 0;
  const find = (id: string) => page?.services.find((s) => s.id === id);
  const finished = (message: string, next: Mode, focus: string) => { say(message); setMode(next); setOpen(false); setFormKey((k) => k + 1); load(); setFocusTo(focus); };
  const current = page ? find(currentId) : undefined;
  const revoke = (id: string, service: string, reason: string, from: string) => { setNotice(""); setMode({ kind: "revoke", id, service, reason, from }); };
  const openIssue = (service: string) => { setNotice(""); setChosen(service); setMode({ kind: "form" }); setOpen(true); setFocusTo("#svc-issue-title"); };
  const discard = (service: string) => {
    const at = attempts.findIndex((e) => e.service === service);
    const next = attempts[at + 1] ?? attempts[at - 1];
    if (!issues.discard(user, service)) return;
    setFocusTo(next ? `[data-attempt="${next.service}"] button` : "#svc-cred-title");
  };
  let panel: ReactNode = null;
  let back = "";
  let backTo = credentialsView ? "#svc-cred-title" : "#svc-list-title";
  if (page && !credentialsView) {
    if (mode.kind === "detail" && find(mode.id)) {
      const s = find(mode.id) as ServiceRow;
      panel = <Detail key={s.id} l={l} s={s} max={max} onAction={(action) => setMode({ kind: "step", id: s.id, action })}
        onCredentials={() => { setChosen(s.id); onView("credentials"); }} onClose={() => { setMode({ kind: "form" }); setFocusTo(`[data-service="${s.id}"]`); }} />;
      backTo = `[data-service="${s.id}"]`;
    } else if (mode.kind === "step" && find(mode.id)) {
      const s = find(mode.id) as ServiceRow;
      const done = mode.id;
      panel = <StateStep key={s.id + mode.action} l={l} api={api} page={page} s={s} action={mode.action} onReload={load}
        onCancel={() => setMode({ kind: "detail", id: done })} onDone={(message) => finished(message, { kind: "detail", id: done }, "#svc-detail-title")} />;
      backTo = `[data-service="${s.id}"]`;
    } else if (wide || mode.kind === "form") {
      panel = <CreateForm key={formKey} l={l} api={api} page={page} user={user} stepping={!wide} onReload={load} onPin={() => setOpen(true)}
        onDone={(id) => finished(v.doneCreated(l, id), { kind: "detail", id }, "#svc-detail-title")} />;
      backTo = ".svc-create-open";
    }
    back = v.backList(l);
  }
  if (page && credentialsView && current) {
    if (mode.kind === "reveal") {
      const shown = mode;
      panel = <Reveal key={shown.ref} l={l} api={api} service={shown.service} refId={shown.ref} credential={shown.credential}
        until={find(shown.service)?.credentials.find((x) => x.id === shown.credential)?.not_after ?? shown.until}
        onLeave={() => { setMode({ kind: "form" }); setOpen(false); load(); setFocusTo("#svc-cred-title"); }}
        onDone={() => { issues.unmark(user, shown.credential); setMode({ kind: "form" }); setOpen(false); setFormKey((k) => k + 1); load(); setFocusTo("#svc-cred-title"); }}
        onRevoke={() => revoke(shown.credential, shown.service, v.revokeUnstoredReason(l), `[data-credential="${shown.credential}"]`)} />;
    } else if (mode.kind === "revoke") {
      const target = mode;
      panel = <RevokeStep key={target.id} l={l} api={api} page={page} id={target.id} service={target.service} initial={target.reason} onReload={load}
        onCancel={() => { setMode({ kind: "form" }); setOpen(false); setFocusTo(target.from); }}
        onDone={(message) => { issues.unmark(user, target.id); issues.revoked(user, target.id); finished(message, { kind: "form" }, "#svc-cred-title"); }} />;
      backTo = target.from;
    } else if (wide || mode.kind === "form") {
      panel = <IssueForm key={current.id + formKey} l={l} api={api} page={page} s={current} user={user} stepping={!wide} onReload={load} onAccounts={() => onView("accounts")}
        onPin={() => { const pin = pinIssue(current.id); setChosen(pin.chosen); setOpen(pin.open); }} onShowSecret={() => showSecret(current.id)} />;
      backTo = ".svc-issue-open";
    }
    back = v.backCreds(l);
  }
  const groups = page ? split(page) : { current: [], retired: [] };
  const creds = current ? credentials(current) : { current: [], ended: [] };
  const selected = mode.kind === "detail" || mode.kind === "step" ? mode.id : mode.kind === "revoke" ? mode.id : "";
  const never = page ? neverCollected(page) : [];
  const neverIds = new Set(never.map((r) => r.credential.id));
  const unstoredShown = unstored.filter((id) => !neverIds.has(id));
  const serviceOf = (id: string) => page?.services.find((x) => x.credentials.some((y) => y.id === id))?.id ?? "";
  const list = !credentialsView ? (
    <section aria-labelledby="svc-list-title">
      <div className="live-toolbar">
        <div className="live-heading">
          <h2 id="svc-list-title" tabIndex={-1}>{v.listTitle(l)}</h2>
          {page && <p className="live-meta">{v.count(l, page.services.length, page.limits.services)}</p>}
          {page && <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>}
        </div>
        <div className="live-tools">
          {!wide && page && <button type="button" className="live-primary svc-create-open" onClick={() => { setNotice(""); setMode({ kind: "form" }); setOpen(true); }}>{v.createTitle(l)}</button>}
          <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
        </div>
      </div>
      {notice && <p className="service-status">{notice}</p>}
      {attempts.length > 0 && (
        <div className="live-note-rule live-attempts-note">
          <p>{v.accountsAttemptsNote(l, attempts.length)} <button type="button" className="service-button-link" onClick={() => { afterView.focus = "#svc-attempts-title"; afterView.view = "credentials"; onView("credentials"); }}>{v.goCredentials(l)}</button></p>
        </div>
      )}
      {status === "loading" && !page && <p role="status" className="live-meta">{v.loading(l)}</p>}
      {loadError}
      {page && (groups.current.length === 0 ? <p className="live-empty">{v.noServices(l)}</p>
        : <ServiceTable l={l} rows={groups.current} max={max} selected={selected} onManage={(s) => { setNotice(""); setMode({ kind: "detail", id: s.id }); }} />)}
      {page && groups.retired.length > 0 && (
        <details className="live-ended"><summary>{v.retiredGroup(l, groups.retired.length)}</summary><ServiceTable l={l} rows={groups.retired} max={max} selected="" /></details>
      )}
    </section>
  ) : (
    <section aria-labelledby="svc-cred-title">
      <div className="live-toolbar">
        <div className="live-heading">
          <h2 id="svc-cred-title" tabIndex={-1}>{v.credTitle(l)}</h2>
          {page && <p className="live-meta">{w.loadedAt(l, when(l, page.now))}</p>}
        </div>
        <div className="live-tools">
          {!wide && current && <button type="button" className="live-primary svc-issue-open" onClick={() => { setNotice(""); setMode({ kind: "form" }); setOpen(true); }}>{v.issueTitle(l)}</button>}
          <button type="button" className="live-secondary" onClick={() => { setNotice(""); load(); }}>{w.reload(l)}</button>
        </div>
      </div>
      {notice && <p className="service-status">{notice}</p>}
      {attempts.length > 0 && (
        <AttemptsRegion l={l} entries={attempts} onOpen={openIssue} onDiscard={discard} onShow={showSecret}
          onRevoke={(e) => e.issued && revoke(e.issued.credential, e.service, v.revokeNotShownReason(l), `[data-attempt-revoke="${e.service}"]`)} />
      )}
      {never.length > 0 && <NeverRegion l={l} rows={never} onRevoke={(service, id) => revoke(id, service, v.revokeNeverReason(l), `[data-never-revoke="${id}"]`)} />}
      {mode.kind !== "reveal" && unstoredShown.length > 0 && (
        <div className="service-summary live-unstored">
          <h3>{v.unstoredTitle(l)}</h3>
          <p>{v.unstoredLead(l)}</p>
          <ul className="live-unstored-list">
            {unstoredShown.map((id) => (
              <li key={id}>
                <span><span className="live-mono">{ltr(id)}</span> · <span className="live-mono">{ltr(serviceOf(id))}</span></span>
                <span className="live-row-buttons">
                  <button type="button" className="live-secondary" data-unstored-revoke={id} onClick={() => revoke(id, serviceOf(id) || current?.id || "", v.revokeUnstoredReason(l), `[data-unstored-revoke="${id}"]`)}>{v.revoke(l)}<span className="visually-hidden"> {id}</span></button>
                  <button type="button" className="live-secondary" onClick={() => issues.unmark(user, id)}>{v.dismiss(l)}<span className="visually-hidden"> {id}</span></button>
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      {status === "loading" && !page && <p role="status" className="live-meta">{v.loading(l)}</p>}
      {loadError}
      {page && !current && (
        <>
          <p className="live-empty">{v.noServicesCred(l)}</p>
          <button type="button" className="live-secondary" onClick={() => onView("accounts")}>{v.goAccounts(l)}</button>
        </>
      )}
      {page && current && (
        <>
          <div className="service-field">
            <label htmlFor="svc-select">{v.colService(l)}</label>
            <select id="svc-select" className="live-select live-mono" value={current.id} onChange={(e) => { setChosen(e.target.value); setMode({ kind: "form" }); setOpen(false); setNotice(""); }}>
              {page.services.map((s) => <option key={s.id} value={s.id}>{s.state === "active" ? s.id : `${s.id} (${serviceState(l, s.state)})`}</option>)}
            </select>
          </div>
          <div className="live-note-rule"><p><strong>{v.prelaunchLead(l)}</strong> {v.prelaunchNote(l)}</p></div>
          <p className="live-meta live-count">{v.credSummary(l, liveCount(current), max)}</p>
          {creds.current.length === 0 ? <p className="live-empty">{v.noCreds(l, current.id)}</p>
            : <CredentialTable l={l} rows={creds.current} retired={current.state === "retired"} selected={selected} onRevoke={(x) => revoke(x.id, current.id, "", `[data-credential="${x.id}"]`)} />}
          {creds.ended.length > 0 && (
            <details className="live-ended"><summary>{v.endedGroup(l, creds.ended.length)}</summary><CredentialTable l={l} rows={creds.ended} selected="" /></details>
          )}
        </>
      )}
    </section>
  );
  const showPanel = panel !== null && panelShown(wide, mode.kind === "form", open);
  return (
    <div className="live-users" data-live={credentialsView ? "credentials" : "accounts"} data-wide={wide ? "true" : "false"}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {!(showPanel && !wide) && <div className="live-list live-sections">{list}</div>}
      {showPanel && (
        <aside className="live-panel" aria-labelledby={undefined} aria-label={credentialsView ? v.credTitle(l) : v.listTitle(l)}>
          {!wide && <button type="button" className="service-button-link live-back" onClick={() => { setMode({ kind: "form" }); setOpen(false); setFocusTo(backTo); }}>{back}</button>}
          {panel}
        </aside>
      )}
    </div>
  );
}
