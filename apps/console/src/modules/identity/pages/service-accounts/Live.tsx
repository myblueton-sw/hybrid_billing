import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, sequence, type AdminClient, type Attempt, type CommandId, type CredentialRow, type ServiceRow, type ServicesPage } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { Summary, type Problems } from "../../../../service/Service";
import { FailureBox, Reason, reasonProblem, settles, useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { failureText } from "../users/live-model";
import { when } from "../authentication/live-model";
import { v } from "./live-copy";
import { credentials, credentialState, initialService, keepsHold, issueBlock, liveCount, prefix, serviceState, split, validName, wholeDays } from "./live-model";
import { flushSync } from "react-dom";
import "../users/live.css";

type Kind = "create" | "state" | "issue" | "revoke";
type Action = "suspend" | "reinstate" | "retire";
type Mode =
  | { kind: "form" }
  | { kind: "detail"; id: string }
  | { kind: "step"; id: string; action: Action }
  | { kind: "reveal"; ref: string; service: string; credential: string; until: string }
  | { kind: "revoke"; id: string; service: string; reason: string };

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

function useCommand(api: AdminClient, revision: string, onReload: () => void, holdUnknown = false) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const held = useRef("");
  const [holding, setHolding] = useState(false);
  useEffect(() => { if (!held.current) tried.current = attempt(); }, [revision]);
  const renew = () => { held.current = ""; setHolding(false); tried.current = attempt(); };
  const run = async (id: CommandId, target: string, reason: string, values: Record<string, string>) => {
    setBusy(true);
    setError(null);
    try {
      const done = await api.command(id, target, held.current || revision, reason, values, tried.current);
      held.current = "";
      setHolding(false);
      return done;
    } catch (err) {
      setError(err);
      if (err instanceof AdminError && (err.failure === "reauth" || err.failure === "reauth_cancelled")) {
        setFocusKey((k) => k + 1);
        return null;
      }
      const unknown = err instanceof AdminError && keepsHold(err.failure, held.current !== "");
      if (holdUnknown && unknown) {
        held.current = held.current || revision;
        setHolding(true);
      } else renew();
      if (settles(err) || unknown) onReload();
      setFocusKey((k) => k + 1);
      return null;
    } finally {
      setBusy(false);
    }
  };
  const hold = (err: AdminError) => {
    held.current = held.current || revision;
    setHolding(true);
    setError(err);
    setFocusKey((k) => k + 1);
    onReload();
  };
  return { busy, error, focusKey, setFocusKey, renew, run, setError, holding, hold };
}

function Facts({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="live-facts">
      {items.map(([k, value]) => <div key={k}><dt>{k}</dt><dd>{value}</dd></div>)}
    </dl>
  );
}

const pending = new Map<string, string[]>();

const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function CreateForm({ l, api, page, stepping, onDone, onReload }: {
  l: Locale; api: AdminClient; page: ServicesPage; stepping: boolean; onDone: (id: string) => void; onReload: () => void;
}) {
  const [name, setName] = useState("");
  const [owner, setOwner] = useState("");
  const [reason, setReason] = useState("");
  const [problems, setProblems] = useState<Problems>([]);
  const cmd = useCommand(api, page.revision, onReload);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  const edit = (f: (x: string) => void) => (x: string) => { cmd.renew(); f(x); };
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (!validName(name)) found.push({ field: "svc-name", message: v.errName(l) });
    if (!owner) found.push({ field: "svc-owner", message: v.errOwner(l) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    cmd.setError(null);
    if (found.length) { cmd.setFocusKey((k) => k + 1); return; }
    const done = await cmd.run("service.create", "services", reason, { name, owner });
    if (done) onDone(prefix + name);
  };
  const failure = cmd.error instanceof AdminError && cmd.error.failure === "invalid" ? cmd.error.fields : [];
  const field = (id: string) => {
    const local = problems.find((x) => x.field === id)?.message;
    if (local) return local;
    if (id === "svc-name" && failure.includes("name")) return v.errNameTaken(l);
    if (id === "svc-owner" && failure.includes("owner")) return v.errOwner(l);
    return "";
  };
  const shownError = failure.length ? null : cmd.error;
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{v.createTitle(l)}</h2>
      <p className="live-impact">{v.createLead(l)}</p>
      {(problems.length > 0 || failure.length > 0) && (
        <Summary locale={l} title={c.check(l)} focusKey={cmd.focusKey}
          problems={problems.length ? problems : (["svc-name", "svc-owner"].filter((id) => field(id)).map((id) => ({ field: id, message: field(id) })))} />
      )}
      {shownError !== null && <FailureBox l={l} error={shownError} focusKey={cmd.focusKey} describe={describeFor("create", page.limits.services)} />}
      <div className="service-field">
        <label htmlFor="svc-name">{v.name(l)}</label>
        <p className="service-hint" id="svc-name-hint">{v.nameHint(l)}</p>
        {field("svc-name") && <p className="service-field-error" id="svc-name-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("svc-name")}</p>}
        <span className="live-unit live-prefixed" dir="ltr">
          <span className="live-mono" aria-hidden="true">{prefix}</span>
          <input id="svc-name" className="live-mono" autoComplete="off" spellCheck={false} value={name} aria-invalid={field("svc-name") ? true : undefined}
            aria-describedby={["svc-name-hint", field("svc-name") ? "svc-name-error" : ""].filter(Boolean).join(" ")} onChange={(e) => edit(setName)(e.target.value)} />
        </span>
        <p className="live-note">{v.idPreview(l)} <bdi dir="ltr" className="live-mono">{prefix + (name || "…")}</bdi></p>
      </div>
      {page.owners.length === 0 ? <p className="live-note">{v.noOwners(l)}</p> : (
        <div className="service-field">
          <label htmlFor="svc-owner">{v.owner(l)}</label>
          <p className="service-hint" id="svc-owner-hint">{v.ownerHint(l)}</p>
          {field("svc-owner") && <p className="service-field-error" id="svc-owner-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{field("svc-owner")}</p>}
          <select id="svc-owner" className="live-select" value={owner} aria-invalid={field("svc-owner") ? true : undefined}
            aria-describedby={["svc-owner-hint", field("svc-owner") ? "svc-owner-error" : ""].filter(Boolean).join(" ")} onChange={(e) => edit(setOwner)(e.target.value)}>
            <option value="">{v.chooseOwner(l)}</option>
            {page.owners.map((o) => <option key={o.user_id} value={o.user_id}>{o.login || o.user_id}</option>)}
          </select>
        </div>
      )}
      <Facts items={[[v.credentialsFact(l), v.credAfter(l)]]} />
      <Reason l={l} value={reason} onChange={edit(setReason)} error={field("live-reason")} />
      <div className="live-buttons">
        <button type="submit" className="live-primary" aria-busy={cmd.busy || undefined} disabled={cmd.busy || page.owners.length === 0}>{cmd.busy ? w.saving(l) : v.createTitle(l)}</button>
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

function IssueForm({ l, api, page, s, stepping, onIssued, onReload, onAccounts, onPin }: {
  l: Locale; api: AdminClient; page: ServicesPage; s: ServiceRow; stepping: boolean;
  onIssued: (ref: string, credential: string, until: string) => void; onReload: () => void; onAccounts: () => void; onPin: () => void;
}) {
  const max = page.limits.credential_days;
  const [days, setDays] = useState(String(max));
  const [reason, setReason] = useState("");
  const [problems, setProblems] = useState<Problems>([]);
  const cmd = useCommand(api, page.revision, onReload, true);
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => { if (stepping) heading.current?.focus(); }, [stepping]);
  const block = cmd.holding ? "" : issueBlock(s, page.limits.credentials_per_service);
  const n = wholeDays(days, max);
  const ends = n ? when(l, new Date(Date.parse(page.now) + n * 86400000).toISOString()) : "";
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: Problems = [];
    if (n === null) found.push({ field: "svc-days", message: v.errDays(l, max) });
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    setProblems(found);
    cmd.setError(null);
    if (found.length) { cmd.setFocusKey((k) => k + 1); return; }
    onPin();
    const done = await cmd.run("service.issue_credential", s.id, reason, { days: String(n) });
    if (!done) return;
    const credential = done.fields.credential ?? "";
    const ref = done.fields.secret_ref ?? "";
    if (!/^cred-[0-9a-f]{16}$/.test(credential) || !/^[A-Za-z0-9_-]{16,64}$/.test(ref)) {
      cmd.hold(new AdminError("unavailable"));
      return;
    }
    onIssued(ref, credential, new Date(Date.parse(page.now) + (n ?? 0) * 86400000).toISOString());
  };
  const field = (id: string) => problems.find((x) => x.field === id)?.message ?? "";
  const daysError = field("svc-days") || (cmd.error instanceof AdminError && cmd.error.failure === "invalid" && cmd.error.fields.includes("days") ? v.errDays(l, max) : "");
  const refused = cmd.error instanceof AdminError && cmd.error.failure === "refused";
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1} ref={heading}>{v.issueTitle(l)}</h2>
      <p className="live-who live-mono">{ltr(s.id)}</p>
      {(block !== "" || refused) && cmd.error !== null && <FailureBox l={l} error={cmd.error} focusKey={cmd.focusKey} describe={describeFor("issue")} />}
      {block === "retired" ? <p className="live-impact">{v.blockedRetired(l)}</p>
        : block === "suspended" || (refused && !cmd.holding && s.state === "suspended") ? (
          <>
            <p className="live-impact">{v.blockedSuspended(l, s.id)}</p>
            <div className="live-buttons"><button type="button" className="live-secondary" onClick={onAccounts}>{v.goAccounts(l)}</button></div>
          </>
        ) : block === "limit" || (refused && !cmd.holding) ? <p className="live-impact">{v.blockedLimit(l, s.id, page.limits.credentials_per_service)}</p> : (
          <>
            <p className="live-impact">{v.issueLead(l)}</p>
            {(problems.length > 0 || daysError) && <Summary locale={l} title={c.check(l)} focusKey={cmd.focusKey} problems={problems.length ? problems : [{ field: "svc-days", message: daysError }]} />}
            {cmd.error !== null && !daysError && <FailureBox l={l} error={cmd.error} focusKey={cmd.focusKey} describe={describeFor("issue")} />}
            <div className="service-field live-minutes">
              <label htmlFor="svc-days">{v.validFor(l)}</label>
              <p className="service-hint" id="svc-days-hint">{v.daysHint(l, max)}</p>
              {daysError && <p className="service-field-error" id="svc-days-error"><span className="visually-hidden">{c.errorPrefix(l)} </span>{daysError}</p>}
              <span className="live-unit">
                <input id="svc-days" inputMode="numeric" dir="ltr" value={days} aria-invalid={daysError ? true : undefined}
                  aria-describedby={["svc-days-hint", daysError ? "svc-days-error" : ""].filter(Boolean).join(" ")} onChange={(e) => { cmd.renew(); setDays(e.target.value); }} />
                <span>{v.days(l)}</span>
              </span>
              {ends && <p className="live-note" aria-live="polite">{v.endsOn(l, ends)}</p>}
            </div>
            <Reason l={l} value={reason} onChange={(x) => { cmd.renew(); setReason(x); }} error={field("live-reason")} />
            <div className="live-buttons">
              <button type="submit" className="live-primary" aria-busy={cmd.busy || undefined} disabled={cmd.busy}>{cmd.busy ? w.saving(l) : cmd.holding ? c.tryAgain(l) : v.issueTitle(l)}</button>
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
      (e) => setState(e instanceof AdminError && e.failure === "unavailable" && !retried.current ? "retry" : "gone"),

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
    const warn = (e: BeforeUnloadEvent) => { e.preventDefault(); };
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

function CredentialTable({ l, rows, selected, onRevoke, retired = false }: { l: Locale; rows: CredentialRow[]; selected: string; onRevoke?: (c: CredentialRow) => void; retired?: boolean }) {
  return (
    <div className="security-table-scroll live-table live-access">
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
            return (
              <tr key={x.id} role="row" className={selected === x.id ? "live-selected" : undefined}>
                <td role="cell" className="live-user live-mono">{ltr(x.id)}</td>
                <td role="cell" data-label={v.colValid(l)}><span className="live-to">{v.from(l, when(l, x.not_before))}</span><span className="live-to">{v.until(l, when(l, x.not_after))}</span></td>
                <td role="cell" data-label={v.colStatus(l)} className={x.state !== "active" || !x.usable ? "live-strong" : ""}>{state}{sub && <span className="live-sub">{sub}</span>}</td>
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

export default function LiveServices({ locale: l, view, session, onSignOut, onView }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<ServicesPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [mode, setMode] = useState<Mode>({ kind: "form" });
  const [chosen, setChosen] = useState("");
  const [unstored, setUnstoredState] = useState<string[]>(() => pending.get(session.user) ?? []);
  const setUnstored = (f: (u: string[]) => string[]) => setUnstoredState((u) => { const next = f(u); pending.set(session.user, next); return next; });
  const [notice, setNotice] = useState("");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [formKey, setFormKey] = useState(0);
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (!focusTo) return;
    (document.querySelector<HTMLElement>(focusTo) ?? document.querySelector<HTMLElement>("#svc-list-title"))?.focus();
    setFocusTo("");
  }, [focusTo]);
  const say = (text: string) => { setNotice(text); setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const load = useCallback(() => {
    const n = order.current.next();
    api.services().then(
      (got) => { if (order.current.latest(n)) { setPage(got); setStatus("ready"); setError(null); } },
      (e) => { if (order.current.latest(n)) { setError(e); setStatus("error"); } },
    );
  }, [api]);
  useEffect(() => { if (view === "accounts" || view === "credentials") load(); }, [load, view]);
  const viewRef = useRef(view);
  useEffect(() => {
    if (viewRef.current !== view) { setMode({ kind: "form" }); setNotice(""); setOpen(false); }
    viewRef.current = view;
  }, [view]);
  useEffect(() => {
    if (unstored.length === 0) return;
    const warn = (e: BeforeUnloadEvent) => { e.preventDefault(); };
    addEventListener("beforeunload", warn);
    return () => removeEventListener("beforeunload", warn);
  }, [unstored.length]);
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
  const credentialsView = view === "credentials";
  const current = page ? find(initialService(page, chosen)) : undefined;
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
      panel = <CreateForm key={formKey} l={l} api={api} page={page} stepping={!wide} onReload={load}
        onDone={(id) => finished(v.doneCreated(l, id), { kind: "detail", id }, "#svc-detail-title")} />;
      backTo = ".svc-create-open";
    }
    back = v.backList(l);
  }
  if (page && credentialsView && current) {
    if (mode.kind === "reveal") {
      panel = <Reveal key={mode.ref} l={l} api={api} service={mode.service} refId={mode.ref} credential={mode.credential}
        until={find(mode.service)?.credentials.find((x) => x.id === mode.credential)?.not_after ?? mode.until}
        onLeave={() => { setMode({ kind: "form" }); setOpen(false); load(); setFocusTo("#svc-cred-title"); }}
        onDone={() => { setUnstored((u) => u.filter((x) => x !== mode.credential)); setMode({ kind: "form" }); setOpen(false); setFormKey((k) => k + 1); load(); setFocusTo("#svc-cred-title"); }}
        onRevoke={() => setMode({ kind: "revoke", id: mode.credential, service: mode.service, reason: v.revokeUnstoredReason(l) })} />;
    } else if (mode.kind === "revoke") {
      const target = mode;
      panel = <RevokeStep key={target.id} l={l} api={api} page={page} id={target.id} service={target.service} initial={target.reason} onReload={load}
        onCancel={() => { setMode({ kind: "form" }); setFocusTo(`[data-credential="${target.id}"]`); }}
        onDone={(message) => { setUnstored((u) => u.filter((x) => x !== target.id)); finished(message, { kind: "form" }, "#svc-cred-title"); }} />;
      backTo = `[data-credential="${target.id}"]`;
    } else if (wide || mode.kind === "form") {
      panel = <IssueForm key={current.id + formKey} l={l} api={api} page={page} s={current} stepping={!wide} onReload={load} onAccounts={() => onView("accounts")} onPin={() => setChosen(current.id)}
        onIssued={(ref, credential, until) => { setNotice(""); setUnstored((u) => (u.includes(credential) ? u : [...u, credential])); load(); setMode({ kind: "reveal", ref, service: current.id, credential, until }); }} />;
      backTo = ".svc-issue-open";
    }
    back = v.backCreds(l);
  }
  const stepping = !wide && (mode.kind !== "form" || open);
  const groups = page ? split(page) : { current: [], retired: [] };
  const creds = current ? credentials(current) : { current: [], ended: [] };
  const selected = mode.kind === "detail" || mode.kind === "step" ? mode.id : mode.kind === "revoke" ? mode.id : "";
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
      {mode.kind !== "reveal" && unstored.length > 0 && (
        <div className="service-summary live-unstored">
          <h3>{v.unstoredTitle(l)}</h3>
          <p>{v.unstoredLead(l)}</p>
          <ul className="live-unstored-list">
            {unstored.map((id) => (
              <li key={id}>
                <span><span className="live-mono">{ltr(id)}</span> · <span className="live-mono">{ltr(page?.services.find((x) => x.credentials.some((y) => y.id === id))?.id ?? "")}</span></span>
                <span className="live-row-buttons">
                  <button type="button" className="live-secondary" onClick={() => setMode({ kind: "revoke", id, service: page?.services.find((x) => x.credentials.some((y) => y.id === id))?.id ?? current?.id ?? "", reason: v.revokeUnstoredReason(l) })}>{v.revoke(l)}<span className="visually-hidden"> {id}</span></button>
                  <button type="button" className="live-secondary" onClick={() => setUnstored((u) => u.filter((x) => x !== id))}>{v.dismiss(l)}<span className="visually-hidden"> {id}</span></button>
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
            <select id="svc-select" className="live-select live-mono" value={current.id} onChange={(e) => { setChosen(e.target.value); setMode({ kind: "form" }); setNotice(""); }}>
              {page.services.map((s) => <option key={s.id} value={s.id}>{s.state === "active" ? s.id : `${s.id} (${serviceState(l, s.state)})`}</option>)}
            </select>
          </div>
          <div className="live-note-rule"><p><strong>{v.prelaunchLead(l)}</strong> {v.prelaunchNote(l)}</p></div>
          <p className="live-meta live-count">{v.credSummary(l, liveCount(current), max)}</p>
          {creds.current.length === 0 ? <p className="live-empty">{v.noCreds(l, current.id)}</p>
            : <CredentialTable l={l} rows={creds.current} retired={current.state === "retired"} selected={selected} onRevoke={(x) => { setNotice(""); setMode({ kind: "revoke", id: x.id, service: current.id, reason: "" }); }} />}
          {creds.ended.length > 0 && (
            <details className="live-ended"><summary>{v.endedGroup(l, creds.ended.length)}</summary><CredentialTable l={l} rows={creds.ended} selected="" /></details>
          )}
        </>
      )}
    </section>
  );
  const showPanel = panel !== null && (wide || stepping);
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
