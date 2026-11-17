import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { AccessPage, LedgerStatus, Revocation } from "../../../../service/admin";
import { when } from "../authentication/live-model";
import { accessText } from "./live-model";
import { v } from "./request-copy";
import { withWho } from "./who";
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

function targetText(l: Locale, page: AccessPage, r: Revocation): ReactNode {
  if (r.target_kind === "delegation") return v.delegationOf(l, r.target);
  const g = page.grants.find((x) => x.id === r.target);
  if (!g) return v.grantGone(l, r.target);
  const who = page.subjects.find((x) => x.user_id === g.subject)?.login || g.subject;
  return withWho(v.grantOf(l, mark, accessText(l, g.action, g.fields)), mark, who);
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

function Record({ l, page, r }: { l: Locale; page: AccessPage; r: Revocation }) {
  const words = stateWords(l, r);
  const groups = cleanupGroups(r.cleanup);
  return (
    <li className="access-record" data-revocation={r.id} data-state={r.state} tabIndex={-1}>
      <h3>{targetText(l, page, r)}</h3>
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

export function RevocationSection({ l, page }: { l: Locale; page: AccessPage }) {
  return (
    <section aria-labelledby="access-revocations">
      <div className="live-heading live-section-head">
        <h2 id="access-revocations" tabIndex={-1}>{v.title(l)}</h2>
      </div>
      <p className="live-impact">{v.lead(l)}</p>
      <LedgerLine l={l} ledger={page.ledger} />
      {page.revocations.length === 0 ? <p className="live-empty">{v.none(l)}</p> : (
        <ol className="access-records">{page.revocations.map((r) => <Record key={r.id} l={l} page={page} r={r} />)}</ol>
      )}
    </section>
  );
}
