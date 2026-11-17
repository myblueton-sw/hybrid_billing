import { useCallback, useEffect, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type Attempt, type CommandId, type OrgMember, type OrgParty, type PlanRow, type PlansPage, type RelationKind } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { FailureBox, Reason, reasonProblem, useWide } from "../../../identity/pages/users/Live";
import { w } from "../../../identity/pages/users/live-copy";
import { o } from "../../../organizations/pages/organizations/live-copy";
import { todayUTC, validDay } from "../../../organizations/pages/organizations/live-model";
import { c } from "./live-copy";
import {
  actionsFor, allMemberChoices, blocked, canActivate, closing, currentOn, findPlan, impactOf, keepsAttempt, kinds, lead, membersKept, minFrom,
  nameOf, needsTarget, partition, react, showOpenCount, steps, targetChoices, type Action, type Choice, type Command, type Reaction,
} from "./live-model";
import "../../../identity/pages/users/live.css";
import "./live.css";

const maxPages = 10;
const pageSize = 99;
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

type Choices = { parties: OrgParty[]; byKind: Partial<Record<RelationKind, OrgMember[]>>; complete: boolean };

async function readChoices(api: AdminClient, on: string): Promise<Choices> {
  const first = await api.organizations({ on, kind: "hierarchy", limit: pageSize });
  const revision = first.relationship_revision;
  const byKind: Partial<Record<RelationKind, OrgMember[]>> = {};
  let complete = true;
  for (const kind of kinds) {
    let page = kind === "hierarchy" ? first : null;
    const members: OrgMember[] = [];
    let after = "";
    for (let n = 0; n < maxPages; n++) {
      try {
        page = page ?? (await api.organizations({ on, kind, revision, after, limit: pageSize }));
      } catch (e) {
        if (e instanceof AdminError && (e.failure === "denied" || e.failure === "unauthenticated")) throw e;
        complete = false;
        break;
      }
      members.push(...page.members);
      after = page.next;
      page = null;
      if (!after) break;
    }
    if (after) complete = false;
    byKind[kind] = members;
  }
  return { parties: first.parties, byKind, complete };
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

function Steps({ l, p }: { l: Locale; p: PlanRow }) {
  const s = steps(p);
  const labels = [
    c.proposed(l),
    p.state === "sealed" || p.state === "rejected" || p.state === "activated" || p.state === "discarded" ? c.stateWord(l, p.state === "activated" || p.state === "discarded" ? "sealed" : p.state) : c.sealedOrRejected(l),
    p.state === "activated" || p.state === "discarded" ? c.stateWord(l, p.state) : p.state === "rejected" ? c.notApplicable(l) : c.pending3(l),
  ];
  return (
    <ol className="o02-steps" aria-label={c.progress(l)}>
      {s.map((state, i) => (
        <li key={i} className="o02-step" aria-current={state === "current" ? "step" : undefined} data-state={state}>
          <span className="o02-step-n">{c.step(l, i + 1)}</span>
          <span className="o02-step-label">{labels[i]}</span>
          <span className="o02-step-state">{c.stepState(l, state)}</span>
        </li>
      ))}
    </ol>
  );
}

function Named({ id, name }: { id: string; name: string }) {
  return <><bdi>{name}</bdi>{name !== id && <span className="live-sub live-mono">{ltr(id)}</span>}</>;
}

function Detail({ l, p, page, parties, members, full, titleRef }: {
  l: Locale; p: PlanRow; page: PlansPage; parties: OrgParty[]; members: OrgMember[]; full: boolean; titleRef?: React.Ref<HTMLHeadingElement>;
}) {
  const named = (id: string) => nameOf(id, parties, members);
  const kindName = o.kind(l, p.change.kind);
  const memberName = p.member_name || named(p.change.member);
  const targetName = p.change.target ? p.target_name || named(p.change.target) : "";
  const ld = lead(p);
  const impact = [page.impact.money, page.impact.publication, page.impact.erp];
  return (
    <>
      <h2 id="o02-plan-title" tabIndex={-1} ref={titleRef}>{c.sentence(l, p.change.action, memberName, targetName, kindName)}</h2>
      <p className="live-sub live-mono">{ltr(p.id)}</p>
      <Steps l={l} p={p} />
      {p.stale && <div className="live-note-rule"><p><strong>{c.staleTag(l)}.</strong> {c.stale(l, page.active_revision, p.base)}</p></div>}
      {p.truncated && <div className="live-note-rule"><p>{c.truncated(l)}</p></div>}
      <dl className="live-facts o02-summary">
        <div><dt>{c.facts.member(l)}</dt><dd><Named id={p.change.member} name={memberName} /></dd></div>
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
                      {body && <p>{body}</p>}
                      <span className="live-sub live-mono">{ltr([v.rule, v.kind, v.member].filter(Boolean).join(" · "))}</span>
                    </li>
                  );
                })}
              </ul>
            </section>
          )}
          <section aria-labelledby="o02-diff-title">
            <h3 id="o02-diff-title">{c.diff(l)}</h3>
            {!membersKept(p) ? <p className="live-note">{c.notKept(l)}</p> : p.members.length === 0 ? <p className="live-note">{c.none(l)}</p> : (
              <>
                <div className="security-table-scroll live-table o02-diff">
                  <table>
                    <colgroup><col className="o02-c20" /><col className="o02-c24" /><col className="o02-c16" /><col className="o02-c20" /><col className="o02-c20" /></colgroup>
                    <thead><tr><th scope="col">{c.colMember(l)}</th><th scope="col">{c.colChange(l)}</th><th scope="col">{c.colRanges(l)}</th><th scope="col">{c.colBefore(l)}</th><th scope="col">{c.colAfter(l)}</th></tr></thead>
                    <tbody>
                      {p.members.map((m) => (
                        <tr key={m.member}>
                          <td data-label={c.colMember(l)}><Named id={m.member} name={m.name || named(m.member)} /></td>
                          <td data-label={c.colChange(l)}>{m.member === p.change.member ? (ld.target ? c.leadTo(l, day(l, ld.from), named(ld.target)) : c.leadNone(l, day(l, ld.from))) : c.none(l)}</td>
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
                <p className="live-note">{c.timelineNote(l)}</p>
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
            <p className="live-impact">{c.impactNote(l)}</p>
            <dl className="o02-impact">
              {c.impactRows(l).map((row, i) => (
                <div key={row}>
                  <dt>{row}</dt>
                  <dd><span className="o02-tag">{impactOf(impact[i]) === "not_computed" ? c.notComputed(l) : c.unknownImpact(l)}</span> <span className="live-sub">{c.notZero(l)}</span></dd>
                </div>
              ))}
            </dl>
          </section>
        </div>
      )}
    </>
  );
}

type Form = { kind: RelationKind | ""; action: Action | ""; member: string; memberById: boolean; target: string; targetById: boolean; from: string; reason: string };
const emptyForm: Form = { kind: "", action: "", member: "", memberById: false, target: "", targetById: false, from: "", reason: "" };
type Errors = Partial<Record<"kind" | "action" | "member" | "target" | "from" | "reason", string>>;

export default function LiveChanges({ locale: l, view, params, session, onSignOut, onView, onParams }: {
  locale: Locale; view: string; params: URLSearchParams; session: Session; onSignOut: () => void;
  onView: (view: string, params?: Record<string, string>) => void; onParams: (params: Record<string, string | null>) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const [page, setPage] = useState<PlansPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [choices, setChoices] = useState<Choices | null>(null);
  const [choicesFailed, setChoicesFailed] = useState(false);
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [stepped, setStepped] = useState(() => !!params.get("plan"));
  const [notice, setNotice] = useState<Notice>(null);
  const [failure, setFailure] = useState<Failure>(null);
  const [denied, setDenied] = useState(false);
  const [form, setForm] = useState<Form>(emptyForm);
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

  const loadChoices = useCallback(async () => {
    const n = ++choiceSeq.current;
    setChoicesFailed(false);
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
    const m = /^(sealed|rejected|discarded|open|activated:(\d+))$/.exec(arrived);
    if (!m) return;
    const what = arrived.split(":")[0];
    if (what !== "open") say(what === "rejected" ? c.aRejected(l) : what === "activated" ? c.aActivated(l, m[2]) : what === "discarded" ? c.aDiscarded(l) : c.aSealed(l));
    setFocusTo("#o02-plan-title");
  }, [arrived, page, status]);
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
  const allMembers = choices ? kinds.flatMap((k) => choices.byKind[k] ?? []) : [];
  const { open, closed } = partition(page?.plans ?? []);
  const selected = findPlan(page?.plans ?? [], selectedId);

  const edit = (patch: Partial<Form>) => {
    attempts.current.propose = undefined;
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
    f === "kind" ? "o02-kind-0" : f === "action" ? "o02-change-0" : f === "member" ? (form.memberById ? "o02-member-id" : "o02-member")
      : f === "target" ? (form.targetById ? "o02-target-id" : "o02-target") : f === "from" ? "o02-from" : "live-reason";

  const validate = (): Errors => {
    const e: Errors = {};
    if (!form.kind) e.kind = c.eKind(l);
    if (!form.action) e.action = c.eChange(l);
    if (!form.member.trim()) e.member = c.eMember(l);
    if (form.action && needsTarget(form.action) && !form.target.trim()) e.target = c.eTarget(l);
    if (!validDay(form.from)) e.from = c.eDate(l);
    else if (!minFrom(today, form.from)) e.from = c.ePast(l);
    const r = reasonProblem(l, form.reason);
    if (r) e.reason = r;
    return e;
  };

  const handle = async (err: unknown, cmd: Command) => {
    const reaction = (err as { reaction?: Reaction }).reaction ?? "failed";
    const cause = (err as { cause?: unknown }).cause;
    const fields = cause instanceof AdminError ? cause.fields : [];
    if (reaction === "denied") { setDenied(true); return; }
    if (reaction === "field") {
      const known: Errors = {};
      for (const f of fields) {
        if (f === "member") known.member = c.eMemberServer(l);
        else if (f === "target") known.target = c.eTargetServer(l);
        else if (f === "from") known.from = c.eFromServer(l);
        else if (f === "action" || f === "kind") known.action = c.eActionServer(l);
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
    if (reaction === "own") { await load(true); setNotice({ text: c.own(l) }); setFocusTo("#o02-notice"); return; }
    if (reaction === "stage") { await load(true); setNotice({ text: c.stageChanged(l) }); setFocusTo("#o02-notice"); return; }
    if (reaction === "conflict") { await load(true); setNotice({ text: c.conflict(l) }); setFocusTo("#o02-notice"); return; }
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
    const values: Record<string, string> = { kind: form.kind, member: form.member.trim(), action: form.action, from: form.from };
    if (form.action && needsTarget(form.action)) values.target = form.target.trim();
    try {
      const out = await run("propose", "relationship.propose", "relationships", form.reason, values);
      if (!out) return;
      const id = out.fields.plan ?? "";
      await load(true);
      setForm(emptyForm);
      setErrors({});
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
      await handle(err, kind);
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

  const problems: Problems = (["kind", "action", "member", "target", "from", "reason"] as const).filter((f) => errors[f]).map((f) => ({ field: fieldId(f), message: errors[f] as string }));
  const errorText = (f: keyof Errors, id: string) => errors[f] && <p className="service-field-error" id={id + "-error"}><span className="visually-hidden">{sc.errorPrefix(l)} </span>{errors[f]}</p>;
  const described = (f: keyof Errors, id: string, hint?: string) => [hint, errors[f] ? id + "-error" : ""].filter(Boolean).join(" ") || undefined;

  const kindMembers = form.kind ? choices?.byKind[form.kind] ?? [] : [];
  const memberList: Choice[] = allMemberChoices(parties, kinds.map((k) => choices?.byKind[k] ?? []));
  const targetList: Choice[] = form.kind ? targetChoices(form.kind, form.member, parties, kindMembers) : [];
  const targetHint = form.kind === "hierarchy" ? c.hintHierarchy(l) : form.kind === "payer" ? c.hintPayer(l) : c.hintPool(l);
  const currentTargets = (on: string): ReactNode => {
    if (!form.kind || !form.member) return null;
    const got = currentOn(form.member, kindMembers, on);
    if (got === null) return <span className="live-quiet">{choices?.complete ? c.noneOfKind(l) : c.notInListing(l)}</span>;
    if (got.length === 0) return <span className="live-quiet">{c.none(l)}</span>;
    return <ul className="o02-ranges">{got.map((t) => <li key={t.id}><Named id={t.id} name={t.name} /></li>)}</ul>;
  };

  const targetsView = (
    <form className="o02-target" onSubmit={(e) => void propose(e)} noValidate aria-busy={busy === "propose" || undefined}>
      <div className="o02-fields">
        {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={summaryKey} />}
        {!choices && !choicesFailed && <p className="live-meta" role="status">{c.loadingChoices(l)}</p>}
        {(choicesFailed || (choices && !choices.complete)) && <p className="live-note">{c.incomplete(l)}</p>}
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
        <div className="service-field">
          <label htmlFor={form.memberById ? "o02-member-id" : "o02-member"}>{form.memberById ? c.memberIdLabel(l) : c.member(l)}</label>
          <p className="service-hint" id="o02-member-hint">{form.memberById ? c.idHint(l) : c.memberHint(l)}</p>
          {errorText("member", form.memberById ? "o02-member-id" : "o02-member")}
          {form.memberById
            ? <input id="o02-member-id" dir="ltr" className="o02-id" value={form.member} aria-invalid={errors.member ? true : undefined} aria-describedby={described("member", "o02-member-id", "o02-member-hint")} onChange={(e) => edit({ member: e.target.value })} />
            : (
              <select id="o02-member" className="live-select" value={form.member} aria-invalid={errors.member ? true : undefined} aria-describedby={described("member", "o02-member", "o02-member-hint")} onChange={(e) => edit({ member: e.target.value })}>
                <option value="">{c.choosePlaceholder(l)}</option>
                {memberList.map((m) => <option key={m.id} value={m.id}>{m.name === m.id ? m.id : `${m.name} (${m.id})`}</option>)}
              </select>
            )}
          <button type="button" className="service-button-link o02-id-toggle" onClick={() => { edit({ memberById: !form.memberById }); setFocusTo(form.memberById ? "#o02-member" : "#o02-member-id"); }}>{form.memberById ? c.pick(l) : c.enterId(l)}</button>
        </div>
        {form.action !== "end" && (
          <div className="service-field">
            <label htmlFor={form.targetById ? "o02-target-id" : "o02-target"}>{form.targetById ? c.targetIdLabel(l) : c.target(l)}</label>
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
            <button type="button" className="service-button-link o02-id-toggle" onClick={() => { edit({ targetById: !form.targetById }); setFocusTo(form.targetById ? "#o02-target" : "#o02-target-id"); }}>{form.targetById ? c.pick(l) : c.enterId(l)}</button>
          </div>
        )}
        <div className="service-field">
          <label htmlFor="o02-from">{c.fromLabel(l)}</label>
          <p className="service-hint" id="o02-from-hint">{c.fromHint(l)}</p>
          {errorText("from", "o02-from")}
          <input id="o02-from" type="date" className="o02-date" min={today} value={form.from} aria-invalid={errors.from ? true : undefined} aria-describedby={described("from", "o02-from", "o02-from-hint")} onChange={(e) => edit({ from: e.target.value })} />
        </div>
        <Reason l={l} value={form.reason} onChange={(v) => edit({ reason: v })} error={errors.reason ?? ""} />
      </div>
      <aside className="o02-current" aria-labelledby="o02-current-title">
        <h3 id="o02-current-title">{c.current(l)}</h3>
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
            const memberName = p.member_name || nameOf(p.change.member, parties, allMembers);
            const targetName = p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
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
    if (p.state !== "sealed") return <p className="live-note">{c.onlySealed(l, c.stateWord(l, p.state))}</p>;
    const b = blocked(p);
    const memberName = p.member_name || nameOf(p.change.member, parties, allMembers);
    const targetName = p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
    const kindName = o.kind(l, p.change.kind);
    const sentence = c.sentence(l, p.change.action, memberName, targetName, kindName);
    const reasonBox = <Reason l={l} value={actReason} onChange={(v) => { setActReason(v); setActError(""); attempts.current.activate = undefined; attempts.current.discard = undefined; }} error={actError} />;
    return (
      <div className="live-sections">
        <section className="live-group" aria-labelledby="o02-activate-title">
          <h3 id="o02-activate-title">{c.activate(l)}</h3>
          <p className="live-impact">{c.restate(l, day(l, p.change.from), sentence, kindName)}</p>
          {!canActivate(p) ? (
            <div className="live-blocked">
              <p><strong>{c.blockedTitle(l)}</strong></p>
              <ul><li>{b === "own" ? c.own(l) : b === "stale" ? c.staleBlock(l) : b === "passed" ? c.passed(l) : c.otherBlock(l)}</li></ul>
              {(b === "stale" || b === "passed") && <p className="live-note">{c.next(l)}</p>}
            </div>
          ) : confirm !== "activate" ? (
            <button type="button" id="o02-open-activate" className="live-secondary" disabled={confirm !== null} onClick={() => { opener.current = "#o02-open-activate"; setConfirm("activate"); setFocusTo("#o02-confirm-title"); }}>{c.reviewActivate(l)}</button>
          ) : (
            <div className="o02-confirm">
              <h3 id="o02-confirm-title" tabIndex={-1}>{c.confirmActivate(l)}</h3>
              <dl className="live-facts">
                <div><dt>{c.facts.from(l)}</dt><dd>{day(l, p.change.from)} (00:00 UTC)</dd></div>
                <div><dt>{c.facts.change(l)}</dt><dd>{sentence}</dd></div>
              </dl>
              {reasonBox}
              <div className="live-buttons">
                <button type="button" className="live-primary" disabled={busy !== null} aria-busy={busy === "activate" || undefined} onClick={() => void decide("activate", p)}>{busy === "activate" ? c.activating(l) : c.activateBtn(l, day(l, p.change.from))}</button>
                <button type="button" className="live-secondary" disabled={busy !== null} onClick={() => { setConfirm(null); setActError(""); setFocusTo(opener.current); }}>{c.cancel(l)}</button>
              </div>
            </div>
          )}
        </section>
        <section className="live-group" aria-labelledby="o02-discard-title">
          <h3 id="o02-discard-title">{c.discard(l)}</h3>
          {confirm !== "discard" ? (
            <button type="button" id="o02-open-discard" className="live-secondary" disabled={confirm !== null} onClick={() => { opener.current = "#o02-open-discard"; setConfirm("discard"); setFocusTo("#o02-confirm-title"); }}>{c.reviewDiscard(l)}</button>
          ) : (
            <div className="o02-confirm">
              <h3 id="o02-confirm-title" tabIndex={-1}>{c.confirmDiscard(l)}</h3>
              <p className="live-impact"><strong>{c.permanent(l)}</strong></p>
              {reasonBox}
              <div className="live-buttons">
                <button type="button" className="live-primary" disabled={busy !== null} aria-busy={busy === "discard" || undefined} onClick={() => void decide("discard", p)}>{busy === "discard" ? c.discarding(l) : c.discardBtn(l)}</button>
                <button type="button" className="live-secondary" disabled={busy !== null} onClick={() => { setConfirm(null); setActError(""); setFocusTo(opener.current); }}>{c.keep(l)}</button>
              </div>
            </div>
          )}
        </section>
      </div>
    );
  };

  let main: ReactNode = null;
  let panel: ReactNode = null;
  let single = false;
  if (view === "impact" || view === "approval") {
    main = list(open);
    panel = selected ? (
      <section className="live-step o02-detail" aria-labelledby="o02-plan-title">
        <Detail l={l} p={selected} page={page} parties={parties} members={allMembers} full={view === "impact"} />
        {view === "impact" ? (
          <div className="live-buttons">
            {selected.state === "sealed" && <button type="button" className="live-primary" onClick={() => onView("approval", { plan: selected.id, done: "open" })}>{c.toApproval(l)}</button>}
            <button type="button" className="service-button-link" onClick={() => onView("targets")}>{c.another(l)}</button>
          </div>
        ) : approvalBody(selected)}
      </section>
    ) : notFound || (open.length > 0 ? <p className="live-empty">{c.choose(l)}</p> : null);
  } else if (view === "result") {
    const cols = c.resultCols(l);
    main = (
      <section className="live-list" aria-busy={status === "loading" || undefined} aria-labelledby="o02-closed-title">
        <div className="live-section-head"><h2 id="o02-closed-title">{c.closed(l)}</h2></div>
        {closed.length === 0 ? <p className="live-empty">{c.noClosed(l)}</p> : (
          <div className="security-table-scroll live-table o02-results">
            <table>
              <colgroup><col className="o02-c22" /><col className="o02-c22" /><col className="o02-c14" /><col className="o02-c22" /><col className="o02-c20" /></colgroup>
              <thead><tr>{cols.map((h) => <th scope="col" key={h}>{h}</th>)}</tr></thead>
              <tbody>
                {closed.map((p) => {
                  const memberName = p.member_name || nameOf(p.change.member, parties, allMembers);
                  const targetName = p.change.target ? p.target_name || nameOf(p.change.target, parties, allMembers) : "";
                  const current = p.id === selectedId;
                  return (
                    <tr key={p.id} className={current ? "live-selected" : undefined}>
                      <td data-label={cols[0]} className="live-user"><button type="button" className="live-open" data-plan={p.id} aria-current={current ? "true" : undefined} onClick={() => select(p.id)}><bdi>{memberName}</bdi></button><span className="live-sub live-mono">{ltr(p.change.member)}</span></td>
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
      </section>
    );
    panel = selected ? (
      <section className="live-step o02-detail" aria-labelledby="o02-plan-title">
        <Detail l={l} p={selected} page={page} parties={parties} members={allMembers} full />
        <p className="live-note">{c.readOnly(l)}</p>
        {selected.state === "activated" && selected.activated_revision > 0 && <p className="o02-link"><a href={o01Href(selected)}>{c.o01Link(l, day(l, selected.change.from), selected.activated_revision)}</a></p>}
        {selected.state === "sealed" && <button type="button" className="service-button-link" onClick={() => onView("approval", { plan: selected.id })}>{c.toApproval(l)}</button>}
      </section>
    ) : notFound || (closed.length > 0 ? <p className="live-empty">{c.choose(l)}</p> : null);
  } else {
    single = true;
    main = targetsView;
  }

  const showPanel = !single && (split ? panel !== null : stepped);
  return (
    <div className={"o02-live" + (single ? " o02-single" : "")} data-live={view}>
      <p className="visually-hidden" id="o02-announce" role="status" aria-live="polite">{announce}</p>
      {scope}
      {noticeBox}
      {failureBox}
      {heading}
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
