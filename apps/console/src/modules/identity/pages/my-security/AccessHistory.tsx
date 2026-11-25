import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type SelfPage } from "../../../../service/admin";
import type { AccessHistoryPage, HistoryEvent } from "../../../../service/access-history";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { whenSeconds } from "./end-copy";
import { deviceLabel, emphasised, eventLabel, eventReason, hc, periods, type Period } from "./history-copy";
import { coverageGap, fromFor, historyState, noteLines, showSizeLimit } from "./history-model";
import { sq } from "./query-copy";
import { useScopedRead } from "./scoped-read";
import "./history.css";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function parts(l: Locale, iso: string): [string, string] {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return ["", ""];
  return [new Intl.DateTimeFormat(tags[l], { dateStyle: "medium" }).format(d).replace(/[\u061c\u200e\u200f]/g, ""), new Intl.DateTimeFormat(tags[l], { timeStyle: "medium" }).format(d).replace(/[\u061c\u200e\u200f]/g, "")];
}

export function HistoryRow({ l, e }: { l: Locale; e: HistoryEvent }) {
  const [date, time] = parts(l, e.at);
  const reason = eventReason(l, e);
  const state = e.session?.current ? hc.thisSession(l) : e.session?.active ? hc.signedInNow(l) : "";
  const device = deviceLabel(l, e);
  return (
    <tr role="row">
      <td role="cell" className="mysec-hist-time">
        <bdi dir="ltr"><bdi dir="ltr" className="mysec-nw">{date}</bdi> <bdi dir="ltr" className="mysec-nw">{time}</bdi></bdi>
      </td>
      <td role="cell" className="mysec-hist-event" data-label={hc.colEvent(l)}>
        <span className={"mysec-hist-label" + (emphasised(e) ? " mysec-hist-strong" : "")}>{eventLabel(l, e)}</span>
        {reason && <span className="mysec-hist-reason">{reason}</span>}
      </td>
      <td role="cell" className="mysec-hist-session" data-label={hc.colSession(l)}>
        {e.session ? (
          <>
            {state && <span className="mysec-hist-state">{state}</span>}
            <span className={"live-mono" + (state ? " mysec-hist-muted" : "")}>{ltr(e.session.handle)}</span>
          </>
        ) : (
          <>—<span className="visually-hidden"> {hc.noSession(l)}</span></>
        )}
      </td>
      <td role="cell" className="mysec-hist-device" data-label={hc.colDevice(l)}>
        {device ? <bdi>{device}</bdi> : <>—<span className="visually-hidden"> {hc.notRecorded(l)}</span></>}
      </td>
    </tr>
  );
}

type HistoryProps = {
  l: Locale; api: AdminClient; self: SelfPage; reloadKey: number; say: (text: string) => void; onSignedOut: () => void;
};

export default function AccessHistory(props: HistoryProps) {
  const client = useRef({ api: props.api, revision: 0 });
  if (client.current.api !== props.api) client.current = { api: props.api, revision: client.current.revision + 1 };
  const key = JSON.stringify([client.current.revision, props.self.user_id, props.self.workspace, props.self.current_session.handle]);
  return <HistoryContent key={key} {...props} />;
}

function HistoryContent({ l, api, self, reloadKey, say, onSignedOut }: HistoryProps) {
  const read = useScopedRead(api);
  const [period, setPeriod] = useState<Period>("all");
  const [page, setPage] = useState<AccessHistoryPage | null>(null);
  const rows = page?.events ?? [];
  const [before, setBefore] = useState<string | undefined>();
  const [trail, setTrail] = useState<(string | undefined)[]>([]);
  const [queryFrom, setQueryFrom] = useState<string | undefined>();
  const [next, setNext] = useState<string | null>(null);
  const [tableLoading, setTableLoading] = useState(false);
  const [failed, setFailed] = useState("");
  const [more, setMore] = useState<"idle" | "busy" | "failed">("idle");
  const [moreRef, setMoreRef] = useState("");
  const [ended, setEnded] = useState(false);
  const [focusTo, setFocusTo] = useState("");
  const serverNow = useRef("");
  const retryPage = useRef<{ target: string | undefined; parents: (string | undefined)[] } | null>(null);
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const fail = (err: unknown) => {
    if (err instanceof AdminError && err.failure === "unauthenticated") {
      setPage(null);
      setNext(null);
      onSignedOut();
    }
  };
  const loadFirst = useCallback((chosen: Period, mode: "background" | "period" | "retry") => {
    const valid = read.begin();
    const request = attempt().request;
    const from = fromFor(chosen, serverNow.current);
    if (mode !== "background") setTableLoading(true);
    api.accessHistory({ from }, request).then(
      (got) => {
        if (!valid()) return;
        serverNow.current = got.now;
        setPage(got);
        setBefore(undefined);
        setTrail([]);
        setQueryFrom(from);
        setNext(got.next);
        setFailed("");
        setTableLoading(false);
        setMore("idle");
        setEnded(false);
        if (mode === "period") say(got.events.length ? hc.showing(l, chosen, got.events.length, got.next !== null) : got.available_from ? hc.empty(l) : hc.noneKept(l));
        if (mode === "retry") setFocusTo("#mysec-history-title");
      },
      (err) => {
        if (!valid()) return;
        fail(err);
        setTableLoading(false);
        setFailed(request);
        if (mode !== "background") say(hc.loadError(l));
      },
    );
  }, [api, l]);
  useEffect(() => {
    loadFirst(period, "background");
  }, [api, reloadKey]);
  const changePeriod = (p: Period) => {
    setPeriod(p);
    setPage(null);
    setNext(null);
    loadFirst(p, "period");
  };
  const loadPage = (target: string | undefined, parents: (string | undefined)[]) => {
    if (more === "busy" || tableLoading) return;
    const valid = read.begin();
    const request = attempt().request;
    retryPage.current = { target, parents };
    setMore("busy");
    api.accessHistory({ from: queryFrom, before: target }, request).then(
      (got) => {
        if (!valid()) return;
        setBefore(target);
        setTrail(parents);
        setNext(got.next);
        setPage(got);
        setMore("idle");
        say(hc.showing(l, period, got.events.length, got.next !== null));
        setEnded(got.next === null);
        setFocusTo("#mysec-history-title");
      },
      (err) => {
        if (!valid()) return;
        fail(err);
        setMore("failed");
        setMoreRef(request);
        say(hc.loadError(l));
        setFocusTo(".mysec-hist-more-failed");
      },
    );
  };
  const showMore = () => {
    if (next) loadPage(next, [...trail, before].slice(-32));
  };
  const previous = () => {
    if (trail.length) loadPage(trail[trail.length - 1], trail.slice(0, -1));
  };
  const time = (iso: string) => whenSeconds(l, iso);
  const state = historyState(page, failed !== "");
  const from = fromFor(period, serverNow.current);
  const lines = page ? noteLines(l, page, time) : [];
  const heading = (
    <div className="mysec-hist-head">
      <h2 id="mysec-history-title" className="live-title" tabIndex={-1}>{hc.title(l)}</h2>
      {page && !failed && <p className="live-meta">{w.loadedAt(l, whenSeconds(l, page.now))}</p>}
    </div>
  );
  if (state === "loading" && !page) {
    return (
      <section className="mysec-hist" aria-labelledby="mysec-history-title">
        {heading}
        <p className="live-meta mysec-hist-loading">{hc.loading(l)}</p>
      </section>
    );
  }
  if (state === "failed" || !page) {
    return (
      <section className="mysec-hist" aria-labelledby="mysec-history-title">
        {heading}
        <div className="service-summary mysec-hist-failed" tabIndex={-1}>
          <p>{hc.loadError(l)} {hc.requestRef(l)} <span className="live-mono">{ltr(failed)}</span></p>
          <button type="button" className="live-secondary" onClick={() => loadFirst(period, "retry")}>{c.tryAgain(l)}</button>
        </div>
      </section>
    );
  }
  const shown = rows.length > 0 ? "rows" : state;
  return (
    <section className="mysec-hist" aria-labelledby="mysec-history-title">
      {heading}
      <div className="mysec-hist-policy">
        <p className="mysec-hist-lead">{hc.retentionLead(l, page.retention_days, page.retention_source === "start_flag")}</p>
        <p>{hc.retentionBody(l)}</p>
        {showSizeLimit(page) && <p className="mysec-hist-strong">{hc.sizeLimit(l, page.effective_days)}</p>}
        <p>{page.available_from ? hc.coverage(l, time(page.available_from)) : hc.noneKept(l)}</p>
        <p className="mysec-hist-muted">{hc.minimisation(l)} {hc.notShown(l)}</p>
      </div>
      {lines.length > 0 && (
        <div className="service-status mysec-hist-notes" aria-labelledby="mysec-hist-notes-title">
          <h3 id="mysec-hist-notes-title" className="visually-hidden">{hc.notesTitle(l)}</h3>
          <ul>
            {lines.map((n) => <li key={n.key} className={n.strong ? "mysec-hist-strong" : undefined}>{n.text}</li>)}
          </ul>
        </div>
      )}
      <div className="mysec-hist-controls">
        <div className="mysec-hist-period">
          <label htmlFor="mysec-hist-period">{hc.period(l)}</label>
          <select id="mysec-hist-period" value={period} onChange={(e) => changePeriod(e.target.value as Period)}>
            {periods.map((p) => <option key={p} value={p}>{hc.periodOption(l, p)}</option>)}
          </select>
          {coverageGap(from, page.available_from) && page.available_from && <p className="live-meta">{hc.nothingBefore(l, time(page.available_from))}</p>}
        </div>
        <div className="mysec-hist-scope">
          <p>{hc.workspace(l)} {ltr(self.workspace)}</p>
          <p className="mysec-hist-muted">{hc.scopeNote(l)}</p>
        </div>
      </div>
      {tableLoading ? (
        <p className="live-meta mysec-hist-loading">{hc.loading(l)}</p>
      ) : shown === "rows" ? (
        <div className="security-table-scroll live-table mysec-hist-table">
          <table role="table">
            <caption className="visually-hidden">{hc.caption(l)}</caption>
            <colgroup><col className="mysec-hist-c1" /><col className="mysec-hist-c2" /><col className="mysec-hist-c3" /><col className="mysec-hist-c4" /></colgroup>
            <thead role="rowgroup"><tr role="row">
              <th scope="col" role="columnheader">{hc.colTime(l)}</th>
              <th scope="col" role="columnheader">{hc.colEvent(l)}</th>
              <th scope="col" role="columnheader">{hc.colSession(l)}</th>
              <th scope="col" role="columnheader">{hc.colDevice(l)}</th>
            </tr></thead>
            <tbody role="rowgroup">
              {rows.map((e, i) => <HistoryRow key={`${e.at}-${e.kind}-${e.session?.handle ?? ""}-${i}`} l={l} e={e} />)}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="live-empty">{shown === "none-kept" ? hc.noneKept(l) : hc.empty(l)}</p>
      )}
      {!tableLoading && (
        <div className="mysec-hist-more">
          <button type="button" className="live-secondary" disabled={more === "busy" || before === undefined} onClick={() => loadPage(undefined, [])}>{sq.first(l)}</button>
          <button type="button" className="live-secondary" disabled={more === "busy" || trail.length === 0} onClick={previous}>{sq.previous(l)}</button>
          {more === "failed" && (
            <div className="service-summary mysec-hist-more-failed" tabIndex={-1}>
              <p>{hc.loadError(l)} {hc.requestRef(l)} <span className="live-mono">{ltr(moreRef)}</span></p>
              <button type="button" className="live-secondary" onClick={() => { const pending = retryPage.current; if (pending) loadPage(pending.target, pending.parents); }}>{c.tryAgain(l)}</button>
            </div>
          )}
          {next !== null && more !== "failed" && (
            <button type="button" className="live-secondary mysec-hist-more-btn" aria-busy={more === "busy" || undefined} disabled={more === "busy"} onClick={showMore}>
              {more === "busy" ? hc.loadingMore(l) : sq.next(l)}
            </button>
          )}
          {ended && <p className="mysec-hist-end mysec-hist-muted" tabIndex={-1}>{hc.endLine(l)}</p>}
        </div>
      )}
    </section>
  );
}
