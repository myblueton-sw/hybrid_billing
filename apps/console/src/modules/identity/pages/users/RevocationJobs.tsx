import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { bulkClient, type BulkDetail, type BulkItem, type BulkJob, type BulkPage, type BulkTarget, type Paged } from "../../../../service/identity-bulk";
import { b } from "./bulk-copy";
import { groups, itemWords, jobWords, number, split } from "./bulk-model";
import { ConfirmDialog, PreviewBody } from "./BulkPrepare";
import { failureText, roleName } from "./live-model";
import { At, fill, Stamp } from "./Stamp";
import { Settings } from "./RolePeriod";
import { w } from "./live-copy";
import { m } from "./match-copy";
import { BasisFacts, BasisLine } from "./Matching";
import { setting } from "./period-model";

export function ReasonCode({ l, code }: { l: Locale; code: string }) {
  const known = b.reasonCode(l, code);
  return known ? <>{known}</> : <>{b.reasonFallback(l)} <bdi dir="ltr" className="live-mono">{code}</bdi></>;
}

function Counts({ l, job }: { l: Locale; job: BulkJob }) {
  const k = job.counts;
  return (
    <>
      <dl className="m01-counts">
        {(["targets", "succeeded", "held", "failed", "unprocessed"] as const).map((key) => <div key={key}><dt>{b.countWords(l, key)}</dt><dd>{k[key]}</dd></div>)}
        <div className="m01-apart"><dt>{b.countWords(l, "excluded")}</dt><dd>{k.excluded}</dd></div>
      </dl>
      <p className="m01-equation">{b.equation(l, k)}</p>
    </>
  );
}

function StateLine({ l, job }: { l: Locale; job: BulkJob }) {
  const words = jobWords(job);
  if (words === "partial") return <div className="m01-state-line"><p><strong>{b.partial(l)}</strong></p><p>{b.error(l)}</p></div>;
  const strong = words === "unconfirmed" || words === "not_started" || words === "waiting";
  return (
    <div className="m01-state-line">
      <p className={strong ? "m01-strong" : undefined}>{b.words(l, words)}</p>
      {words === "unconfirmed" && <p className="m01-ink">{b.unconfirmedBody(l)}</p>}
      {words === "finished" && job.counts.held > 0 && <p>{b.heldLine(l, job.counts.held)}</p>}
    </div>
  );
}

function Item({ l, item }: { l: Locale; item: BulkItem }) {
  const words = itemWords(item);
  const g = groups(item.cleanup);
  const who = item.login || item.user_id;
  return (
    <li className="m01-item">
      <div>
        <p className="live-strong"><bdi dir="ltr">{who}</bdi></p>
        <p className="live-sub live-mono"><bdi dir="ltr">{item.user_id}</bdi></p>
        {item.role && <p className="live-sub">{roleName(l, item.role)}</p>}
      </div>
      <div>
        <p className={words === "complete" || words === "excluded" ? undefined : "live-strong"}>
          {b.itemWords(l, words)}{item.outcome === "excluded" && item.reason ? " · " + b.reason(l, item.reason) : ""}{item.outcome === "failed" && item.reason && <> · <ReasonCode l={l} code={item.reason} /></>}
        </p>
        {words === "held" && <p>{b.prepareHeld(l)}</p>}
        {item.receipt.sequence > 0 ? <p className="live-mono">{b.record(l, item.receipt.sequence)}</p> : item.receipt.reason && <p className="live-strong">{fill(b.notConfirmed(l, "{at}"), <ReasonCode l={l} code={item.receipt.reason} />)}</p>}
        {item.cleanup.length > 0 && (
          <details className="m01-cleanup">
            <summary>{b.cleanupFor(l, who)}</summary>
            {([["done", g.done], ["na", g.not_applicable], ["outside", g.outside_scope]] as const).map(([key, list]) => list.length > 0 && (
              <div key={key}><h4>{key === "done" ? b.done(l) : key === "na" ? b.na(l) : b.outside(l)}</h4><ul>{list.map((x) => <li key={x.item}>{b.item(l, x.item)}</li>)}</ul></div>
            ))}
          </details>
        )}
        {item.replay_note && <p className="live-sub">{b.replay(l, item.replay_note)}</p>}
      </div>
    </li>
  );
}

function Scope({ l }: { l: Locale }) {
  return (
    <section className="m01-scope" aria-labelledby="m01-scope-h">
      <h3 id="m01-scope-h">{b.covers(l)}</h3>
      <h4>{b.done(l)}</h4><ul>{["B1", "B2", "B3", "B4"].map((x) => <li key={x}>{b.item(l, x)}</li>)}</ul>
      <h4>{b.na(l)}</h4><ul>{["B5", "B6"].map((x) => <li key={x}>{b.item(l, x)}</li>)}</ul>
      <h4>{b.outside(l)}</h4><ul>{["B7", "B8", "B9", "B10"].map((x) => <li key={x} className={x === "B10" ? "live-strong" : undefined}>{b.item(l, x)}</li>)}</ul>
    </section>
  );
}

type Opened = { detail: BulkDetail; items: Paged<BulkItem>; targets: Paged<BulkTarget> | null };

function Detail({ l, api, opened, revision, wide, onClose, onReload, onMore, moreFailed, outcome, onOutcome, self, days }: {
  l: Locale; api: AdminClient; opened: Opened; revision: string; wide: boolean; onClose: () => void; onReload: () => void; onMore: () => void; moreFailed: boolean;
  outcome: string; onOutcome: (o: string) => void; self: string; days: number;
}) {
  const job = opened.detail.job;
  const heading = useRef<HTMLHeadingElement>(null);
  const [confirming, setConfirming] = useState(false);
  useEffect(() => heading.current?.focus(), [job.id]);
  const owner = job.preparer === self;
  const [discardFailed, setDiscardFailed] = useState(false);
  useEffect(() => setDiscardFailed(false), [job.id, job.state]);
  const discard = async () => {
    setDiscardFailed(false);
    try {
      await api.command("bulk_revocation.discard", job.id, revision, b.discardReason(l), {}, { request: "m01-d-" + job.id + Date.now(), key: "m01-d-" + job.id + Date.now() });
    } catch (e) {
      if (!(e instanceof AdminError && e.failure === "stage")) setDiscardFailed(true);
    } finally {
      onReload();
    }
  };
  return (
    <>
      <div className="m01-detail-head">
        <h2 id="m01-job-title" tabIndex={-1} ref={heading}>{b.bulkN(l, number(job.id))}</h2>
        {job.effect && <p className="live-sub">{job.effect === "suspend_user" ? b.suspendUser(l) : b.suspendMembership(l)}</p>}
        {wide && <button type="button" className="live-secondary" onClick={onClose}>{b.close(l)}</button>}
      </div>
      {job.state === "prepared" ? (
        <>
          {job.selection !== "filter" && <BasisFacts l={l} job={job} days={days} />}
          <PreviewBody l={l} api={api} prepared={{ detail: opened.detail, targets: opened.targets }} onMore={onMore} moreFailed={moreFailed} days={days} />
          <div className="live-buttons">
            {owner && <button type="button" className="live-primary m01-continue" onClick={() => setConfirming(true)}>{b.continue(l)}</button>}
            {owner && <button type="button" className="live-secondary" onClick={() => void discard()}>{b.discard(l)}</button>}
          </div>
          {discardFailed && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
          {!owner && <p className="live-meta m01-not-owner">{b.notOwner(l)}</p>}
          {confirming && <ConfirmDialog l={l} api={api} prepared={{ detail: opened.detail, targets: opened.targets }} revision={revision} defaultReason=""
            onClose={() => setConfirming(false)} onReload={onReload} onAccepted={() => { setConfirming(false); onReload(); }} />}
        </>
      ) : job.state === "draft" ? (
        <>
          <BasisFacts l={l} job={job} days={days} />
          <StateLine l={l} job={job} />
          {owner && <button type="button" className="live-secondary" onClick={() => void discard()}>{b.discardDraft(l)}</button>}
          {discardFailed && <p className="live-meta" role="alert">{b.discardFailed(l)}</p>}
        </>
      ) : job.state !== "executed" ? (
        <>
          <BasisFacts l={l} job={job} days={days} />
          <StateLine l={l} job={job} />
        </>
      ) : (
        <>
          <BasisFacts l={l} job={job} days={days} />
          <StateLine l={l} job={job} />
          <Counts l={l} job={job} />
          <dl className="m01-sum">
            <div><dt>{b.preparedBy(l)}</dt><dd><bdi dir="ltr">{job.preparer}</bdi></dd></div>
            <div><dt>{b.preparedAtL(l)}</dt><dd><Stamp l={l} iso={job.prepared_at} /></dd></div>
            <div><dt>{b.executedBy(l)}</dt><dd><bdi dir="ltr">{job.executor}</bdi></dd></div>
            <div><dt>{b.executedAt(l)}</dt><dd><Stamp l={l} iso={job.executed_at} /></dd></div>
            <div><dt>{b.blockedAt(l)}</dt><dd><Stamp l={l} iso={job.deny_at} /></dd></div>
            <div><dt>{b.ledger(l)}</dt><dd>{job.ledger.reason ? <strong>{fill(b.notConfirmed(l, "{at}"), <ReasonCode l={l} code={job.ledger.reason} />)}</strong>
              : <>{b.recordsL(l)} <bdi dir="ltr" className="live-mono">{job.ledger.first_sequence}–{job.ledger.last_sequence}</bdi></>}</dd></div>
            <div><dt>{b.cleanup(l)}</dt><dd>{b.cleanupOf(l, job.cleanup.done, job.cleanup.total)}</dd></div>
          </dl>
          {job.origin === "ledger_replay" && <p className="live-meta">{b.fromEarlier(l)}</p>}
          <Scope l={l} />
          <section className="m01-items" aria-labelledby="m01-items-h">
            <h3 id="m01-items-h">{b.peopleN(l, opened.items.total)}</h3>
            <div className="service-field m01-filter">
              <label htmlFor="m01-outcome">{b.show(l)}</label>
              <select id="m01-outcome" value={outcome} onChange={(e) => onOutcome(e.target.value)}>
                <option value="">{b.all(l)}</option>
                {(["succeeded", "held", "failed", "unprocessed", "excluded"] as const).map((o) => <option key={o} value={o}>{b.countWords(l, o === "excluded" ? "excluded" : o)}</option>)}
              </select>
            </div>
            <ol className="m01-items-list">{opened.items.rows.map((item) => <Item key={item.user_id + item.outcome} l={l} item={item} />)}</ol>
            <p className="live-meta">{b.showing(l, opened.items.rows.length, opened.items.total)}</p>
            {opened.items.next && <button type="button" className="live-secondary" onClick={onMore}>{b.morePeople(l)}</button>}
            {moreFailed && <p className="live-meta" role="alert">{b.moreFailed(l)}</p>}
          </section>
          <details className="m01-expert">
            <summary>{b.technical(l)}</summary>
            <p><bdi dir="ltr" className="live-mono">{job.id}</bdi></p>
            <p className="m01-digest"><bdi dir="ltr" className="live-mono">{job.digest}</bdi></p>
            <p><bdi dir="ltr" className="live-mono">{job.ledger.watermark}</bdi></p>
          </details>
        </>
      )}
    </>
  );
}

export function JobTable({ l, jobs, open, onOpen, days }: { l: Locale; jobs: BulkJob[]; open: string; onOpen: (id: string) => void; days: number }) {
  const strong = (j: BulkJob) => ["unconfirmed", "not_started", "waiting", "partial"].includes(jobWords(j));
  const people = (j: BulkJob) =>
    j.state === "draft" ? <>{b.notCounted(l)}{j.staged > 0 && <span className="live-sub m01-line">{b.sentSoFar(l, j.staged)}</span>}</>
      : j.state !== "executed" && !j.prepared_at ? <>{b.neverCounted(l)}</>
      : <>{j.state === "executed" ? b.equation(l, j.counts) : b.people(l, j.counts.targets)}{j.counts.excluded > 0 && <span className="live-sub m01-line">{b.excludedCount(l, j.counts.excluded)}</span>}</>;
  const when = (j: BulkJob) =>
    j.state === "prepared" ? <At l={l} template={b.validUntil(l)} iso={j.expires_at} zone={false} /> : j.executed_at ? <At l={l} template={b.executed(l)} iso={j.executed_at} zone={false} />
      : j.state === "draft" && j.selection === "filter" && j.fixed_at ? <At l={l} template={m.fixedWhen(l)} iso={j.fixed_at} zone={false} /> : "—";
  return (
    <div className="security-table-scroll">
      <table className="m01-jobs-table" role="table">
        <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{b.colJob(l)}</th><th scope="col" role="columnheader">{b.colState(l)}</th><th scope="col" role="columnheader">{b.countWords(l, "targets")}</th><th scope="col" role="columnheader">{b.colWhen(l)}</th></tr></thead>
        <tbody role="rowgroup">
          {jobs.map((j) => (
            <tr key={j.id} role="row" className={j.id === open ? "live-selected" : undefined}>
              <td role="cell" data-label={b.colJob(l)}>
                <span className="m01-opener">
                  <button type="button" className="m01-open" data-job={j.id} aria-current={j.id === open ? "true" : undefined} onClick={() => onOpen(j.id)}>{b.bulkN(l, number(j.id))}</button>
                  {j.id === open && <span className="m01-tag" aria-hidden="true">{b.open(l)}</span>}
                </span>
                {j.effect && <span className="live-sub m01-line">{j.effect === "suspend_user" ? b.suspendUser(l) : b.suspendMembership(l)}</span>}
                {j.origin === "ledger_replay" && <span className="live-sub m01-line">{b.fromEarlier(l)}</span>}
                <BasisLine l={l} job={j} days={days} />
                <span className={"live-sub m01-compact" + (strong(j) ? " m01-strong" : "")}>{b.words(l, jobWords(j))}</span>
                <span className="live-sub m01-compact">{people(j)}</span>
              </td>
              <td role="cell" data-label={b.colState(l)} className={strong(j) ? "m01-strong" : undefined}>{b.words(l, jobWords(j))}</td>
              <td role="cell" data-label={b.countWords(l, "targets")}>{people(j)}</td>
              <td role="cell" data-label={b.colWhen(l)}>{when(j)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export default function RevocationJobs({ l, api, wide, initial, self, revision, onSignOut }: {
  l: Locale; api: AdminClient; wide: boolean; initial: string; self: string; revision: string; onSignOut: () => void;
}) {
  const bulk = bulkClient(api);
  const [page, setPage] = useState<BulkPage | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [open, setOpen] = useState(initial);
  const [opened, setOpened] = useState<Opened | null>(null);
  const [outcome, setOutcome] = useState("");
  const [at, setAt] = useState<Date | null>(null);
  const load = useCallback(() => {
    bulk.list("", 100).then((p) => { setPage(p); setError(null); setAt(new Date()); }, setError);
  }, [api]);
  const moreJobs = async () => {
    if (!page?.next) return;
    try {
      const next = await bulk.list(page.next, 100);
      setPage({ ...next, jobs: [...page.jobs, ...next.jobs] });
    } catch (e) {
      setError(e);
    }
  };
  useEffect(() => { load(); }, [load]);
  const [moreFailed, setMoreFailed] = useState(false);
  const shown = useRef("");
  const turn = useRef(0), loadingMore = useRef(-1);
  const openJob = useCallback(async (id: string, filter = outcome): Promise<boolean> => {
    const mine = ++turn.current;
    setOpen(id);
    setMoreFailed(false);
    if (shown.current !== id) { shown.current = ""; setOpened(null); }
    try {
      const detail = await bulk.detail(id);
      const items = detail.job.state === "executed" ? await bulk.items(id, 0, 50, filter) : { total: 0, next: "", rows: [] };
      const targets = detail.job.state === "prepared" && detail.job.preparer === self ? await bulk.targets(id) : null;
      if (mine !== turn.current) return false;
      shown.current = id;
      setOpened({ detail, items, targets });
      return true;
    } catch (e) {
      if (mine !== turn.current) return false;
      if (shown.current !== id) setOpen((now) => (now === id ? "" : now));
      setError(e);
      return false;
    }
  }, [api, outcome, self]);
  useEffect(() => { if (initial) void openJob(initial); }, [initial]);
  const more = async () => {
    const mine = turn.current;
    if (!opened || loadingMore.current === mine) return;
    const id = opened.detail.job.id;
    loadingMore.current = mine;
    try {
      if (opened.detail.job.state === "prepared") {
        if (!opened.targets) return;
        const from = opened.targets.rows, t = await bulk.targets(id, from.length);
        if (mine !== turn.current) return;
        setOpened({ ...opened, targets: { ...t, rows: [...from, ...t.rows] } });
      } else {
        const from = opened.items.rows, i = await bulk.items(id, from.length, 50, outcome);
        if (mine !== turn.current) return;
        setOpened({ ...opened, items: { ...i, rows: [...from, ...i.rows] } });
      }
      setMoreFailed(false);
    } catch {
      if (mine === turn.current) setMoreFailed(true);
    } finally {
      if (loadingMore.current === mine) loadingMore.current = -1;
    }
  };
  if (error instanceof AdminError && error.failure === "unauthenticated") {
    return <section className="live-error" role="alert"><p>{w.ended(l)}</p><button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button></section>;
  }
  if (error instanceof AdminError && error.failure === "denied") {
    return <section className="live-unavailable"><h2>{w.deniedTitle(l)}</h2><p>{w.denied(l)}</p></section>;
  }
  const groupsOf = split(page?.jobs ?? []);
  const days = setting(page?.settings ?? [], "role_period_warning_days", 14);
  const close = () => { const id = open; turn.current++; shown.current = ""; setOpen(""); setOpened(null); setMoreFailed(false); setTimeout(() => document.querySelector<HTMLElement>(`[data-job="${CSS.escape(id)}"]`)?.focus(), 0); };
  const detail = opened && open ? (
    <Detail l={l} api={api} opened={opened} revision={page?.revision ?? revision} wide={wide} onClose={close} onReload={() => { load(); void openJob(open); }} onMore={() => void more()} moreFailed={moreFailed}
      outcome={outcome} onOutcome={(o) => { void openJob(open, o).then((ok) => { if (ok) setOutcome(o); }); }} self={self} days={days} />
  ) : null;
  const stepping = !wide && detail;
  return (
    <section className="m01-jobs" aria-labelledby="m01-jobs-title">
      <div className="live-toolbar">
        <div className="live-heading">
          <h2 id="m01-jobs-title">{b.jobs(l)}</h2>
          {at && <p className="live-meta">{fill(w.loadedAt(l, "{at}"), <bdi className="m01-part">{new Intl.DateTimeFormat(l === "ko" ? "ko-KR" : l === "ar" ? "ar" : "en-US", { timeStyle: "short" }).format(at)}</bdi>)}</p>}
        </div>
        <div className="live-tools"><button type="button" className="live-secondary" onClick={load}>{w.reload(l)}</button></div>
      </div>
      {page?.started_at && <p className="m01-note"><At l={l} template={b.notKept(l)} iso={page.started_at} /></p>}
      {error !== null && <section className="live-error" role="alert"><h2>{w.loadFailed(l)}</h2><p>{failureText(l, error).body}</p><button type="button" className="live-secondary" onClick={load}>{c.tryAgain(l)}</button></section>}
      {!page && error === null && <p role="status" className="live-meta">{w.loading(l)}</p>}
      {page && open && !opened && error === null && <p role="status" className="live-meta m01-job-loading">{w.loading(l)}</p>}
      {page && (
        <div className="m01-jobs-split" data-mode={wide ? "split" : "step"} data-open={detail ? "true" : "false"}>
          {stepping ? (
            <section className="m01-job-step">
              <button type="button" className="service-button-link live-back" onClick={close}>{b.backJobs(l)}</button>
              {detail}
            </section>
          ) : (
            <>
              <div className="m01-jobs-list">
                {page.jobs.length === 0 && <p className="live-empty">{b.noJobsYet(l)}</p>}
                {page.jobs.length > 0 && <>
                  <h3>{b.notExecuted(l, groupsOf.open.length)}</h3>
                  {groupsOf.open.length ? <JobTable l={l} jobs={groupsOf.open} open={open} days={days} onOpen={(id) => void openJob(id)} /> : <p className="live-empty">{b.noOpen(l)}</p>}
                  <h3>{b.jobsN(l, groupsOf.executed.length)}</h3>
                  {groupsOf.executed.length ? <JobTable l={l} jobs={groupsOf.executed} open={open} days={days} onOpen={(id) => void openJob(id)} /> : <p className="live-empty">{b.noExecuted(l)}</p>}
                </>}
                {groupsOf.ended.length > 0 && (
                  <details className="m01-ended">
                    <summary>{b.endedN(l, groupsOf.ended.length)}</summary>
                    <JobTable l={l} jobs={groupsOf.ended} open={open} days={days} onOpen={(id) => void openJob(id)} />
                  </details>
                )}
                {page.next && <><p className="live-meta">{b.loadedJobs(l, page.jobs.length)}</p><button type="button" className="live-secondary" onClick={() => void moreJobs()}>{b.moreJobs(l)}</button></>}
              </div>
              {detail && <aside className="m01-job" tabIndex={0} aria-labelledby="m01-job-title">{detail}</aside>}
            </>
          )}
        </div>
      )}
      {page && <Settings l={l} settings={page.settings} />}
    </section>
  );
}
