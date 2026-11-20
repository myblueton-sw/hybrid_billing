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
import type { RecoveryPage } from "../../../../service/recovery";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { commandsAllowed, drop, forget, isProposalCommand, pruneRefusals, rowId, selectionOf, setScope, vanished, viewOf } from "./live-model";
import { Known, Rich, Stamp, ltr } from "./live-format";
import type { Announce, Shared } from "./LiveCommand";
import { SetsView } from "./LiveSets";
import { IsolationView } from "./LiveIsolation";
import { ReconciliationView } from "./LiveReconciliation";
import { MigrationView } from "./LiveMigration";
import { ResumeView } from "./LiveResume";
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

const prefixes: Record<string, string> = { sets: "set", isolation: "iso", reconciliation: "rec", resume: "res" };

export default function LiveRecovery({ locale: l, view: rawView, params, session, onSignOut }: Props) {
  const source = useAdminClient();
  const { api, detail } = useCommandDiagnostics(source, JSON.stringify([session.workspace, session.user, session.csrf, rawView]), () => { setNotice(""); setNoticeDetail(null); setTerminalDetail(null); });
  setScope(session.workspace, session.user, session.csrf);
  const view = viewOf(rawView);
  const [page, setPage] = useState<RecoveryPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [denied, setDenied] = useState(false);
  const [signedOut, setSignedOut] = useState(false);
  const [notice, setNotice] = useState("");
  const [readComplete, setReadComplete] = useState(false);
  const [noticeDetail, setNoticeDetail] = useState<CommandDiagnostic | null>(null);
  const [terminalDetail, setTerminalDetail] = useState<CommandDiagnostic | null>(null);
  const [busy, setBusy] = useState(false);
  const [, setTick] = useState(0);
  const seq = useRef(0);
  const scrolled = useRef("");
  const selection = selectionOf(params, page, view);
  const selectedId = selection.proposal ? "p03-proposal-" + selection.proposal : selection.installation && prefixes[view] ? rowId(prefixes[view], selection.installation) : "";

  const load = useCallback(async (userRefresh = false): Promise<RecoveryPage | null> => {
    const n = ++seq.current;
    setReadComplete(false);
    const generation = currentScope();
    setBusy(true);
    try {
      const got = await api.recovery();
      if (n !== seq.current || generation !== currentScope()) return null;
      pruneRefusals(got);
      setPage(got);
      setStatus("ready");
      setReadComplete(userRefresh);
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

  const refresh = (fromError = false) => {
    if (fromError) document.querySelector<HTMLButtonElement>(".p03-reload")?.focus();
    void load(true);
  };

  useLayoutEffect(() => { void load(); return () => { seq.current++; }; }, [load]);
  useEffect(() => {
    if (!selectedId || scrolled.current === selectedId) return;
    scrolled.current = selectedId;
    document.getElementById(selectedId)?.scrollIntoView({ block: "center" });
  }, [selectedId]);

  const showNotice = (text: string, reference?: CommandDiagnostic) => {
    setNoticeDetail(reference ?? null);
    setNotice(text);
    requestAnimationFrame(() => document.getElementById("p03-notice")?.focus());
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
      <div className="p03-live">
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
        <section className="live-unavailable" aria-labelledby="p03-denied">
          <h2 id="p03-denied">{k.deniedTitle(l)}</h2>
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="p03-live">
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
        l, page, api, me: session.user, selection, canCommand: commandsAllowed(page),
        onDone: (text) => { void announce(text); },
        onNotice: showNotice,
        onReload: load,
        onSignedOut: (reference?: CommandDiagnostic) => { setTerminalDetail(reference ?? null); setSignedOut(true); },
      }
    : null;
  const sectionState = !shared ? "" : view === "reconciliation" ? shared.page.reconciliation.state : view === "migration" ? shared.page.migration.state
    : view === "isolation" ? shared.page.isolation.state : view === "resume" ? shared.page.resume.state : shared.page.sets.state;
  const expected = view === "migration" ? "not_connected" : "configured";
  const gone = shared ? vanished(shared.page, view) : [];
  const failure = failureText(l, loadError);
  const lead = view === "isolation" ? k.isolationLead(l) : view === "reconciliation" ? k.reconciliationLead(l) : view === "resume" ? k.resumeLead(l) : view === "migration" ? k.migrationLead(l) : k.setsLead(l);

  return (
    <div className="p03-live" data-live={view} aria-busy={busy || undefined}>
      <p className="visually-hidden p03-read-status" role="status" aria-live="polite">{readComplete ? sc.dataReloaded(l) : ""}</p>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
      {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
      <div className="p03-scope">
        <dl className="p03-scope-facts">
          <div><dt>{k.workspace(l)}</dt><dd>{page ? <Known l={l} value={session.workspace} /> : "—"}</dd></div>
          <div><dt>{k.revision(l)}</dt><dd>{page ? <Known l={l} value={page.revision} /> : "—"}</dd></div>
          <div><dt>{k.now(l)}</dt><dd>{page ? <Stamp l={l} at={page.now} /> : "—"}</dd></div>
        </dl>
        <button type="button" className="live-secondary p03-reload" onClick={() => refresh()}>{k.reload(l)}</button>
      </div>
      <p className="live-note-rule p03-statement"><strong>{k.statementLead(l)}</strong> {k.statement(l)}</p>
      <p className="p03-permission">{k.permission(l)}</p>
      {notice && <div id="p03-notice" className="service-status" tabIndex={-1} role="status"><p><Rich text={notice} /></p>{noticeDetail && <DiagnosticDetails l={l} detail={noticeDetail} />}</div>}
      {status === "error" && (
        <div className="live-error p03-failure" role="alert">
          <h2>{diagnosticCopy.load(l, "recovery")}</h2>
          {failure && <p>{failure}</p>}
          <button type="button" className="live-secondary" onClick={() => refresh(true)}>{sc.tryAgain(l)}</button>
        </div>
      )}
      {gone.length > 0 && (
        <div className="p03-vanished" role="status">
          {gone.map((g) => (
            <div key={g.command + g.target} className="live-note-rule">
              <p>{k.vanished(l)} {ltr(isProposalCommand(g.command) && g.context.proposal ? "#" + g.context.proposal : g.target)}</p>
              <button type="button" className="live-secondary" onClick={() => { drop(g.command, g.target); setTick((x) => x + 1); }}>{k.dismissNotice(l)}</button>
            </div>
          ))}
        </div>
      )}
      <div className="live-toolbar"><h2 id="p03-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2></div>
      {status === "loading" && !page && <p className="live-meta" role="status">{k.loading(l)}</p>}
      {page && page.state === "not_configured" && (
        <section className="live-note-rule p03-empty-site"><h2>{k.notConfigured(l)}</h2></section>
      )}
      {page && !configured && page.state !== "not_configured" && (
        <section className="live-note-rule p03-empty-site"><p><Rich text={k.stateUnknown(l, page.state)} /></p></section>
      )}
      {shared && (
        <>
          {(view !== "migration" || sectionState === "not_connected") && (
            <p className="live-note-rule p03-lead">
              {lead}
              {view === "resume" && <> <strong>{k.resumeLeadStrong(l)}</strong></>}
            </p>
          )}
          {!shared.canCommand && view !== "migration" && <p className="live-note-rule p03-readonly" role="status">{k.readOnly(l)}</p>}
          {sectionState !== expected && !(view === "reconciliation" && sectionState === "not_configured") ? (
            <p className="live-empty">{sectionState ? <>{k.unknown(l)} ({ltr(sectionState)})</> : k.unknown(l)}</p>
          ) : view === "sets" ? (
            <SetsView {...shared} />
          ) : view === "isolation" ? (
            <IsolationView {...shared} />
          ) : view === "reconciliation" ? (
            sectionState === "configured" ? <ReconciliationView {...shared} /> : <p className="live-empty">{k.reconciliationEmpty(l)}</p>
          ) : view === "migration" ? (
            <MigrationView {...shared} />
          ) : (
            <ResumeView {...shared} />
          )}
        </>
      )}
    </div>
  );
}
