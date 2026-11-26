import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient, LedgerStatus } from "../../../../service/admin";
import type { ListedRevocation } from "../../../../service/access-lists";
import { when } from "../authentication/live-model";
import { a, kindLabel } from "./live-copy";
import { accessText } from "./live-model";
import { v } from "./request-copy";
import { breakable } from "./who";
import { ListFailure, Paging, Reference, usePagedList } from "./paged";
import { qcopy } from "../service-accounts/query-copy";
import { cleanupGroups, failureWords, itemWords, jobWords, ledgerLines, settled, stateWords, type Groups } from "./request-model";

export function LedgerLine({ l, ledger }: { l: Locale; ledger: LedgerStatus }) {
  const lines = ledgerLines(l, ledger, (at) => when(l, at));
  return (
    <div className="access-ledger" data-ledger={ledger.state}>
      {lines.map((line, i) => <p key={i} className={i === 0 && ledger.state !== "ok" ? "live-strong" : "live-note"}>{line}</p>)}
      <p className="live-note">{v.ledgerChain(l)}</p>
      <details className="live-ended">
        <summary>{v.details(l)}</summary>
        <dl className="live-facts">
          <div><dt>{v.ledgerId(l)}</dt><dd className="live-mono" dir="ltr">{ledger.ledger_id || "—"}</dd></div>
          <div><dt>{v.createdAt(l)}</dt><dd>{ledger.created_at ? when(l, ledger.created_at) : "—"}</dd></div>
          <div><dt>{v.watermark(l)}</dt><dd className="live-mono access-wrap" dir="ltr">{ledger.watermark || "—"}</dd></div>
          <div><dt>{v.checkedAt(l)}</dt><dd>{ledger.checked_at ? when(l, ledger.checked_at) : "—"}</dd></div>
          <div><dt>{v.file(l)}</dt><dd className="live-mono" dir="ltr">{ledger.file || "—"}</dd></div>
        </dl>
      </details>
    </div>
  );
}

const mark = "\u0000who\u0000";

function Target({ l, r }: { l: Locale; r: ListedRevocation }) {
  if (r.target_action === "") return <h3>{r.target_kind === "delegation" ? v.delegationOf(l, r.target) : v.grantGone(l, r.target)}</h3>;
  const access = accessText(l, r.target_action, r.target_fields);
  const text = r.target_kind === "delegation" ? v.delegationTo(l, mark, access) : v.grantOf(l, mark, access);
  const [before, after] = text.split(mark);
  return (
    <>
      <h3>{before}<bdi dir="ltr">{breakable(r.target_subject_login || r.target_subject)}</bdi>{after}</h3>
      <p className="live-meta access-target">{kindLabel(l, r.target_resource_kind) ?? r.target_resource_kind} · <bdi dir="ltr" className="live-mono" translate="no">{r.target}</bdi></p>
    </>
  );
}

function Cleanup({ l, groups }: { l: Locale; groups: Groups }) {
  const order: (keyof Groups)[] = ["done", "not_applicable", "outside_scope", "not_started"];
  return (
    <div className="access-cleanup">
      {order.filter((k) => groups[k].length > 0).map((k) => (
        <div key={k}>
          <h4>{v.group[k](l)}</h4>
          <ul role="list" className="access-bullets">
            {groups[k].map((it) => {
              const words = itemWords(l, it);
              return <li key={it.item}><span className="live-strong">{words.name}</span> — {words.why}{it.refs.length > 0 && <span className="live-mono" dir="ltr"> ({it.refs.join(", ")})</span>}</li>;
            })}
          </ul>
        </div>
      ))}
    </div>
  );
}

function Record({ l, r }: { l: Locale; r: ListedRevocation }) {
  const words = stateWords(l, r);
  const groups = cleanupGroups(r.cleanup);
  return (
    <li className="access-record" data-revocation={r.id} data-state={r.state} tabIndex={-1}>
      <Target l={l} r={r} />
      <p><span className="live-strong">{words.name}</span> · {jobWords(l, r)}</p>
      <p>{words.copy}</p>
      {r.state === "decision_unconfirmed" && <p className="live-strong">{v.thisProcess(l)}</p>}
      <p className="live-meta">
        {r.origin === "ledger_replay" ? <>{v.earlierRun(l)} · {v.recordedAt(l, when(l, r.recorded_at))}{r.replay && <> · {v.replay[r.replay](l)}</>}</> : v.receivedAt(l, when(l, r.received_at))}
      </p>
      {r.ended_delegations.length > 0 && <p className="live-meta">{v.endedDelegs(l)}: <span className="live-mono" dir="ltr">{r.ended_delegations.join(", ")}</span></p>}
      {r.cleanup.length > 0 && r.state !== "decision_unconfirmed" && <p className="live-meta">{v.progress(l, settled(r.cleanup), r.cleanup.length)}</p>}
      <Cleanup l={l} groups={groups} />
      <details className="live-ended">
        <summary>{v.details(l)}</summary>
        <dl className="live-facts">
          <div><dt>{v.revocationId(l)}</dt><dd className="live-mono" dir="ltr">{r.id}</dd></div>
          {r.request && <div><dt>{v.request(l)}</dt><dd className="live-mono" dir="ltr">{r.request}</dd></div>}
          <div><dt>{v.sequence(l)}</dt><dd className="live-mono" dir="ltr">{r.receipt.sequence || "—"}</dd></div>
          <div><dt>{v.watermark(l)}</dt><dd className="live-mono access-wrap" dir="ltr">{r.receipt.watermark || "—"}</dd></div>
          <div><dt>{v.denyRevision(l)}</dt><dd className="live-mono" dir="ltr">{r.deny_generation || "—"}</dd></div>
          {r.receipt.reason && <div><dt>{v.failure(l)}</dt><dd>{failureWords(l, r.receipt.reason)}</dd></div>}
        </dl>
      </details>
    </li>
  );
}

export function RevocationRecords({ l, rows }: { l: Locale; rows: ListedRevocation[] }) {
  return <ol className="access-records">{rows.map((r) => <Record key={r.id} l={l} r={r} />)}</ol>;
}

export function RevocationSection({ l, api, generation, ledger, onSignOut }: { l: Locale; api: AdminClient; generation: number; ledger: LedgerStatus; onSignOut: () => void }) {
  const list = usePagedList((after) => api.accessRevocations({ limit: 10, ...(after ? { after } : {}) }), "revocations", generation);
  const page = list.read.state === "ready" ? list.read.page : null;
  return (
    <section aria-labelledby="access-revocations" className="access-revocation-section">
      <div className="live-heading live-section-head">
        <h2 id="access-revocations" tabIndex={-1}>{v.title(l)}</h2>
        {page && <p className="live-meta" data-revocations={page.count.value}>{page.count.certainty === "exact" ? v.revocationsShown(l, Number(page.count.value)) : v.revocationsAtLeast(l, Number(page.count.value))}</p>}
      </div>
      <p className="live-impact">{v.lead(l)}</p>
      <LedgerLine l={l} ledger={ledger} />
      {list.read.state === "loading" && <p role="status" className="live-meta">{a.loading(l)}</p>}
      <ListFailure l={l} title={v.revocationsFailed(l)} list={list} onSignOut={onSignOut} />
      {page && (page.rows.length === 0 ? <p className="live-empty">{page.next ? qcopy(l, "incomplete") : v.none(l)}</p> : <RevocationRecords l={l} rows={page.rows} />)}
      {page && <Reference l={l} page={page} list={list} />}
      <Paging l={l} label={v.title(l)} list={list} />
    </section>
  );
}
