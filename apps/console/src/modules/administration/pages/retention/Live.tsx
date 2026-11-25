import { DiagnosticDetails } from "../../../../service/CommandDiagnostics";
import type { CommandDiagnostic } from "../../../../service/command-diagnostics";
import { currentScope } from "./live-model";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import "../../../../service/command-diagnostics.css";
import { CommandDiagnostics, ErrorDiagnostics, useCommandDiagnostics } from "../../../../service/CommandDiagnostics";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { ArtifactEntry, RetentionPage } from "../../../../service/retention";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { artifactIdsInUse, commandsAllowed, drop, forget, isProposalCommand, missingArtifacts, pruneRefusals, rowId, selectionOf, setScope, validId, vanished, viewOf, withHeldArtifacts, withMoreReferences } from "./live-model";
import { Known, Rich, Stamp, ltr } from "./live-format";
import type { Announce, Shared } from "./LiveCommand";
import { PoliciesView } from "./LivePolicies";
import { ExpiryView } from "./LiveExpiry";
import { HoldsView } from "./LiveHolds";
import { DeletionView } from "./LiveDeletion";
import { RestoreView } from "./LiveRestore";
import { DecisionsView } from "./LiveDecisions";
import "../../../../shared/domain-screen-ui/work/screen.css";
import "../../../identity/pages/users/live.css";
import "./live.css";

type Props = {
  locale: Locale;
  view: string;
  params: URLSearchParams;
  session: Session;
  onSignOut: () => void;
};

function failureText(l: Locale, e: unknown): string {
  if (e instanceof AdminError) {
    if (e.failure === "limited") return sc.limited(l, retryPhrase(l, e.retryAfter || 60));
    if (e.failure === "busy" || e.failure === "unavailable") return diagnosticCopy.unavailable(l);
  }
  return "";
}

function selectedOf(view: string, s: ReturnType<typeof selectionOf>): string {
  if (s.proposal) return "p04-proposal-" + s.proposal;
  if (view === "expiry" && s.artifact) return rowId("item", s.artifact);
  if (view === "expiry" && s.cls) return rowId("class", s.cls);
  if (view === "policies" && s.cls) return rowId("policy", s.cls);
  if (view === "holds" && s.hold) return rowId("hold", s.hold);
  if (view === "deletion" && s.run) return rowId("run", s.run);
  return "";
}

type ListState = { next: string | null; status: "idle" | "loading" | "failed"; reset: boolean };
const idleList: ListState = { next: null, status: "idle", reset: false };
const idBatch = 20;

export default function LiveRetention({ locale: l, view: rawView, params, session, onSignOut }: Props) {
  const source = useAdminClient();
  const { api, detail } = useCommandDiagnostics(source, JSON.stringify([session.workspace, session.user, session.csrf, rawView]), () => { setNotice(""); setNoticeDetail(null); setTerminalDetail(null); });
  setScope(session.workspace, session.user, session.csrf);
  const view = viewOf(rawView);
  const [page, setPage] = useState<RetentionPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [denied, setDenied] = useState(false);
  const [signedOut, setSignedOut] = useState(false);
  const [notice, setNotice] = useState("");
  const [noticeDetail, setNoticeDetail] = useState<CommandDiagnostic | null>(null);
  const [terminalDetail, setTerminalDetail] = useState<CommandDiagnostic | null>(null);
  const [busy, setBusy] = useState(false);
  const [, setTick] = useState(0);
  const seq = useRef(0);
  const listSeq = useRef<{ epoch: number; of: Record<string, number> }>({ epoch: 0, of: {} });
  const [lists, setLists] = useState<Record<string, ListState>>({});
  const wantedArtifact = view === "expiry" && validId(params.get("artifact") ?? "") ? params.get("artifact") ?? "" : "";
  const wantedRef = useRef(wantedArtifact);
  wantedRef.current = wantedArtifact;
  const scrolled = useRef("");
  const selection = selectionOf(params, page, view);
  const selectedId = selectedOf(view, selection);

  const load = useCallback(async (): Promise<RetentionPage | null> => {
    const n = ++seq.current;
    const generation = currentScope();
    setBusy(true);
    try {
      const main = await api.retention();
      if (n !== seq.current || generation !== currentScope()) return null;
      const held: ArtifactEntry[] = [];
      if (main.state === "configured") {
        const missing = missingArtifacts(main, [...artifactIdsInUse(), wantedRef.current]);
        for (let at = 0; at < missing.length; at += idBatch) held.push(...(await api.retentionArtifacts({ ids: missing.slice(at, at + idBatch) })).rows);
        if (n !== seq.current || generation !== currentScope()) return null;
      }
      const got = withHeldArtifacts(main, held);
      listSeq.current = { epoch: listSeq.current.epoch + 1, of: {} };
      pruneRefusals(got);
      setPage(got);
      setLists((before) => Object.fromEntries(Object.entries(before).filter(([, x]) => x.next !== null).map(([key]) => [key, { next: null, status: "idle" as const, reset: true }])));
      setStatus("ready");
      setLoadError(null);
      return got;
    } catch (e) {
      if (n !== seq.current || generation !== currentScope()) return null;
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut(true);
      setLoadError(e);
      setStatus("error");
      return null;
    } finally {
      if (n === seq.current && generation === currentScope()) setBusy(false);
    }
  }, [api]);

  useLayoutEffect(() => { void load(); return () => { seq.current++; }; }, [load]);
  const more = async (key: string, first: string | null) => {
    const was = lists[key] ?? idleList;
    const after = was.next ?? first ?? "";
    if (!page || was.status === "loading" || !after) return;
    const epoch = listSeq.current.epoch;
    const n = (listSeq.current.of[key] = (listSeq.current.of[key] ?? 0) + 1);
    const generation = currentScope();
    const current = () => epoch === listSeq.current.epoch && n === listSeq.current.of[key] && generation === currentScope();
    const set = (x: ListState) => setLists((before) => ({ ...before, [key]: x }));
    set({ next: was.next, status: "loading", reset: false });
    try {
      let next = "";
      if (key === "refs") {
        const answer = await api.retentionReferences(after);
        if (!current()) return;
        setPage((p) => (p ? withMoreReferences(p, answer.rows) : p));
        next = answer.next;
      } else {
        const answer = await api.retentionArtifacts({ cls: key, after });
        if (!current()) return;
        setPage((p) => (p ? withHeldArtifacts(p, answer.rows) : p));
        next = answer.next;
      }
      set({ next, status: "idle", reset: false });
    } catch (e) {
      if (!current()) return;
      if (e instanceof AdminError && (e.failure === "denied" || e.failure === "unauthenticated")) { set({ next: was.next, status: "idle", reset: false }); void load(); return; }
      set({ next: was.next, status: "failed", reset: false });
    }
  };
  const pageRevision = page?.revision ?? "";
  useEffect(() => {
    if (!page || page.state !== "configured" || !wantedArtifact || missingArtifacts(page, [wantedArtifact]).length === 0) return;
    const n = seq.current;
    const generation = currentScope();
    void api.retentionArtifacts({ ids: [wantedArtifact] }).then((answer) => {
      if (n === seq.current && generation === currentScope() && answer.rows.length > 0) setPage((p) => (p ? withHeldArtifacts(p, answer.rows) : p));
    }).catch(() => undefined);
  }, [wantedArtifact, pageRevision]); // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (!selectedId || scrolled.current === selectedId) return;
    scrolled.current = selectedId;
    document.getElementById(selectedId)?.scrollIntoView({ block: "center" });
  }, [selectedId]);

  const showNotice = (text: string, reference?: CommandDiagnostic) => {
    setNoticeDetail(reference ?? null);
    setNotice(text);
    requestAnimationFrame(() => document.getElementById("p04-notice")?.focus());
  };

  const announce = async (text: Announce) => {
    if (typeof text === "string") {
      showNotice(text);
      void load();
      return;
    }
    const got = await load();
    showNotice(text(got));
  };

  if (denied) {
    return (
      <div className="p04-live">
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
        <section className="live-unavailable" aria-labelledby="p04-denied">
          <h2 id="p04-denied">{k.deniedTitle(l)}</h2>
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="p04-live">
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
        <div className="live-error" role="alert">
          <h2>{k.sessionEnded(l)}</h2>
          <button type="button" className="live-secondary" onClick={() => { forget(); onSignOut(); }}>{k.signInAgain(l)}</button>
        </div>
      </div>
    );
  }

  const configured = page?.state === "configured";
  const shared: Shared | null = page && configured
    ? {
        l, page, api, me: session.user, workspace: session.workspace, selection, canCommand: commandsAllowed(page) && status !== "error",
        onDone: (text) => { void announce(text); },
        onNotice: showNotice,
        onReload: load,
        onSignedOut: (reference?: CommandDiagnostic) => { setTerminalDetail(reference ?? null); setSignedOut(true); },
        listOf: (key, first) => {
          const x = lists[key] ?? idleList;
          return { more: (x.next ?? first ?? "") !== "", status: x.status, reset: x.reset, onMore: () => { void more(key, first); } };
        },
      }
    : null;
  const gone = shared ? vanished(shared.page, view) : [];
  const failure = failureText(l, loadError);

  return (
    <div className="p04-live" data-live={view} aria-busy={busy || undefined}>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
      {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
      <div className="p04-scope">
        <dl className="p04-scope-facts">
          <div><dt>{k.workspace(l)}</dt><dd>{page ? <Known l={l} value={session.workspace} /> : "—"}</dd></div>
          <div><dt>{k.revision(l)}</dt><dd>{page ? <Known l={l} value={page.revision} /> : "—"}</dd></div>
          <div><dt>{k.now(l)}</dt><dd>{page ? <Stamp l={l} at={page.now} /> : "—"}</dd></div>
        </dl>
        <button type="button" className="live-secondary p04-reload" onClick={() => void load()}>{k.reload(l)}</button>
      </div>
      <p className="live-note-rule p04-statement"><strong>{k.statementLead(l)}</strong> {k.statement(l)}</p>
      <p className="p04-note">{k.valuesNote(l)}</p>
      <p className="p04-note p04-permission">{k.permission(l)}</p>
      {notice && <div id="p04-notice" className="service-status" tabIndex={-1} role="status"><p><Rich text={notice} /></p>{noticeDetail && <DiagnosticDetails l={l} detail={noticeDetail} />}</div>}
      {status === "error" && (
        <div className="live-error p04-failure" role="alert">
          <h2>{diagnosticCopy.load(l, "retention")}</h2>
          {failure && <p>{failure}</p>}
          <button type="button" className="live-secondary" onClick={() => void load()}>{sc.tryAgain(l)}</button>
        </div>
      )}
      {status === "error" && page && <p id="p04-stale" className="live-note-rule p04-readonly" role="status">{diagnosticCopy.stale(l)}</p>}
      {gone.length > 0 && (
        <div className="p04-vanished" role="status">
          {gone.map((g) => (
            <div key={g.command + g.target} className="live-note-rule">
              <p>{k.vanished(l)} {ltr(isProposalCommand(g.command) && g.context.proposal ? "#" + g.context.proposal : g.target)}</p>
              <button type="button" className="live-secondary" onClick={() => { drop(g.command, g.target); setTick((x) => x + 1); }}>{k.dismissNotice(l)}</button>
            </div>
          ))}
        </div>
      )}
      <div className="live-toolbar"><h2 id="p04-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2></div>
      {status === "loading" && !page && <p className="live-meta" role="status">{k.loading(l)}</p>}
      {page && page.state === "not_configured" && (
        <section className="live-note-rule p04-empty-site"><h2>{k.notConfigured(l)}</h2></section>
      )}
      {page && !configured && page.state !== "not_configured" && (
        <section className="live-note-rule p04-empty-site"><p><Rich text={k.stateUnknown(l, page.state)} /></p></section>
      )}
      {shared && (
        <>
          {view === "policies" && <p className="live-note-rule p04-lead">{k.policiesLead(l)}</p>}
          {view === "expiry" && <p className="live-note-rule p04-lead">{k.expiryLead(l)}</p>}
          {view === "holds" && <p className="live-note-rule p04-lead">{k.holdsLeadA(l)} <strong>{k.holdsLeadB(l)}</strong> {k.holdsLeadC(l)}</p>}
          {view === "deletion" && <p className="live-note-rule p04-lead">{k.deletionLead(l)}</p>}
          {view === "decisions" && <p className="live-note-rule p04-lead">{k.decisionsLead(l)}</p>}
          {!shared.canCommand && status !== "error" && view !== "restore" && view !== "decisions" && <p className="live-note-rule p04-readonly" role="status">{k.readOnly(l)}</p>}
          {view === "policies" ? (
            <PoliciesView {...shared} />
          ) : view === "expiry" ? (
            <ExpiryView {...shared} />
          ) : view === "holds" ? (
            <HoldsView {...shared} />
          ) : view === "deletion" ? (
            <DeletionView {...shared} />
          ) : view === "restore" ? (
            <RestoreView {...shared} />
          ) : (
            <DecisionsView {...shared} />
          )}
        </>
      )}
    </div>
  );
}
