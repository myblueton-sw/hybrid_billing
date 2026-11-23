import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type OrgMember, type OrgPage, type OrgParty, type OrgProfile, type OrgQuery, type OrgTargetRow, type RelationKind, type TargetsPage, type TargetsQuery } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { useWide } from "../../../identity/pages/users/Live";
import { w } from "../../../identity/pages/users/live-copy";
import { o } from "./live-copy";
import {
  accountRows, appended, appliedKeys, applies, cleared, clearedAll, countOf, covering, filterParams, FilterRefused, firstPage, joinKinds, label, legacyKind, linkedId, loaded, narrowing,
  newer, nextFocus, pinnedRead, readFilters, refusal, todayUTC, tree, validDay, type AccountRow, type FilterKey, type Filters, type Paged, type Progress, type Side, type TreeNode,
} from "./live-model";
import { countText, FilterBand, FilteredEmpty, MoreBar, SearchForm } from "./filters";
import { fill, handoffHref } from "../changes/live-model";
import { DraftPanel, DraftsSection, HeaderCheck, RowCheck, SelectionBar, useDraftEditor, useDraftList } from "./drafts";
import { allShown, header, toggled } from "./drafts-model";
import "../../../identity/pages/users/live.css";
import "./live.css";

const maxPages = 10;
const pageSize = 99;
const listPage = 50;
const searchPage = 10;

type Loaded = { members: OrgMember[]; next: string; complete: boolean; failed: boolean };
const progressOf = (c: Loaded | undefined): Progress => ({ complete: !!c && c.complete && !c.failed, cursor: c && c.members.length ? c.members[c.members.length - 1].member : "" });
type Base = { parties: OrgParty[]; profiles: OrgProfile[]; workspace: { id: string; profile: string }; revision: number; active: number };

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
function day(l: Locale, iso: string): string {
  if (!iso) return "";
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeZone: "UTC" }).format(new Date(iso + "T00:00:00Z"));
}

const ltr = (s: string) => <bdi dir="ltr">{s}</bdi>;

function Dated({ l, iso }: { l: Locale; iso: string }) {
  return <time dateTime={iso}>{day(l, iso)}</time>;
}

export type PlanTarget = { kind: RelationKind; member: string; base: number };

export function PlanActions({ l, plan, on, today, lang, children }: { l: Locale; plan: PlanTarget | null; on: string; today: string; lang: string; children?: ReactNode }) {
  if (!plan && !children) return null;
  const from = plan && on >= today ? on : "";
  const r = plan && <bdi dir="ltr">{plan.base}</bdi>;
  const kind = plan ? o.kind(l, plan.kind) : "";
  return (
    <>
      <div className="live-buttons">
        {plan && <a id="o01-create-plan" className="live-secondary o01-plan-link" href={handoffHref({ ...plan, from, lang })} aria-describedby="o01-plan-hint">{o.createPlan(l)}</a>}
        {children}
      </div>
      {plan && (
        <p id="o01-plan-hint" className="live-note o01-plan-hint">
          {from ? fill(o.planHint(l, kind, "{r}", "{d}"), { r, d: <Dated l={l} iso={from} /> }) : fill(o.planHintNoDate(l, kind, "{r}"), { r })}
        </p>
      )}
    </>
  );
}

export function planNode(nodes: TreeNode[], id: string): TreeNode | undefined {
  for (const n of nodes) {
    if (n.id === id) return n;
    const found = planNode(n.children, id);
    if (found) return found;
  }
  return undefined;
}

export const planMember = (n: TreeNode | undefined) => !!n && (n.kind === "internal_unit" || n.kind === "account");

function kindName(l: Locale, kind: string): string {
  return kind === "legal_entity" ? o.legalEntity(l) : kind === "internal_unit" ? o.internalUnit(l) : kind === "account" ? o.account(l) : o.unknown(l);
}

function Facts({ items }: { items: [string, ReactNode][] }) {
  return (
    <dl className="live-facts">
      {items.map(([k, v]) => <div key={k}><dt>{k}</dt><dd>{v}</dd></div>)}
    </dl>
  );
}

const invalidRead = (err: unknown) => err instanceof AdminError && err.failure === "invalid";

function useStore(api: AdminClient, on: string, initialPin: number | null, onPin: (revision: number | null) => void) {
  const [base, setBase] = useState<Base | null>(null);
  const [cache, setCache] = useState<Partial<Record<RelationKind, Loaded>>>({});
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [newest, setNewest] = useState(0);
  const [notice, setNotice] = useState("");
  const [inconsistent, setInconsistent] = useState(false);
  const [epoch, setEpoch] = useState(0);
  const state = useRef("");
  const pinned = useRef<number | null>(null);
  const restarted = useRef(false);
  const token = useRef(0);
  const busy = useRef(new Set<RelationKind>());
  const reset = useCallback((bump = true) => {
    token.current += 1;
    pinned.current = null;
    busy.current.clear();
    setBase(null);
    setCache({});
    setNewest(0);
    setInconsistent(false);
    setError(null);
    state.current = "";
    setStatus("loading");
    if (bump) setEpoch((e) => e + 1);
  }, []);
  const seeded = useRef<{ on: string; pin: number | null } | null>(null);
  useEffect(() => {
    restarted.current = false;
    const keep = seeded.current === null || seeded.current.on === on;
    const pin = seeded.current === null ? initialPin : seeded.current.pin;
    reset(seeded.current !== null);
    if (keep) pinned.current = pin;
    seeded.current = { on, pin: keep ? pin : null };
  }, [on, reset]); // eslint-disable-line react-hooks/exhaustive-deps
  const pin = (relationship: number, active: number) => {
    if (pinned.current === null) {
      pinned.current = relationship;
      onPin(relationship);
    }
    if (newer(active, relationship)) setNewest(active);
  };
  const absorb = (page: OrgPage) => {
    pin(page.relationship_revision, page.active_revision);
    if (state.current && state.current !== page.revision) setInconsistent(true);
    state.current = page.revision;
    setBase({ parties: page.parties, profiles: page.profiles, workspace: page.workspace, revision: page.relationship_revision, active: page.active_revision });
  };
  const l0 = useRef<Locale>("en");
  const fail = (err: unknown): boolean => {
    if (err instanceof AdminError && err.failure === "invalid" && pinned.current !== null && !restarted.current) {
      restarted.current = true;
      setNotice(o.revisionReset(l0.current, pinned.current));
      reset();
      onPin(null);
      return true;
    }
    if (err instanceof AdminError && (err.failure === "denied" || err.failure === "unauthenticated")) {
      setError(err);
      setStatus("error");
      return true;
    }
    return false;
  };
  const load = useCallback(async (kind: RelationKind, more = false) => {
    if (busy.current.has(kind)) return;
    const mine = token.current;
    busy.current.add(kind);
    let current: Loaded = more ? (cache[kind] as Loaded) : { members: [], next: "", complete: false, failed: false };
    try {
      for (let i = 0; i < maxPages; i++) {
        const page = await api.organizations({ on, kind, revision: pinned.current ?? undefined, after: current.next || undefined, limit: pageSize });
        if (mine !== token.current) return;
        absorb(page);
        current = { members: [...current.members, ...page.members], next: page.next, complete: !page.next, failed: false };
        setCache((c) => ({ ...c, [kind]: current }));
        setStatus("ready");
        if (!page.next) break;
      }
    } catch (err) {
      if (mine !== token.current) return;
      if (fail(err)) return;
      if (current.members.length > 0) setCache((c) => ({ ...c, [kind]: { ...current, failed: true } }));
      else { setError(err); setStatus("error"); }
    } finally {
      if (mine === token.current) busy.current.delete(kind);
    }
  }, [api, on, cache, reset]); // eslint-disable-line react-hooks/exhaustive-deps
  const org = useCallback(async (query: OrgQuery): Promise<OrgPage> => {
    const mine = token.current;
    const page = await pinnedRead((revision) => api.organizations({ ...query, on, revision }), pinned.current, invalidRead);
    if (mine === token.current) { absorb(page); setStatus("ready"); }
    return page;
  }, [api, on]); // eslint-disable-line react-hooks/exhaustive-deps
  const targets = useCallback(async (query: TargetsQuery): Promise<TargetsPage> => {
    const mine = token.current;
    const page = await pinnedRead((revision) => api.targets({ ...query, on, revision }), pinned.current, invalidRead);
    if (mine === token.current) { pin(page.relationship_revision, page.active_revision); setStatus("ready"); }
    return page;
  }, [api, on]); // eslint-disable-line react-hooks/exhaustive-deps
  const fresh = useCallback(() => { restarted.current = false; reset(); onPin(null); }, [reset]); // eslint-disable-line react-hooks/exhaustive-deps
  return { base, cache, status, error, newest, notice, setNotice, load, reset: fresh, l0, inconsistent, epoch, org, targets, fail };
}

type Store = ReturnType<typeof useStore>;
type Fetched<T> = { rows: T[]; next: string; stopped: boolean };
type Pager<T> = Paged<T> & { key: string | null; current: boolean; more: () => void; retry: () => void };

function usePages<T>(key: string | null, fetch: (after: string) => Promise<Fetched<T>>, onError: (err: unknown) => boolean): Pager<T> {
  const [state, setState] = useState<Paged<T> & { key: string | null }>(() => ({ ...firstPage<T>(), key: null }));
  const token = useRef(0);
  const fetchRef = useRef(fetch);
  fetchRef.current = fetch;
  const errorRef = useRef(onError);
  errorRef.current = onError;
  const run = useCallback(async (owner: string, after: string, prev: Paged<T> | null) => {
    const mine = ++token.current;
    try {
      const page = await fetchRef.current(after);
      if (mine !== token.current) return;
      setState({ ...appended(prev ?? firstPage<T>(), page.rows, page.next, page.stopped), key: owner });
    } catch (err) {
      if (mine !== token.current) return;
      if (errorRef.current(err)) return;
      setState(prev ? { ...prev, status: "moreFailed", key: owner } : { ...firstPage<T>(), status: "error", error: err, key: owner });
    }
  }, []);
  useEffect(() => {
    token.current += 1;
    if (key === null) return;
    setState({ ...firstPage<T>(), key });
    void run(key, "", null);
  }, [key, run]);
  const current = key !== null && state.key === key;
  const more = () => {
    if (!current || state.status === "more" || !state.next) return;
    const prev = { ...state, status: "ready" as const };
    setState({ ...state, status: "more" });
    void run(state.key as string, state.next, prev);
  };
  const retry = () => {
    if (key === null) return;
    setState({ ...firstPage<T>(), key });
    void run(key, "", null);
  };
  const shown: Paged<T> = current ? state : firstPage<T>();
  return { ...shown, key, current, more, retry };
}

function Toggle({ l, node, open, onToggle }: { l: Locale; node: TreeNode; open: boolean; onToggle: () => void }) {
  if (node.children.length === 0) return <span className="o01-toggle-space" aria-hidden="true" />;
  return (
    <button type="button" className="o01-toggle" aria-expanded={open} aria-controls={open ? "o01-branch-" + node.id : undefined} aria-label={open ? o.collapse(l, node.name) : o.expand(l, node.name)} onClick={onToggle}>
      <span aria-hidden="true">{open ? "▼" : "▶"}</span>
    </button>
  );
}

function TreeList({ l, nodes, depth, open, setOpen, selected, onSelect, branchId, label: named = false }: {
  l: Locale; nodes: TreeNode[]; depth: number; open: Set<string>; setOpen: (id: string) => void; selected: string; onSelect: (n: TreeNode) => void; branchId?: string; label?: boolean;
}) {
  return (
    <ul className={depth === 0 ? "o01-tree" : "o01-branch"} aria-label={depth === 0 && named ? o.treeTitle(l) : undefined} id={branchId}>
      {nodes.map((n) => (
        <li key={n.id}>
          <div className={"o01-row" + (selected === n.id ? " live-selected" : "")} style={{ ["--o01-indent" as string]: `${Math.min(depth, 4) * 24}px` }}>
            <Toggle l={l} node={n} open={open.has(n.id)} onToggle={() => setOpen(n.id)} />
            <button type="button" className={"live-open o01-name" + (n.kind === "legal_entity" ? " o01-strong" : "")} data-id={n.id} aria-current={selected === n.id ? "true" : undefined} onClick={() => onSelect(n)}>
              <bdi>{n.name}</bdi>
            </button>
            <span className="o01-kind">{kindName(l, n.kind)}{n.state && n.state !== "active" ? ` · ${o.stateSuspended(l)}` : ""}{n.source === "registry" ? ` · ${o.registryParent(l)}` : ""}</span>
            <span className="o01-id live-mono">{ltr(n.id)}</span>
          </div>
          {n.children.length > 0 && open.has(n.id) && <TreeList l={l} nodes={n.children} depth={depth + 1} open={open} setOpen={setOpen} selected={selected} onSelect={onSelect} branchId={"o01-branch-" + n.id} />}
        </li>
      ))}
    </ul>
  );
}

function all(nodes: TreeNode[], out: string[] = []): string[] {
  for (const n of nodes) {
    if (n.children.length) out.push(n.id);
    all(n.children, out);
  }
  return out;
}

function conflictLabel(l: Locale, n: number): ReactNode {
  const text = o.conflict(l, n);
  const at = text.indexOf("(");
  return at < 0 ? text : <>{text.slice(0, at)}<span className="o01-nowrap">{text.slice(at)}</span></>;
}

function sideText(l: Locale, s: Side, none: string, empty: string): ReactNode {
  if (s.state === "target") return <bdi>{label(s.target.name, s.target.id)}</bdi>;
  if (s.state === "conflict") return <><strong>{conflictLabel(l, s.targets.length)}</strong><span className="live-sub">{s.targets.map((x) => label(x.name, x.id)).join(", ")}</span></>;
  return s.state === "unattributed" ? <strong>{empty}</strong> : <span className="live-quiet">{none}</span>;
}

function useMemberFacts(store: Store, selected: string, enabled: boolean, onRefused: () => void) {
  const [facts, setFacts] = useState<{ id: string; key: string; hier: OrgMember | undefined } | null>(null);
  const [failedKey, setFailedKey] = useState("");
  const [attempt, setAttempt] = useState(0);
  const key = `${store.epoch}|${selected}`;
  useEffect(() => {
    if (!enabled || !selected) return;
    let live = true;
    store.org({ kind: "hierarchy", member: [selected] })
      .then((page) => { if (live) { setFailedKey(""); setFacts({ id: selected, key, hier: page.members[0] }); } })
      .catch((err) => {
        if (!live) return;
        if (err instanceof FilterRefused) onRefused();
        else { if (!store.fail(err)) setFacts(null); setFailedKey(key); }
      });
    return () => { live = false; };
  }, [key, enabled, attempt]); // eslint-disable-line react-hooks/exhaustive-deps
  return { facts: facts && facts.key === key ? facts : null, failed: failedKey === key, retry: () => { setFailedKey(""); setAttempt((n) => n + 1); } };
}

export default function LiveOrganizations({ locale: l, view, params, session, onSignOut, onView, onParams, onDirty }: {
  locale: Locale; view: string; params: URLSearchParams; session: Session; onSignOut: () => void;
  onView: (view: string, params?: Record<string, string>) => void; onParams: (params: Record<string, string | null>) => void; onDirty?: (dirty: boolean) => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const [on, setOn] = useState(() => (validDay(params.get("on") ?? "") ? (params.get("on") as string) : todayUTC(new Date())));
  const [draft, setDraft] = useState(on);
  const [dateError, setDateError] = useState("");
  const initialPin = /^\d+$/.test(params.get("revision") ?? "") ? Number(params.get("revision")) : null;
  const store = useStore(api, on, initialPin, (r) => onParams({ revision: r === null ? null : String(r) }));
  store.l0.current = l;
  const { base, cache, status, error, newest } = store;
  const [filters, setFilters] = useState<Filters>(() => {
    const linked = readFilters(params).filters;
    const legacy = legacyKind(params);
    return legacy ? { ...linked, rk: legacy } : linked;
  });
  const [badFilter, setBadFilter] = useState(false);
  const [selected, setSelected] = useState(() => linkedId(params, "sel").id);
  const [selectedGroup, setSelectedGroup] = useState("");
  const [stepped, setStepped] = useState(false);
  const [open, setOpenSet] = useState<Set<string>>(new Set());
  const [bandOpen, setBandOpen] = useState<Set<string>>(new Set());
  const [groupKind, setGroupKind] = useState<RelationKind>("view_group");
  const [historyMember, setHistoryMember] = useState(() => linkedId(params, "member").id);
  const [historyQ, setHistoryQ] = useState("");
  const [picking, setPicking] = useState(() => !linkedId(params, "member").id);
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const [checked, setChecked] = useState<Set<string>>(new Set());
  const [selCleared, setSelCleared] = useState(false);
  const draftList = useDraftList(api, view === "accounts" || view === "groups");
  const editor = useDraftEditor({ api, l, list: draftList, onDirty, say: setAnnounce, focus: setFocusTo });
  useEffect(() => { if (editor.leave && !wide) setStepped(true); }, [editor.leave]); // eslint-disable-line react-hooks/exhaustive-deps
  const removed = useRef(false);
  const opened = useRef(false);
  const focusHistory = useRef(params.get("from") === "link");
  useEffect(() => {
    const removedKeys: Record<string, string | null> = Object.fromEntries(readFilters(params).invalid.map((k) => [k, null]));
    for (const key of ["sel", "member"]) if (linkedId(params, key).invalid) removedKeys[key] = null;
    const legacy = legacyKind(params);
    const changes: Record<string, string | null> = legacy ? { ...removedKeys, kind: null, rk: legacy === "hierarchy" ? null : legacy } : params.has("kind") ? { ...removedKeys, kind: null } : removedKeys;
    if (Object.keys(changes).length > 0) onParams(changes);
    if (Object.keys(removedKeys).length > 0) setBadFilter(true);
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  const filterKey = ["entity", "rk", "ua", "q"].map((k) => params.get(k) ?? "").join("|");
  useEffect(() => {
    const { filters: linked, invalid } = readFilters(params);
    if (invalid.length === 0) setFilters((f) => (f.entity === linked.entity && f.rk === linked.rk && f.ua === linked.ua && f.q === linked.q ? f : linked));
  }, [filterKey]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    const member = linkedId(params, "member").id;
    if (member === historyMember) return;
    setHistoryMember(member);
    setPicking(!member);
  }, [linkedId(params, "member").id]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (view === "tree" && !cache.hierarchy) void store.load("hierarchy");
  }, [view, on, cache]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { setStepped(false); }, [view]);
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const dropLinked = (changes: Record<string, string | null>) => {
    onParams(changes);
    setBadFilter(true);
  };
  const fail = (err: unknown) => {
    const action = refusal(err, filters.entity);
    if (action === "store") return store.fail(err);
    if (action === "error") return false;
    setFilters((f) => ({ ...f, entity: "" }));
    dropLinked({ entity: null });
    return true;
  };
  const failMember = (err: unknown) => {
    const action = refusal(err, historyMember);
    if (action === "store") return store.fail(err);
    if (action === "error") return false;
    setHistoryMember("");
    setPicking(true);
    dropLinked({ member: null });
    return true;
  };
  const scope = `${store.epoch}|${on}`;
  const rk = filters.rk;
  const listKey = `${scope}|${base?.revision ?? ""}|${rk}|${filters.entity}|${filters.ua}|${filters.q}|${groupKind}`;
  const lastList = useRef(listKey);
  const clearedNote = useRef(false);
  useEffect(() => {
    if (lastList.current === listKey) return;
    lastList.current = listKey;
    setChecked((c) => {
      if (c.size === 0) return c;
      setSelCleared(true);
      clearedNote.current = true;
      setAnnounce(o.selCleared(l));
      return new Set();
    });
  }, [listKey]); // eslint-disable-line react-hooks/exhaustive-deps
  const accounts = usePages<AccountRow>(view === "accounts" ? `a|${scope}|${rk}|${filters.entity}|${filters.ua}|${filters.q}` : null, async (after) => {
    const page = await store.org({ kind: rk, subject: "account", entity: filters.entity || undefined, unattributed: filters.ua, q: filters.q || undefined, after: after || undefined, limit: listPage });
    const ids = page.members.map((m) => m.member);
    const joined: Partial<Record<RelationKind, OrgMember[]>> = {};
    for (const k of ids.length ? joinKinds(rk) : []) joined[k] = (await store.org({ kind: k, member: ids })).members;
    return { rows: accountRows(page.members, rk, joined, on), next: page.next, stopped: page.scan_stopped };
  }, fail);
  const groupList = usePages<OrgTargetRow>(view === "groups" ? `g|${scope}|${groupKind}|${filters.q}` : null, async (after) => {
    const page = await store.targets({ kind: groupKind, q: filters.q || undefined, after: after || undefined, limit: listPage });
    return { rows: page.targets, next: page.next, stopped: page.scan_stopped };
  }, fail);
  const memberFilter = { entity: filters.entity || undefined };
  const outside = usePages<OrgMember>(view === "groups" ? `o|${scope}|${groupKind}|${filters.entity}` : null, async (after) => {
    const page = await store.org({ kind: groupKind, subject: "account", unattributed: true, entity: memberFilter.entity, after: after || undefined, limit: listPage });
    return { rows: page.members, next: page.next, stopped: page.scan_stopped };
  }, fail);
  const groupMembers = usePages<OrgMember>(view === "groups" && selectedGroup ? `m|${scope}|${groupKind}|${selectedGroup}|${filters.entity}` : null, async (after) => {
    const page = await store.org({ kind: groupKind, target: selectedGroup, ...memberFilter, after: after || undefined, limit: listPage });
    return { rows: page.members, next: page.next, stopped: page.scan_stopped };
  }, fail);
  const results = usePages<OrgMember>(view === "history" && picking && historyQ ? `s|${scope}|${rk}|${historyQ}` : null, async (after) => {
    const page = await store.org({ kind: rk, q: historyQ, after: after || undefined, limit: searchPage });
    return { rows: page.members, next: page.next, stopped: page.scan_stopped };
  }, fail);
  const timeline = usePages<OrgMember>(view === "history" && historyMember ? `t|${scope}|${rk}|${historyMember}` : null, async () => {
    const page = await store.org({ kind: rk, member: [historyMember] });
    return { rows: page.members, next: "", stopped: false };
  }, failMember);
  const { facts, failed: factsFailed, retry: retryFacts } = useMemberFacts(store, selected, view === "accounts" || view === "groups", () => { setSelected(""); dropLinked({ sel: null }); });
  const primary: Pager<unknown> | null = view === "accounts" ? accounts : view === "groups" ? groupList : null;
  const seen = useRef<{ key: string | null; n: number }>({ key: null, n: 0 });
  useEffect(() => {
    if (!primary || !primary.current || primary.status !== "ready") return;
    const count = countText(l, countOf(primary), view === "groups");
    const n = primary.rows.length;
    if (seen.current.key === primary.key) {
      if (n > seen.current.n) {
        setAnnounce(o.moreShown(l, n - seen.current.n, count));
        const row = primary.rows[seen.current.n] as AccountRow | OrgTargetRow;
        setFocusTo(`#o01-list .live-open[data-id="${row.id}"]`);
      }
    } else {
      const shown = count || o.noneShown(l).replace(/\.$/, "");
      const read = removed.current ? o.filterRemoved(l, shown) : o.readAnnounce(l, shown, appliedKeys(view, filters).length);
      setAnnounce(clearedNote.current ? `${o.selCleared(l)} ${read}` : read);
      clearedNote.current = false;
    }
    removed.current = false;
    seen.current = { key: primary.key, n };
  }, [primary?.key, primary?.status, primary?.rows.length]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (view === "tree" && cache.hierarchy?.complete && base) setAnnounce(o.announce(l, base.revision, on, cache.hierarchy.members.length));
  }, [view, cache.hierarchy, base?.revision, on]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!focusHistory.current || view !== "history" || !timeline.current || timeline.status !== "ready") return;
    focusHistory.current = false;
    if (params.get("from") === "link") onParams({ from: null });
    setFocusTo("#o01-history-title");
  }, [view, timeline.current, timeline.status]); // eslint-disable-line react-hooks/exhaustive-deps
  const built = useMemo(() => (base && cache.hierarchy ? tree(base.parties, cache.hierarchy.members, progressOf(cache.hierarchy)) : null), [base, cache.hierarchy]);
  useEffect(() => {
    if (built && !opened.current) {
      opened.current = true;
      setOpenSet(new Set(built.roots.filter((r) => r.kind === "legal_entity").map((r) => r.id)));
    }
  }, [built]);
  if (status === "error" && error instanceof AdminError && error.failure === "denied") {
    return <div className="live-users"><section className="live-unavailable" aria-labelledby="o01-denied"><h2 id="o01-denied">{o.deniedTitle(l)}</h2><p>{o.denied(l)}</p></section></div>;
  }
  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (status === "loading") return;
    if (!validDay(draft)) { setDateError(o.badDate(l)); setFocusTo("#o01-on"); return; }
    setDateError("");
    opened.current = false;
    store.setNotice("");
    if (draft !== on) {
      onParams({ on: draft, revision: null });
      setOn(draft);
    }
  };
  const apply = (next: Filters) => {
    setFilters(next);
    onParams(filterParams(next));
  };
  const fallbackFocus = wide ? (view === "history" ? "#o01-kind" : "#o01-q") : "#o01-filters-toggle";
  const removeFilter = (key: FilterKey) => {
    const to = nextFocus(appliedKeys(view, filters), key);
    removed.current = true;
    apply(cleared(filters, key));
    setFocusTo(to ? "#o01-chip-" + to : fallbackFocus);
  };
  const clearAll = () => {
    removed.current = true;
    apply(clearedAll(view, filters));
    setFocusTo(fallbackFocus);
  };
  const selectNode = (id: string) => editor.guarded(() => {
    editor.drop();
    setSelected(id);
    if (id !== selected) onParams({ sel: id });
    setSelectedGroup("");
    if (!wide) { setStepped(true); setFocusTo("#o01-detail-title"); }
  });
  const selectGroup = (id: string) => editor.guarded(() => {
    editor.drop();
    setSelectedGroup(id);
    if (!wide) { setStepped(true); setFocusTo("#o01-detail-title"); }
  });
  const back = () => {
    setStepped(false);
    setFocusTo(editor.target ? "#o01-return-draft" : `.live-open[data-id="${selectedGroup || selected}"]`);
  };
  const stepIn = () => { if (!wide) setStepped(true); };
  const toHistory = (member: string) => {
    setHistoryMember(member);
    setPicking(false);
    focusHistory.current = true;
    onView("history", { on, member, from: "link" });
  };
  const scopeForm = (
    <form className="o01-scope" onSubmit={submit} noValidate>
      <div className="service-field">
        <label htmlFor="o01-on">{o.asOf(l)}</label>
        {dateError && <p className="service-field-error" id="o01-on-error">{dateError}</p>}
        <input id="o01-on" type="date" value={draft} aria-invalid={dateError ? true : undefined} aria-describedby={dateError ? "o01-on-error" : undefined} onChange={(e) => setDraft(e.target.value)} />
      </div>
      <div className="live-buttons o01-scope-buttons">
        <button type="submit" className="live-primary" aria-disabled={status === "loading" || undefined}>{o.show(l)}</button>
        <button type="button" className="live-secondary" aria-disabled={status === "loading" || undefined} onClick={() => { if (status === "loading") return; const d = todayUTC(new Date()); setDraft(d); setDateError(""); if (d !== on) { onParams({ on: d, revision: null }); setOn(d); } }}>{o.today(l)}</button>
      </div>
      {base && (
        <dl className="o01-scope-facts">
          <div><dt>{o.revision(l)}</dt><dd>{base.revision}{base.revision === base.active ? ` · ${o.active(l)}` : ""}</dd></div>
          <div><dt>{o.workspace(l)}</dt><dd><span className="live-mono">{ltr(base.workspace.id)}</span> · {base.workspace.profile}</dd></div>
        </dl>
      )}
      <button type="button" className="live-secondary o01-reload" onClick={() => { opened.current = false; store.setNotice(""); store.reset(); }}>{o.reload(l)}</button>
    </form>
  );
  const notices = (
    <>
      {store.notice && <p className="service-status" role="status">{store.notice}</p>}
      {badFilter && <p className="service-status" role="status">{o.badFilter(l)}</p>}
      {store.inconsistent && (
        <div className="o01-newer" role="status">
          <p>{o.inconsistent(l)}</p>
          <button type="button" className="live-secondary" onClick={() => { opened.current = false; store.reset(); }}>{o.reload(l)}</button>
        </div>
      )}
      {base && newest > base.revision && (
        <div className="o01-newer" role="status">
          <p>{o.newer(l, newest, base.revision)}</p>
          <button type="button" className="live-secondary" onClick={() => { opened.current = false; store.reset(); setFocusTo("#o01-view-title"); }}>{o.showRevision(l, newest)}</button>
        </div>
      )}
    </>
  );
  const pageError = status === "error" ? (
    <section className="live-error" role="alert">
      <h2>{o.failedTitle(l)}</h2>
      <p>{error instanceof AdminError && error.failure === "unauthenticated" ? w.signInAgain(l) : o.failed(l)}</p>
      {error instanceof AdminError && error.failure === "unauthenticated"
        ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
        : <button type="button" className="live-secondary" onClick={() => store.reset()}>{o.tryAgain(l)}</button>}
    </section>
  ) : null;
  const pagerState = (p: Pager<unknown>, loading: string): ReactNode => {
    if (p.status === "loading") return <p role="status" className="live-meta">{loading}</p>;
    if (p.status !== "error") return null;
    return (
      <section className="live-error" role="alert">
        <h2>{o.failedTitle(l)}</h2>
        <p>{o.failed(l)}</p>
        <button type="button" className="live-secondary" onClick={p.retry}>{o.tryAgain(l)}</button>
      </section>
    );
  };
  const listHead = (title: string, meta: ReactNode, end: ReactNode = null) => (
    <div className="o01-list-head">
      <div className="live-heading">
        <h2 id="o01-view-title" tabIndex={-1}>{title}</h2>
        {meta && <p className="live-meta">{meta}</p>}
      </div>
      {end}
    </div>
  );
  const pagedMeta = (p: Pager<unknown>, groups = false): ReactNode =>
    p.current && p.status !== "loading" && p.status !== "error" ? <>{countOf(p).n > 0 && <>{countText(l, countOf(p), groups)} · </>}{dated(o.relationshipsOn(l, day(l, on)))}</> : null;
  const more = (p: Pager<unknown>, groups = false) => (
    <MoreBar l={l} next={p.next} stopped={p.stopped} status={p.status} count={countText(l, countOf(p), groups)} compact={!wide} onMore={p.more} />
  );
  const dated = (text: string): ReactNode => {
    const at = text.indexOf(day(l, on));
    if (at < 0) return text;
    return <>{text.slice(0, at)}<time dateTime={on}>{day(l, on)}</time>{text.slice(at + day(l, on).length)}</>;
  };
  const filteredEmpty = <FilteredEmpty l={l} date={day(l, on)} onClear={clearAll} />;
  const party = (id: string) => base?.parties.find((p) => p.id === id);
  const memberOf = (kind: RelationKind, id: string) => cache[kind]?.members.find((m) => m.member === id);
  const today = todayUTC(new Date());
  const lang = params.get("lang") ?? "";
  const planFor = (kind: RelationKind, member: string): PlanTarget | null => (base && status === "ready" ? { kind, member, base: base.revision } : null);
  const checking = draftList.status === "ready" && (view === "accounts" || (view === "groups" && groupKind === "view_group"));
  const pickRows: { id: string; name: string }[] = view === "accounts" ? accounts.rows.map((r) => ({ id: r.id, name: r.name })) : view === "groups" ? outside.rows.map((m) => ({ id: m.member, name: m.name })) : [];
  const editing = editor.editing;
  const cap = Math.max(0, draftList.maxMembers - (editing ? editor.count : 0));
  const toggle = (id: string) => { setSelCleared(false); setChecked((c) => toggled(c, id, cap)); };
  const toggleAll = () => { setSelCleared(false); setChecked((c) => allShown(c, pickRows.map((r) => r.id), cap)); };
  const picked = () => pickRows.filter((r) => checked.has(r.id)).map((r) => ({ id: r.id, name: r.name === r.id ? "" : r.name }));
  const startDraft = () => {
    const rows = picked();
    editor.openNew(rows);
    setChecked(new Set());
    setAnnounce(o.movedToDraft(l, rows.length));
    stepIn();
  };
  const addToDraft = () => {
    const got = editor.addRows(picked());
    setChecked(new Set());
    setAnnounce([o.addedToDraft(l, got.added, got.already), got.over > 0 ? o.overCap(l, got.over, draftList.maxMembers) : ""].filter(Boolean).join(" "));
  };
  const openDraft = (d: Parameters<typeof editor.openDraft>[0]) => { setSelectedGroup(""); editor.openDraft(d); stepIn(); };
  const newDraft = () => { setSelectedGroup(""); editor.openNew([]); stepIn(); };
  const bar = checking ? (
    <SelectionBar l={l} n={checked.size} k={pickRows.length} cap={cap} atCap={checked.size > 0 && checked.size >= cap} limitReached={draftList.limit > 0 && draftList.open >= draftList.limit}
      limit={draftList.limit} editing={editing} cleared={selCleared} wide={wide} onClear={() => setChecked(new Set())} onStart={startDraft} onAdd={addToDraft} onSelectAll={toggleAll} />
  ) : null;
  const returnLink = editor.target && !wide && !stepped ? (
    <button type="button" id="o01-return-draft" className="service-button-link" onClick={() => { setStepped(true); setFocusTo("#o01-draft-title"); }}>{o.returnToDraft(l)}</button>
  ) : null;
  const headEnd = bar || returnLink ? <div className="o01-head-end">{returnLink}{bar}</div> : null;
  const checkCell = (id: string, name: string) => (
    <RowCheck l={l} id={id} name={name} checked={checked.has(id)} disabled={!checked.has(id) && checked.size >= cap} stacked onToggle={() => toggle(id)} />
  );
  const memberDetail = (hier: OrgMember | undefined, hierLoaded: boolean, fallbackName: string, isAccount: boolean, plan: PlanTarget | null, pending = false) => {
    if (!selected || !base) return null;
    const p = party(selected);
    const parentFact: [string, ReactNode] = hier
      ? [o.parentOn(l, day(l, on)), hier.targets.length > 1 ? <strong>{conflictLabel(l, hier.targets.length)}</strong>
        : hier.targets[0] ? <bdi>{label(hier.targets[0].name, hier.targets[0].id)}</bdi> : <strong>{o.unattributed(l)}</strong>]
      : !hierLoaded ? [o.parentOn(l, day(l, on)), factsFailed && pending
        ? <><span className="live-quiet" role="alert">{o.factsFailed(l)}</span> <button type="button" className="service-button-link" onClick={retryFacts}>{o.tryAgain(l)}</button></>
        : <span className="live-quiet">{pending ? o.loadingMore(l) : o.notLoaded(l)}</span>]
        : p?.parent ? [o.registryParent(l), <bdi>{label(party(p.parent)?.name ?? "", p.parent)}</bdi>]
          : [o.parentOn(l, day(l, on)), <span className="live-quiet">{o.noHierarchy(l)}</span>];
    const since = hier && hier.targets.length === 1 ? covering(hier.history, on)?.from ?? "" : "";
    const roles = base.profiles.filter((pr) => [pr.issuer, pr.buyer, pr.payer, pr.recipient].includes(selected));
    const name = p ? label(p.name, p.id) : label(hier?.name || fallbackName, selected);
    return (
      <section className="live-step" aria-labelledby="o01-detail-title">
        <h2 id="o01-detail-title" tabIndex={-1}><bdi>{name}</bdi></h2>
        <Facts items={[
          [o.idLabel(l), <span className="live-mono">{ltr(selected)}</span>],
          [o.kindLabel(l), kindName(l, p ? p.kind : isAccount ? "account" : "unknown")],
          parentFact,
          ...(p ? [[o.state(l), p.state === "active" ? o.stateActive(l) : o.stateSuspended(l)] as [string, ReactNode]] : []),
          ...(since ? [[o.entryFrom(l), <Dated l={l} iso={since} />] as [string, ReactNode]] : []),
        ]} />
        {p && (
          <>
            <h3>{o.profiles(l)}</h3>
            <p className="live-note">{o.currentRegistry(l)}</p>
            {roles.length === 0 ? <p className="live-note">{o.noProfiles(l)}</p> : (
              <Facts items={roles.map((pr) => [label(pr.name, pr.id), <>
                {[pr.issuer === selected && o.issuer(l), pr.buyer === selected && o.buyer(l), pr.payer === selected && o.payer(l), pr.recipient === selected && o.recipient(l)].filter(Boolean).join(", ")}
                {" · "}{pr.path === "external_invoice" ? o.external(l) : o.chargeback(l)}{" · "}{pr.currency}
              </>] as [string, ReactNode])} />
            )}
          </>
        )}
        <PlanActions l={l} plan={plan} on={on} today={today} lang={lang}>
          <button type="button" className="service-button-link" onClick={() => toHistory(selected)}>{o.viewHistory(l)}</button>
        </PlanActions>
      </section>
    );
  };
  const notInView = (
    <p className="o01-not-loaded" role="status">{o.selected(l)} <bdi dir="ltr" className="live-mono">{selected}</bdi>. {o.selNotLoaded(l)}</p>
  );
  let detail: ReactNode = null;
  let main: ReactNode = null;
  if (view === "tree") {
    const hp = progressOf(cache.hierarchy);
    const hier = memberOf("hierarchy", selected);
    const node = built ? planNode([...built.roots, ...built.unattributed, ...built.conflicts.map((x) => x.node), ...built.cycles], selected) : undefined;
    detail = memberDetail(hier, loaded(selected, hp), "", !!hier, planMember(node) && loaded(selected, hp) ? planFor("hierarchy", selected) : null);
    const complete = !!cache.hierarchy && cache.hierarchy.complete && !cache.hierarchy.failed;
    const ready = !!cache.hierarchy;
    main = (
      <section aria-labelledby="o01-view-title">
        <div className="live-toolbar">
          <div className="live-heading">
            <h2 id="o01-view-title" tabIndex={-1}>{o.treeTitle(l)}</h2>
            {ready && <p className="live-meta">{o.loaded(l, cache.hierarchy?.members.length ?? 0)} · <Dated l={l} iso={on} /></p>}
          </div>
        </div>
        {status === "loading" && !ready ? <p role="status" className="live-meta">{o.loading(l, day(l, on))}</p> : pageError}
        {built && (
          <>
            {(() => {
              const toggle = (id: string) => setOpenSet((st) => { const n = new Set(st); if (n.has(id)) n.delete(id); else n.add(id); return n; });
              const sub = (nodes: TreeNode[]) => <TreeList l={l} nodes={nodes} depth={0} open={open} selected={selected} onSelect={(n) => selectNode(n.id)} setOpen={toggle} label={false} />;
              const everything = [...built.roots, ...built.unattributed, ...built.conflicts.map((x) => x.node), ...built.unloaded, ...built.cycles];
              if (everything.length === 0) {
                return <p className="live-empty">{!complete ? o.incomplete(l) : base && base.parties.length === 0 ? o.emptyAll(l) : base && base.revision === 0 ? o.emptyRevision(l) : o.emptyKind(l, o.kind(l, "hierarchy"), base?.revision ?? 0)}</p>;
              }
              return (
                <>
                  <div className="live-tools o01-tree-tools">
                    <button type="button" className="live-secondary" onClick={() => setOpenSet(new Set(all(everything)))}>{o.expandAll(l)}</button>
                    <button type="button" className="live-secondary" onClick={() => setOpenSet(new Set())}>{o.collapseAll(l)}</button>
                  </div>
                  {base?.revision === 0 && <p className="live-note" role="status">{o.noRevisionNote(l)}</p>}
                  {built.roots.length > 0 && <TreeList l={l} nodes={built.roots} depth={0} open={open} selected={selected} onSelect={(n) => selectNode(n.id)} setOpen={toggle} label />}
                  {built.conflicts.length > 0 && (
                    <section className="o01-unattributed" aria-labelledby="o01-conflicts">
                      <h3 id="o01-conflicts">{o.conflictTitle(l, day(l, on))}</h3>
                      {built.conflicts.map((c) => (
                        <div key={c.node.id}>
                          {sub([c.node])}
                          <p className="live-note">{conflictLabel(l, c.targets.length)}: {c.targets.map((x) => label(x.name, x.id)).join(", ")}</p>
                        </div>
                      ))}
                    </section>
                  )}
                  {built.cycles.length > 0 && (
                    <section className="o01-unattributed" aria-labelledby="o01-cycles">
                      <h3 id="o01-cycles">{o.cycleTitle(l)}</h3>
                      {sub(built.cycles)}
                    </section>
                  )}
                  {built.unloaded.length > 0 && (
                    <section className="o01-unattributed" aria-labelledby="o01-unloaded">
                      <h3 id="o01-unloaded">{o.notLoadedTitle(l)}</h3>
                      {sub(built.unloaded)}
                    </section>
                  )}
                  {(built.unattributed.length > 0 || !complete) && (
                    <section className="o01-unattributed" aria-labelledby="o01-noorg">
                      <h3 id="o01-noorg">{o.noOrg(l, day(l, on))}</h3>
                      {built.unattributed.length > 0 && sub(built.unattributed)}
                      {!complete && <p className="live-note">{o.incomplete(l)}</p>}
                    </section>
                  )}
                </>
              );
            })()}
            {ready && !complete && (
              <div className="live-note-rule o01-incomplete" role="status">
                <p>{o.incomplete(l)}</p>
                <button type="button" className="live-secondary" onClick={() => void store.load("hierarchy", true)}>{cache.hierarchy?.failed ? o.continueLoading(l) : o.loadMore(l)}</button>
              </div>
            )}
          </>
        )}
      </section>
    );
  } else if (view === "accounts") {
    const rows = accounts.rows;
    const row = rows.find((r) => r.id === selected);
    detail = !selected ? null : row ? memberDetail(facts?.hier, !!facts, row.name, true, planFor(rk, selected), !facts) : notInView;
    const other = rk !== "hierarchy" && rk !== "payer";
    const kindLabel = o.kind(l, rk);
    main = (
      <section aria-labelledby="o01-view-title">
        {listHead(o.accountsTitle(l), pagedMeta(accounts), headEnd)}
        {pageError ?? pagerState(accounts, o.loadingAccounts(l, day(l, on)))}
        {accounts.current && accounts.status !== "loading" && accounts.status !== "error" && (
          rows.length === 0 ? (
            accounts.stopped ? null : narrowing(view, filters).length > 0 ? filteredEmpty : <p className="live-empty">{o.noAccounts(l)}</p>
          ) : (
            <div className={"security-table-scroll live-table o01-accounts" + (checking ? " o01-checkable" : "")}>
              <table role="table">

                <thead role="rowgroup"><tr role="row">
                  {checking && <th scope="col" role="columnheader" className="o01-check"><HeaderCheck l={l} state={header(checked, rows.map((r) => r.id))} onToggle={toggleAll} /></th>}
                  <th scope="col" role="columnheader">{o.colAccount(l)}</th>
                  <th scope="col" role="columnheader">{o.colOrganization(l)}</th>
                  <th scope="col" role="columnheader">{o.payer(l)}</th>
                  {other && <th scope="col" role="columnheader">{kindLabel}</th>}
                  <th scope="col" role="columnheader">{o.entryFrom(l)}</th>
                </tr></thead>
                <tbody role="rowgroup">
                  {rows.map((r) => (
                    <tr key={r.id} role="row" className={selected === r.id ? "live-selected" : undefined}>
                      {checking && <td role="cell" className="o01-check">{checkCell(r.id, r.name)}</td>}
                      <td role="cell" className="live-user"><button type="button" className="live-open" data-id={r.id} aria-current={selected === r.id ? "true" : undefined} onClick={() => selectNode(r.id)}><bdi className={r.name === r.id ? "o01-id-name" : undefined}>{r.name}</bdi></button><span className="live-sub live-mono">{ltr(r.id)}</span></td>
                      <td role="cell" data-label={o.orgOn(l, day(l, on))}>{sideText(l, r.unit, o.noHierarchy(l), o.unattributed(l))}</td>
                      <td role="cell" data-label={o.payerOn(l, day(l, on))}>{sideText(l, r.payer, o.noPayer(l), o.noPayerDay(l))}</td>
                      {other && r.other && <td role="cell" data-label={o.kindOn(l, kindLabel, day(l, on))}>{sideText(l, r.other, o.noKind(l), o.unattributed(l))}</td>}
                      <td role="cell" data-label={o.entryFrom(l)}>{r.entryFrom ? <Dated l={l} iso={r.entryFrom} /> : ""}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
        {accounts.current && rows.length === 0 && accounts.stopped && <p className="live-empty">{o.noMatchesYet(l)}</p>}
        {accounts.current && more(accounts)}
      </section>
    );
  } else if (view === "groups") {
    const groupsRows = groupList.rows;
    const outsideRow = outside.rows.find((m) => m.member === selected);
    const group = groupsRows.find((g) => g.id === selectedGroup);
    detail = selectedGroup ? (
      <section className="live-step" aria-labelledby="o01-detail-title">
        <h2 id="o01-detail-title" tabIndex={-1}><bdi>{group ? label(group.name, group.id) : selectedGroup}</bdi></h2>
        <Facts items={[
          [o.idLabel(l), <span className="live-mono">{ltr(selectedGroup)}</span>],
          [o.kindLabel(l), o.kind(l, groupKind)],
          ...(group ? [[o.membersAll(l, day(l, on)), String(group.members_on)] as [string, ReactNode]] : []),
        ]} />
        <h3>{o.groupMembers(l)}</h3>
        {pagerState(groupMembers, o.loadingMembers(l, day(l, on)))}
        {groupMembers.current && groupMembers.status !== "loading" && groupMembers.status !== "error" && (
          groupMembers.rows.length === 0 ? (groupMembers.stopped ? null : <p className="live-empty">{filters.entity ? o.groupMembersFiltered(l, day(l, on)) : o.groupMembersNone(l, day(l, on))}</p>) : (
            <>
              <p className="live-meta">{countText(l, countOf(groupMembers))}</p>
              <ul className="o01-plain">{groupMembers.rows.map((m) => <li key={m.member}><bdi>{label(m.name, m.member)}</bdi> <span className="live-mono live-quiet">{ltr(m.member)}</span></li>)}</ul>
            </>
          )
        )}
        {groupMembers.current && groupMembers.rows.length === 0 && groupMembers.stopped && <p className="live-empty">{o.noMatchesYet(l)}</p>}
        {groupMembers.current && <MoreBar l={l} next={groupMembers.next} stopped={groupMembers.stopped} status={groupMembers.status} count={countText(l, countOf(groupMembers))} compact={false} onMore={groupMembers.more} />}
      </section>
    ) : !selected ? null : outsideRow ? memberDetail(facts?.hier, !!facts, outsideRow.name, true, planFor(groupKind, selected), !facts) : notInView;
    main = (
      <section aria-labelledby="o01-view-title">
        {listHead(o.groupsTitle(l), pagedMeta(groupList, true), returnLink)}
        <div className="live-note-rule"><p>{groupKind === "view_group" ? o.viewNote(l) : o.poolNote(l)}</p></div>
        {groupKind === "view_group" && <DraftsSection l={l} list={draftList} lang={lang} current={editor.draft?.id ?? ""} onOpen={openDraft} onNew={newDraft} />}
        {pageError ?? pagerState(groupList, o.loadingGroups(l, day(l, on)))}
        {groupList.current && groupList.status !== "loading" && groupList.status !== "error" && (
          groupsRows.length === 0 ? (groupList.stopped ? <p className="live-empty">{o.noMatchesYet(l)}</p> : filters.q ? filteredEmpty : <p className="live-empty">{o.noGroupsOn(l, day(l, on))}</p>) : (
            <div className="security-table-scroll live-table o01-groups">
              <table role="table">
                <thead role="rowgroup"><tr role="row">
                  <th scope="col" role="columnheader">{o.colGroup(l)}</th>
                  <th scope="col" role="columnheader">{dated(o.membersAll(l, day(l, on)))}</th>
                </tr></thead>
                <tbody role="rowgroup">
                  {groupsRows.map((g) => (
                    <tr key={g.id} role="row" className={selectedGroup === g.id ? "live-selected" : undefined}>
                      <td role="cell" className="live-user"><button type="button" className="live-open" data-id={g.id} aria-current={selectedGroup === g.id ? "true" : undefined} onClick={() => selectGroup(g.id)}><bdi className={g.name ? undefined : "o01-id-name"}>{label(g.name, g.id)}</bdi></button><span className="live-sub live-mono">{ltr(g.id)}</span></td>
                      <td role="cell" data-label={o.membersAll(l, day(l, on))}>{g.members_on}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        )}
        {groupList.current && more(groupList, true)}
        <section aria-labelledby="o01-outside" className="o01-outside">
          <div className="o01-list-head o01-sub-head">
            <div className="live-heading">
              <h3 id="o01-outside">{dated(groupKind === "view_group" ? o.notInView(l, day(l, on)) : o.notInPool(l, day(l, on)))}</h3>
              {outside.current && outside.status !== "loading" && outside.status !== "error" && outside.rows.length > 0 && <p className="live-meta">{countText(l, countOf(outside))}</p>}
            </div>
            {bar}
          </div>
          {pagerState(outside, groupKind === "view_group" ? o.loadingOutsideView(l, day(l, on)) : o.loadingOutsidePool(l, day(l, on)))}
          {outside.current && outside.status !== "loading" && outside.status !== "error" && (
            outside.rows.length === 0 ? (outside.stopped ? <p className="live-empty">{o.noMatchesYet(l)}</p> : filters.entity ? filteredEmpty : <p className="live-empty">{o.noneUnattributed(l)}</p>) : (
              <>
                <ul className="o01-plain">{outside.rows.map((m) => (
                  <li key={m.member} className={checking ? "o01-pick" : undefined}>{checking && checkCell(m.member, label(m.name, m.member))}<button type="button" className="live-open" data-id={m.member} aria-current={selected === m.member && !selectedGroup ? "true" : undefined} onClick={() => selectNode(m.member)}><bdi>{label(m.name, m.member)}</bdi></button> <span className="live-mono live-quiet">{ltr(m.member)}</span></li>
                ))}</ul>
              </>
            )
          )}
          {outside.current && more(outside)}
        </section>
      </section>
    );
  } else {
    const chosen = timeline.rows[0];
    const applying = chosen ? chosen.history.filter((h) => applies(h, on)) : [];
    const conflicting = applying.length > 1 && (rk === "hierarchy" || rk === "payer");
    const pick = (m: OrgMember) => {
      setHistoryMember(m.member);
      setPicking(false);
      focusHistory.current = true;
      onParams({ member: m.member });
    };
    main = (
      <section aria-labelledby="o01-view-title">
        {listHead(o.historyTitle(l), null)}
        {pageError}
        {picking && historyQ && (
          <>
            {pagerState(results, o.searchingMembers(l))}
            {results.current && results.status !== "loading" && results.status !== "error" && (results.rows.length === 0 ? (results.stopped ? null : <p className="live-empty">{o.noResults(l)}</p>) : (
              <>
                <p className="live-meta">{countText(l, countOf(results))}</p>
                <ul className="o01-plain o01-results">{results.rows.map((m) => (
                  <li key={m.member}><button type="button" className="live-open" data-id={m.member} onClick={() => pick(m)}><bdi>{label(m.name, m.member)}</bdi> <span className="live-mono live-quiet">{ltr(m.member)}</span></button></li>
                ))}</ul>
              </>
            ))}
            {results.current && results.rows.length === 0 && results.stopped && <p className="live-empty">{o.noMatchesYet(l)}</p>}
            {results.current && <MoreBar l={l} next={results.next} stopped={results.stopped} status={results.status} count={countText(l, countOf(results))} compact={!wide} onMore={results.more} />}
          </>
        )}
        {historyMember && (
          <>
            {pagerState(timeline, o.loadingTimeline(l))}
            {timeline.current && timeline.status === "ready" && !chosen && <p className="live-empty">{o.noTimeline(l)}</p>}
            {chosen && (
              <section aria-labelledby="o01-history-title">
                <h3 id="o01-history-title" tabIndex={-1}><bdi>{label(chosen.name, chosen.member)}</bdi> <span className="live-mono live-quiet">{ltr(chosen.member)}</span></h3>
                <div className="security-table-scroll live-table o01-history">
                  <table role="table">
                    <colgroup><col className="o01-c40" /><col className="o01-c20" /><col className="o01-c20" /><col className="o01-c20" /></colgroup>
                    <thead role="rowgroup"><tr role="row">
                      <th scope="col" role="columnheader">{o.colTarget(l)}</th>
                      <th scope="col" role="columnheader">{o.colFrom(l)}</th>
                      <th scope="col" role="columnheader">{o.colUntil(l)}</th>
                      <th scope="col" role="columnheader">{dated(o.colOn(l, day(l, on)))}</th>
                    </tr></thead>
                    <tbody role="rowgroup">
                      {[...chosen.history].sort((a, b) => a.from.localeCompare(b.from)).map((h) => (
                        <tr key={h.target + h.from} role="row">
                          <td role="cell" className="live-user"><bdi>{label(h.name, h.target)}</bdi><span className="live-sub live-mono">{ltr(h.target)}</span></td>
                          <td role="cell" data-label={o.colFrom(l)}><Dated l={l} iso={h.from} /></td>
                          <td role="cell" data-label={o.colUntil(l)}>{h.until ? <Dated l={l} iso={h.until} /> : o.noEnd(l)}</td>
                          <td role="cell" data-label={o.colOn(l, day(l, on))}>{applying.includes(h) ? <strong>{o.applies(l)}</strong> : ""}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {conflicting && <p className="live-note"><strong>{conflictLabel(l, applying.length)}</strong></p>}
                <p className="live-note">{o.endNote(l)}</p>
                <PlanActions l={l} plan={planFor(rk, chosen.member)} on={on} today={today} lang={lang} />
              </section>
            )}
          </>
        )}
      </section>
    );
  }
  const historyPick = view !== "history" ? null : picking ? (
    <SearchForm l={l} id="o01-history-q" labelText={o.search(l)} value={historyQ} onSearch={setHistoryQ} />
  ) : (
    <p className="o01-member-chosen">
      <span className="o01-provider-label">{o.memberLabel(l)}</span>{" "}
      <bdi>{label(timeline.rows[0]?.name ?? "", historyMember)}</bdi> <span className="live-mono live-quiet">{ltr(historyMember)}</span>{" "}
      <button type="button" className="service-button-link" onClick={() => { setPicking(true); setFocusTo("#o01-history-q"); }}>{o.change(l)}</button>
    </p>
  );
  const entities = (base?.parties ?? []).filter((p) => p.kind === "legal_entity");
  const band = (
    <FilterBand l={l} view={view} wide={wide} open={bandOpen.has(view)} onToggle={() => setBandOpen((s) => { const n = new Set(s); if (n.has(view)) n.delete(view); else n.add(view); return n; })}
      filters={filters} entities={entities} onChange={apply} onRemove={removeFilter} onClearAll={clearAll}
      groupKind={groupKind} onGroupKind={(k) => { setGroupKind(k); setSelectedGroup(""); }} history={historyPick} />
  );
  if (editor.target && (view === "accounts" || view === "groups")) {
    const searchAccounts = async (q: string, after: string) => {
      const page = await store.org({ kind: "view_group", subject: "account", q: q || undefined, after: after || undefined, limit: 10 });
      return { rows: page.members, next: page.next };
    };
    const searchGroups = async (q: string) => {
      const page = await store.targets({ kind: "view_group", q: q || undefined, limit: 10 });
      return { rows: page.targets, next: "" };
    };
    detail = <DraftPanel l={l} e={editor} list={draftList} lang={lang} on={on} today={today} base={base?.revision ?? draftList.active} searchAccounts={searchAccounts} searchGroups={searchGroups} />;
  }
  const showPanel = view !== "history" && (wide || stepped);
  return (
    <div className={"o01-live" + (view === "history" ? " o01-single" : "")} data-live={view}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {scopeForm}
      {notices}
      {band}
      <div className="live-users" data-wide={wide ? "true" : "false"} data-mode={editor.target && view === "groups" ? "draft" : view === "accounts" && checking ? "pick" : undefined}>
        {!(stepped && !wide) && <div className="live-list live-sections" id="o01-list">{main}</div>}
        {showPanel && (
          <aside className="live-panel" aria-label={o.selectHint(l)}>
            {stepped && !wide && <button type="button" className="service-button-link live-back" onClick={back}>{o.back(l)}</button>}
            {detail ?? <p className="live-empty">{o.selectHint(l)}</p>}
          </aside>
        )}
      </div>
    </div>
  );
}
