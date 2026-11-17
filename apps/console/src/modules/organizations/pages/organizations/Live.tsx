import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, type AdminClient, type OrgMember, type OrgPage, type OrgParty, type OrgProfile, type RelationKind } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { useWide } from "../../../identity/pages/users/Live";
import { w } from "../../../identity/pages/users/live-copy";
import { o } from "./live-copy";
import { accounts, applies, covering, groups, label, loaded, newer, todayUTC, tree, validDay, type Progress, type Side, type TreeNode } from "./live-model";
import "../../../identity/pages/users/live.css";
import "./live.css";

const maxPages = 10;
const pageSize = 99;

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

function useStore(api: AdminClient, on: string, initialPin: number | null, onPin: (revision: number | null) => void) {
  const [base, setBase] = useState<Base | null>(null);
  const [cache, setCache] = useState<Partial<Record<RelationKind, Loaded>>>({});
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [newest, setNewest] = useState(0);
  const [notice, setNotice] = useState("");
  const [inconsistent, setInconsistent] = useState(false);
  const state = useRef("");
  const pinned = useRef<number | null>(null);
  const restarted = useRef(false);
  const token = useRef(0);
  const busy = useRef(new Set<RelationKind>());
  const reset = useCallback(() => {
    token.current += 1;
    pinned.current = null;
    busy.current.clear();
    setBase(null);
    setCache({});
    setNewest(0);
    setInconsistent(false);
    state.current = "";
    setStatus("loading");
  }, []);
  const seeded = useRef<{ on: string; pin: number | null } | null>(null);
  useEffect(() => {
    restarted.current = false;
    const keep = seeded.current === null || seeded.current.on === on;
    const pin = seeded.current === null ? initialPin : seeded.current.pin;
    reset();
    if (keep) pinned.current = pin;
    seeded.current = { on, pin: keep ? pin : null };
  }, [on, reset]); // eslint-disable-line react-hooks/exhaustive-deps
  const absorb = (page: OrgPage) => {
    if (pinned.current === null) {
      pinned.current = page.relationship_revision;
      onPin(page.relationship_revision);
    }
    if (state.current && state.current !== page.revision) setInconsistent(true);
    state.current = page.revision;
    setBase({ parties: page.parties, profiles: page.profiles, workspace: page.workspace, revision: page.relationship_revision, active: page.active_revision });
    if (newer(page.active_revision, page.relationship_revision)) setNewest(page.active_revision);
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
      if (err instanceof AdminError && err.failure === "invalid" && pinned.current !== null && !restarted.current) {
        restarted.current = true;
        setNotice(o.revisionReset(l0.current, pinned.current));
        reset();
        onPin(null);
        return;
      }
      if (current.members.length > 0) setCache((c) => ({ ...c, [kind]: { ...current, failed: true } }));
      else { setError(err); setStatus("error"); }
    } finally {
      if (mine === token.current) busy.current.delete(kind);
    }
  }, [api, on, cache, reset]); // eslint-disable-line react-hooks/exhaustive-deps
  const l0 = useRef<Locale>("en");
  const fresh = useCallback(() => { restarted.current = false; reset(); onPin(null); }, [reset]); // eslint-disable-line react-hooks/exhaustive-deps
  return { base, cache, status, error, newest, notice, setNotice, load, reset: fresh, l0, inconsistent };
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

function sideText(l: Locale, s: Side, none: string, empty: string): ReactNode {
  if (s.state === "target") return <bdi>{label(s.target.name, s.target.id)}</bdi>;
  if (s.state === "conflict") return <><strong>{o.conflict(l, s.targets.length)}</strong><span className="live-sub">{s.targets.map((x) => label(x.name, x.id)).join(", ")}</span></>;
  if (s.state === "unloaded") return <span className="live-quiet">{o.notLoaded(l)}</span>;
  return s.state === "unattributed" ? <strong>{empty}</strong> : <span className="live-quiet">{none}</span>;
}

export default function LiveOrganizations({ locale: l, view, params, session, onSignOut, onView, onParams }: {
  locale: Locale; view: string; params: URLSearchParams; session: Session; onSignOut: () => void;
  onView: (view: string, params?: Record<string, string>) => void; onParams: (params: Record<string, string | null>) => void;
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
  const [selected, setSelected] = useState("");
  const [stepped, setStepped] = useState(false);
  const [open, setOpenSet] = useState<Set<string>>(new Set());
  const [onlyUnattributed, setOnlyUnattributed] = useState(false);
  const [groupKind, setGroupKind] = useState<RelationKind>("view_group");
  const [historyKind, setHistoryKind] = useState<RelationKind>(() => {
    const k = params.get("kind");
    return k === "payer" || k === "contract_pool" || k === "allocation_pool" || k === "view_group" ? k : "hierarchy";
  });
  const [historyMember, setHistoryMember] = useState(params.get("member") ?? "");
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const opened = useRef(false);
  const focusedHistory = useRef(false);
  useEffect(() => {
    if (focusedHistory.current || view !== "history" || params.get("from") !== "link" || !cache[historyKind]) return;
    focusedHistory.current = true;
    onParams({ from: null });
    document.getElementById("o01-history-member")?.focus();
  }, [view, cache, historyKind]); // eslint-disable-line react-hooks/exhaustive-deps
  const needs: RelationKind[] = view === "tree" ? ["hierarchy"] : view === "accounts" ? ["hierarchy", "payer"] : view === "groups" ? [groupKind, "hierarchy"] : [historyKind];
  useEffect(() => {
    for (const k of needs) {
      if (!cache[k]) void store.load(k);
      if (!base) break;
    }
  }, [view, groupKind, historyKind, on, cache]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => { setSelected(""); setStepped(false); }, [view, on]);
  useEffect(() => {
    if (!focusTo) return;
    document.querySelector<HTMLElement>(focusTo)?.focus();
    setFocusTo("");
  }, [focusTo]);
  const primary = view === "groups" ? [groupKind] : needs;
  const loadedKinds = primary.map((k) => cache[k]).filter(Boolean) as Loaded[];
  const ready = loadedKinds.length === primary.length;
  const complete = ready && loadedKinds.every((c) => c.complete && !c.failed);
  const failedMid = loadedKinds.some((c) => c.failed);
  const count = loadedKinds.reduce((n, c) => Math.max(n, c.members.length), 0);
  useEffect(() => {
    if (ready && complete && base) setAnnounce(o.announce(l, base.revision, on, count));
  }, [ready, complete, base?.revision, on, count]); // eslint-disable-line react-hooks/exhaustive-deps
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
  const selectNode = (id: string) => {
    setSelected(id);
    if (!wide) { setStepped(true); setFocusTo("#o01-detail-title"); }
  };
  const back = () => { setStepped(false); setFocusTo(`.live-open[data-id="${selected}"]`); };
  const toHistory = (kind: RelationKind, member: string) => {
    setHistoryKind(kind);
    setHistoryMember(member);
    onView("history", { on, kind, member, from: "link" });
    setFocusTo("#o01-history-member");
  };
  const scope = (
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
  const loadState = status === "loading" && !ready ? <p role="status" className="live-meta">{o.loading(l, day(l, on))}</p>
    : status === "error" ? (
      <section className="live-error" role="alert">
        <h2>{o.failedTitle(l)}</h2>
        <p>{error instanceof AdminError && error.failure === "unauthenticated" ? w.signInAgain(l) : o.failed(l)}</p>
        {error instanceof AdminError && error.failure === "unauthenticated"
          ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
          : <button type="button" className="live-secondary" onClick={() => store.reset()}>{o.tryAgain(l)}</button>}
      </section>
    ) : null;
  const incompleteNote = ready && !complete && (
    <div className="live-note-rule o01-incomplete" role="status">
      <p>{o.incomplete(l)}</p>
      <button type="button" className="live-secondary" onClick={() => { for (const k of primary) if (!cache[k]?.complete) void store.load(k, true); }}>{failedMid ? o.continueLoading(l) : o.loadMore(l)}</button>
    </div>
  );
  const toolbar = (title: string) => (
    <div className="live-toolbar">
      <div className="live-heading">
        <h2 id="o01-view-title" tabIndex={-1}>{title}</h2>
        {ready && <p className="live-meta">{o.loaded(l, count)} · <Dated l={l} iso={on} /></p>}
      </div>
    </div>
  );
  const party = (id: string) => base?.parties.find((p) => p.id === id);
  const memberOf = (kind: RelationKind, id: string) => cache[kind]?.members.find((m) => m.member === id);
  const detail = (() => {
    if (!selected || !base) return null;
    const p = party(selected);
    const hier = memberOf("hierarchy", selected);
    const hp = progressOf(cache.hierarchy);
    const parentFact: [string, ReactNode] = hier
      ? [o.parentOn(l, day(l, on)), hier.targets.length > 1 ? <strong>{o.conflict(l, hier.targets.length)}</strong>
        : hier.targets[0] ? <bdi>{label(hier.targets[0].name, hier.targets[0].id)}</bdi> : <strong>{o.unattributed(l)}</strong>]
      : !loaded(selected, hp) ? [o.parentOn(l, day(l, on)), <span className="live-quiet">{o.notLoaded(l)}</span>]
        : p?.parent ? [o.registryParent(l), <bdi>{label(party(p.parent)?.name ?? "", p.parent)}</bdi>]
          : [o.parentOn(l, day(l, on)), <span className="live-quiet">{o.noHierarchy(l)}</span>];
    const since = hier && hier.targets.length === 1 ? covering(hier.history, on)?.from ?? "" : "";
    const roles = base.profiles.filter((pr) => [pr.issuer, pr.buyer, pr.payer, pr.recipient].includes(selected));
    const name = p ? label(p.name, p.id) : label(hier?.name ?? memberOf("payer", selected)?.name ?? "", selected);
    return (
      <section className="live-step" aria-labelledby="o01-detail-title">
        <h2 id="o01-detail-title" tabIndex={-1}><bdi>{name}</bdi></h2>
        <Facts items={[
          [o.idLabel(l), <span className="live-mono">{ltr(selected)}</span>],
          [o.kindLabel(l), kindName(l, p ? p.kind : hier || memberOf("payer", selected) || cache[groupKind]?.members.some((x) => x.member === selected) ? "account" : "unknown")],
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
        <div className="live-buttons">
          <button type="button" className="service-button-link" onClick={() => toHistory("hierarchy", selected)}>{o.viewHistory(l)}</button>
        </div>
      </section>
    );
  })();
  let main: ReactNode = null;
  if (view === "tree") {
    main = (
      <section aria-labelledby="o01-view-title">
        {toolbar(o.treeTitle(l))}
        {loadState}
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
                          <p className="live-note">{o.conflict(l, c.targets.length)}: {c.targets.map((x) => label(x.name, x.id)).join(", ")}</p>
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
            {incompleteNote}
          </>
        )}
      </section>
    );
  } else if (view === "accounts") {
    const rows = base && cache.hierarchy && cache.payer ? accounts(base.parties, cache.hierarchy.members, cache.payer.members, on, progressOf(cache.hierarchy), progressOf(cache.payer)) : null;
    const shown = rows?.filter((r) => !onlyUnattributed || r.unit.state === "unattributed");
    main = (
      <section aria-labelledby="o01-view-title">
        {toolbar(o.accountsTitle(l))}
        {loadState}
        {rows && (
          <>
            <label className="live-check o01-filter"><input type="checkbox" checked={onlyUnattributed} onChange={(e) => setOnlyUnattributed(e.target.checked)} />{o.onlyUnattributed(l)}</label>
            {shown && shown.length === 0 ? <p className="live-empty">{onlyUnattributed ? o.noneUnattributed(l) : !complete ? o.incomplete(l) : o.noAccounts(l)}</p> : (
              <div className="security-table-scroll live-table o01-accounts">
                <table role="table">
                  <colgroup><col className="o01-c30" /><col className="o01-c26" /><col className="o01-c26" /><col className="o01-c18" /></colgroup>
                  <thead role="rowgroup"><tr role="row">
                    <th scope="col" role="columnheader">{o.colAccount(l)}</th>
                    <th scope="col" role="columnheader">{o.orgOn(l, day(l, on))}</th>
                    <th scope="col" role="columnheader">{o.payerOn(l, day(l, on))}</th>
                    <th scope="col" role="columnheader">{o.entryFrom(l)}</th>
                  </tr></thead>
                  <tbody role="rowgroup">
                    {shown?.map((r) => (
                      <tr key={r.id} role="row" className={selected === r.id ? "live-selected" : undefined}>
                        <td role="cell" className="live-user"><button type="button" className="live-open" data-id={r.id} onClick={() => selectNode(r.id)}><bdi>{r.name}</bdi></button><span className="live-sub live-mono">{ltr(r.id)}</span></td>
                        <td role="cell" data-label={o.orgOn(l, day(l, on))}>{sideText(l, r.unit, o.noHierarchy(l), o.unattributed(l))}</td>
                        <td role="cell" data-label={o.payerOn(l, day(l, on))}>{sideText(l, r.payer, o.noPayer(l), o.noPayerDay(l))}</td>
                        <td role="cell" data-label={o.entryFrom(l)}>{r.entryFrom ? <Dated l={l} iso={r.entryFrom} /> : ""}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {incompleteNote}
          </>
        )}
      </section>
    );
  } else if (view === "groups") {
    const g = cache[groupKind] ? groups(cache[groupKind]!.members) : null;
    main = (
      <section aria-labelledby="o01-view-title">
        {toolbar(o.groupsTitle(l))}
        <fieldset className="live-roles o01-kinds">
          <legend>{o.relationship(l)}</legend>
          {(["view_group", "allocation_pool"] as RelationKind[]).map((k) => (
            <label key={k}><input type="radio" name="o01-group-kind" value={k} checked={groupKind === k} onChange={() => { setGroupKind(k); setSelected(""); }} />{o.kind(l, k)}</label>
          ))}
        </fieldset>
        {loadState}
        {g && (
          <>
            <div className="live-note-rule"><p>{groupKind === "view_group" ? o.viewNote(l) : o.poolNote(l)}</p></div>
            {g.groups.length === 0 ? <p className="live-empty">{!complete ? o.incomplete(l) : g.outside.length > 0 ? o.noGroupsOn(l, day(l, on)) : o.emptyKind(l, o.kind(l, groupKind), base?.revision ?? 0)}</p> : (
              <div className="security-table-scroll live-table o01-groups">
                <table role="table">
                  <colgroup><col className="o01-c34" /><col /><col className="o01-c14" /></colgroup>
                  <thead role="rowgroup"><tr role="row">
                    <th scope="col" role="columnheader">{o.colGroup(l)}</th>
                    <th scope="col" role="columnheader">{o.membersOn(l, day(l, on))}</th>
                    <th scope="col" role="columnheader">{o.membersLoaded(l)}</th>
                  </tr></thead>
                  <tbody role="rowgroup">
                    {g.groups.map((grp) => (
                      <tr key={grp.id} role="row">
                        <td role="cell" className="live-user"><bdi>{grp.name}</bdi><span className="live-sub live-mono">{ltr(grp.id)}</span></td>
                        <td role="cell" data-label={o.membersOn(l, day(l, on))}>
                          <ul className="o01-inline">
                            {grp.members.map((m) => <li key={m.id}><button type="button" className="live-open" data-id={m.id} onClick={() => selectNode(m.id)}><bdi>{m.name}</bdi></button></li>)}
                          </ul>
                        </td>
                        <td role="cell" data-label={o.membersLoaded(l)}>{grp.members.length}{complete ? "" : ` ${o.incompleteMark(l)}`}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {g.outside.length > 0 && (
              <section aria-labelledby="o01-outside">
                <h3 id="o01-outside">{groupKind === "view_group" ? o.notInView(l, day(l, on)) : o.notInPool(l, day(l, on))}</h3>
                <ul className="o01-plain">{g.outside.map((m) => <li key={m.member}><bdi>{label(m.name, m.member)}</bdi> <span className="live-mono live-quiet">{ltr(m.member)}</span></li>)}</ul>
              </section>
            )}
            {incompleteNote}
          </>
        )}
      </section>
    );
  } else {
    const list = cache[historyKind]?.members ?? [];
    const chosen = list.find((m) => m.member === historyMember);
    const applying = chosen ? chosen.history.filter((h) => applies(h, on)) : [];
    const conflicting = applying.length > 1 && (historyKind === "hierarchy" || historyKind === "payer");
    main = (
      <section aria-labelledby="o01-view-title">
        {toolbar(o.historyTitle(l))}
        <div className="o01-history-pick">
          <div className="service-field">
            <label htmlFor="o01-history-kind">{o.relationship(l)}</label>
            <select id="o01-history-kind" className="live-select" value={historyKind} onChange={(e) => setHistoryKind(e.target.value as RelationKind)}>
              {(["hierarchy", "payer", "contract_pool", "allocation_pool", "view_group"] as RelationKind[]).map((k) => <option key={k} value={k}>{o.kind(l, k)}</option>)}
            </select>
          </div>
          <div className="service-field">
            <label htmlFor="o01-history-member">{o.member(l)}</label>
            <select id="o01-history-member" className="live-select" value={historyMember} onChange={(e) => setHistoryMember(e.target.value)}>
              <option value="">{o.chooseMember(l)}</option>
              {list.map((m) => <option key={m.member} value={m.member}>{label(m.name, m.member)} ({m.member})</option>)}
            </select>
          </div>
        </div>
        {loadState}
        {historyMember && cache[historyKind] && !chosen && <p className="live-empty">{loaded(historyMember, progressOf(cache[historyKind])) ? o.noTimeline(l) : o.notLoadedMember(l)}</p>}
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
                  <th scope="col" role="columnheader">{o.colOn(l, day(l, on))}</th>
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
            {conflicting && <p className="live-note"><strong>{o.conflict(l, applying.length)}</strong></p>}
            <p className="live-note">{o.endNote(l)}</p>
          </section>
        )}
        {incompleteNote}
      </section>
    );
  }
  const showPanel = view !== "history" && (wide || stepped);
  return (
    <div className={"o01-live" + (view === "history" ? " o01-single" : "")} data-live={view}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {scope}
      {notices}
      <div className="live-users" data-wide={wide ? "true" : "false"}>
        {!(stepped && !wide) && <div className="live-list live-sections">{main}</div>}
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
