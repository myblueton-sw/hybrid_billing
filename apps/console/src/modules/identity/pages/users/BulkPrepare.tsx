import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type Attempt, type UserRow } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { Summary } from "../../../../service/Service";
import { bulkClient, stage, stagedBatches, StageError, type BulkDetail, type BulkExclusion, type BulkTarget, type Paged } from "../../../../service/identity-bulk";
import { b } from "./bulk-copy";
import { digestShown, expired, number } from "./bulk-model";
import { FailureBox, Reason, reasonProblem } from "./Live";
import { failureText, roleName } from "./live-model";
import { w } from "./live-copy";
import { Emph, PeriodCell } from "./RolePeriod";
import { At, Stamp } from "./Stamp";
import { m } from "./match-copy";
import { emptyFilter, type Fixed } from "./match-model";
import { FixedChips, FixedFacts, fillAll } from "./Matching";

export type Prepared = { detail: BulkDetail; targets: Paged<BulkTarget> | null };
type Stage =
  | { kind: "effect" }
  | { kind: "sending"; sent: number; waiting: boolean; resent?: boolean }
  | { kind: "failed"; error: unknown; draft: string; staged: number; revision: string; cause: "" | "draft" | "limit" }
  | { kind: "blocked"; draft: string }
  | { kind: "closed"; draft: string }
  | { kind: "preview"; prepared: Prepared };

function Sum({ rows }: { rows: [string, React.ReactNode, boolean?][] }) {
  return (
    <dl className="m01-sum">
      {rows.map(([dt, dd, strong]) => <div key={dt}><dt>{dt}</dt><dd className={strong ? "live-strong" : undefined}>{dd}</dd></div>)}
    </dl>
  );
}

function Excluded({ l, api, id, total }: { l: Locale; api: AdminClient; id: string; total: number }) {
  const [rows, setRows] = useState<Paged<BulkExclusion> | null>(null);
  const [failed, setFailed] = useState(false);
  const more = async () => {
    const page = await bulkClient(api).excluded(id, rows?.rows.length ?? 0).catch(() => null);
    setFailed(!page);
    if (page) setRows({ ...page, rows: [...(rows?.rows ?? []), ...page.rows] });
  };
  const [unread, setUnread] = useState(false);
  const first = () => { setRows(null); setUnread(false); setFailed(false); if (total > 0) void bulkClient(api).excluded(id).then(setRows, () => { setRows(null); setUnread(true); }); };
  useEffect(first, [id, total]);
  if (unread) return <div className="live-meta" role="alert"><p>{b.excludedFailed(l)}</p><button type="button" className="live-secondary" onClick={first}>{c.tryAgain(l)}</button></div>;
  if (!rows || rows.rows.length === 0) return null;
  return (
    <>
      <ul className="m01-people">{rows.rows.map((x) => <li key={x.user_id}><bdi dir="ltr">{x.login || x.user_id}</bdi> · {b.reason(l, x.reason)}</li>)}</ul>
      <p className="live-meta">{b.showing(l, rows.rows.length, rows.total)}</p>
      {rows.next && <button type="button" className="live-secondary" onClick={() => void more()}>{b.moreExcluded(l)}</button>}
      {failed && <p className="live-meta" role="alert">{b.moreFailed(l)}</p>}
    </>
  );
}

const filterReasons = ["self", "not_found", "no_longer_matches", "no_membership", "already_revoked"];
const explicitReasons = ["self", "not_found", "no_membership", "already_revoked"];

export function PreviewBody({ l, api, prepared, onMore, moreFailed = false, days = 14 }: { l: Locale; api: AdminClient; prepared: Prepared; onMore: () => void; moreFailed?: boolean; days?: number }) {
  const { detail, targets } = prepared;
  const i = detail.impact;
  const job = detail.job;
  return (
    <>
      <dl className="m01-sum m01-when">
        <div><dt>{b.preparedAtL(l)}</dt><dd><Stamp l={l} iso={job.prepared_at} /></dd></div>
        <div><dt>{b.validUntilL(l)}</dt><dd><Stamp l={l} iso={job.expires_at} /></dd></div>
      </dl>
      <p className="m01-lead m01-ink"><strong>{b.scope(l, job.effect, job.counts.targets)}</strong></p>
      <dl className="m01-counts">
        <div><dt>{b.toRevoke(l)}</dt><dd>{job.counts.targets}</dd></div>
        <div><dt>{b.excluded(l)}</dt><dd>{job.counts.excluded}</dd></div>
        <div><dt>{b.duplicates(l)}</dt><dd>{job.counts.duplicates}</dd></div>
      </dl>
      {job.selection === "filter" && <FixedFacts l={l} job={job} days={days} exclusions={detail.exclusions} />}
      <div className="m01-preview-grid">
        <section className="m01-impact" aria-labelledby="m01-impact-h">
          <h3 id="m01-impact-h">{b.impact(l)}</h3>
          <p className="live-meta"><At l={l} template={b.computedAt(l)} iso={i.computed_at} date={false} zone={false} /></p>
          <h4>{b.peopleRoles(l)}</h4>
          <Sum rows={[[b.countWords(l, "targets"), i.people], [b.administrators(l), i.administrators], [b.members(l), i.members]]} />
          <h4>{b.rights(l)}</h4>
          <Sum rows={[[b.installationAdmins(l), i.installation_administrators], [b.trustHolders(l), i.trust_right_holders], [b.bulkHolders(l), i.bulk_right_holders]]} />
          <h4>{b.sessionsKeys(l)}</h4>
          <Sum rows={[[b.sessions(l), i.sessions], [b.pickups(l), i.pickups], [b.apiKeys(l), i.api_keys === "not_applicable" ? b.noKeys(l) : b.impactUnknown(l)]]} />
          <h4>{b.pendingWork(l)}</h4>
          <Sum rows={[...i.pending.map((s): [string, React.ReactNode, boolean] => [b.store(l, s.store),
            <><span className="m01-line">{s.count}</span><span className={"m01-line" + (s.rechecked ? "" : " live-strong")}>{s.store === "result_evidence_proposals" ? s.rechecked ? b.resultProposerRechecked(l) : b.resultProposerNotRechecked(l) : s.store === "result_applications_unknown" ? s.rechecked ? b.resultRecoveryRechecked(l) : b.resultRecoveryNotRechecked(l) : s.store === "document_evidence_proposals" ? s.rechecked ? b.documentProposerRechecked(l) : b.documentProposerNotRechecked(l) : (s.store === "financial_charge_requests" || s.store === "erp_final_requests") ? b.chargeRequesterNotRechecked(l) : s.store === "issuance_setup_proposals" ? s.rechecked ? b.setupProposerRechecked(l) : b.setupProposerNotRechecked(l) : s.store === "issuance_protected_steps" ? s.rechecked ? b.nextStepRechecked(l) : b.nextStepNotRechecked(l) : s.rechecked ? b.rechecked(l) : b.notRechecked(l)}</span></>, false]),
          [b.pendingJobs(l), i.pending_jobs === "not_applicable" ? b.noJobs(l) : b.impactUnknown(l)]]} />
          <h4>{b.serviceUsers(l)}</h4>
          <Sum rows={[[b.serviceUsers(l), <><span className="m01-line">{i.service_users}</span><span className="m01-line">{b.serviceKeep(l)}</span></>]]} />
          <h4>{b.afterwards(l)}</h4>
          <p className={i.anchors_after <= 1 ? "live-strong" : undefined}>{b.anchors(l, i.anchors_after)}</p>
          {i.other_workspaces && <><h4>{b.scopeTitle(l)}</h4><p>{b.otherWorkspaces(l)}</p></>}
        </section>
        <section className="m01-excl" aria-labelledby="m01-excl-h">
          <h3 id="m01-excl-h">{b.excludedN(l, job.counts.excluded)}</h3>
          <Sum rows={(job.selection === "filter" ? filterReasons : explicitReasons).map((r): [string, React.ReactNode] => [b.reason(l, r), detail.exclusions[r] ?? 0])} />
          <Excluded l={l} api={api} id={job.id} total={job.counts.excluded} />
        </section>
      </div>
      {targets ? (
        <section className="m01-targets" aria-labelledby="m01-targets-h">
          <h3 id="m01-targets-h">{b.toRevokeN(l, targets.total)}</h3>
          <ul className="m01-people">
            {targets.rows.map((t) => (
              <li key={t.user_id}><bdi dir="ltr">{t.login || t.user_id}</bdi> · {roleName(l, t.role)}{t.installation_administrator ? " · " + w.installationAdmin(l) : ""}</li>
            ))}
          </ul>
          <p className="live-meta">{b.showing(l, targets.rows.length, targets.total)}</p>
          {targets.next && <button type="button" className="live-secondary" onClick={onMore}>{b.showMore(l)}</button>}
          {moreFailed && <p className="live-meta" role="alert">{b.moreFailed(l)}</p>}
        </section>
      ) : <p className="m01-owner-only">{b.ownerOnlyPeople(l)}</p>}
      <section className="m01-outside" aria-labelledby="m01-outside-h">
        <h3 id="m01-outside-h">{b.notChanged(l)}</h3>
        <ul>{detail.outside_scope.map((code) => <li key={code} className={code === "B10" ? "live-strong" : undefined}>{b.item(l, code)}</li>)}</ul>
      </section>
    </>
  );
}

export function ConfirmDialog({ l, api, prepared, revision, defaultReason, onClose, onAccepted, onReload }: {
  l: Locale; api: AdminClient; prepared: Prepared; revision: string; defaultReason: string; onClose: () => void; onAccepted: (id: string) => void; onReload: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const heading = useRef<HTMLHeadingElement>(null);
  const [reason, setReason] = useState(defaultReason);
  const [checked, setChecked] = useState(false);
  const [problems, setProblems] = useState<{ field: string; message: string }[]>([]);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [focusKey, setFocusKey] = useState(0);
  const tried = useRef<Attempt>(attempt());
  const job = prepared.detail.job;
  useEffect(() => {
    const d = dialog.current;
    if (d && !d.open && typeof d.showModal === "function") d.showModal();
    heading.current?.focus();
  }, []);
  const submit = async (e: FormEvent) => {
    e.preventDefault();
    const found: { field: string; message: string }[] = [];
    const r = reasonProblem(l, reason);
    if (r) found.push({ field: "live-reason", message: r });
    if (!checked) found.push({ field: "m01-checked", message: b.confirmCheck(l) });
    setProblems(found);
    setError(null);
    if (found.length) { setFocusKey((k) => k + 1); return; }
    setBusy(true);
    try {
      const out = await api.command("bulk_revocation.execute", job.id, revision, reason, { digest: job.digest }, tried.current);
      dialog.current?.close();
      onAccepted(out.job_id || job.id);
    } catch (err) {
      setError(err);
      setFocusKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };
  const final = error instanceof AdminError && ["stage", "invalid", "refused", "conflict", "stale", "denied"].includes(error.failure);
  const describe = (lc: Locale, e: unknown) => (e instanceof AdminError && e.failure === "refused" ? { title: b.refused(lc), body: b.blocked(lc) } : failureText(lc, e));
  const close = () => { if (busy) return; dialog.current?.close(); if (final) onReload(); onClose(); };
  return (
    <dialog className="m01-confirm" ref={dialog} aria-labelledby="m01-confirm-title" onCancel={(e) => { e.preventDefault(); close(); }}>
      <form onSubmit={submit} noValidate>
        <h2 id="m01-confirm-title" tabIndex={-1} ref={heading}>{b.confirmTitle(l)}</h2>
        {problems.length > 0 && <Summary locale={l} title={c.check(l)} problems={problems} focusKey={focusKey} />}
        {error !== null && <FailureBox l={l} error={error} focusKey={focusKey} describe={describe} />}
        <dl className="m01-confirm-facts">
          <div><dt>{b.list(l)}</dt><dd>{b.bulkN(l, number(job.id))} <bdi dir="ltr" className="live-mono">{job.id}</bdi></dd></div>
          <div><dt>{b.effect(l)}</dt><dd>{job.effect === "suspend_user" ? b.suspendUser(l) : b.suspendMembership(l)}</dd></div>
          <div><dt>{b.toRevoke(l)}</dt><dd>{job.counts.targets}</dd></div>
          <div><dt>{b.excluded(l)}</dt><dd>{job.counts.excluded}</dd></div>
          <div><dt>{b.preparedBy(l)}</dt><dd><bdi dir="ltr">{job.preparer}</bdi></dd></div>
          <div><dt>{b.preparedAtL(l)}</dt><dd><Stamp l={l} iso={job.prepared_at} /></dd></div>
          <div><dt>{b.validUntilL(l)}</dt><dd><Stamp l={l} iso={job.expires_at} /></dd></div>
          <div className="m01-wide"><dt>{b.digest(l)}</dt><dd><bdi dir="ltr" className="live-mono">{digestShown(job.digest)}</bdi></dd></div>
        </dl>
        <p className="m01-scope m01-ink"><strong>{b.scope(l, job.effect, job.counts.targets)}</strong></p>
        <h3>{b.notChanged(l)}</h3>
        <ul>{prepared.detail.outside_scope.map((code) => <li key={code}>{b.item(l, code)}</li>)}</ul>
        <div className="m01-consequence"><p className="m01-ink"><strong>{b.consequence(l)}</strong></p><p>{b.reversal(l, job.effect)}</p></div>
        <Reason l={l} value={reason} onChange={setReason} error={problems.find((x) => x.field === "live-reason")?.message ?? ""} />
        <label className="live-check">
          <input id="m01-checked" type="checkbox" checked={checked} onChange={(e) => setChecked(e.target.checked)} />
          {b.checked(l)}
        </label>
        <div className="live-buttons m01-confirm-actions">
          {!final && <button type="submit" className="live-primary" aria-busy={busy || undefined} disabled={busy}>{b.revokeFor(l, job.counts.targets)}</button>}
          <button type="button" className="live-secondary" onClick={close} disabled={busy}>{final ? b.reloadClose(l) : b.goBack(l)}</button>
        </div>
      </form>
    </dialog>
  );
}

export function BulkFlow({ l, api, ids, names, admins, rows, serverTime, revision, reload, onBack, onJobs, onExecuted, onDeselect, say, fixed = null, days = 14, onPrepared, onDiscarded }: {
  l: Locale; api: AdminClient; ids: string[]; names: Record<string, string>; admins: string[]; rows: UserRow[]; serverTime: string; revision: string; reload: () => Promise<string>;
  onBack: () => void; onJobs: () => void; onExecuted: (id: string, targets: string[]) => void; onDeselect: (id: string) => void; say: (text: string) => void;
  fixed?: Fixed | null; days?: number; onPrepared?: () => void; onDiscarded?: (closed: boolean) => void;
}) {
  const bulk = bulkClient(api);
  const [effect, setEffect] = useState<"suspend_membership" | "suspend_user">("suspend_membership");
  const [reason, setReason] = useState("");
  const [problem, setProblem] = useState("");
  const [state, setState] = useState<Stage>({ kind: "effect" });
  const [confirming, setConfirming] = useState(false);
  const [rev, setRev] = useState(revision);
  const [focusKey, setFocusKey] = useState(0);
  const tries = useRef<Attempt[]>([]);
  const heading = useRef<HTMLHeadingElement>(null);
  const progress = useRef<HTMLParagraphElement>(null);
  useEffect(() => heading.current?.focus(), [state.kind === "preview"]);
  const load = async (id: string, after = 0, previous?: Paged<BulkTarget>): Promise<Prepared> => {
    const detail = await bulk.detail(id);
    const page = await bulk.targets(id, after);
    return { detail, targets: previous ? { ...page, rows: [...previous.rows, ...page.rows] } : page };
  };
  const [trouble, setTrouble] = useState<"" | "discard" | "more">("");
  const prepareFixed = async (f: Fixed) => {
    setTrouble("");
    setState({ kind: "sending", sent: 0, waiting: false });
    setTimeout(() => progress.current?.focus(), 0);
    say(m.checking(l, f.matched));
    try {
      const prepared = await api.command("bulk_revocation.prepare", f.draft, await reload(), reason, { effect }, attempt());
      onPrepared?.();
      setRev(prepared.revision || (await reload()));
      setState({ kind: "preview", prepared: await load(prepared.fields.bulk) });
      say(b.impactTitle(l));
    } catch (e) {
      if (e instanceof AdminError && e.failure === "refused") setState({ kind: "blocked", draft: f.draft });
      else if (e instanceof AdminError && e.failure === "stage") setState({ kind: "closed", draft: f.draft });
      else setState({ kind: "failed", error: e, draft: "", staged: 0, revision: "", cause: "" });
      setFocusKey((k) => k + 1);
    }
  };
  const send = async (draft = "", from = 0, startRevision = rev) => {
    setTrouble("");
    setState({ kind: "sending", sent: 0, waiting: false });
    setTimeout(() => progress.current?.focus(), 0);
    say(b.sending(l, 0, ids.length));
    let out;
    try {
      out = await stage(api, ids, { rationale: reason, revision: startRevision, reload, draft, from, tries: tries.current,
        progress: (pr) => { setState({ kind: "sending", sent: pr.sent, waiting: pr.waiting, resent: pr.resent }); if (pr.waiting) say(b.limited(l, pr.sent, ids.length)); } });
    } catch (e) {
      const s = e as StageError;
      const refused = s.cause instanceof AdminError && s.cause.failure === "refused" && !s.draft;
      const cause = refused ? await bulk.list("", 1).then((pg) => (pg.open_draft ? "draft" as const : "limit" as const), () => "" as const) : "";
      setState({ kind: "failed", error: s.cause, draft: s.draft, staged: s.staged, revision: s.revision, cause });
      setFocusKey((k) => k + 1);
      return;
    }
    try {
      const prepared = await api.command("bulk_revocation.prepare", out.draft, out.revision, reason, { effect }, attempt());
      setRev(prepared.revision || (await reload()));
      setState({ kind: "preview", prepared: await load(prepared.fields.bulk) });
      say(b.impactTitle(l));
    } catch (e) {
      if (e instanceof AdminError && e.failure === "refused") setState({ kind: "blocked", draft: out.draft });
      else setState({ kind: "failed", error: e, draft: out.draft, staged: ids.length, revision: await reload(), cause: "" });
      setFocusKey((k) => k + 1);
    }
  };
  const start = (e: FormEvent) => {
    e.preventDefault();
    const pr = reasonProblem(l, reason);
    setProblem(pr);
    if (pr) { setFocusKey((k) => k + 1); return; }
    if (fixed) { void prepareFixed(fixed); return; }
    tries.current = Array.from({ length: Math.max(1, Math.ceil(ids.length / 150) + 2) }, () => attempt());
    void send();
  };
  const discard = async (id: string) => {
    try {
      await api.command("bulk_revocation.discard", id, await reload(), reason || b.discardReason(l), {}, attempt());
    } catch {
      setTrouble("discard");
      return;
    }
    setTrouble("");
    say(b.discarded(l));
    onBack();
  };
  const discardFixed = async (id: string) => {
    try {
      await api.command("bulk_revocation.discard", id, await reload(), reason || b.discardReason(l), {}, attempt());
    } catch (e) {
      if (!(e instanceof AdminError && e.failure === "stage")) { setTrouble("discard"); return; }
      setTrouble("");
      onDiscarded?.(true);
      return;
    }
    setTrouble("");
    onDiscarded?.(false);
  };
  const sending = state.kind === "sending";
  const proceed = async () => {
    if (state.kind !== "preview") return;
    const fresh = await bulk.detail(state.prepared.detail.job.id).catch(() => null);
    if (fresh && expired(fresh.job, fresh.server_time)) {
      setState({ kind: "preview", prepared: { ...state.prepared, detail: fresh } });
      setFocusKey((k) => k + 1);
      return;
    }
    setConfirming(true);
  };
  return (
    <section className="m01-bulk" aria-labelledby="m01-bulk-title" data-bulk={state.kind === "preview" ? "preview" : "effect"}>
      {!sending && <button type="button" className="service-button-link live-back" onClick={onBack}>{b.back(l)}</button>}
      {state.kind !== "preview" ? (
        <>
          <h2 id="m01-bulk-title" tabIndex={-1} ref={heading}>{b.prepare(l)}</h2>
          <p className="m01-stepline">{b.step1(l)}</p>
          {problem && <Summary locale={l} title={c.check(l)} problems={[{ field: "live-reason", message: problem }]} focusKey={focusKey} />}
          {state.kind === "failed" && (
            <div className="service-summary" role="alert" tabIndex={-1}>
              <h2>{b.notPrepared(l)}</h2>
              {state.cause ? (
                <>
                  <p>{state.cause === "draft" ? b.openDraft(l) : b.pendingLimit(l)}</p>
                  <p><button type="button" className="service-button-link m01-link" onClick={onJobs}>{b.openJobs(l)}</button></p>
                </>
              ) : (
                <>
                  <p>{failureText(l, state.error).body}</p>
                  <p>{b.staged(l, state.staged, ids.length)}</p>
                </>
              )}
              <p><strong>{b.nothingRevoked(l)}</strong></p>
              <div className="live-buttons">
                {state.draft && <button type="button" className="live-secondary" onClick={() => void send(state.draft, stagedBatches(ids, state.staged), state.revision)}>{b.resume(l)}</button>}
                {state.draft && <button type="button" className="live-secondary" onClick={() => void discard(state.draft)}>{b.discardDraft(l)}</button>}
              </div>
              {trouble === "discard" && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
            </div>
          )}
          {fixed && (state.kind === "blocked" || state.kind === "closed") && (
            <div className="service-summary" role="alert" tabIndex={-1} ref={(el) => el?.focus()}>
              <h2>{state.kind === "blocked" ? b.refused(l) : b.notPrepared(l)}</h2>
              <p className="m01-ink">{state.kind === "blocked" ? <Emph l={l} text={m.blockedFilter(l, `${w.workspaceRole(l)}: ${roleName(l, "member")}`)} /> : m.listClosed(l)}</p>
              <p><strong>{b.nothingRevoked(l)}</strong></p>
              <button type="button" className="live-secondary m01-discard-list" onClick={() => void discardFixed(state.draft)}>{m.discardList(l)}</button>
              {trouble === "discard" && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
            </div>
          )}
          {!fixed && state.kind === "blocked" && (
            <div className="service-summary" role="alert" tabIndex={-1}>
              <h2>{b.refused(l)}</h2>
              <p className="m01-ink"><Emph l={l} text={b.blocked(l)} /></p>
              <h3>{b.adminsInSelection(l)}</h3>
              <ul className="m01-admins">
                {admins.map((id) => {
                  const row = rows.find((u) => u.user_id === id);
                  return (
                    <li key={id}>
                      <span className="m01-admin-who"><bdi dir="ltr">{names[id] || id}</bdi> · {roleName(l, row?.membership?.role ?? "administrator")}</span>
                      {row && <span className="m01-admin-period"><PeriodCell l={l} u={row} serverTime={serverTime} /></span>}
                      <button type="button" className="live-secondary" onClick={() => onDeselect(id)}>{b.removeFromSelection(l)}</button>
                    </li>
                  );
                })}
              </ul>
              <p>{b.previewAgain(l)}</p>
              <button type="button" className="live-secondary" onClick={() => void discard(state.draft)}>{b.discardDraft(l)}</button>
              {trouble === "discard" && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
            </div>
          )}
          <div className="m01-bulk-grid">
            <form className="m01-effect-form" onSubmit={start} noValidate>
              <fieldset className="live-roles m01-effects">
                <legend>{b.effect(l)}</legend>
                {(["suspend_membership", "suspend_user"] as const).map((e) => (
                  <label key={e}>
                    <input type="radio" name="m01-effect" value={e} checked={effect === e} aria-disabled={sending || undefined} onChange={() => { if (!sending) setEffect(e); }} />
                    <span className="m01-opt">{e === "suspend_user" ? b.suspendUser(l) : b.suspendMembership(l)}</span>
                    <span className="m01-opt-hint">{e === "suspend_user" ? b.suspendUserHint(l) : b.suspendMembershipHint(l)}</span>
                  </label>
                ))}
              </fieldset>
              <Reason l={l} value={reason} onChange={(v) => { if (!sending) setReason(v); }} error={problem} />
              {sending && (
                <p className="m01-progress" tabIndex={-1} ref={progress}>
                  {fixed ? m.checking(l, fixed.matched) : state.waiting ? b.limited(l, state.sent, ids.length) : b.sending(l, state.sent, ids.length)}
                  {state.resent && !state.waiting && <> {b.resent(l)}</>}
                </p>
              )}
              {!sending && state.kind !== "blocked" && state.kind !== "closed" && !(state.kind === "failed" && state.draft) && <div className="live-buttons"><button type="submit" className="live-primary">{b.previewImpact(l)}</button></div>}
            </form>
            {fixed ? (
              <section className="m01-selsum" aria-labelledby="m01-selsum-h">
                <h3 id="m01-selsum-h">{m.everyoneMatching(l, fixed.matched)}</h3>
                <p>{fillAll(m.fixedShort(l, fixed.matched), { "{n}": <strong>{fixed.matched}</strong>, "{at}": <Stamp l={l} iso={fixed.fixed_at} zone={false} /> })}</p>
                {!emptyFilter(fixed.filter) && <div className="m01-fixedfilters"><span className="m01-fixedlead">{m.fixedFilters(l)}</span><FixedChips l={l} filter={fixed.filter} masked={false} days={days} /></div>}
                <p className="m01-fixedmeta">{b.bulkN(l, number(fixed.draft))} · {m.baseRevision(l)} <bdi dir="ltr" className="live-mono">{fixed.fixed_revision}</bdi></p>
                <p><strong>{b.nothingRevoked(l)}</strong> {m.serverChecks(l)}</p>
              </section>
            ) : (
              <section className="m01-selsum" aria-labelledby="m01-selsum-h">
                <h3 id="m01-selsum-h">{b.selectedPeople(l, ids.length)}</h3>
                <p><strong>{b.nothingRevoked(l)}</strong> {b.serverChecks(l)}</p>
                <details className="m01-sellist">
                  <summary>{b.showSelected(l, ids.length)}</summary>
                  <ul className="m01-ids">{ids.map((id) => <li key={id}><bdi dir="ltr">{names[id] || id}</bdi></li>)}</ul>
                </details>
              </section>
            )}
          </div>
        </>
      ) : (
        <>
          <h2 id="m01-bulk-title" tabIndex={-1} ref={heading}>{b.impactTitle(l)} <span className="live-meta">{b.bulkN(l, number(state.prepared.detail.job.id))}</span></h2>
          <p className="m01-stepline">{b.step2(l)}</p>
          {expired(state.prepared.detail.job, state.prepared.detail.server_time) && <div className="service-summary" role="alert"><p><Emph l={l} text={b.expired(l)} /></p></div>}
          <PreviewBody l={l} api={api} prepared={state.prepared} days={days} moreFailed={trouble === "more"} onMore={async () => { try { const prepared = await load(state.prepared.detail.job.id, state.prepared.targets?.rows.length ?? 0, state.prepared.targets ?? undefined); setTrouble(""); setState({ kind: "preview", prepared }); } catch { setTrouble("more"); } }} />
          <div className="live-buttons">
            {expired(state.prepared.detail.job, state.prepared.detail.server_time)
              ? <button type="button" className="live-primary" onClick={() => { setTrouble(""); setState({ kind: "effect" }); }}>{b.prepareAgain(l)}</button>
              : <button type="button" className="live-primary m01-continue" onClick={() => void proceed()}>{b.continue(l)}</button>}
            <button type="button" className="live-secondary" onClick={() => void discard(state.prepared.detail.job.id)}>{b.discard(l)}</button>
          </div>
          {trouble === "discard" && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
          {confirming && (
            <ConfirmDialog l={l} api={api} prepared={state.prepared} revision={rev} defaultReason={reason}
              onClose={() => { setConfirming(false); setTimeout(() => document.querySelector<HTMLElement>(".m01-continue")?.focus(), 0); }}
              onReload={() => void reload().then(setRev)}
              onAccepted={async (id) => {
                const targets: string[] = [];
                for (let after = 0; ; ) {
                  const page = await bulk.targets(state.prepared.detail.job.id, after, 100).catch(() => ({ rows: [], next: "", total: 0 }));
                  targets.push(...page.rows.map((t) => t.user_id));
                  if (!page.next) break;
                  after = Number(page.next);
                }
                setConfirming(false);
                say(b.started(l));
                onExecuted(id, targets);
              }} />
          )}
        </>
      )}
    </section>
  );
}

