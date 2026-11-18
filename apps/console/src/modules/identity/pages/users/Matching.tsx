import { useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type UsersFilter, type UsersQuery } from "../../../../service/admin";
import { c as copy } from "../../../../service/copy";
import { bulkClient, matchValues, type BulkJob, type MatchCount } from "../../../../service/identity-bulk";
import { b } from "./bulk-copy";
import { fc } from "./filter-copy";
import { appliedKeys, type UserFilters } from "./filter-model";
import { chipText } from "./filters";
import { failureText } from "./live-model";
import { w } from "./live-copy";
import { m } from "./match-copy";
import { confirmable, countNote, countView, emptyFilter, fixedOfFields, keepsOwn, refusalCause, storedFilters, type Fixed, type RefusalCause } from "./match-model";
import { At, fill, Stamp } from "./Stamp";
import "./match.css";

export function fillAll(template: string, parts: Record<string, ReactNode>): ReactNode {
  const tokens = Object.keys(parts);
  const out: ReactNode[] = [];
  let rest = template;
  for (;;) {
    let at = -1;
    let token = "";
    for (const k of tokens) {
      const i = rest.indexOf(k);
      if (i >= 0 && (at < 0 || i < at)) { at = i; token = k; }
    }
    if (at < 0) break;
    out.push(rest.slice(0, at), <span key={out.length}>{parts[token]}</span>);
    rest = rest.slice(at + token.length);
  }
  out.push(rest);
  return <>{out}</>;
}

function terms(l: Locale, filter: UsersFilter, masked: boolean, days: number): { key: string; node: ReactNode }[] {
  const f = storedFilters(filter);
  const out: { key: string; node: ReactNode }[] = [];
  if (masked) out.push({ key: "q", node: m.searchApplied(l) });
  for (const k of appliedKeys(f)) {
    if (k === "q") out.push({ key: k, node: <span>{fill(fc.chip(fc.searchChip(l), "{q}"), <bdi dir="auto">{f.q}</bdi>, "{q}")}</span> });
    else out.push({ key: k, node: chipText(l, "users", k, f, days) });
  }
  return out;
}

export function FixedChips({ l, filter, masked, days }: { l: Locale; filter: UsersFilter; masked: boolean; days: number }) {
  const list = terms(l, filter, masked, days);
  if (list.length === 0) return <span>{m.everyoneInList(l)}</span>;
  return <ul className="m01-chips">{list.map((x) => <li key={x.key}><span className="m01-chip m01-chip-fixed">{x.node}</span></li>)}</ul>;
}

export function BasisLine({ l, job, days }: { l: Locale; job: BulkJob; days: number }) {
  if (job.selection !== "filter") return <span className="live-sub m01-line m01-basis">{m.basis(l, job.selection)}</span>;
  const list = job.filter ? terms(l, job.filter, job.search_masked, days) : [];
  return (
    <span className="live-sub m01-line m01-basis">
      {m.basis(l, "filter")}: {list.length === 0 ? m.everyoneInList(l) : list.map((x, i) => <span key={x.key}>{i > 0 && " · "}{x.node}</span>)}
    </span>
  );
}

export function FixedFacts({ l, job, days, exclusions }: { l: Locale; job: BulkJob; days: number; exclusions?: Record<string, number> }) {
  const gone = exclusions?.no_longer_matches ?? 0;
  return (
    <section className="m01-fixedfacts" aria-labelledby="m01-fixedfacts-h">
      <h3 id="m01-fixedfacts-h">{m.fixedList(l)}</h3>
      <dl className="m01-sum">
        <div><dt>{m.filters(l)}</dt><dd>{job.filter ? <FixedChips l={l} filter={job.filter} masked={job.search_masked} days={days} /> : m.everyoneInList(l)}</dd></div>
        <div><dt>{m.fixedAt(l)}</dt><dd><Stamp l={l} iso={job.fixed_at} /></dd></div>
        <div><dt>{m.matchedWhenFixed(l)}</dt><dd className="m01-num">{job.matched}</dd></div>
        <div><dt>{m.baseRevision(l)}</dt><dd><bdi dir="ltr" className="live-mono">{job.fixed_revision}</bdi></dd></div>
      </dl>
      {exclusions && <p>{m.arithmetic(l, job.matched, job.counts.targets, job.counts.excluded)}</p>}
      {gone > 0 && <p className="live-strong">{m.noLongerMatch(l, gone)}</p>}
      {exclusions && <p><At l={l} template={m.notAdded(l)} iso={job.fixed_at} zone={false} /></p>}
    </section>
  );
}

export function BasisFacts({ l, job, days }: { l: Locale; job: BulkJob; days: number }) {
  if (job.selection === "filter") return <FixedFacts l={l} job={job} days={days} />;
  return (
    <dl className="m01-sum m01-basisfacts">
      <div><dt>{m.basisL(l)}</dt><dd>{m.basis(l, job.selection)}</dd></div>
    </dl>
  );
}

export function fixedLine(l: Locale, f: Fixed): string {
  const note = countNote(f);
  const template = note === "differs" ? m.fixedDiffers(l, f.matched, f.counted ?? 0) : note === "below" ? m.fixedBelow(l, f.matched, f.counted ?? 0)
    : emptyFilter(f.filter) ? m.fixedEveryone(l, f.matched) : m.fixed(l, f.matched);
  return template;
}

function FixedLine({ l, f }: { l: Locale; f: Fixed }) {
  return <>{fillAll(fixedLine(l, f), { "{n}": <strong className="m01-num">{f.matched}</strong>, "{at}": <Stamp l={l} iso={f.fixed_at} zone={false} /> })}</>;
}

export function plainFixedLine(l: Locale, f: Fixed, at: string): string {
  return fixedLine(l, f).split("{n}").join(String(f.matched)).split("{at}").join(at);
}

export function FixedBar({ l, f, filters, days, number, error, onPrepare, onDiscard, say }: {
  l: Locale; f: Fixed; filters: UserFilters; days: number; number: string; error: unknown; onPrepare: () => void; onDiscard: () => void; say: (text: string) => void;
}) {
  const own = keepsOwn(f, filters);
  const was = useRef(own);
  useEffect(() => {
    if (own && !was.current) say(m.keepsOwn(l));
    was.current = own;
  }, [own]);
  return (
    <section className="m01-selbar m01-fixedbar" aria-labelledby="m01-selbar-h" data-fixed="true">
      <h3 id="m01-selbar-h" className="visually-hidden">{b.barHeading(l)}</h3>
      <p className="m01-selcount" tabIndex={-1}><FixedLine l={l} f={f} /></p>
      <div className="m01-selactions">
        <button type="button" className="live-secondary m01-prepare" onClick={onPrepare}>{b.prepare(l)}</button>
        <button type="button" className="live-secondary m01-discard-list" onClick={onDiscard}>{m.discardList(l)}</button>
      </div>
      {!emptyFilter(f.filter) && (
        <div className="m01-fixedfilters">
          <span className="m01-fixedlead">{m.fixedFilters(l)}</span>
          <FixedChips l={l} filter={f.filter} masked={false} days={days} />
        </div>
      )}
      <p className="m01-fixedmeta">{b.bulkN(l, number)} · {m.baseRevision(l)} <bdi dir="ltr" className="live-mono">{f.fixed_revision}</bdi></p>
      <p className="m01-selnote">{m.fixedNoChecks(l)}</p>
      {own && <p className="m01-selnote m01-ink" role="status">{m.keepsOwn(l)}</p>}
      {error !== null && error !== undefined && <p className="m01-limit" role="alert">{failureText(l, error).body}</p>}
    </section>
  );
}

type Region =
  | { kind: "closed" }
  | { kind: "counting" }
  | { kind: "counted"; count: MatchCount }
  | { kind: "failed"; error: unknown }
  | { kind: "fixing"; count: MatchCount }
  | { kind: "refused"; cause: RefusalCause | "other"; error: unknown; limit: number };

export function MatchAction({ l, api, query, filters, picked, pendingLimit, reload, onFixed, onJobs, say }: {
  l: Locale; api: AdminClient; query: UsersQuery; filters: UserFilters; picked: number; pendingLimit: number; reload: () => Promise<string>;
  onFixed: (f: Fixed) => void; onJobs: () => void; say: (text: string) => void;
}) {
  const bulk = bulkClient(api);
  const [region, setRegion] = useState<Region>({ kind: "closed" });
  const asked = useRef(0);
  const trigger = useRef<HTMLButtonElement>(null);
  const alert = useRef<HTMLDivElement>(null);
  const key = JSON.stringify(query);
  useEffect(() => { asked.current++; setRegion({ kind: "closed" }); }, [key]);
  useEffect(() => { if (region.kind === "refused") alert.current?.focus(); }, [region.kind]);
  const close = () => { asked.current++; setRegion({ kind: "closed" }); trigger.current?.focus(); };
  const count = async () => {
    const n = ++asked.current;
    setRegion({ kind: "counting" });
    say(m.counting(l));
    try {
      const c = await bulk.count(query);
      if (n !== asked.current) return;
      setRegion({ kind: "counted", count: c });
      const v = countView(c);
      say(v === "over" ? m.tooMany(l, c.limit) : v === "none" ? m.noneMatch(l) : v === "exact" ? m.selectCounted(l, c.count) : m.selectAtLeast(l, c.count));
    } catch (e) {
      if (n === asked.current) setRegion({ kind: "failed", error: e });
    }
  };
  const toggle = () => (region.kind === "closed" ? void count() : close());
  const fix = async (c: MatchCount) => {
    const n = ++asked.current;
    setRegion({ kind: "fixing", count: c });
    const tried = attempt();
    for (let retry = 0; ; retry++) {
      try {
        const out = await api.command("bulk_revocation.add_matching", "bulk-revocations", await reload(), m.matchRationale(l), matchValues(query), tried);
        if (n !== asked.current) return;
        setRegion({ kind: "closed" });
        onFixed(fixedOfFields(out.fields, c.count, !c.complete));
        return;
      } catch (e) {
        if (e instanceof AdminError && e.failure === "conflict" && retry === 0) continue;
        if (n !== asked.current) return;
        if (e instanceof AdminError && e.failure === "reauth_cancelled") { setRegion({ kind: "counted", count: c }); return; }
        if (!(e instanceof AdminError) || e.failure !== "refused") { setRegion({ kind: "refused", cause: "other", error: e, limit: c.limit }); return; }
        const page = await bulk.list("", 1).catch(() => null);
        const again = page?.open_draft ? null : await bulk.count(query).catch(() => null);
        if (n !== asked.current) return;
        setRegion({ kind: "refused", cause: refusalCause(page, pendingLimit, again), error: e, limit: again?.limit || c.limit });
        return;
      }
    }
  };
  const label = appliedKeys(filters).length > 0 ? m.selectMatching(l) : m.selectAll(l);
  const escape = (e: KeyboardEvent) => { if (e.key === "Escape") { e.preventDefault(); close(); } };
  const cancel = <button type="button" className="live-secondary" onClick={close}>{w.cancel(l)}</button>;
  const closeButton = <button type="button" className="live-secondary" onClick={close}>{b.close(l)}</button>;
  let body: ReactNode = null;
  if (region.kind === "counting") {
    body = <><p>{m.counting(l)}</p><div className="m01-match-actions">{cancel}</div></>;
  } else if (region.kind === "failed") {
    const text = failureText(l, region.error);
    body = (
      <div role="alert">
        <p className="live-strong">{text.title}</p>
        <p>{text.body}</p>
        <div className="m01-match-actions"><button type="button" className="live-secondary" onClick={() => void count()}>{copy.tryAgain(l)}</button>{cancel}</div>
      </div>
    );
  } else if (region.kind === "counted" || region.kind === "fixing") {
    const c = region.count;
    const v = countView(c);
    if (!confirmable(c)) {
      body = <><p className={v === "over" ? "live-strong" : undefined}>{v === "over" ? m.tooMany(l, c.limit) : m.noneMatch(l)}</p><div className="m01-match-actions">{closeButton}</div></>;
    } else {
      const fixing = region.kind === "fixing";
      body = (
        <>
          <p className="live-meta">{v === "exact" ? <At l={l} template={m.countedAt(l)} iso={c.server_time} date={false} zone={false} /> : m.lowerBound(l, c.scan_budget, c.count, c.limit)}</p>
          {picked > 0 && <p className="live-strong" id="m01-replace">{m.replaceConfirm(l, picked)}</p>}
          <div className="m01-match-actions">
            <button type="button" className="live-secondary m01-match-confirm" aria-describedby={picked > 0 ? "m01-replace" : undefined} aria-disabled={fixing || undefined}
              onClick={() => { if (!fixing) void fix(c); }}>{fixing ? m.selecting(l) : v === "exact" ? m.selectCounted(l, c.count) : m.selectAtLeast(l, c.count)}</button>
            {!fixing && cancel}
          </div>
        </>
      );
    }
  } else if (region.kind === "refused") {
    const text = region.cause === "openDraft" ? b.openDraft(l) : region.cause === "pending" ? b.pendingLimit(l) : region.cause === "none" ? m.noneMatch(l)
      : region.cause === "over" ? m.tooMany(l, region.limit) : region.cause === "unknown" ? m.combined(l) : failureText(l, region.error).body;
    body = (
      <div role="alert" tabIndex={-1} ref={alert} className="m01-match-refusal">
        <p>{text}</p>
        <p className="live-strong">{b.nothingRevoked(l)}</p>
        <div className="m01-match-actions">
          {(region.cause === "openDraft" || region.cause === "pending") && <button type="button" className="service-button-link m01-link" onClick={onJobs}>{b.openJobs(l)}</button>}
          {closeButton}
        </div>
      </div>
    );
  }
  return (
    <>
      <button type="button" ref={trigger} className="live-secondary m01-match-open" aria-expanded={region.kind !== "closed"} aria-controls={region.kind !== "closed" ? "m01-match" : undefined} onClick={toggle}>{label}</button>
      {region.kind !== "closed" && <div id="m01-match" className="m01-match" data-state={region.kind} onKeyDown={escape}>{body}</div>}
    </>
  );
}
