import { useCallback, useEffect, useRef, useState, type ReactNode, type RefObject } from "react";
import { flushSync } from "react-dom";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt, type AdminClient, type DraftMember, type GroupDraft, type OrgMember, type OrgTargetRow } from "../../../../service/admin";
import { Summary, type Problems } from "../../../../service/Service";
import { c as sc } from "../../../../service/copy";
import { Reason, reasonProblem } from "../../../identity/pages/users/Live";
import { w } from "../../../identity/pages/users/live-copy";
import { c as cc } from "../changes/live-copy";
import { fill } from "../changes/live-model";
import { o } from "./live-copy";
import { label } from "./live-model";
import { SearchForm } from "./filters";
import {
  afterRefusal, callRevision, conflictCopy, isLatest, partitionPrecheck, readOutcome, sameView, savePlan,
  addPending, allowed, block, chunks, dirty, draftHref, draftsPage, editCalls, lastPlanNote, parseIds, planHref, refusalOf, settingValue, shownStep, stateKey, validName,
  type Snapshot,
} from "./drafts-model";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
const stampText = (l: Locale, iso: string) => {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "short", timeZone: "UTC" }).format(d) + " UTC";
};
const dayText = (l: Locale, iso: string) => {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return iso;
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeZone: "UTC" }).format(d);
};
const mono = (s: string) => <bdi dir="ltr" className="live-mono o01-nowrap">{s}</bdi>;

export const stateText = (l: Locale, s: GroupDraft["state"]) => o[stateKey(s)](l);

export type DraftList = {
  status: "idle" | "loading" | "ready" | "error";
  drafts: GroupDraft[];
  next: string;
  more: "idle" | "more" | "failed";
  open: number;
  limit: number;
  maxMembers: number;
  revision: string;
  active: number;
  reload: () => Promise<GroupDraft[] | null>;
  loadMore: () => void;
  find: (id: string) => Promise<GroupDraft | null>;
};

export function useDraftList(api: AdminClient, enabled: boolean): DraftList {
  const [state, setState] = useState({ status: "idle" as DraftList["status"], drafts: [] as GroupDraft[], next: "", more: "idle" as DraftList["more"], open: 0, limit: 0, maxMembers: 0, revision: "", active: 0 });
  const seq = useRef(0);
  const reload = useCallback(async () => {
    const n = ++seq.current;
    setState((s) => ({ ...s, status: s.status === "ready" ? "ready" : "loading" }));
    try {
      const page = await api.viewGroupDrafts("", draftsPage);
      if (n !== seq.current) return null;
      setState({ status: "ready", drafts: page.drafts, next: page.next, more: "idle", open: page.open, limit: page.open_limit, maxMembers: settingValue(page.settings, "view_group_draft_members", 1000), revision: page.revision, active: page.active_revision });
      return page.drafts;
    } catch {
      if (n === seq.current) setState((s) => ({ ...s, status: "error" }));
      return null;
    }
  }, [api]);
  useEffect(() => { if (enabled && state.status === "idle") void reload(); }, [enabled]); // eslint-disable-line react-hooks/exhaustive-deps
  const loadMore = () => {
    if (!state.next || state.more === "more") return;
    const n = seq.current;
    setState((s) => ({ ...s, more: "more" }));
    api.viewGroupDrafts(state.next, draftsPage)
      .then((page) => { if (n === seq.current) setState((s) => ({ ...s, drafts: [...s.drafts, ...page.drafts], next: page.next, more: "idle", revision: page.revision, active: page.active_revision })); })
      .catch(() => { if (n === seq.current) setState((s) => ({ ...s, more: "failed" })); });
  };
  const find = async (id: string): Promise<GroupDraft | null> => {
    let after = "";
    for (let i = 0; i < 40; i++) {
      const page = await api.viewGroupDrafts(after, 50);
      const hit = page.drafts.find((d) => d.id === id);
      if (hit) return hit;
      if (!page.next) return null;
      after = page.next;
    }
    return null;
  };
  return { ...state, reload, loadMore, find };
}

export function RowCheck({ l, id, name, checked, disabled, stacked, onToggle }: { l: Locale; id: string; name: string; checked: boolean; disabled: boolean; stacked: boolean; onToggle: () => void }) {
  return (
    <label className={"o01-check-hit" + (stacked ? " o01-check-stacked" : "")}>
      <input type="checkbox" className="o01-check-box" data-check={id} checked={checked} disabled={disabled} aria-label={o.selectRow(l, name, id)} onChange={onToggle} />
      {stacked && <span aria-hidden="true">{o.selectWord(l)}</span>}
    </label>
  );
}

export function HeaderCheck({ l, state, onToggle }: { l: Locale; state: "none" | "some" | "all"; onToggle: () => void }) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => { if (ref.current) ref.current.indeterminate = state === "some"; }, [state]);
  return (
    <label className="o01-check-hit">
      <input ref={ref} type="checkbox" className="o01-check-box" id="o01-check-all" checked={state === "all"} aria-label={o.selectAllRows(l)} onChange={onToggle} />
    </label>
  );
}

export function SelectionBar({ l, n, k, cap, atCap, limitReached, limit, editing, cleared, wide, onClear, onStart, onAdd, onSelectAll }: {
  l: Locale; n: number; k: number; cap: number; atCap: boolean; limitReached: boolean; limit: number; editing: boolean; cleared: boolean; wide: boolean;
  onClear: () => void; onStart: () => void; onAdd: () => void; onSelectAll: () => void;
}) {
  if (n === 0) {
    return (
      <div className="o01-selbar">
        <p className="o01-selcount">{cleared ? o.selCleared(l) : o.selNone(l)}</p>
        {!wide && k > 0 && <button type="button" className="service-button-link" onClick={onSelectAll}>{o.selectAllShown(l)}</button>}
      </div>
    );
  }
  return (
    <div className="o01-selbar">
      <p className="o01-selcount o01-selcount-on" data-n={n}>{fill(o.selCount(l), { n: <strong>{n}</strong>, k: <span>{k}</span> })}</p>
      <button type="button" className="service-button-link" onClick={onClear}>{o.clearSel(l)}</button>
      {editing ? <button type="button" className="live-secondary o01-bar-action" onClick={onAdd}>{o.addToDraft(l, n)}</button>
        : limitReached ? <p className="o01-selnote">{o.openLimit(l, limit)}</p>
          : <button type="button" className="live-secondary o01-bar-action" onClick={onStart}>{o.startDraft(l)}</button>}
      {atCap && <p className="o01-selnote">{editing ? o.selCapDraft(l, cap) : o.selCap(l, cap)}</p>}
    </div>
  );
}

function PlanRef({ l, id, lang }: { l: Locale; id: string; lang: string }) {
  return <a href={planHref(id, lang)} className="o01-planref">{fill(o.planLink(l), { p: mono(id) })}</a>;
}

function useNarrow(ref: RefObject<HTMLElement | null>, below: number): boolean {
  const [narrow, setNarrow] = useState(false);
  useEffect(() => {
    const el = ref.current;
    if (!el || typeof ResizeObserver === "undefined") return;
    const watch = new ResizeObserver(([entry]) => setNarrow(entry.contentRect.width < below));
    watch.observe(el);
    return () => watch.disconnect();
  }, [ref, below]);
  return narrow;
}

export function DraftRows({ l, drafts, lang, current, onOpen }: { l: Locale; drafts: GroupDraft[]; lang: string; current: string; onOpen: (d: GroupDraft) => void }) {
  return (
    <ul className="o01-draft-rows">
      {drafts.map((d) => {
        const note = lastPlanNote(d);
        return (
          <li key={d.id} className={current === d.id ? "live-selected" : undefined}>
            <div className="o01-draft-row1">
              <button type="button" className="live-open" data-draft={d.id} aria-current={current === d.id ? "true" : undefined} onClick={() => onOpen(d)}><bdi>{d.new_group ? d.name || d.id : label(d.group_label, d.group)}</bdi></button>
              <span className="o01-draft-rowstate">{stateText(l, d.state)}{note && <> · {note === "rejected" ? o.lastRejected(l) : o.lastDiscarded(l)}</>} · <span className="o01-nowrap">{o.nMembers(l, d.member_count)}</span></span>
            </div>
            <p className="o01-draft-row2">{mono(d.id)} · {d.new_group ? o.newGroup(l) : o.existingGroup(l)} {mono(d.group)} · {d.plan ? <a href={planHref(d.plan.id, lang)} className="o01-planref">{mono(d.plan.id)}</a> : o.noChangePlan(l)}</p>
          </li>
        );
      })}
    </ul>
  );
}

export function DraftsSection({ l, list, lang, current, onOpen, onNew }: {
  l: Locale; list: DraftList; lang: string; current: string; onOpen: (d: GroupDraft) => void; onNew: () => void;
}) {
  const box = useRef<HTMLElement>(null);
  const narrow = useNarrow(box, 760);
  const full = list.status === "ready" && list.limit > 0 && list.open >= list.limit;
  return (
    <section className="o01-drafts" aria-labelledby="o01-drafts-title" ref={box}>
      <div className="o01-list-head o01-sub-head">
        <div className="live-heading">
          <h3 id="o01-drafts-title">{o.drafts(l)}</h3>
          {list.status === "ready" && <p className="live-meta">{o.openDrafts(l, list.open, list.limit)}{full && <> · {o.openLimit(l, list.limit)}</>}</p>}
        </div>
        {list.status === "ready" && !full && <button type="button" id="o01-new-draft" className="live-secondary o01-bar-action" onClick={onNew}>{o.newDraft(l)}</button>}
      </div>
      {list.status === "loading" || list.status === "idle" ? <p role="status" className="live-meta">{o.loadingDrafts(l)}</p>
        : list.status === "error" ? (
          <div className="live-error" role="alert"><p>{o.draftsFailed(l)}</p><button type="button" className="live-secondary" onClick={() => void list.reload()}>{o.tryAgain(l)}</button></div>
        ) : list.drafts.length === 0 ? <p className="live-empty">{o.noDrafts(l)}</p> : narrow ? <DraftRows l={l} drafts={list.drafts} lang={lang} current={current} onOpen={onOpen} /> : (
          <div className="security-table-scroll live-table o01-drafts-table">
            <table role="table">
              <thead role="rowgroup"><tr role="row">
                <th scope="col" role="columnheader">{o.colDraft(l)}</th>
                <th scope="col" role="columnheader">{o.groupLegend(l)}</th>
                <th scope="col" role="columnheader">{o.colMembers(l)}</th>
                <th scope="col" role="columnheader">{o.colState(l)}</th>
                <th scope="col" role="columnheader">{o.colPlan(l)}</th>
              </tr></thead>
              <tbody role="rowgroup">
                {list.drafts.map((d) => {
                  const note = lastPlanNote(d);
                  return (
                    <tr key={d.id} role="row" className={current === d.id ? "live-selected" : undefined}>
                      <td role="cell" className="live-user">
                        <button type="button" className="live-open" data-draft={d.id} aria-current={current === d.id ? "true" : undefined} onClick={() => onOpen(d)}><bdi>{d.new_group ? d.name || d.id : label(d.group_label, d.group)}</bdi></button>
                        <span className="live-sub">{mono(d.id)}</span>
                      </td>
                      <td role="cell" data-label={o.groupLegend(l)}>{d.new_group ? o.newGroup(l) : o.existingGroup(l)}<span className="live-sub">{mono(d.group)}</span></td>
                      <td role="cell" data-label={o.colMembers(l)}>{d.member_count}</td>
                      <td role="cell" data-label={o.colState(l)}>{stateText(l, d.state)}{note && <span className="live-sub">{note === "rejected" ? o.lastRejected(l) : o.lastDiscarded(l)}</span>}</td>
                      <td role="cell" data-label={o.colPlan(l)}>{d.plan ? <PlanRef l={l} id={d.plan.id} lang={lang} /> : o.none(l)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      {list.status === "ready" && list.more === "failed" && <div className="o01-more" role="alert"><p>{o.nextFailed(l)}</p><button type="button" className="live-secondary o01-more-button" onClick={list.loadMore}>{o.tryAgain(l)}</button></div>}
      {list.status === "ready" && list.next && list.more !== "failed" && (
        <div className="o01-more"><button type="button" className="live-secondary o01-more-button" aria-disabled={list.more === "more" || undefined} onClick={list.loadMore}>{list.more === "more" ? o.loadingMore(l) : o.loadMore(l)}</button></div>
      )}
    </section>
  );
}

type Target = { kind: "new" } | { kind: "draft"; draft: GroupDraft } | { kind: "gone"; id: string } | { kind: "unread"; id: string };
type Errors = Partial<Record<"name" | "group" | "members" | "ids" | "reason", string>>;
type Partial2 = { done: number; total: number; rest: number; reason: string; kind: "new" | "edit"; draftId: string };
const fieldIds: Record<keyof Errors, string> = { name: "o01-draft-name", group: "o01-group-new", members: "o01-draft-members", ids: "o01-draft-ids", reason: "live-reason" };
const memberPage = 99;
const lookupPage = 50;

export type Editor = ReturnType<typeof useDraftEditor>;

export function useDraftEditor({ api, l, list, onDirty, say, focus }: {
  api: AdminClient; l: Locale; list: DraftList; onDirty?: (d: boolean) => void; say: (t: string) => void; focus: (sel: string) => void;
}) {
  const [target, setTarget] = useState<Target | null>(null);
  const [start, setStart] = useState<Snapshot>({ name: "", mode: "new", group: "", adds: [], removes: [] });
  const [name, setName] = useState("");
  const [mode, setMode] = useState<"new" | "existing">("new");
  const [group, setGroup] = useState<{ id: string; label: string } | null>(null);
  const [adds, setAdds] = useState<DraftMember[]>([]);
  const [removes, setRemoves] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [errors, setErrors] = useState<Errors>({});
  const [summaryKey, setSummaryKey] = useState(0);
  const [progress, setProgress] = useState<{ done: number; total: number; changes: boolean } | null>(null);
  const [partial, setPartial] = useState<Partial2 | null>(null);
  const [notice, setNotice] = useState<{ text: string; alert: boolean } | null>(null);
  const [confirm, setConfirm] = useState(false);
  const [discarding, setDiscarding] = useState(false);
  const [leave, setLeave] = useState<(() => void) | null>(null);
  const [members, setMembers] = useState<{ rows: DraftMember[]; next: string; status: "idle" | "loading" | "ready" | "error" }>({ rows: [], next: "", status: "idle" });
  const lastEdited = useRef("");
  const generation = useRef(0);
  const memberSeq = useRef(0);
  const checkedInfo = useRef("");
  const now: Snapshot = { name, mode, group: group?.id ?? "", adds: adds.map((a) => a.id), removes };
  const isDirty = !!target && target.kind !== "gone" && dirty(start, now);
  useEffect(() => { onDirty?.(isDirty); }, [isDirty]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => () => onDirty?.(false), []); // eslint-disable-line react-hooks/exhaustive-deps
  const draft = target?.kind === "draft" ? target.draft : null;
  const count = draft ? draft.member_count - removes.length + adds.length : adds.length;
  const editing = target?.kind === "new" || draft?.state === "open";

  const loadMembers = useCallback(async (id: string, after = "") => {
    const n = ++memberSeq.current;
    setMembers((m) => ({ ...m, status: "loading" }));
    try {
      const page = await api.draftMembers(id, after, shownStep);
      if (!isLatest(n, memberSeq.current)) return;
      if (page === "missing") { setTarget({ kind: "gone", id }); return; }
      setMembers((m) => ({ rows: after ? [...m.rows, ...page.members] : page.members, next: page.next, status: "ready" }));
    } catch {
      if (isLatest(n, memberSeq.current)) setMembers((m) => ({ ...m, status: "error" }));
    }
  }, [api]);

  const reset = (t: Target | null, initial: DraftMember[] = []) => {
    generation.current += 1;
    memberSeq.current += 1;
    const d = t?.kind === "draft" ? t.draft : null;
    const snap: Snapshot = { name: d?.name ?? "", mode: d && !d.new_group ? "existing" : "new", group: d && !d.new_group ? d.group : "", adds: [], removes: [] };
    setTarget(t);
    setStart(snap);
    setName(snap.name);
    setMode(snap.mode);
    setGroup(d && !d.new_group ? { id: d.group, label: d.group_label } : null);
    setAdds(initial);
    setRemoves([]);
    setErrors({});
    setProgress(null);
    setPartial(null);
    setNotice(null);
    setConfirm(false);
    setLeave(null);
    setMembers({ rows: [], next: "", status: "idle" });
    if (d && !d.pruned) void loadMembers(d.id);
  };

  const guarded = (action: () => void) => {
    if (!isDirty) { action(); return; }
    setLeave(() => action);
    focus("#o01-leave");
  };
  const discardInput = () => {
    const pending = leave;
    flushSync(() => { setLeave(null); setAdds([]); setRemoves([]); setName(start.name); setMode(start.mode); onDirty?.(false); });
    pending?.();
  };
  const keepEditing = () => { setLeave(null); focus(lastEdited.current || "#o01-draft-title"); };

  const openNew = (initial: DraftMember[] = []) => guarded(() => { reset({ kind: "new" }, initial); focus("#o01-draft-title"); });
  const openDraft = (d: GroupDraft) => guarded(() => { reset({ kind: "draft", draft: d }); focus("#o01-draft-title"); });
  const close = () => guarded(() => reset(null));
  const touch = (sel: string) => { lastEdited.current = sel; };

  const addRows = (rows: DraftMember[]) => {
    const existing = new Set(draft ? members.rows.map((m) => m.id) : []);
    const got = addPending(adds, rows, existing, list.maxMembers - (draft ? draft.member_count - removes.length : 0) + existing.size);
    setAdds(got.list);
    setErrors((e) => ({ ...e, members: undefined }));
    return got;
  };

  const fresh = async (): Promise<string> => (await api.viewGroupDrafts("", 1)).revision;

  const refreshAfterStage = async (id: string, dropped: number, cause: "conflict" | "stage" | "denied") => {
    let found: GroupDraft | null = null;
    let read: "found" | "missing" | "denied" | "signin" | "failed" = "missing";
    try {
      found = await list.find(id);
      read = found ? "found" : "missing";
    } catch (e) {
      read = readOutcome(e instanceof AdminError ? e.failure : "");
    }
    const next = afterRefusal(read, cause, found?.state === "open");
    if (next.view === "gone") { await list.reload(); setTarget({ kind: "gone", id }); return; }
    if (next.view !== "reloaded") {
      setNotice({ text: next.view === "denied" ? o.denied(l) : next.view === "signin" ? w.signInAgain(l) : o.saveFailed(l), alert: true });
      focus("#o01-draft-notice");
      return;
    }
    await list.reload();
    if (!found) return;
    setTarget({ kind: "draft", draft: found });
    void loadMembers(found.id);
    if (next.notice !== "draftChangedClosed") {
      setNotice({ text: next.notice === "dataChanged" ? o.dataChanged(l) : o.draftChanged(l), alert: true });
      return;
    }
    setAdds([]);
    setRemoves([]);
    setName(found.name);
    setStart({ name: found.name, mode: found.new_group ? "new" : "existing", group: found.new_group ? "" : found.group, adds: [], removes: [] });
    setNotice({ text: o.draftChangedClosed(l, dropped), alert: true });
  };

  const refused = (e: unknown, fieldOf: (f: string) => keyof Errors | null, id = "") => {
    const err = e instanceof AdminError ? e : null;
    const kind = refusalOf(err?.failure ?? "", err?.code ?? "");
    if (kind === "field") {
      const found: Errors = {};
      for (const f of err?.fields ?? []) {
        const k = fieldOf(f);
        if (k === "name") found.name = o.eName(l);
        else if (k === "group") found.group = o.eGroup(l);
        else if (k === "members") found.members = o.eMembersServer(l);
      }
      if (Object.keys(found).length === 0) found.reason = cc.eServer(l);
      setErrors(found);
      setSummaryKey((k) => k + 1);
      return;
    }
    if (kind === "missing") { setTarget({ kind: "gone", id }); return; }
    if (id && (kind === "stage" || kind === "conflict" || kind === "denied")) {
      void refreshAfterStage(id, adds.length + removes.length, kind);
      focus("#o01-draft-notice");
      return;
    }
    const text = kind === "limit" ? (list.limit && list.open >= list.limit ? o.openLimit(l, list.limit) : o.capRefused(l, list.maxMembers))
      : kind === "conflict" ? o[conflictCopy(!!id)](l) : kind === "dependency" ? o.depFailed(l) : kind === "unknown" ? o.outcomeUnknown(l) : o.saveFailed(l);
    setNotice({ text, alert: true });
    focus("#o01-draft-notice");
  };

  const validate = (): Errors => {
    const e: Errors = {};
    if (target?.kind === "new" && mode === "new" && !validName(name)) e.name = o.eName(l);
    if (draft?.new_group && name !== start.name && !validName(name)) e.name = o.eName(l);
    if (target?.kind === "new" && mode === "existing" && !group) e.group = o.eGroup(l);
    if (target?.kind === "new" && adds.length === 0 && !partial) e.members = o.eMembers(l);
    if (draft && count < 1) e.members = o.eRemoveAll(l);
    const r = reasonProblem(l, reason);
    if (r) e.reason = r;
    return e;
  };

  const precheck = async (pending: DraftMember[], draftId: string): Promise<DraftMember[] | null> => {
    try {
      const saved = new Set<string>();
      for (let after = "", i = 0; draftId && i < 200; i++) {
        const page = await api.draftMembers(draftId, after, memberPage);
        if (page === "missing") break;
        for (const m of page.members) saved.add(m.id);
        if (!page.next) break;
        after = page.next;
      }
      const known = new Set<string>();
      for (const part of chunks(pending.filter((m) => !saved.has(m.id)).map((m) => m.id), lookupPage)) {
        const page = await api.organizations({ kind: "view_group", subject: "account", member: part });
        for (const m of page.members) known.add(m.member);
      }
      const { keep, already, strangers } = partitionPrecheck(pending, saved, known);
      const lines = [already > 0 ? o.alreadyInDraft(l, already) : "", strangers.length > 0 ? o.notAccount(l, strangers.join(", ")) : ""].filter(Boolean);
      checkedInfo.current = lines.join(" ");
      if (lines.length) { setNotice({ text: checkedInfo.current, alert: false }); setAdds(keep); }
      return keep;
    } catch {
      setNotice({ text: o.depFailed(l), alert: true });
      focus("#o01-draft-notice");
      return null;
    }
  };

  const finish = async (id: string, text: string, mine: number) => {
    let found: GroupDraft | null = null;
    let unread = false;
    try { found = (await list.find(id)) ?? null; } catch { unread = true; }
    await list.reload();
    if (!sameView(mine, generation.current)) { say(text); return; }
    if (found) reset({ kind: "draft", draft: found });
    else reset(unread ? { kind: "unread", id } : { kind: "gone", id });
    setReason("");
    if (checkedInfo.current) setNotice({ text: checkedInfo.current, alert: false });
    flushSync(() => onDirty?.(false));
    say([text, checkedInfo.current].filter(Boolean).join(" "));
    checkedInfo.current = "";
    focus("#o01-draft-title");
  };

  const saveNew = async (pending: DraftMember[], created: string, done: number, total: number) => {
    const mine = generation.current;
    const batches = chunks(pending.map((a) => a.id));
    let id = created;
    let saved = done;
    let rev = "";
    setProgress({ done: saved, total, changes: false });
    for (let i = 0; i < batches.length; i++) {
      try {
        rev = callRevision(i, !!created, list.revision) ?? (await fresh());
        if (!id) {
          const values: Record<string, string> = { members: batches[i].join("\n") };
          if (mode === "existing" && group) values.group = group.id;
          else values.name = name.trim();
          const out = await api.command("view_group.draft", "view-group-drafts", rev, reason, values, attempt());
          id = out.fields.draft ?? "";
          if (!id) {
            setProgress(null);
            await list.reload();
            reset({ kind: "unread", id: "" });
            setReason("");
            flushSync(() => onDirty?.(false));
            say(o.draftUnread(l));
            focus("#o01-draft-title");
            return;
          }
        } else {
          await api.command("view_group.edit", id, rev, reason, { add: batches[i].join("\n") }, attempt());
        }
        saved += batches[i].length;
        setProgress({ done: saved, total, changes: false });
        say(o.saving(l, saved, total));
      } catch (e) {
        setProgress(null);
        if (!id) { refused(e, (f) => (f === "name" || f === "group" || f === "members" ? f : null)); return; }
        const rest = pending.slice(i * 31);
        const found = await list.find(id).catch(() => null);
        await list.reload();
        const err = e instanceof AdminError ? e : null;
        const why = err?.failure === "invalid" ? o.eMembersServer(l) : err?.failure === "busy" ? o.partialBusy(l) : refusalOf(err?.failure ?? "", err?.code ?? "") === "unknown" ? o.partialUnknown(l) : o.partialFailed(l);
        if (found) {
          setTarget({ kind: "draft", draft: found });
          setStart({ name: found.name, mode: found.new_group ? "new" : "existing", group: found.new_group ? "" : found.group, adds: [], removes: [] });
          void loadMembers(id);
        }
        setAdds(rest);
        setPartial({ done: saved, total, rest: rest.length, reason: why, kind: "new", draftId: id });
        focus("#o01-draft-partial");
        return;
      }
    }
    setProgress(null);
    await finish(id, o.draftSaved(l, total), mine);
  };

  const saveEdit = async (d: GroupDraft, pending: DraftMember[]) => {
    const mine = generation.current;
    const renamed = d.new_group && name.trim() !== start.name.trim();
    const calls = editCalls(d.member_count, pending.map((a) => a.id), removes, list.maxMembers, renamed);
    if (!calls) { setErrors({ members: d.member_count + pending.length - removes.length < 1 ? o.eRemoveAll(l) : o.capRefused(l, list.maxMembers) }); setSummaryKey((k) => k + 1); return; }
    if (calls.length === 0) return;
    const total = pending.length + removes.length;
    let done = 0;
    let rev = "";
    setProgress({ done, total, changes: true });
    for (let i = 0; i < calls.length; i++) {
      const call = calls[i];
      const values: Record<string, string> = {};
      if (call.name) values.name = name.trim();
      if (call.add.length) values.add = call.add.join("\n");
      if (call.remove.length) values.remove = call.remove.join("\n");
      try {
        rev = callRevision(i, false, list.revision) ?? (await fresh());
        await api.command("view_group.edit", d.id, rev, reason, values, attempt());
        done += call.add.length + call.remove.length;
        setProgress({ done, total, changes: true });
        say(o.savingChanges(l, done, total));
      } catch (e) {
        setProgress(null);
        if (i === 0) { refused(e, (f) => (f === "name" ? "name" : f === "add" || f === "remove" ? "members" : null), d.id); return; }
        const doneAdds = new Set(calls.slice(0, i).flatMap((x) => x.add));
        const doneRemoves = new Set(calls.slice(0, i).flatMap((x) => x.remove));
        const found = await list.find(d.id).catch(() => null);
        if (found) { setTarget({ kind: "draft", draft: found }); void loadMembers(found.id); }
        setAdds((a) => a.filter((x) => !doneAdds.has(x.id)));
        setRemoves((r) => r.filter((x) => !doneRemoves.has(x)));
        if (calls[0].name) setStart((s) => ({ ...s, name: name.trim() }));
        setPartial({ done, total, rest: total - done, reason: e instanceof AdminError && e.failure === "busy" ? o.partialBusy(l) : refusalOf(e instanceof AdminError ? e.failure : "", e instanceof AdminError ? e.code : "") === "unknown" ? o.partialUnknown(l) : o.partialFailed(l), kind: "edit", draftId: d.id });
        focus("#o01-draft-partial");
        return;
      }
    }
    setProgress(null);
    await finish(d.id, o.changesSaved(l), mine);
  };

  const save = async () => {
    if (progress) return;
    const found = validate();
    setErrors(found);
    if (Object.keys(found).length) { setSummaryKey((k) => k + 1); return; }
    setNotice(null);
    const plan = savePlan(partial, target?.kind ?? null, draft?.id ?? "");
    if (plan.mode === "none") return;
    const keep = adds.length ? await precheck(adds, plan.mode === "new" ? "" : plan.draftId) : adds;
    if (keep === null) return;
    if (plan.mode === "continue") { setPartial(null); await saveNew(keep, plan.draftId, plan.done, plan.done + keep.length); return; }
    if (plan.mode === "new") {
      if (keep.length === 0) { setErrors({ members: o.eMembers(l) }); setSummaryKey((k) => k + 1); return; }
      await saveNew(keep, "", 0, keep.length);
      return;
    }
    if (draft) await saveEdit({ ...draft }, keep);
  };

  const retryRest = async () => {
    if (!partial) return;
    if (partial.kind === "new") { await save(); return; }
    setPartial(null);
    if (draft) await saveEdit(draft, adds);
  };

  const discardChanges = () => {
    if (!draft) return;
    flushSync(() => { reset({ kind: "draft", draft }); onDirty?.(false); });
    focus("#o01-draft-title");
  };
  const askDiscard = () => { setConfirm(true); focus("#o01-confirm"); };
  const keepDraft = () => { setConfirm(false); focus("#o01-discard-draft"); };

  const discardDraft = async () => {
    if (!draft || discarding) return;
    const r = reasonProblem(l, reason);
    if (r) { setErrors({ reason: r }); setSummaryKey((k) => k + 1); return; }
    const mine = generation.current;
    setDiscarding(true);
    try {
      await api.command("view_group.discard_draft", draft.id, list.revision, reason, {}, attempt());
      setReason("");
      await finish(draft.id, o.draftDiscarded(l), mine);
    } catch (e) {
      setConfirm(false);
      refused(e, () => null, draft.id);
    } finally {
      setDiscarding(false);
    }
  };

  return {
    target, draft, name, setName, mode, setMode, group, setGroup, adds, setAdds, removes, setRemoves, reason, setReason, errors, setErrors, summaryKey, progress, partial,
    notice, confirm, setConfirm, discarding, leave, members, loadMembers, count, isDirty, start, editing,
    openNew, openDraft, close, guarded, drop: () => reset(null), discardInput, keepEditing, touch, addRows, save, retryRest, discardDraft, discardChanges, askDiscard, keepDraft,
  };
}

type Searcher<T> = (q: string, after: string) => Promise<{ rows: T[]; next: string }>;

function useSearch<T>(run: Searcher<T>) {
  const [q, setQ] = useState("");
  const [state, setState] = useState<{ rows: T[]; next: string; status: "idle" | "loading" | "ready" | "error" }>({ rows: [], next: "", status: "idle" });
  const seq = useRef(0);
  const go = async (term: string, after = "") => {
    const n = ++seq.current;
    setQ(term);
    setState((s) => ({ ...s, rows: after ? s.rows : [], next: after ? s.next : "", status: "loading" }));
    try {
      const got = await run(term, after);
      if (n === seq.current) setState((s) => ({ rows: after ? [...s.rows, ...got.rows] : got.rows, next: got.next, status: "ready" }));
    } catch {
      if (n === seq.current) setState((s) => ({ ...s, status: "error" }));
    }
  };
  return { q, ...state, search: (term: string) => void go(term), more: () => void go(q, state.next) };
}

function MemberRows({ l, rows, removing, onRemove, onUndo, editable }: {
  l: Locale; rows: DraftMember[]; removing: Set<string>; onRemove: (id: string) => void; onUndo: (id: string) => void; editable: boolean;
}) {
  return (
    <ul className="o01-draft-members">
      {rows.map((m) => (
        <li key={m.id}>
          <p className="o01-member-text">{m.name ? <><bdi>{m.name}</bdi> {mono(m.id)}</> : mono(m.id)}</p>
          {removing.has(m.id) && <span className="o01-will-remove">{o.willRemove(l)}</span>}
          {editable && (removing.has(m.id)
            ? <button type="button" className="service-button-link" data-undo={m.id} onClick={() => onUndo(m.id)}>{o.undo(l)}</button>
            : <button type="button" className="service-button-link" data-remove={m.id} onClick={() => onRemove(m.id)}>{o.removeFromDraft(l)}</button>)}
        </li>
      ))}
    </ul>
  );
}

function AddMembers({ l, e, searchAccounts, max, inDraft }: { l: Locale; e: Editor; searchAccounts: Searcher<OrgMember>; max: number; inDraft: Set<string> }) {
  const results = useSearch(searchAccounts);
  const [ids, setIds] = useState("");
  const [entering, setEntering] = useState(false);
  const [shown, setShown] = useState(shownStep);
  const [said, setSaid] = useState("");
  const pending = new Set(e.adds.map((a) => a.id));
  const full = e.count >= max;
  const addIds = () => {
    const parsed = parseIds(ids);
    if (parsed.bad.length) { e.setErrors({ ...e.errors, ids: o.badIds(l, parsed.bad.join(", ")) }); return; }
    const got = e.addRows(parsed.ids.map((id) => ({ id, name: "" })));
    e.touch("#o01-draft-ids");
    e.setErrors({ ...e.errors, ids: undefined });
    setSaid([o.idsAdded(l, got.added, got.already), got.over > 0 ? o.overCap(l, got.over, max) : ""].filter(Boolean).join(" "));
    setIds("");
  };
  return (
    <section className="o01-draft-add" aria-labelledby="o01-add-title">
      <h3 id="o01-add-title">{o.membersToAdd(l)}</h3>
      <p className="live-meta o01-add-meta" id="o01-draft-members" tabIndex={-1}>{e.draft ? o.inDraftMeta(l, e.count, max) : o.toAddMeta(l, e.count, max)}</p>
      {full ? <p className="live-note">{o.draftFull(l, max)}</p> : (
        <>
          <SearchForm l={l} id="o01-draft-q" labelText={o.searchAccounts(l)} value={results.q} onSearch={results.search} />
          {results.status === "loading" && <p role="status" className="live-meta">{o.searchingMembers(l)}</p>}
          {results.status === "error" && <p className="live-error" role="alert">{o.failed(l)}</p>}
          {results.status === "ready" && results.rows.length === 0 && <p className="live-empty">{o.noResults(l)}</p>}
          {results.rows.length > 0 && (
            <ul className="o01-draft-results">
              {results.rows.map((m) => (
                <li key={m.member}>
                  <p className="o01-member-text">{m.name ? <><bdi>{m.name}</bdi> {mono(m.member)}</> : mono(m.member)}</p>
                  {pending.has(m.member) || inDraft.has(m.member) ? <span className="live-quiet">{o.added(l)}</span>
                    : <button type="button" className="live-secondary o01-bar-action" data-add={m.member} onClick={() => { e.addRows([{ id: m.member, name: m.name }]); e.touch(`[data-add]`); }}>{o.add(l)}</button>}
                </li>
              ))}
            </ul>
          )}
          {results.next && <div className="o01-more"><button type="button" className="live-secondary o01-more-button" onClick={results.more}>{o.loadMore(l)}</button></div>}
          {!entering ? <button type="button" className="service-button-link o01-enter-ids" onClick={() => setEntering(true)}>{o.enterIds(l)}</button> : (
            <div className="service-field o01-ids">
              <label htmlFor="o01-draft-ids">{o.idsLabel(l)}</label>
              <p className="service-hint" id="o01-draft-ids-hint">{o.idsHint(l)}</p>
              {e.errors.ids && <p className="service-field-error" id="o01-draft-ids-error">{e.errors.ids}</p>}
              <textarea id="o01-draft-ids" dir="ltr" rows={6} value={ids} aria-invalid={e.errors.ids ? true : undefined}
                aria-describedby={["o01-draft-ids-hint", e.errors.ids ? "o01-draft-ids-error" : ""].filter(Boolean).join(" ")} onChange={(ev) => setIds(ev.target.value)} />
              <div className="live-buttons"><button type="button" className="live-secondary o01-bar-action" onClick={addIds}>{o.addIds(l)}</button></div>
              {said && <p className="live-meta" role="status">{said}</p>}
            </div>
          )}
        </>
      )}
      {e.adds.length > 0 && (
        <>
          <MemberRows l={l} rows={e.adds.slice(0, shown)} removing={new Set()} editable onRemove={(id) => { e.setAdds(e.adds.filter((a) => a.id !== id)); e.touch(""); }} onUndo={() => undefined} />
          {e.adds.length > shown && <button type="button" className="service-button-link" onClick={() => setShown((s) => s + shownStep)}>{o.showMore(l)}</button>}
        </>
      )}
    </section>
  );
}

function StateBlock({ l, d, lang }: { l: Locale; d: GroupDraft; lang: string }) {
  const b = block(d);
  if (!b) return null;
  const p = d.plan ? <a href={planHref(d.plan.id, lang)} className="o01-planref">{mono(d.plan.id)}</a> : <>{o.none(l)}</>;
  const ruleText = (code: string) => { const [title, body] = cc.rule(l, code); return body || (/[.!?。]$/.test(title) ? title : title + "."); };
  const rule = d.plan?.rule === "plan_limit" ? o.planLimitCause(l) : d.plan?.rule ? ruleText(d.plan.rule) : "";
  const text = b === "rejected" ? (rule ? fill(o.bRejected(l).replace("{s}", rule), { p }) : fill(o.bRejectedBare(l), { p }))
    : b === "planDiscarded" ? fill(o.bPlanDiscarded(l), { p })
      : b === "held" ? fill(o.bHeld(l), { p })
        : b === "openPlan" ? fill(o.bOpenPlan(l), { p })
          : b === "activated" ? fill(o.bActivated(l), { r: <bdi dir="ltr">{d.activated_revision}</bdi>, p })
            : fill(o.bDiscarded(l), { d: <time dateTime={d.discarded}>{dayText(l, d.discarded)}</time> });
  return (
    <div className="live-note-rule o01-draft-state">
      <p>{text}</p>
      {b === "activated" && d.pruned && <p>{o.bPruned(l)}</p>}
    </div>
  );
}

export function DraftPanel({ l, e, list, lang, on, today, base, searchAccounts, searchGroups, onBack }: {
  l: Locale; e: Editor; list: DraftList; lang: string; on: string; today: string; base: number; searchAccounts: Searcher<OrgMember>; searchGroups: Searcher<OrgTargetRow>; onBack?: ReactNode;
}) {
  const groups = useSearch(searchGroups);
  if (!e.target) return null;
  if (e.target.kind === "unread") {
    return (
      <section className="o01-draft" aria-labelledby="o01-draft-title">
        <h2 id="o01-draft-title" tabIndex={-1}>{o.drafts(l)}</h2>
        <p className="live-note-rule" role="alert">{o.draftUnread(l)}</p>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={() => e.close()}>{o.back(l)}</button></div>
      </section>
    );
  }
  if (e.target.kind === "gone") {
    return (
      <section className="o01-draft" aria-labelledby="o01-draft-title">
        <h2 id="o01-draft-title" tabIndex={-1}>{o.drafts(l)}</h2>
        <p className="live-note-rule">{o.draftGone(l)}</p>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={() => e.close()}>{o.back(l)}</button></div>
      </section>
    );
  }
  const d = e.draft;
  const isNew = e.target.kind === "new";
  const can = d ? allowed(d) : { edit: true, plan: false, discard: false, openPlan: false };
  const max = list.maxMembers;
  const removing = new Set(e.removes);
  const inDraft = new Set(e.members.rows.map((m) => m.id));
  const problems: Problems = (["name", "group", "members", "reason"] as const).filter((f) => e.errors[f]).map((f) => ({ field: fieldIds[f], message: e.errors[f] as string }));
  const saving = !!e.progress;
  const title = isNew ? o.newDraft(l) : d ? (d.new_group ? d.name || d.group : label(d.group_label, d.group)) : "";
  const renamed = !!d && d.new_group && e.name.trim() !== e.start.name.trim();
  const leaveBox = e.leave && (
    <div className="live-note-rule o01-leave" id="o01-leave" tabIndex={-1}>
      <p><strong>{o.discardInput(l)}</strong></p>
      <div className="live-buttons">
        <button type="button" className="live-secondary" onClick={e.discardInput}>{o.discardWord(l)}</button>
        <button type="button" className="live-secondary" onClick={e.keepEditing}>{o.keepEditing(l)}</button>
      </div>
    </div>
  );
  const nameField = (
    <div className="service-field o01-draft-name">
      <label htmlFor="o01-draft-name">{o.draftNameLabel(l)}</label>
      <p className="service-hint" id="o01-draft-name-hint">{o.nameHint(l)}</p>
      {e.errors.name && <p className="service-field-error" id="o01-draft-name-error">{e.errors.name}</p>}
      <input id="o01-draft-name" dir="auto" maxLength={80} value={e.name} aria-invalid={e.errors.name ? true : undefined}
        aria-describedby={["o01-draft-name-hint", e.errors.name ? "o01-draft-name-error" : ""].filter(Boolean).join(" ")}
        onChange={(ev) => { e.setName(ev.target.value); e.touch("#o01-draft-name"); }} />
    </div>
  );
  return (
    <section className="o01-draft" aria-labelledby="o01-draft-title">
      {onBack}
      <h2 id="o01-draft-title" tabIndex={-1}><bdi>{title}</bdi></h2>
      {d && <p className="live-meta">{o.draftSub(l, stateText(l, d.state))}</p>}
      {leaveBox}
      <div className="live-note-rule o01-not-ownership"><p>{o.notOwnership(l)}</p></div>
      {problems.length > 0 && <Summary locale={l} title={sc.check(l)} problems={problems} focusKey={e.summaryKey} />}
      {e.notice && <div className={e.notice.alert ? "live-error" : "service-status"} id="o01-draft-notice" tabIndex={-1} role={e.notice.alert ? "alert" : "status"}><p>{e.notice.text}</p></div>}
      {e.partial && (
        <div className="live-error o01-partial" id="o01-draft-partial" tabIndex={-1} role="alert">
          <p>{e.partial.kind === "edit" ? o.partialSaveChanges(l, e.partial.done, e.partial.total, e.partial.rest) : o.partialSave(l, e.partial.done, e.partial.total, e.partial.rest)}</p>
          <p>{e.partial.reason}</p>
          <button type="button" className="live-secondary" onClick={() => void e.retryRest()}>{o.retryRest(l, e.partial.rest)}</button>
        </div>
      )}
      {d && <StateBlock l={l} d={d} lang={lang} />}
      {d && (
        <dl className="live-facts">
          {d.new_group && <div><dt>{o.draftNameLabel(l)}</dt><dd><bdi>{d.name || o.none(l)}</bdi></dd></div>}
          <div><dt>{o.groupLegend(l)}</dt><dd>{d.new_group ? o.newGroup(l) : <bdi>{label(d.group_label, d.group)}</bdi>} {mono(d.group)}</dd></div>
          <div><dt>{o.colMembers(l)}</dt><dd>{o.membersOf(l, d.member_count, max)}</dd></div>
          <div><dt>{o.colState(l)}</dt><dd>{stateText(l, d.state)}</dd></div>
          <div><dt>{o.fCreated(l)}</dt><dd><bdi>{d.author_login || d.author}</bdi><br /><time dateTime={d.created}>{stampText(l, d.created)}</time></dd></div>
          <div><dt>{o.fChanged(l)}</dt><dd><time dateTime={d.updated}>{stampText(l, d.updated)}</time></dd></div>
          <div><dt>{o.colPlan(l)}</dt><dd>{d.plan ? <PlanRef l={l} id={d.plan.id} lang={lang} /> : o.none(l)}</dd></div>
          {d.state === "activated" && <div><dt>{o.fActivatedIn(l)}</dt><dd><bdi dir="ltr">{d.activated_revision}</bdi></dd></div>}
        </dl>
      )}
      {isNew && (
        <fieldset className="live-roles o01-group-mode">
          <legend>{o.groupLegend(l)}</legend>
          {e.errors.group && <p className="service-field-error" id="o01-group-error">{e.errors.group}</p>}
          <label><input id="o01-group-new" type="radio" name="o01-group-mode" checked={e.mode === "new"} onChange={() => { e.setMode("new"); e.touch("#o01-group-new"); }} />{o.newGroup(l)}</label>
          <label><input id="o01-group-existing" type="radio" name="o01-group-mode" checked={e.mode === "existing"} onChange={() => { e.setMode("existing"); e.touch("#o01-group-existing"); }} />{o.existingGroup(l)}</label>
        </fieldset>
      )}
      {isNew && e.mode === "new" && nameField}
      {isNew && e.mode === "existing" && (e.group ? (
        <p className="o01-member-chosen"><span className="o01-provider-label">{o.groupChosen(l)}</span> <bdi>{label(e.group.label, e.group.id)}</bdi> {mono(e.group.id)}{" "}
          <button type="button" className="service-button-link" onClick={() => e.setGroup(null)}>{o.change(l)}</button></p>
      ) : (
        <div className="o01-group-pick">
          <SearchForm l={l} id="o01-group-q" labelText={o.searchGroups(l)} value={groups.q} onSearch={groups.search} />
          {groups.status === "loading" && <p role="status" className="live-meta">{o.loadingMore(l)}</p>}
          {groups.status === "error" && <p className="live-error" role="alert">{o.failed(l)}</p>}
          {groups.status === "ready" && groups.rows.length === 0 && <p className="live-empty">{o.noResults(l)}</p>}
          <ul className="o01-plain o01-results">{groups.rows.slice(0, 10).map((g) => (
            <li key={g.id}><button type="button" className="live-open" data-group={g.id} onClick={() => { e.setGroup({ id: g.id, label: g.name }); e.touch("#o01-group-existing"); }}><bdi>{label(g.name, g.id)}</bdi> {mono(g.id)}</button></li>
          ))}</ul>
        </div>
      ))}
      {d && d.new_group && can.edit && nameField}
      {d && !d.pruned && (
        <section className="o01-draft-current" aria-labelledby="o01-members-title">
          <h3 id="o01-members-title">{o.groupMembers(l)}</h3>
          <p className="live-meta">{o.inDraft(l, d.member_count)}</p>
          {e.members.status === "loading" && e.members.rows.length === 0 && <p role="status" className="live-meta">{o.loadingMembers(l, dayText(l, today))}</p>}
          {e.members.status === "error" && <div className="live-error" role="alert"><p>{o.failed(l)}</p><button type="button" className="live-secondary" onClick={() => void e.loadMembers(d.id)}>{o.tryAgain(l)}</button></div>}
          <MemberRows l={l} rows={e.members.rows} removing={removing} editable={can.edit} onRemove={(id) => { e.setRemoves([...e.removes, id]); e.touch(`[data-undo="${id}"]`); }}
            onUndo={(id) => { e.setRemoves(e.removes.filter((x) => x !== id)); e.touch(`[data-remove="${id}"]`); }} />
          {e.members.next && <button type="button" className="service-button-link" onClick={() => void e.loadMembers(d.id, e.members.next)}>{o.showMore(l)}</button>}
        </section>
      )}
      {(isNew || can.edit) && (
        <>
          <AddMembers l={l} e={e} searchAccounts={searchAccounts} max={max} inDraft={inDraft} />
          <Reason l={l} value={e.reason} onChange={(v) => { e.setReason(v); e.touch("#live-reason"); }} error={e.errors.reason ?? ""} />
        </>
      )}
      {isNew && <p className="live-meta o01-batch">{o.batchHint(l)}</p>}
      {e.progress && <p className="o01-draft-progress" role="status">{e.progress.changes ? o.savingChanges(l, e.progress.done, e.progress.total) : o.saving(l, e.progress.done, e.progress.total)}</p>}
      {isNew && (
        <div className="live-buttons">
          <button type="button" className="live-primary" aria-disabled={saving || undefined} onClick={() => void e.save()}>{saving ? o.savingButton(l) : o.saveDraft(l)}</button>
          {!saving && <button type="button" className="live-secondary" onClick={() => e.close()}>{o.cancel(l)}</button>}
        </div>
      )}
      {d && can.edit && e.isDirty && (
        <div className="o01-unsaved">
          <p><strong>{o.pendingLead(l)}</strong> {o.pendingSummary(l, e.adds.length, e.removes.length, renamed)}</p>
          <div className="live-buttons">
            <button type="button" className="live-primary" aria-disabled={saving || undefined} onClick={() => void e.save()}>{saving ? o.savingButton(l) : o.saveChanges(l)}</button>
            {!saving && <button type="button" className="live-secondary" onClick={e.discardChanges}>{o.discardChanges(l)}</button>}
          </div>
        </div>
      )}
      {d && can.plan && !e.isDirty && !e.confirm && (
        <div className="live-buttons o01-draft-actions">
          <a id="o01-draft-plan" className="live-secondary o01-plan-link" href={draftHref(d.id, on, today, base, lang)}>{o.createPlan(l)}</a>
          <button type="button" id="o01-discard-draft" className="live-secondary" onClick={e.askDiscard}>{o.discardDraft(l)}</button>
        </div>
      )}
      {d && can.edit && e.isDirty && <p className="live-meta">{o.saveFirst(l)}</p>}
      {d && e.confirm && (
        <div className="live-note-rule o01-confirm" id="o01-confirm" tabIndex={-1}>
          <p><strong>{o.discardQ(l)}</strong> {o.discardEffect(l)}</p>
          <div className="live-buttons">
            <button type="button" className="live-secondary" aria-disabled={e.discarding || undefined} onClick={() => void e.discardDraft()}>{e.discarding ? o.savingButton(l) : o.discardDraft(l)}</button>
            <button type="button" className="live-secondary" onClick={e.keepDraft}>{o.keepDraft(l)}</button>
          </div>
        </div>
      )}
      {d && can.openPlan && d.plan && <div className="live-buttons"><a className="live-secondary o01-plan-link" href={planHref(d.plan.id, lang)}>{o.openPlan(l)}</a></div>}
    </section>
  );
}
