import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import { flushSync } from "react-dom";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type Attempt, type CommandId, type DraftMember, type GroupDraft, type OrgMember, type OrgParty, type PlanMember, type PlanRow, type PlansPage, type RelationKind } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import { appended } from "../../../../shared/more-rows";
import type { Session } from "../../../../service/api";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import { FailureBox, Reason, reasonProblem, useWide } from "../../../identity/pages/users/Live";
import { w } from "../../../identity/pages/users/live-copy";
import { o } from "../../../organizations/pages/organizations/live-copy";
import { todayUTC, validDay } from "../../../organizations/pages/organizations/live-model";
import { c } from "./live-copy";
import {
  actionsFor, adopted, afterConflict, baseBlocks, baseField, baseStale, blocked, canActivate, closing, currentOn, currentView, fill, findPlan, arrival, formStep, memberMode, held, impactDone, impactOf, impactUnread, keepsAttempt, kinds, lead, membersKept, minFrom,
  nameOf, needsTarget, partition, prefilled, react, readHandoff, refusal, showOpenCount, steps, targetChoices, type Action, type Blocked, type Choice, type Command, type Carried, type Reaction,
  draftLoad, groupRefusal, planTitle, type DraftLoad,
} from "./live-model";
import { Named, Picker, resultPage, unitChoices, usePresent } from "./choices";
import { usePages } from "../organizations/use-pages";
import { MoreBar, SearchForm } from "../organizations/filters";
import "../../../identity/pages/users/live.css";
import "./live.css";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

function day(l: Locale, iso: string): string {
  if (!validDay(iso)) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeZone: "UTC" }).format(new Date(iso + "T00:00:00Z"));
}

function stamp(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(d) + " UTC";
}

const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

type Choices = { parties: OrgParty[]; revision: number };

async function readChoices(api: AdminClient, on: string): Promise<Choices> {
  const first = await api.organizations({ on, kind: "hierarchy", limit: 1 });
  return { parties: first.parties, revision: first.relationship_revision };
}

function namedAs(rows: Choice[]): OrgMember[] {
  return rows.filter((r) => r.name && r.name !== r.id).map((r) => ({ member: r.id, name: r.name, targets: [], history: [] }));
}

type Notice = { text: string; alert?: boolean } | null;
type Failure = { error: unknown; reaction: Reaction; key: number } | null;

function failureFor(l: Locale, reaction: Reaction, e: unknown): { title: string; body: string } {
  const op = e instanceof AdminError ? e.operation : "";
  if (reaction === "wait") return { title: c.failed(l), body: sc.limited(l, retryPhrase(l, e instanceof AdminError && e.retryAfter ? e.retryAfter : 60)) };
  if (reaction === "retry") return { title: c.failed(l), body: c.retry(l) };
  if (reaction === "renew") return { title: c.failed(l), body: c.renew(l) };
  if (reaction === "reauth") return { title: c.failed(l), body: sc.reauthText(l, e instanceof AdminError ? e.failure : "") };
  if (reaction === "signin") return { title: c.failed(l), body: w.signInAgain(l) };
  return { title: c.failed(l), body: op ? c.stopped(l, op) : "" };
}

export function Steps({ l, p }: { l: Locale; p: PlanRow }) {
  const s = steps(p);
  const names = c.stepNames(l);
  const note = (i: number): string => {
    if (s[i] === "held") return i === 1 ? (impactUnread(p) ? c.unknownImpact(l) : c.notComputed(l)) : c.approvalHeld(l, blocked(p));
    if (s[i] === "stopped") return c.stateWord(l, p.state);
    return "";
  };
  return (
    <ol className="o02-steps" aria-label={c.progress(l)}>
      {s.map((state, i) => (
        <li key={i} className="o02-step" aria-current={state === "current" ? "step" : undefined} data-state={state}>
          <span className="o02-step-n">{c.step(l, i + 1)}</span>
          <span className="o02-step-label">{names[i]}</span>
          <span className="o02-step-state">{c.stepState(l, state)}</span>
          {note(i) && <span className="o02-step-note">{note(i)}</span>}
        </li>
      ))}
    </ol>
  );
}

export function Held({ l, b }: { l: Locale; b: Blocked }) {
  const unread = b === "impact_unknown";
  return (
    <div className="live-blocked o02-held">
      <p className="o02-held-lead">{unread ? c.heldUnknownLead(l) : c.heldLead(l)}</p>
      <p>{unread ? c.heldUnknownNext(l) : c.heldNext(l)}</p>
    </div>
  );
}

const impactWord = (l: Locale, v: string) => {
  const i = impactOf(v);
  return i === "computed" ? c.computedImpact(l) : i === "not_computed" ? c.notComputed(l) : c.unknownImpact(l);
};

type FurtherMembers = { key: string; rows: PlanMember[]; next: Record<string, string>; status: Record<string, "loading" | "failed"> };

export function Detail({ l, p, page, parties, members, full, titleRef, readMembers, onRefused }: {
  l: Locale; p: PlanRow; page: PlansPage; parties: OrgParty[]; members: OrgMember[]; full: boolean; titleRef?: React.Ref<HTMLHeadingElement>;
  readMembers?: (plan: string, kind: string, after: string) => Promise<{ rows: PlanMember[]; next: string }>;
  onRefused?: (e: unknown) => boolean;
}) {
  const named = (id: string) => nameOf(id, parties, members);
  const furtherKey = page.revision + "\u0000" + p.id;
  const blankFurther = (): FurtherMembers => ({ key: furtherKey, rows: [], next: {}, status: {} });
  const [extra, setExtra] = useState<FurtherMembers>(blankFurther);
  const further = extra.key === furtherKey ? extra : blankFurther();
  const furtherToken = useRef(0);
  useEffect(() => { furtherToken.current += 1; }, [furtherKey]);
  const shownMembers = appended(p.members, further.rows, (m) => (m.kind ?? "") + "\u0000" + m.member);
  const cutKinds = readMembers ? p.kinds.filter((k) => (k.kind in further.next ? further.next[k.kind] : k.next ?? "") !== "") : [];
  const changedTotal = p.kinds.reduce((sum, k) => sum + k.changed, 0);
  const severalKinds = p.kinds.length > 1;
  const moreMembers = (kind: string) => {
    const after = kind in further.next ? further.next[kind] : p.kinds.find((k) => k.kind === kind)?.next ?? "";
    if (!readMembers || !after || further.status[kind] === "loading") return;
    const mine = furtherToken.current;
    setExtra({ ...further, status: { ...further.status, [kind]: "loading" } });
    readMembers(p.id, kind, after)
      .then((got) => {
        if (mine !== furtherToken.current) return;
        setExtra((was) => {
          const now = was.key === furtherKey ? was : blankFurther();
          const status = { ...now.status };
          delete status[kind];
          return { key: furtherKey, rows: [...now.rows, ...got.rows.map((m) => ({ ...m, kind: m.kind ?? kind }))], next: { ...now.next, [kind]: got.next }, status };
        });
      })
      .catch((e: unknown) => {
        if (mine !== furtherToken.current) return;
        const refused = onRefused?.(e) === true;
        setExtra((was) => {
          const now = was.key === furtherKey ? was : blankFurther();
          const status = { ...now.status };
          if (refused) delete status[kind];
          else status[kind] = "failed";
          return { ...now, status };
        });
      });
  };
  const kindName = o.kind(l, p.change.kind);
  const memberName = p.member_name || named(p.change.member);
  const targetName = p.change.target ? p.target_name || named(p.change.target) : "";
  const ld = lead(p);
  const impact = [p.impact.money, p.impact.publication, p.impact.erp];
  const b = blocked(p);
  return (
    <>
      <h2 id="o02-plan-title" tabIndex={-1} ref={titleRef}>{p.draft ? fill(c.groupLead(l, "{g}", "{n}", "{d}"), { g: <bdi dir="ltr">{planTitle(p)}</bdi>, n: <>{p.draft.member_count}</>, d: <time dateTime={p.change.from}>{day(l, p.change.from)}</time> }) : c.sentence(l, p.change.action, memberName, targetName, kindName)}</h2>
      {p.draft?.new_group && <p className="o02-draft-name"><span className="o02-draft-label">{c.draftName(l)}</span> <bdi>{p.draft.name}</bdi></p>}
      <p className="live-sub live-mono">{ltr(p.id)}</p>
      <Steps l={l} p={p} />
      {p.stale && <div className="live-note-rule"><p><strong>{c.staleTag(l)}.</strong> {c.stale(l, page.active_revision, p.base)}</p></div>}
      <dl className="live-facts o02-summary">
        {p.draft ? <div><dt>{c.membersFact(l)}</dt><dd>{p.draft.member_count}</dd></div> : <div><dt>{c.facts.member(l)}</dt><dd><Named id={p.change.member} name={memberName} /></dd></div>}
        <div><dt>{c.facts.kind(l)}</dt><dd>{kindName}</dd></div>
        <div><dt>{c.facts.change(l)}</dt><dd>{c.actionShort(l, p.change.action)}</dd></div>
        <div><dt>{c.facts.target(l)}</dt><dd>{p.change.target ? <Named id={p.change.target} name={targetName} /> : c.none(l)}</dd></div>
        <div><dt>{c.facts.from(l)}</dt><dd><time dateTime={p.change.from}>{day(l, p.change.from)}</time></dd></div>
        <div><dt>{c.facts.state(l)}</dt><dd><span className="o02-tag">{c.stateWord(l, p.state)}</span></dd></div>
        <div><dt>{c.facts.base(l)}</dt><dd>{p.base}</dd></div>
        <div><dt>{c.facts.proposer(l)}</dt><dd><bdi>{p.proposer_login || p.proposer}</bdi></dd></div>
        <div><dt>{c.facts.at(l)}</dt><dd>{stamp(l, p.created)}</dd></div>
        <div className="o02-wide"><dt>{c.facts.reason(l)}</dt><dd><bdi>{p.reason}</bdi></dd></div>
      </dl>
      {full && (
        <div className="live-sections o02-sections">
          {p.violations.length > 0 && (
            <section aria-labelledby="o02-violations-title">
              <h3 id="o02-violations-title">{c.violations(l)}</h3>
              <ul className="o02-violations">
                {p.violations.map((v, i) => {
                  const [title, body] = c.rule(l, v.rule);
                  return (
                    <li key={i}>
                      <strong>{title}</strong>
                      {p.draft && v.rule === "plan_limit" ? <p>{c.planLimit(l)}</p> : body && <p>{body}</p>}
                      <span className="live-sub live-mono">{ltr([v.rule, v.kind, v.member].filter(Boolean).join(" · "))}</span>
                    </li>
                  );
                })}
              </ul>
              {p.truncated && <p className="live-sub o02-violations-cut">{c.truncated(l)}</p>}
            </section>
          )}
          <section aria-labelledby="o02-diff-title">
            <h3 id="o02-diff-title">{c.diff(l)}</h3>
            {!membersKept(p) ? <p className="live-note">{c.notKept(l)}</p> : shownMembers.length === 0 ? <p className="live-note">{c.none(l)}</p> : (
              <>
                {p.draft ? (
                  <div className="security-table-scroll live-table o02-diff o02-group-diff">
                    <table>
                      <thead><tr><th scope="col">{c.colMember(l)}</th><th scope="col">{c.colRanges(l)}</th><th scope="col">{c.colBefore(l)}</th><th scope="col">{c.colAfter(l)}</th></tr></thead>
                      <tbody>
                        {shownMembers.map((m) => (
                          <tr key={(m.kind ?? "") + m.member}>
                            <td data-label={c.colMember(l)}><Named id={m.member} name={m.name || named(m.member)} />{severalKinds && m.kind && <span className="live-sub o02-member-kind">{o.kind(l, m.kind)}</span>}</td>
                            <td data-label={c.colRanges(l)}>{m.differs.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.differs.map((r) => <li key={r.from + r.until}><DatedRange l={l} from={r.from} until={r.until} /></li>)}</ul>}</td>
                            <td data-label={c.colBefore(l)}>{m.before.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.before.map((id) => <li key={id}><Named id={id} name={named(id)} /></li>)}</ul>}</td>
                            <td data-label={c.colAfter(l)}>{m.after.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.after.map((id) => <li key={id}><Named id={id} name={named(id)} /></li>)}</ul>}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                ) : (
                <div className="security-table-scroll live-table o02-diff">
                  <table>
                    <colgroup><col className="o02-c20" /><col className="o02-c24" /><col className="o02-c16" /><col className="o02-c20" /><col className="o02-c20" /></colgroup>
                    <thead><tr><th scope="col">{c.colMember(l)}</th><th scope="col">{c.colChange(l)}</th><th scope="col">{c.colRanges(l)}</th><th scope="col">{c.colBefore(l)}</th><th scope="col">{c.colAfter(l)}</th></tr></thead>
                    <tbody>
                      {shownMembers.map((m) => (
                        <tr key={(m.kind ?? "") + m.member}>
                          <td data-label={c.colMember(l)}><Named id={m.member} name={m.name || named(m.member)} />{severalKinds && m.kind && <span className="live-sub o02-member-kind">{o.kind(l, m.kind)}</span>}</td>
                          <td data-label={c.colChange(l)}>{m.member === p.change.member ? (ld.target ? c.leadTo(l, day(l, ld.from), (ld.target === p.change.target && p.target_name) || named(ld.target)) : c.leadNone(l, day(l, ld.from))) : c.none(l)}</td>
                          <td data-label={c.colRanges(l)}>
                            {m.differs.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.differs.map((r) => <li key={r.from + r.until}>{r.until ? c.rangeClosed(l, day(l, r.from), day(l, r.until)) : c.rangeOpen(l, day(l, r.from))}</li>)}</ul>}
                          </td>
                          <td data-label={c.colBefore(l)}>{m.before.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.before.map((id) => <li key={id}><Named id={id} name={named(id)} /></li>)}</ul>}</td>
                          <td data-label={c.colAfter(l)}>{m.after.length === 0 ? c.none(l) : <ul className="o02-ranges">{m.after.map((id) => <li key={id}><Named id={id} name={named(id)} /></li>)}</ul>}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                )}
                <p className="live-note">{c.timelineNote(l)}</p>
                {cutKinds.length > 0 && (
                  <div className="o02-members-foot">
                    <p className="live-note o02-members-count">{c.membersShown(l, shownMembers.length, changedTotal)}</p>
                    {cutKinds.map((k) => {
                      const status = further.status[k.kind] ?? "idle";
                      return (
                        <div key={k.kind} className="o02-members-more">
                          {status === "failed" && <p role="alert">{c.membersFailed(l)}</p>}
                          <button type="button" className="live-secondary o02-members-more-button" aria-disabled={status === "loading" || undefined} onClick={() => { if (status !== "loading") moreMembers(k.kind); }}>
                            {status === "failed" ? c.membersTryAgain(l, o.kind(l, k.kind)) : status === "loading" ? c.membersLoading(l) : c.membersMore(l, o.kind(l, k.kind))}
                          </button>
                        </div>
                      );
                    })}
                  </div>
                )}
              </>
            )}
          </section>
          {p.kinds.length > 0 && (
            <section aria-labelledby="o02-kinds-title">
              <h3 id="o02-kinds-title">{c.kinds(l)}</h3>
              <div className="security-table-scroll live-table o02-kinds">
                <table>
                  <thead><tr>{c.kindCols(l).map((h) => <th scope="col" key={h}>{h}</th>)}</tr></thead>
                  <tbody>
                    {p.kinds.map((k) => {
                      const h = c.kindCols(l);
                      return (
                        <tr key={k.kind}>
                          <td data-label={h[0]}>{o.kind(l, k.kind)}</td>
                          <td data-label={h[1]} className="o02-num">{k.changed}</td>
                          <td data-label={h[2]} className="o02-num">{k.entries_before}</td>
                          <td data-label={h[3]} className="o02-num">{k.entries_after}</td>
                          <td data-label={h[4]}>{k.earliest ? <time dateTime={k.earliest}>{day(l, k.earliest)}</time> : c.none(l)}</td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </section>
          )}
          <section aria-labelledby="o02-impact-title">
            <h3 id="o02-impact-title">{c.impact(l)}</h3>
            {held(b) && <p className="o02-held-lead">{b === "impact_unknown" ? c.heldUnknownLead(l) : c.heldLead(l)}</p>}
            {!impactDone(p) && <p className="live-impact">{c.impactNote(l)}</p>}
            <dl className="o02-impact">
              {c.impactRows(l).map((row, i) => (
                <div key={row}>
                  <dt>{row}</dt>
                  <dd><span className="o02-tag">{impactWord(l, impact[i])}</span>{impactOf(impact[i]) !== "computed" && <> <span className="live-sub">{c.notZero(l)}</span></>}</dd>
                </div>
              ))}
            </dl>
          </section>
        </div>
      )}
    </>
  );
}

type Decision = "activate" | "discard";

export function Approval({ l, p, sentence, kindName, confirm, busy, reason, onOpen, onDecide, onCancel }: {
  l: Locale; p: PlanRow; sentence: string; kindName: string; confirm: Decision | null; busy: Command | null; reason: ReactNode;
  onOpen: (k: Decision) => void; onDecide: (k: Decision) => void; onCancel: () => void;
}) {
  if (p.state !== "sealed") return <p className="live-note">{c.onlySealed(l, c.stateWord(l, p.state))}</p>;
  const b = blocked(p);
  return (
    <div className="live-sections">
      <section className="live-group" aria-labelledby="o02-activate-title">
        <h3 id="o02-activate-title">{c.activate(l)}</h3>
        {held(b) && <Held l={l} b={b} />}
        <p className="live-impact">{c.restate(l, day(l, p.change.from), sentence, kindName)}</p>
        {held(b) ? null : !canActivate(p) ? (
          <div className="live-blocked">
            <p><strong>{c.blockedTitle(l)}</strong></p>
            <ul><li>{b === "own" ? c.own(l) : b === "own_draft" ? c.ownDraft(l) : b === "stale" ? c.staleBlock(l) : b === "passed" ? c.passed(l) : c.otherBlock(l)}</li></ul>
            {(b === "stale" || b === "passed") && <p className="live-note">{c.next(l)}</p>}
          </div>
        ) : confirm !== "activate" ? (
          <button type="button" id="o02-open-activate" className="live-secondary" disabled={confirm !== null} onClick={() => onOpen("activate")}>{c.reviewActivate(l)}</button>
        ) : (
          <div className="o02-confirm">
            <h3 id="o02-confirm-title" tabIndex={-1}>{c.confirmActivate(l)}</h3>
            <dl className="live-facts">
              <div><dt>{c.facts.from(l)}</dt><dd>{day(l, p.change.from)} (00:00 UTC)</dd></div>
              <div><dt>{c.facts.change(l)}</dt><dd>{sentence}</dd></div>
            </dl>
            {reason}
            <div className="live-buttons">
              <button type="button" className="live-primary" disabled={busy !== null} aria-busy={busy === "activate" || undefined} onClick={() => onDecide("activate")}>{busy === "activate" ? c.activating(l) : c.activateBtn(l, day(l, p.change.from))}</button>
              <button type="button" className="live-secondary" disabled={busy !== null} onClick={onCancel}>{c.cancel(l)}</button>
            </div>
          </div>
        )}
      </section>
      <section className="live-group" aria-labelledby="o02-discard-title">
        <h3 id="o02-discard-title">{c.discard(l)}</h3>
        {confirm !== "discard" ? (
          <button type="button" id="o02-open-discard" className="live-secondary" disabled={confirm !== null} onClick={() => onOpen("discard")}>{c.reviewDiscard(l)}</button>
        ) : (
          <div className="o02-confirm">
            <h3 id="o02-confirm-title" tabIndex={-1}>{c.confirmDiscard(l)}</h3>
            <p className="live-impact"><strong>{c.permanent(l)}</strong></p>
            {reason}
            <div className="live-buttons">
              <button type="button" className="live-primary" disabled={busy !== null} aria-busy={busy === "discard" || undefined} onClick={() => onDecide("discard")}>{busy === "discard" ? c.discarding(l) : c.discardBtn(l)}</button>
              <button type="button" className="live-secondary" disabled={busy !== null} onClick={onCancel}>{c.keep(l)}</button>
            </div>
          </div>
        )}
      </section>
    </div>
  );
}

export const decisionLabel = (l: Locale, p: PlanRow) => (held(blocked(p)) ? c.toDiscard(l) : c.toApproval(l));

export function ImpactActions({ l, p, onDecision, onAnother }: { l: Locale; p: PlanRow; onDecision: () => void; onAnother: () => void }) {
  return (
    <div className="live-buttons">
      {p.state === "sealed" && <button type="button" className="live-primary" onClick={onDecision}>{decisionLabel(l, p)}</button>}
      <button type="button" className="service-button-link" onClick={onAnother}>{c.another(l)}</button>
    </div>
  );
}

type Form = { kind: RelationKind | ""; action: Action | ""; member: string; memberById: boolean; target: string; targetById: boolean; from: string; reason: string };
const emptyForm: Form = { kind: "", action: "", member: "", memberById: false, target: "", targetById: false, from: "", reason: "" };
type Errors = Partial<Record<"base" | "kind" | "action" | "member" | "target" | "from" | "reason", string>>;
const listViews = ["impact", "approval", "result"];

export function HandoffNote({ l, h, active, onUseBase }: { l: Locale; h: Carried; active: number; onUseBase: () => void }) {
  const stale = baseStale(h.base, active);
  const rev = (n: number) => <bdi dir="ltr">{n}</bdi>;
  return (
    <div id="o02-handoff" className="live-note-rule o02-handoff" tabIndex={-1}>
      {h.dropped && <p>{c.linkDropped(l)}</p>}
      {prefilled(h) && <p>{h.base !== null ? fill(c.prefilled(l, "{r}"), { r: rev(h.base) }) : c.prefilledNoBase(l)}</p>}
      {h.passed && <p>{fill(c.fromPassed(l, "{d}"), { d: <time dateTime={h.passed}>{day(l, h.passed)}</time> })}</p>}
      {stale && h.base !== null && (
        <>
          <p><strong>{c.staleTag(l)}.</strong> {fill((h.refused ? c.baseRefused : c.baseChanged)(l, "{r}", "{a}"), { r: rev(h.base), a: rev(active) })}</p>
          <button type="button" id="o02-use-base" className="live-secondary o02-use-base" onClick={onUseBase}><span>{fill(c.useBase(l, "{a}"), { a: rev(active) })}</span></button>
        </>
      )}
    </div>
  );
}

export function DatedRange({ l, from, until }: { l: Locale; from: string; until: string }) {
  const text = until ? c.rangeClosed(l, "{d}", "{e}") : c.rangeOpen(l, "{d}");
  return <>{text.split(/(\{[de]\})/).map((x, i) => (x === "{d}" ? <time key={i} className="o02-nowrap" dateTime={from}>{day(l, from)}</time>
    : x === "{e}" ? <time key={i} className="o02-nowrap" dateTime={until}>{day(l, until)}</time> : x))}</>;
}

export type GroupLoad = { status: DraftLoad; draft: GroupDraft | null; members: DraftMember[] };

export function GroupHandoff({ l, h, group, active, onUseBase, onRetry }: { l: Locale; h: Carried; group: GroupLoad; active: number; onUseBase: () => void; onRetry: () => void }) {
  const stale = baseStale(h.base, active);
  const rev = (n: number) => <bdi dir="ltr">{n}</bdi>;
  const d = group.status === "ready" ? group.draft : null;
  return (
    <div id="o02-handoff" className="live-note-rule o02-handoff o02-group-handoff" tabIndex={-1}>
      <p><strong>{c.groupTitle(l)}</strong></p>
      {h.dropped && <p>{c.linkDropped(l)}</p>}
      {h.passed && <p>{fill(c.fromPassed(l, "{d}"), { d: <time dateTime={h.passed}>{day(l, h.passed)}</time> })}</p>}
      {group.status === "loading" && <p className="live-meta">{c.draftLoading(l)}</p>}
      {group.status === "missing" && <p>{c.draftMissing(l)}</p>}
      {group.status === "closed" && <p>{c.draftNotOpen(l)}</p>}
      {group.status === "error" && <><p>{c.draftFailed(l)}</p><button type="button" className="live-secondary" onClick={onRetry}>{o.tryAgain(l)}</button></>}
      {d && (
        <>
          <dl className="live-facts">
            {d.new_group && <div><dt>{c.draftNameFact(l)}</dt><dd><bdi>{d.name}</bdi></dd></div>}
            <div><dt>{c.groupFact(l)}</dt><dd>{d.new_group ? c.newGroup(l) : <bdi>{d.group_label || d.group}</bdi>} <bdi dir="ltr" className="live-mono o02-nowrap">{d.group}</bdi></dd></div>
            <div><dt>{c.membersFact(l)}</dt><dd>{d.member_count}</dd></div>
          </dl>
          <p>{c.notOwnership(l)}</p>
          <p>{c.draftEffect(l, d.member_count)}</p>
          <h3>{c.membersFirst(l, group.members.length, d.member_count)}</h3>
          <ul className="o02-group-members">
            {group.members.map((m) => <li key={m.id}><bdi>{m.name || m.id}</bdi> <bdi dir="ltr" className="live-mono o02-nowrap">{m.id}</bdi></li>)}
          </ul>
        </>
      )}
      {stale && h.base !== null && (
        <>
          <p><strong>{c.staleTag(l)}.</strong> {fill((h.refused ? c.baseRefused : c.baseChanged)(l, "{r}", "{a}"), { r: rev(h.base), a: rev(active) })}</p>
          <button type="button" id="o02-use-base" className="live-secondary o02-use-base" onClick={onUseBase}><span>{fill(c.useBase(l, "{a}"), { a: rev(active) })}</span></button>
        </>
      )}
    </div>
  );
}

export function BaseFact({ l, base, active }: { l: Locale; base: number; active: number }) {
  return (
    <dl className="live-facts o02-base">
      <div><dt>{c.baseFact(l)}</dt><dd><bdi dir="ltr">{base}</bdi> · {baseStale(base, active) ? c.baseNotActive(l, active) : c.baseActive(l)}</dd></div>
    </dl>
  );
}

export default function LiveChanges({ locale: l, view, params, session, onSignOut, onView, onParams, onDirty }: {
  locale: Locale; view: string; params: URLSearchParams; session: Session; onSignOut: () => void;
  onView: (view: string, params?: Record<string, string>) => void; onParams: (params: Record<string, string | null>) => void; onDirty?: (dirty: boolean) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const [page, setPage] = useState<PlansPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [choices, setChoices] = useState<Choices | null>(null);
  const [choicesFailed, setChoicesFailed] = useState(false);
  const [choiceTurn, setChoiceTurn] = useState(0);
  const [seen, setSeen] = useState<OrgMember[]>([]);
  const [carried, setCarried] = useState<boolean | null>(null);
  const remember = (rows: Choice[]) => setSeen((was) => {
    const fresh = namedAs(rows).filter((m) => !was.some((x) => x.member === m.member));
    return fresh.length === 0 ? was : [...was, ...fresh].slice(-200);
  });
  const refusedRead = (e: unknown) => {
    if (e instanceof AdminError && e.failure === "denied") { setDenied(true); return true; }
    return false;
  };
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [stepped, setStepped] = useState(() => !!params.get("plan"));
  const [notice, setNotice] = useState<Notice>(null);
  const [failure, setFailure] = useState<Failure>(null);
  const [denied, setDenied] = useState(false);
  const [historyQ, setHistoryQ] = useState("");
  const [historyMore, setHistoryMore] = useState(false);
  const [matching, setMatching] = useState<Record<string, number>>({});
  const [historyReset, setHistoryReset] = useState(false);
  const lastRevision = useRef("");
  const historyKey = view !== "result" || !page ? null : historyQ ? `q|${page.revision}|${historyQ}` : historyMore && page.closed.next ? `more|${page.revision}` : null;
  const history = usePages<PlanRow>(historyKey, async (after) => {
    const asked = historyKey ?? "";
    const got = await api.closedPlans({ q: historyQ || undefined, after: after || (historyQ ? undefined : page?.closed.next), limit: 50 });
    setMatching((was) => ({ ...was, [asked]: got.matching }));
    return { rows: got.plans, next: got.next, stopped: false };
  }, (e) => { if (e instanceof AdminError && e.failure === "denied") { setDenied(true); return true; } return false; });
  const pageRevision = page?.revision ?? "";
  useEffect(() => {
    if (lastRevision.current && lastRevision.current !== pageRevision && historyMore) setHistoryReset(true);
    lastRevision.current = pageRevision;
    setHistoryMore(false);
  }, [pageRevision, historyQ]); // eslint-disable-line react-hooks/exhaustive-deps
  const [initial] = useState(() => (listViews.includes(view) ? null : readHandoff(params, todayUTC(new Date()))));
  const [handoff, setHandoff] = useState<Carried | null>(initial && { ...initial, refused: false });
  const [form, setForm] = useState<Form>(initial ? { ...emptyForm, kind: initial.kind, member: initial.member, from: initial.from } : emptyForm);
  const [touched, setTouched] = useState(false);
  const present = usePresent(api, form.member, form.kind, choices ? choices.revision : null, refusedRead);
  const resolveMember = useRef(!!initial?.member);
  const greetedHandoff = useRef(false);
  const [errors, setErrors] = useState<Errors>({});
  const [summaryKey, setSummaryKey] = useState(0);
  const [busy, setBusy] = useState<Command | null>(null);
  const [confirm, setConfirm] = useState<"activate" | "discard" | null>(null);
  const [actReason, setActReason] = useState("");
  const [actError, setActError] = useState("");
  const seq = useRef(0);
  const choiceSeq = useRef(0);
  const attempts = useRef<Partial<Record<Command, { tried: Attempt; expected: string; target: string }>>>({});
  const opener = useRef("");
  const today = todayUTC(new Date());
  const selectedId = params.get("plan") ?? "";
  const [arrived] = useState(() => params.get("done") ?? "");
  const [group, setGroup] = useState<{ status: DraftLoad; draft: GroupDraft | null; members: DraftMember[] }>({ status: "loading", draft: null, members: [] });
  const draftId = handoff?.draft ?? "";

  const load = useCallback(async (quiet = false): Promise<PlansPage | null> => {
    const n = ++seq.current;
    if (!quiet) setStatus((s) => (s === "ready" ? "ready" : "loading"));
    try {
      const got = await api.plans();
      if (n !== seq.current) return null;
      setPage(got);
      setStatus("ready");
      setLoadError(null);
      return got;
    } catch (e) {
      if (n !== seq.current) return null;
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      setLoadError(e);
      setStatus("error");
      return null;
    }
  }, [api]);
  const membersRefused = (e: unknown): boolean => {
    if (!(e instanceof AdminError) || (e.failure !== "denied" && e.failure !== "unauthenticated")) return false;
    if (e.failure === "denied") setDenied(true);
    else void load(true);
    return true;
  };

  const loadChoices = useCallback(async () => {
    const n = ++choiceSeq.current;
    setChoicesFailed(false);
    setChoices(null);
    setChoiceTurn((t) => t + 1);
    try {
      const got = await readChoices(api, todayUTC(new Date()));
      if (n === choiceSeq.current) setChoices(got);
    } catch (e) {
      if (n !== choiceSeq.current) return;
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      setChoicesFailed(true);
    }
  }, [api]);

  useEffect(() => { void load(); void loadChoices(); }, [load, loadChoices]);
  useEffect(() => {
    if (!focusTo) return;
    const el = document.querySelector<HTMLElement>(focusTo);
    if (!el) return;
    el.focus();
    setFocusTo("");
  });
  const greeted = useRef(false);
  const say = (text: string) => setAnnounce((prev) => (prev === text ? text + "\u00a0" : text));
  useEffect(() => {
    if (greeted.current || !arrived || (!page && status !== "error")) return;
    greeted.current = true;
    onParams({ done: null });
    if (!page) return;
    const got = arrival(arrived, findPlan(page.plans, selectedId));
    if (!got) return;
    const what = got.what;
    if (what !== "open") say(what === "rejected" ? c.aRejected(l) : what === "activated" ? c.aActivated(l, String(got.revision)) : what === "discarded" ? c.aDiscarded(l) : c.aSealed(l));
    setFocusTo("#o02-plan-title");
  }, [arrived, page, status]);
  useEffect(() => {
    if (initial) onParams({ kind: null, member: null, from: null, base: null, draft: null });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { onDirty?.(touched); }, [touched]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => () => onDirty?.(false), []); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!page || !handoff || greetedHandoff.current) return;
    greetedHandoff.current = true;
    setFocusTo("#o02-handoff");
  }, [page]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!resolveMember.current) return;
    const unit = !!choices && unitChoices(choices.parties, form.kind, "").some((m) => m.id === form.member);
    const mode = memberMode(choicesFailed || unit || carried !== null, unit || carried === true, touched);
    if (mode === null) return;
    resolveMember.current = false;
    if (mode === "id") setForm((f) => ({ ...f, memberById: true }));
  }, [choices, choicesFailed, carried]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!resolveMember.current || !choices) return;
    let live = true;
    api.organizations({ kind: "hierarchy", subject: "account", member: [form.member], revision: choices.revision })
      .then((got) => { if (live) { remember(got.members.map((m) => ({ id: m.member, name: m.name }))); setCarried(got.members.length > 0); } })
      .catch((e) => { if (!live) return; if (e instanceof AdminError && e.failure === "denied") setDenied(true); setCarried(false); });
    return () => { live = false; };
  }, [choices]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!listViews.includes(view)) return;
    setHandoff(null);
    const step = formStep("left", touched);
    if (step.resetForm) {
      setForm(emptyForm);
      setErrors({});
    }
    setTouched(step.touched);
  }, [view]); // eslint-disable-line react-hooks/exhaustive-deps
  const loadGroup = useCallback(async (id: string) => {
    setGroup((g) => ({ ...g, status: "loading" }));
    try {
      let after = "";
      let found: GroupDraft | null = null;
      for (let i = 0; i < 40 && !found; i++) {
        const got = await api.viewGroupDrafts(after, 50);
        found = got.drafts.find((d) => d.id === id) ?? null;
        after = got.next;
        if (!after) break;
      }
      const members = found ? await api.draftMembers(id, "", 50) : "missing";
      setGroup({ status: draftLoad(found), draft: found, members: members === "missing" ? [] : members.members });
    } catch (e) {
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      setGroup({ status: draftLoad("error"), draft: null, members: [] });
    }
  }, [api]);
  useEffect(() => { if (initial?.draft) void loadGroup(initial.draft); }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const lastActive = useRef(0);
  useEffect(() => {
    if (!page) return;
    if (lastActive.current && page.active_revision !== lastActive.current) void loadChoices();
    lastActive.current = page.active_revision;
  }, [page, loadChoices]);

  if (denied) {
    return <div className="live-users"><section className="live-unavailable" aria-labelledby="o02-denied"><h2 id="o02-denied">{c.deniedTitle(l)}</h2><p>{o.denied(l)}</p></section></div>;
  }

  const split = wide && view !== "result";
  const parties = choices?.parties ?? [];
  const allMembers = seen;
  const { open, closed: firstClosed } = partition(page?.plans ?? []);
  const searching = historyQ !== "";
  const closed = searching ? history.rows : history.current ? [...firstClosed, ...history.rows.filter((p) => !firstClosed.some((x) => x.id === p.id))] : firstClosed;
  const selected = findPlan([...(page?.plans ?? []), ...history.rows], selectedId);

  const edit = (patch: Partial<Form>) => {
    const step = formStep("edit", touched);
    if (step.clearAttempt) attempts.current.propose = undefined;
    setTouched(step.touched);
    setForm((f) => ({ ...f, ...patch }));
    setErrors((e) => {
      const next = { ...e };
      for (const k of Object.keys(patch)) {
        const f = k === "memberById" ? "member" : k === "targetById" ? "target" : k;
        delete next[f as keyof Errors];
      }
      return next;
    });
  };

  const run = async (cmd: Command, id: CommandId, target: string, reason: string, values: Record<string, string>) => {
    if (!page || busy) return null;
    const kept = attempts.current[cmd];
    const a = kept && kept.target === target ? kept : { tried: attempt(), expected: page.revision, target };
    attempts.current[cmd] = a;
    setBusy(cmd);
    setFailure(null);
    setNotice(null);
    try {
      const out = await api.command(id, target, a.expected, reason, values, a.tried);
      attempts.current[cmd] = undefined;
      return out;
    } catch (e) {
      const failureCode = e instanceof AdminError ? e.failure : "failed";
      const reaction = react(cmd, failureCode, e instanceof AdminError ? e.fields : []);
      if (!keepsAttempt(reaction)) attempts.current[cmd] = undefined;
      throw Object.assign(new Error("command"), { reaction, cause: e });
    } finally {
      setBusy(null);
    }
  };

  const fieldId = (f: keyof Errors): string =>
    f === "base" ? "o02-use-base" : f === "kind" ? "o02-kind-0" : f === "action" ? "o02-change-0" : f === "member" ? (form.memberById ? "o02-member-id" : choices ? "o02-member" : "o02-member-toggle")
      : f === "target" ? (form.targetById ? "o02-target-id" : choices || (form.kind !== "contract_pool" && form.kind !== "allocation_pool" && form.kind !== "view_group") ? "o02-target" : "o02-target-toggle") : f === "from" ? "o02-from" : "live-reason";

  const validate = (): Errors => {
    const e: Errors = {};
    if (handoff && handoff.base !== null && page && baseBlocks(handoff, page.active_revision)) e.base = c.eBase(l, handoff.base, page.active_revision);
    if (draftId && handoff?.base === null) e.base = c.eServer(l);
    if (!draftId && !form.kind) e.kind = c.eKind(l);
    if (!draftId && !form.action) e.action = c.eChange(l);
    if (!draftId && !form.member.trim()) e.member = c.eMember(l);
    if (!draftId && form.action && needsTarget(form.action) && !form.target.trim()) e.target = c.eTarget(l);
    if (!validDay(form.from)) e.from = c.eDate(l);
    else if (!minFrom(today, form.from)) e.from = c.ePast(l);
    const r = reasonProblem(l, form.reason);
    if (r) e.reason = r;
    return e;
  };

  const handle = async (err: unknown, cmd: Command, target = "") => {
    const reaction = (err as { reaction?: Reaction }).reaction ?? "failed";
    const cause = (err as { cause?: unknown }).cause;
    const fields = cause instanceof AdminError ? cause.fields : [];
    if (reaction === "denied") { setDenied(true); return; }
    if (cmd === "propose" && draftId) {
      const kind = groupRefusal(reaction, fields);
      if (kind === "draft") { setNotice({ text: c.draftRefused(l), alert: true }); setFocusTo("#o02-notice"); return; }
      if (reaction === "stage") { await loadGroup(draftId); setNotice({ text: c.draftNotOpen(l) }); setFocusTo("#o02-notice"); return; }
    }
    if (reaction === "field") {
      const known: Errors = {};
      for (const f of fields) {
        if (f === "member") known.member = c.eMemberServer(l);
        else if (f === "target") known.target = c.eTargetServer(l);
        else if (f === "from") known.from = c.eFromServer(l);
        else if (f === "action" || f === "kind") known.action = c.eActionServer(l);
        else if (f === "base") known.base = c.eServer(l);
        else known.reason = c.eServer(l);
      }
      setErrors(known);
      setSummaryKey((k) => k + 1);
      if (known.member || known.target) void loadChoices();
      return;
    }
    if (reaction === "reason") {
      if (cmd === "propose") { setErrors({ reason: c.eServer(l) }); setSummaryKey((k) => k + 1); } else { setActError(c.eServer(l)); setFocusTo("#live-reason"); }
      return;
    }
    if (reaction === "limit") { await load(true); setNotice({ text: c.limit(l, page?.open_limit ?? 0) }); setFocusTo("#o02-notice"); return; }
    if (reaction === "refused") {
      const got = await load(true);
      setNotice({ text: c.refused(l, refusal(findPlan(got?.plans ?? [], target))) });
      setFocusTo("#o02-notice");
      return;
    }
    if (reaction === "stage") { await load(true); setNotice({ text: c.stageChanged(l) }); setFocusTo("#o02-notice"); return; }
    if (reaction === "conflict") {
      const got = await load(true);
      const next = afterConflict(handoff, cmd, got ? got.active_revision : null);
      if (next.shown === "base") {
        setHandoff(next.handoff);
        setFocusTo("#o02-handoff");
        return;
      }
      setNotice({ text: c.conflict(l) });
      setFocusTo("#o02-notice");
      return;
    }
    if (reaction === "partial" && cmd === "propose") {
      await load(true);
      setNotice({ text: c.partialPropose(l, cause instanceof AdminError ? cause.operation : ""), alert: true });
      setFocusTo("#o02-notice");
      return;
    }
    if (reaction === "partial") void load(true);
    setFailure({ error: cause, reaction, key: Date.now() });
  };

  const propose = async (e: FormEvent) => {
    e.preventDefault();
    const found = validate();
    setErrors(found);
    if (Object.keys(found).length) { setSummaryKey((k) => k + 1); return; }
    const values: Record<string, string> = draftId ? { from: form.from } : { kind: form.kind, member: form.member.trim(), action: form.action, from: form.from };
    if (!draftId && form.action && needsTarget(form.action)) values.target = form.target.trim();
    Object.assign(values, baseField(handoff));
    try {
      const out = await run("propose", draftId ? "view_group.propose" : "relationship.propose", draftId || "relationships", form.reason, values);
      if (!out) return;
      const id = out.fields.plan ?? "";
      await load(true);
      const step = formStep("proposed", touched);
      if (step.resetForm) {
        setForm(emptyForm);
        setErrors({});
      }
      setHandoff(null);
      flushSync(() => { setTouched(step.touched); onDirty?.(step.touched); });
      onView("impact", { plan: id, done: out.state === "rejected" ? "rejected" : "sealed" });
    } catch (err) {
      await handle(err, "propose");
    }
  };

  const decide = async (kind: "activate" | "discard", p: PlanRow) => {
    const problem = reasonProblem(l, actReason);
    if (problem) { setActError(problem); setFocusTo("#live-reason"); return; }
    setActError("");
    try {
      const out = await run(kind, kind === "activate" ? "relationship.activate" : "relationship.discard", p.id, actReason, {});
      if (!out) return;
      await load(true);
      setActReason("");
      setConfirm(null);
      onView("result", { plan: p.id, done: kind === "activate" ? `activated:${out.fields.revision ?? ""}` : "discarded" });
    } catch (err) {
      await handle(err, kind, p.id);
    }
  };

  const select = (id: string) => {
    attempts.current.activate = undefined;
    attempts.current.discard = undefined;
    setConfirm(null);
    setActReason("");
    setActError("");
    setNotice(null);
    setFailure(null);
    onParams({ plan: id });
    if (!split) setStepped(true);
    setFocusTo("#o02-plan-title");
  };
  const back = () => { setStepped(false); setFocusTo(`[data-plan="${typeof CSS !== "undefined" && CSS.escape ? CSS.escape(selectedId) : selectedId}"]`); };

  const adoptBase = () => {
    if (!page || !handoff) return;
    const step = formStep("adopt", touched);
    if (step.clearAttempt) attempts.current.propose = undefined;
    setTouched(step.touched);
    setHandoff(adopted(handoff, page.active_revision));
    setErrors((e) => {
      const next = { ...e };
      delete next.base;
      return next;
    });
    say(c.baseAdopted(l, page.active_revision));
    setFocusTo(draftId ? "#o02-from" : "#o02-current-title");
  };

  const reload = async () => {
    const got = await load();
    void loadChoices();
    if (got) say(c.aReloaded(l));
  };

  const failureBox = failure && (
    <div className="o02-failure">
      <FailureBox l={l} error={failure.error} focusKey={failure.key} describe={(ll) => failureFor(ll, failure.reaction, failure.error)} />
      {failure.reaction === "signin" && <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>}
    </div>
  );
  const noticeBox = notice && (
    <div className={notice.alert ? "live-error" : "service-status"} role={notice.alert ? "alert" : "status"} id="o02-notice" tabIndex={-1}><p>{notice.text}</p></div>
  );

  const scope = (
    <div className="o02-scope">
      {page && (
        <dl className="o02-scope-facts">
          <div><dt>{c.revision(l)}</dt><dd>{c.active(l, page.active_revision)}</dd></div>
          {showOpenCount(page.open_limit) && <div><dt>{c.openPlans(l)}</dt><dd>{c.openCount(l, page.open_plans, page.open_limit)}</dd></div>}
        </dl>
      )}
      <button type="button" className="live-secondary o02-reload" onClick={() => void reload()}>{c.reload(l)}</button>
    </div>
  );

  const titles: Record<string, string> = { targets: c.targetsTitle(l), impact: c.impactTitle(l), approval: c.approvalTitle(l), result: c.resultTitle(l) };
  const heading = <div className="live-toolbar"><div className="live-heading"><h2 id="o02-view-title" tabIndex={-1}>{titles[view] ?? titles.targets}</h2></div></div>;

  if (status === "loading" && !page) {
    return <div className="o02-live">{scope}<p role="status" className="live-meta">{c.loading(l)}</p></div>;
  }
  if (status === "error" && !page) {
    const signin = loadError instanceof AdminError && loadError.failure === "unauthenticated";
    return (
      <div className="o02-live">
        <section className="live-error" role="alert">
          <p>{c.loadFailed(l)}</p>
          {signin ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button> : <button type="button" className="live-secondary" onClick={() => void load()}>{o.tryAgain(l)}</button>}
        </section>
      </div>
    );
  }
  if (!page) return null;

  const problems: Problems = (["base", "kind", "action", "member", "target", "from", "reason"] as const).filter((f) => errors[f]).map((f) => ({ field: fieldId(f), message: errors[f] as string }));
  const errorText = (f: keyof Errors, id: string) => errors[f] && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors[f]}</p>;
  const described = (f: keyof Errors, id: string, hint?: string) => [hint, errors[f] ? id + "-error" : ""].filter(Boolean).join(" ") || undefined;

  const pooled = form.kind === "contract_pool" || form.kind === "allocation_pool" || form.kind === "view_group";
  const targetList: Choice[] = form.kind && !pooled ? targetChoices(form.kind, form.member, parties, []) : [];
  const turn = `${choiceTurn}|${choices?.revision ?? ""}`;
  const searchMembers = async (term: string, after: string) => {
    const got = await api.organizations({ kind: "hierarchy", subject: "account", q: term || undefined, after: after || undefined, limit: resultPage, revision: choices?.revision });
    return { rows: got.members.map((m) => ({ id: m.member, name: m.name || m.member })), next: got.next, stopped: got.scan_stopped };
  };
  const searchTargets = async (term: string, after: string) => {
    const got = await api.targets({ kind: form.kind as RelationKind, q: term || undefined, after: after || undefined, limit: resultPage, revision: choices?.revision });
    return { rows: got.targets.map((t) => ({ id: t.id, name: t.name || t.id })), next: got.next, stopped: got.scan_stopped };
  };
  const targetHint = form.kind === "hierarchy" ? c.hintHierarchy(l) : form.kind === "payer" ? c.hintPayer(l) : c.hintPool(l);
  const currentTargets = (on: string): ReactNode => {
    if (!form.kind || !form.member) return null;
    if (!present.valid) return null;
    if (present.status === "failed") return <><span className="live-quiet" role="alert">{o.factsFailed(l)}</span> <button type="button" className="service-button-link" onClick={present.retry}>{o.tryAgain(l)}</button></>;
    const got = present.history === null ? null : currentOn(form.member.trim(), [{ member: form.member.trim(), name: "", targets: [], history: present.history }], on);
    const shown = currentView(present.status === "ready", got);
    if (shown === "loading") return <span className="live-quiet">{c.loadingCurrent(l)}</span>;
    if (shown === "none_of_kind") return <span className="live-quiet">{c.noneOfKind(l)}</span>;
    if (shown === "none" || !got) return <span className="live-quiet">{c.none(l)}</span>;
    return <ul className="o02-ranges">{got.map((t) => <li key={t.id}><Named id={t.id} name={t.name} /></li>)}</ul>;
  };

  const groupView = (
    <form className="o02-target o02-group" onSubmit={(e) => void propose(e)} noValidate aria-busy={busy === "propose" || undefined}>
      <div className="o02-fields">
        {handoff && handoff.base !== null && <BaseFact l={l} base={handoff.base} active={page.active_revision} />}
        {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
        <div className="service-field">
          <label htmlFor="o02-from">{c.fromLabel(l)}</label>
          <p className="service-hint" id="o02-from-hint">{c.fromHint(l)}</p>
          {errorText("from", "o02-from")}
          <input id="o02-from" type="date" className="o02-date" min={today} value={form.from} aria-invalid={errors.from ? true : undefined} aria-describedby={described("from", "o02-from", "o02-from-hint")} onChange={(e) => edit({ from: e.target.value })} />
        </div>
        <Reason l={l} value={form.reason} onChange={(v) => edit({ reason: v })} error={errors.reason ?? ""} />
      </div>
      {group.status === "ready" && (
        <div className="live-buttons o02-form-actions">
          <button type="submit" className="live-primary" disabled={busy === "propose"} aria-busy={busy === "propose" || undefined}>{busy === "propose" ? c.proposing(l) : c.propose(l)}</button>
        </div>
      )}
    </form>
  );

  const targetsView = (
    <form className="o02-target" onSubmit={(e) => void propose(e)} noValidate aria-busy={busy === "propose" || undefined}>
      <div className="o02-fields">
        {handoff && handoff.base !== null && <BaseFact l={l} base={handoff.base} active={page.active_revision} />}
        {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
        {choicesFailed && <p className="live-note" role="alert">{c.choicesFailed(l)} <button type="button" className="service-button-link" onClick={() => void loadChoices()}>{o.tryAgain(l)}</button></p>}
        <fieldset className="live-roles" aria-describedby={errors.kind ? "o02-kind-error" : undefined}>
          <legend>{c.relationship(l)}</legend>
          {errorText("kind", "o02-kind")}
          {kinds.map((k, i) => (
            <label key={k}>
              <input id={`o02-kind-${i}`} type="radio" name="o02-kind" value={k} checked={form.kind === k} onChange={() => edit({ kind: k, action: "", target: "" })} />
              {o.kind(l, k)}
            </label>
          ))}
        </fieldset>
        <fieldset className="live-roles" aria-describedby={["o02-change-hint", errors.action ? "o02-change-error" : ""].filter(Boolean).join(" ")}>
          <legend>{c.change(l)}</legend>
          <p className="service-hint" id="o02-change-hint">{c.changeHint(l)}</p>
          {errorText("action", "o02-change")}
          {(form.kind ? actionsFor(form.kind) : (["set", "end"] as Action[])).map((a, i) => (
            <label key={a}>
              <input id={`o02-change-${i}`} type="radio" name="o02-change" value={a} checked={form.action === a} disabled={!form.kind} onChange={() => edit({ action: a, target: a === "end" ? "" : form.target })} />
              {c.action(l, a)}
            </label>
          ))}
        </fieldset>
        {form.memberById ? (
          <div className="service-field">
            <label htmlFor="o02-member-id" id="o02-member-label">{c.memberIdLabel(l)}</label>
            <p className="service-hint" id="o02-member-hint">{c.idHint(l)}</p>
            {errorText("member", "o02-member-id")}
            <input id="o02-member-id" dir="ltr" className="o02-id" value={form.member} aria-invalid={errors.member ? true : undefined} aria-describedby={described("member", "o02-member-id", "o02-member-hint")} onChange={(e) => edit({ member: e.target.value })} />
            <button type="button" id="o02-member-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-member-toggle o02-member-label" onClick={() => { edit({ memberById: !form.memberById }); setFocusTo(form.memberById ? "#o02-member" : "#o02-member-id"); }}>{form.memberById ? c.pick(l) : c.enterId(l)}</button>
          </div>
        ) : choices ? (
          <Picker l={l} id="o02-member" label={c.member(l)} hint={c.memberHint(l)} empty={c.noMember(l)} value={form.member} name={form.member ? nameOf(form.member, parties, seen) : ""} error={errorText("member", "o02-member")} invalid={!!errors.member}
            turn={turn} lead={(term) => unitChoices(parties, form.kind, term)} search={searchMembers} onError={refusedRead} onChoose={(m) => edit({ member: m.id })} onSeen={remember} onFocus={setFocusTo}>
            <button type="button" id="o02-member-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-member-toggle o02-member-label" onClick={() => { edit({ memberById: !form.memberById }); setFocusTo(form.memberById ? "#o02-member" : "#o02-member-id"); }}>{form.memberById ? c.pick(l) : c.enterId(l)}</button>
          </Picker>
        ) : (
          <div className="service-field">
            <span className="o02-field-label" id="o02-member-label">{c.member(l)}</span>
            {form.member && <p className="o02-chosen"><span className="o02-chosen-name"><Named id={form.member} name={form.member} /></span></p>}
            {!choicesFailed && <p className="live-meta" role="status">{c.loadingChoices(l)}</p>}
            <button type="button" id="o02-member-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-member-toggle o02-member-label" onClick={() => { edit({ memberById: !form.memberById }); setFocusTo(form.memberById ? "#o02-member" : "#o02-member-id"); }}>{form.memberById ? c.pick(l) : c.enterId(l)}</button>
          </div>
        )}
        {form.action !== "end" && (form.targetById || !pooled ? (
          <div className="service-field">
            <label htmlFor={form.targetById ? "o02-target-id" : "o02-target"} id="o02-target-label">{form.targetById ? c.targetIdLabel(l) : c.target(l)}</label>
            <p className="service-hint" id="o02-target-hint">{form.targetById ? c.idHint(l) : targetHint}</p>
            {errorText("target", form.targetById ? "o02-target-id" : "o02-target")}
            {form.targetById
              ? <input id="o02-target-id" dir="ltr" className="o02-id" value={form.target} aria-invalid={errors.target ? true : undefined} aria-describedby={described("target", "o02-target-id", "o02-target-hint")} onChange={(e) => edit({ target: e.target.value })} />
              : (
                <select id="o02-target" className="live-select" value={form.target} disabled={!form.kind} aria-invalid={errors.target ? true : undefined} aria-describedby={described("target", "o02-target", "o02-target-hint")} onChange={(e) => edit({ target: e.target.value })}>
                  <option value="">{c.choosePlaceholder(l)}</option>
                  {targetList.map((m) => <option key={m.id} value={m.id}>{m.name === m.id ? m.id : `${m.name} (${m.id})`}</option>)}
                </select>
              )}
            <button type="button" id="o02-target-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-target-toggle o02-target-label" onClick={() => { edit({ targetById: !form.targetById }); setFocusTo(form.targetById ? "#o02-target" : "#o02-target-id"); }}>{form.targetById ? c.pick(l) : c.enterId(l)}</button>
          </div>
        ) : !choices ? (
          <div className="service-field">
            <span className="o02-field-label" id="o02-target-label">{c.target(l)}</span>
            {form.target && <p className="o02-chosen"><span className="o02-chosen-name"><Named id={form.target} name={form.target} /></span></p>}
            {!choicesFailed && <p className="live-meta" role="status">{c.loadingChoices(l)}</p>}
            <button type="button" id="o02-target-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-target-toggle o02-target-label" onClick={() => { edit({ targetById: !form.targetById }); setFocusTo(form.targetById ? "#o02-target" : "#o02-target-id"); }}>{form.targetById ? c.pick(l) : c.enterId(l)}</button>
          </div>
        ) : (
          <Picker key={form.kind} l={l} id="o02-target" label={c.target(l)} hint={targetHint} empty={c.noTarget(l)} value={form.target} name={form.target ? nameOf(form.target, parties, seen) : ""} error={errorText("target", "o02-target")} invalid={!!errors.target}
            turn={`${turn}|${form.kind}`} exclude={form.member.trim()} search={searchTargets} onError={refusedRead} onChoose={(t) => edit({ target: t.id })} onSeen={remember} onFocus={setFocusTo}>
            <button type="button" id="o02-target-toggle" className="service-button-link o02-id-toggle" aria-labelledby="o02-target-toggle o02-target-label" onClick={() => { edit({ targetById: !form.targetById }); setFocusTo(form.targetById ? "#o02-target" : "#o02-target-id"); }}>{form.targetById ? c.pick(l) : c.enterId(l)}</button>
          </Picker>
        ))}
        <div className="service-field">
          <label htmlFor="o02-from">{c.fromLabel(l)}</label>
          <p className="service-hint" id="o02-from-hint">{c.fromHint(l)}</p>
          {errorText("from", "o02-from")}
          <input id="o02-from" type="date" className="o02-date" min={today} value={form.from} aria-invalid={errors.from ? true : undefined} aria-describedby={described("from", "o02-from", "o02-from-hint")} onChange={(e) => edit({ from: e.target.value })} />
        </div>
        <Reason l={l} value={form.reason} onChange={(v) => edit({ reason: v })} error={errors.reason ?? ""} />
      </div>
      <aside className="o02-current" aria-labelledby="o02-current-title">
        <h3 id="o02-current-title" tabIndex={-1}>{c.current(l)}</h3>
        {!form.kind || !form.member ? <p className="live-empty">{c.pickFirst(l)}</p> : (
          <>
            <dl className="live-facts">
              {validDay(form.from) && <div><dt>{c.onFrom(l, day(l, form.from))}</dt><dd>{currentTargets(form.from)}</dd></div>}
              <div><dt>{c.today(l, day(l, today))}</dt><dd>{currentTargets(today)}</dd></div>
            </dl>
            <p className="live-note">{c.currentNote(l)}</p>
          </>
        )}
      </aside>
      <div className="live-buttons o02-form-actions">
        <button type="submit" className="live-primary" disabled={busy === "propose"} aria-busy={busy === "propose" || undefined}>{busy === "propose" ? c.proposing(l) : c.propose(l)}</button>
      </div>
    </form>
  );

  const list = (rows: PlanRow[]) => (
    <section className="live-list" aria-busy={status === "loading" || undefined} aria-labelledby="o02-open-title">
      <div className="live-section-head">
        <h2 id="o02-open-title">{c.openPlans(l)}</h2>
        {showOpenCount(page.open_limit) && <p className="live-meta">{c.openCount(l, page.open_plans, page.open_limit)}</p>}
      </div>
      {showOpenCount(page.open_limit) && page.open_plans >= page.open_limit && <div className="live-note-rule"><p>{c.limit(l, page.open_limit)}</p></div>}
      {selected && !open.some((x) => x.id === selected.id) && <p className="live-note">{c.closedHere(l)} <button type="button" className="o02-inline-link" onClick={() => onView("result", { plan: selected.id })}>{c.resultTitle(l)}</button></p>}
      {rows.length === 0 ? (
        <p className="live-empty">{c.noOpen(l)} <button type="button" className="service-button-link" onClick={() => onView("targets")}>{c.proposeLink(l)}</button></p>
      ) : (
        <ul className="o02-plans">
          {rows.map((p) => {
            const memberName = p.draft ? planTitle(p) : p.member_name || nameOf(p.change.member, parties, allMembers);
            const targetName = p.draft ? "" : p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
            const current = p.id === selectedId;
            return (
              <li key={p.id}>
                <button type="button" className="o02-plan" data-plan={p.id} aria-current={current ? "true" : undefined} onClick={() => select(p.id)}>
                  <span className="o02-plan-name"><bdi>{memberName}</bdi></span>
                  <span className="o02-plan-line">{c.actionShort(l, p.change.action)} · {o.kind(l, p.change.kind)}{targetName && <> · <bdi>{targetName}</bdi></>}</span>
                  <span className="o02-plan-line">{c.from(l, day(l, p.change.from))}{p.stale && <> <span className="o02-tag">{c.staleTag(l)}</span></>}{current && <> <span className="o02-tag">{c.selectedTag(l)}</span></>}</span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );

  const notFound = selectedId && !selected && (
    <div className="live-note-rule" role="status"><p>{c.notFound(l)}</p><button type="button" className="service-button-link" onClick={() => { setStepped(false); onParams({ plan: null }); onView(view === "result" ? "result" : "impact"); }}>{c.showOpen(l)}</button></div>
  );

  const lang = params.get("lang");
  const o01Href = (p: PlanRow) => `#/organizations?${new URLSearchParams({ ...(lang ? { lang } : {}), view: "tree", on: p.change.from, revision: String(p.activated_revision) }).toString()}`;

  const approvalBody = (p: PlanRow) => {
    const memberName = p.member_name || nameOf(p.change.member, parties, allMembers);
    const targetName = p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
    const kindName = o.kind(l, p.change.kind);
    const sentence = p.draft ? c.groupLead(l, planTitle(p), p.draft.member_count, day(l, p.change.from)) : c.sentence(l, p.change.action, memberName, targetName, kindName);
    const reasonBox = <Reason l={l} value={actReason} onChange={(v) => { setActReason(v); setActError(""); attempts.current.activate = undefined; attempts.current.discard = undefined; }} error={actError} />;
    return (
      <Approval l={l} p={p} sentence={sentence} kindName={kindName} confirm={confirm} busy={busy} reason={reasonBox}
        onOpen={(k) => { opener.current = k === "activate" ? "#o02-open-activate" : "#o02-open-discard"; setConfirm(k); setFocusTo("#o02-confirm-title"); }}
        onDecide={(k) => void decide(k, p)} onCancel={() => { setConfirm(null); setActError(""); setFocusTo(opener.current); }} />
    );
  };

  let main: ReactNode = null;
  let panel: ReactNode = null;
  let single = false;
  if (view === "impact" || view === "approval") {
    main = list(open);
    panel = selected ? (
      <section className="live-step o02-detail" aria-labelledby="o02-plan-title">
        <Detail l={l} p={selected} page={page} parties={parties} members={allMembers} full={view === "impact"} readMembers={api.planMembers} onRefused={membersRefused} />
        {view === "impact" ? (
          <ImpactActions l={l} p={selected} onDecision={() => onView("approval", { plan: selected.id, done: "open" })} onAnother={() => onView("targets")} />
        ) : approvalBody(selected)}
      </section>
    ) : notFound || (open.length > 0 ? <p className="live-empty">{c.choose(l)}</p> : null);
  } else if (view === "result") {
    const cols = c.resultCols(l);
    main = (
      <section className="live-list" aria-busy={status === "loading" || undefined} aria-labelledby="o02-closed-title">
        <div className="live-section-head">
          <h2 id="o02-closed-title">{c.closed(l)}</h2>
          {page && closed.length > 0 && <p className="live-meta o02-closed-count">{searching ? c.closedMatching(l, closed.length, matching[history.key ?? ""] ?? closed.length) : c.closedShown(l, closed.length, page.closed.total)}</p>}
        </div>
        {page && (page.closed.total > 0 || searching) && <div className="o02-closed-search"><SearchForm l={l} id="o02-closed-q" labelText={c.closedSearch(l)} value={historyQ} onSearch={setHistoryQ} /></div>}
        {searching && (
          <p className="live-meta o02-closed-applied"><bdi>{c.closedApplied(l, historyQ)}</bdi> <button type="button" className="service-button-link o02-closed-clear" onClick={() => { setHistoryQ(""); setFocusTo("#o02-closed-q"); }}>{c.closedClear(l)}</button></p>
        )}
        {historyReset && !searching && <p className="live-note o02-closed-reset" role="status">{c.closedReset(l)}</p>}
        {searching && history.status === "loading" ? <p role="status" className="live-meta">{c.searching(l)}</p>
          : searching && history.status === "error" ? <p role="alert">{c.closedFailed(l)} <button type="button" className="service-button-link" onClick={history.retry}>{o.tryAgain(l)}</button></p>
          : closed.length === 0 ? <p className="live-empty">{searching ? c.closedNoMatch(l) : c.noClosed(l)}</p> : (
          <div className="security-table-scroll live-table o02-results">
            <table>
              <colgroup><col className="o02-c22" /><col className="o02-c22" /><col className="o02-c14" /><col className="o02-c22" /><col className="o02-c20" /></colgroup>
              <thead><tr>{cols.map((h) => <th scope="col" key={h}>{h}</th>)}</tr></thead>
              <tbody>
                {closed.map((p) => {
                  const memberName = p.draft ? planTitle(p) : p.member_name || nameOf(p.change.member, parties, allMembers);
                  const targetName = p.draft ? "" : p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
                  const current = p.id === selectedId;
                  return (
                    <tr key={p.id} className={current ? "live-selected" : undefined}>
                      <td data-label={cols[0]} className="live-user"><button type="button" className="live-open" data-plan={p.id} aria-current={current ? "true" : undefined} onClick={() => select(p.id)}><bdi>{memberName}</bdi></button><span className="live-sub live-mono">{ltr(p.change.member || p.change.target)}</span></td>
                      <td data-label={cols[1]}>{c.actionShort(l, p.change.action)} · {o.kind(l, p.change.kind)}{targetName && <span className="live-sub"><bdi>{targetName}</bdi></span>}</td>
                      <td data-label={cols[2]}><time dateTime={p.change.from}>{day(l, p.change.from)}</time></td>
                      <td data-label={cols[3]}><span className="o02-tag">{c.stateWord(l, p.state)}</span><span className="live-sub">{c.closing(l, closing(p), p.activated_revision)}</span></td>
                      <td data-label={cols[4]}><bdi>{p.proposer_login || p.proposer}</bdi><span className="live-sub">{stamp(l, p.created)}</span></td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        {page && !searching && !history.current && page.closed.next && (
          <div className="o01-more"><button type="button" className="live-secondary o01-more-button o02-closed-more" onClick={() => { setHistoryReset(false); setHistoryMore(true); }}>{o.loadMore(l)}</button></div>
        )}
        {history.current && history.status === "loading" && !searching && <p role="status" className="live-meta">{o.loadingMore(l)}</p>}
        {history.current && history.status === "error" && !searching && <p role="alert">{o.nextFailed(l)} <button type="button" className="service-button-link" onClick={history.retry}>{o.tryAgain(l)}</button></p>}
        {history.current && history.status !== "loading" && history.status !== "error" && <MoreBar l={l} next={history.next} stopped={false} status={history.status} count="" compact={false} onMore={history.more} />}
        {page && page.closed.dropped > 0 && closed.length > 0 && (searching ? history.current && !history.next && history.status === "ready" : history.current ? !history.next && history.status === "ready" : !page.closed.next) && (
          <p className="live-note o02-closed-end">{c.closedEnd(l, page.closed.kept)}</p>
        )}
      </section>
    );
    panel = selected ? (
      <section className="live-step o02-detail" aria-labelledby="o02-plan-title">
        <Detail l={l} p={selected} page={page} parties={parties} members={allMembers} full readMembers={api.planMembers} onRefused={membersRefused} />
        <p className="live-note">{c.readOnly(l)}</p>
        {selected.state === "activated" && selected.activated_revision > 0 && <p className="o02-link"><a href={o01Href(selected)}>{c.o01Link(l, day(l, selected.change.from), selected.activated_revision)}</a></p>}
        {selected.state === "sealed" && <button type="button" className="service-button-link" onClick={() => onView("approval", { plan: selected.id })}>{decisionLabel(l, selected)}</button>}
      </section>
    ) : notFound || (closed.length > 0 ? <p className="live-empty">{c.choose(l)}</p> : null);
  } else {
    single = true;
    main = draftId ? groupView : targetsView;
  }

  const showPanel = !single && (split ? panel !== null : stepped);
  return (
    <div className={"o02-live" + (single ? " o02-single" : "")} data-live={view}>
      <p className="visually-hidden" id="o02-announce" role="status" aria-live="polite">{announce}</p>
      {scope}
      {status === "error" && page && <div className="live-error o02-kept" role="alert"><p>{c.loadFailed(l)} {diagnosticCopy.kept(l)}</p><button type="button" className="live-secondary" onClick={() => void reload()}>{o.tryAgain(l)}</button></div>}
      {noticeBox}
      {failureBox}
      {heading}
      {single && handoff && (draftId ? <GroupHandoff l={l} h={handoff} group={group} active={page.active_revision} onUseBase={adoptBase} onRetry={() => void loadGroup(draftId)} />
        : <HandoffNote l={l} h={handoff} active={page.active_revision} onUseBase={adoptBase} />)}
      {single ? main : (
        <div className={"live-users" + (split ? " o02-open" : "")} data-wide={split ? "true" : "false"}>
          {!(stepped && !split) && <div className="live-list live-sections">{main}</div>}
          {showPanel && (
            <aside className="live-panel" aria-label={c.choose(l)} aria-busy={status === "loading" || undefined}>
              {stepped && !split && <button type="button" className="service-button-link live-back" onClick={back}>{view === "result" ? c.backClosed(l) : c.back(l)}</button>}
              {panel}
            </aside>
          )}
        </div>
      )}
    </div>
  );
}
