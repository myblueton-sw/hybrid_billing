import { DiagnosticDetails } from "../../../../service/CommandDiagnostics";
import type { CommandDiagnostic } from "../../../../service/command-diagnostics";
import { currentScope } from "./live-model";
import "../../../../service/command-diagnostics.css";
import { CommandDiagnostics, ErrorDiagnostics, useCommandDiagnostics } from "../../../../service/CommandDiagnostics";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, attempt } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { EvidencePage } from "../../../../service/release-evidence";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import {
  commandsAllowed, drop, forget, isProposalCommand, modeOf, pruneRefusals, readDenied, readSucceeded, revokedNow, rowId, selectionOf, setScope, vanished, viewOf, type Selection,
} from "./live-model";
import { Absent, Known, Stamp, Template, Unexpected, ltr, takeIntent, useAvailableWidth } from "./live-format";
import { ErrorDetail, type Announce, type Shared } from "./LiveCommand";
import { MatrixView } from "./LiveMatrix";
import { RequirementsView } from "./LiveRequirements";
import { ArtifactsView } from "./LiveArtifacts";
import { LoadView } from "./LiveLoad";
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

type Failure = { error: unknown; request: string };

function failureLines(l: Locale, e: unknown): { text: string; code: string; next: string } {
  if (e instanceof AdminError) {
    if (e.failure === "limited") {
      const when = retryPhrase(l, e.retryAfter || 60);
      return { text: k.failedBusy(l), code: e.code, next: k.nextWait(l, when) };
    }
    return { text: e.failure === "busy" || e.failure === "unavailable" ? k.failedBusy(l) : "", code: e.code, next: k.nextTry(l) };
  }
  return { text: "", code: "", next: k.nextTry(l) };
}

function selectedId(view: string, s: Selection): string {
  if (view === "matrix" && s.path) return rowId("path", s.path);
  if (view === "requirements" && s.requirement) return rowId("requirement", s.requirement);
  if (view === "artifacts" && s.build) return rowId("build", s.build);
  if (view === "load" && s.profile) return rowId("profile", s.profile);
  if (view === "decisions" && s.proposal) return rowId("proposal", s.proposal);
  return "";
}

function originOf(page: EvidencePage, view: string): string {
  if (view === "requirements") return page.requirements.origin;
  if (view === "artifacts") return page.artifacts.origin;
  if (view === "load") return page.load.origin;
  if (view === "decisions") return page.decisions.origin;
  return page.matrix.origin;
}

function OriginNote({ l, origin }: { l: Locale; origin: string }) {
  if (origin === "sample_seed") return <p className="p05-note"><strong>{k.originSampleLead(l)}</strong> {k.originSample(l)}</p>;
  if (origin === "empty") return <p className="p05-note">{k.originEmpty(l)}</p>;
  return <p className="p05-note"><Unexpected l={l} code="origin" /></p>;
}

function CandidateLine({ l, page }: { l: Locale; page: EvidencePage }) {
  const c = page.candidate;
  if (!c) return <p className="p05-candidate">{k.noCandidate(l)}</p>;
  return (
    <>
      <p className="p05-candidate"><Template text={k.candidateLine(l)} slots={{ version: <Known l={l} value={c.version} />, build: <Known l={l} value={c.build} /> }} /></p>
      {c.release !== "not_connected" && (
        <p className="p05-candidate"><Unexpected l={l} code="release" /></p>
      )}
    </>
  );
}

export default function LiveReleaseEvidence({ locale: l, view: rawView, params, session, onSignOut }: Props) {
  const source = useAdminClient();
  const { api, detail } = useCommandDiagnostics(source, JSON.stringify([session.workspace, session.user, session.csrf, rawView]), () => { setNotice(""); setNoticeDetail(null); setTerminalDetail(null); });
  setScope(session.workspace, session.user, session.csrf);
  const view = viewOf(rawView);
  const [page, setPage] = useState<EvidencePage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [failure, setFailure] = useState<Failure | null>(null);
  const [access, setAccess] = useState<"ok" | "denied" | "revoked">("ok");
  const [signedOut, setSignedOut] = useState(false);
  const [notice, setNotice] = useState("");
  const [noticeDetail, setNoticeDetail] = useState<CommandDiagnostic | null>(null);
  const [terminalDetail, setTerminalDetail] = useState<CommandDiagnostic | null>(null);
  const [announce, setAnnounce] = useState("");
  const [busy, setBusy] = useState(false);
  const [, setTick] = useState(0);
  const seq = useRef(0);
  const scrolled = useRef("");
  const root = useRef<HTMLDivElement>(null);
  const width = useAvailableWidth(root, access);
  const mode = modeOf(width);
  const selection = selectionOf(params, page, view);
  const selected = selectedId(view, selection);
  const detailKey = view === "matrix" ? selection.path : view === "requirements" ? selection.requirement : "";
  const lastDetail = useRef("");
  const lastMode = useRef(mode);

  const load = useCallback(async (): Promise<EvidencePage | null> => {
    const n = ++seq.current;
    const generation = currentScope();
    const request = attempt().request;
    setBusy(true);
    try {
      const got = await api.releaseEvidence(request);
      if (n !== seq.current || generation !== currentScope()) return null;
      readSucceeded();
      pruneRefusals(got);
      setPage(got);
      setAccess("ok");
      setStatus("ready");
      setFailure(null);
      return got;
    } catch (e) {
      if (n !== seq.current || generation !== currentScope()) return null;
      if (e instanceof AdminError && e.failure === "denied") {
        setFailure({ error: e, request });
        const kind = readDenied();
        if (kind === "revoked") setPage(null);
        setAccess(kind);
        setStatus("error");
        return null;
      }
      if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut(true);
      setFailure({ error: e, request });
      setStatus("error");
      return null;
    } finally {
      if (n === seq.current && generation === currentScope()) setBusy(false);
    }
  }, [api]);

  useLayoutEffect(() => { void load(); return () => { seq.current++; }; }, [load]);
  useEffect(() => {
    if (!selected || scrolled.current === selected) return;
    scrolled.current = selected;
    document.getElementById(selected)?.scrollIntoView({ block: "nearest" });
  }, [selected]);
  useEffect(() => {
    const intent = takeIntent();
    const before = lastDetail.current;
    lastDetail.current = detailKey;
    if (detailKey && intent?.kind === "open") {
      document.getElementById("p05-detail-title")?.focus({ focusVisible: !intent.pointer } as FocusOptions);
      return;
    }
    if (!detailKey && before && intent?.kind === "close") {
      const target = document.getElementById(intent.opener) ?? document.getElementById("p05-view-title");
      target?.focus({ focusVisible: !intent.pointer } as FocusOptions);
      return;
    }
    if (detailKey && detailKey !== before) setAnnounce(k.opened(l));
    else if (!detailKey && before) setAnnounce(k.closedDetail(l));
  }, [detailKey, l]);
  useEffect(() => {
    if (lastMode.current === mode) return;
    lastMode.current = mode;
    if (!detailKey) return;
    const active = document.activeElement;
    if (!active || active === document.body || !active.isConnected) document.getElementById("p05-detail-title")?.focus();
  }, [mode, detailKey]);

  const showNotice = (text: string, reference?: CommandDiagnostic) => {
    setNoticeDetail(reference ?? null);
    setNotice(text);
    requestAnimationFrame(() => document.getElementById("p05-notice")?.focus());
  };

  const done = async (text: Announce) => {
    if (typeof text === "string") {
      showNotice(text);
      void load();
      return;
    }
    const got = await load();
    showNotice(text(got));
  };

  const scopeBar = (
    <div className="p05-scope">
      <dl className="p05-scope-facts">
        {page && (
          <>
            <div><dt>{k.workspace(l)}</dt><dd><Known l={l} value={session.workspace} /></dd></div>
            <div><dt>{k.revision(l)}</dt><dd><Known l={l} value={page.revision} /></dd></div>
            <div><dt>{k.now(l)}</dt><dd><Stamp l={l} at={page.now} /></dd></div>
            {page.state === "configured" && (
              <div>
                <dt>{k.candidateFact(l)}</dt>
                <dd>{page.candidate ? <Template text={k.candidateValue(l)} slots={{ version: <Known l={l} value={page.candidate.version} />, build: <Known l={l} value={page.candidate.build} /> }} /> : <Absent>{k.noneSelected(l)}</Absent>}</dd>
              </div>
            )}
          </>
        )}
      </dl>
      <button type="button" className="live-secondary p05-reload" onClick={() => void load()}>{k.reload(l)}</button>
    </div>
  );

  if (access === "revoked" || (access === "denied" && revokedNow())) {
    return (
      <div className="p05-live" ref={root} aria-busy={busy || undefined}>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {failure && <ErrorDiagnostics l={l} error={failure.error} />}
        {scopeBar}
        <section className="live-unavailable p05-revoked" aria-labelledby="p05-revoked">
          <h2 id="p05-revoked">{k.revokedTitle(l)}</h2>
          <p>{k.revoked(l)}</p>
        </section>
      </div>
    );
  }
  if (access === "denied") {
    return (
      <div className="p05-live" ref={root} aria-busy={busy || undefined}>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {failure && <ErrorDiagnostics l={l} error={failure.error} />}
        {scopeBar}
        <section className="live-unavailable" aria-labelledby="p05-denied">
          <h2 id="p05-denied">{k.deniedTitle(l)}</h2>
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="p05-live" ref={root}>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {failure && <ErrorDiagnostics l={l} error={failure.error} />}
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
        l, page, api, me: session.user, selection, canCommand: commandsAllowed(page) && status !== "error", mode,
        onDone: (text) => { void done(text); },
        onNotice: showNotice,
        onReload: load,
        onSignedOut: (reference?: CommandDiagnostic) => { setTerminalDetail(reference ?? null); setSignedOut(true); },
      }
    : null;
  const gone = shared ? vanished(shared.page, view) : [];
  const lines = failure ? failureLines(l, failure.error) : null;
  const step = mode === "step" && !!detailKey;

  return (
    <div className="p05-live" data-live={view} data-mode={mode} ref={root} aria-busy={busy || undefined}>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
      {scopeBar}
      {page && configured && (
        <div className="p05-frame">
          <div className="p05-frame-main">
            <p className="p05-statement"><strong>{k.statementLead(l)}</strong> {k.statement(l)}</p>
            <CandidateLine l={l} page={page} />
          </div>
          <div className="p05-frame-notes">
            <OriginNote l={l} origin={originOf(page, view)} />
            <p className="p05-note">{k.openDecisions(l)}</p>
            <p className="p05-note">{k.permission(l)}</p>
          </div>
        </div>
      )}
      <p className="visually-hidden" role="status">{announce}</p>
      {notice && <div id="p05-notice" className="p05-notice" tabIndex={-1} role="status"><p>{notice}</p>{noticeDetail && <DiagnosticDetails l={l} detail={noticeDetail} />}</div>}
      {status === "error" && failure && lines && (
        <div className="live-error p05-failure" role="alert">
          <h2>{k.failedTitle(l)}</h2>
          {lines.text && <p>{lines.text}</p>}
          <p>{k.failedSpec(l)}</p>
          <ErrorDetail l={l} code={lines.code} request={failure.error instanceof AdminError ? failure.error.request : ""} client={failure.request} next={lines.next} />
          <button type="button" className="live-secondary" onClick={() => void load()}>{sc.tryAgain(l)}</button>
        </div>
      )}
      {gone.length > 0 && !step && (
        <div className="p05-vanished" role="status">
          {gone.map((g) => (
            <div key={g.command + g.target} className="p05-vanished-item">
              <p>{k.vanished(l)} {ltr(isProposalCommand(g.command) && g.context.proposal ? "#" + g.context.proposal : g.target)}</p>
              <button type="button" className="live-secondary" onClick={() => { drop(g.command, g.target); setTick((x) => x + 1); }}>{k.dismiss(l)}</button>
            </div>
          ))}
        </div>
      )}
      <h2 id="p05-view-title" className="p05-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2>
      {status === "loading" && !page && <p className="p05-loading" role="status">{k.loading(l)}</p>}
      {page && page.state === "not_configured" && <p className="p05-site">{k.notConfigured(l)}</p>}
      {page && !configured && page.state !== "not_configured" && (
        <p className="p05-site"><Unexpected l={l} code="page.state" /></p>
      )}
      {shared && (
        <>
          {!commandsAllowed(shared.page) && !step && view !== "matrix" && view !== "artifacts" && <p className="p05-readonly" role="status">{k.readOnly(l)}</p>}
          {view === "requirements" ? (
            <RequirementsView {...shared} />
          ) : view === "artifacts" ? (
            <ArtifactsView {...shared} />
          ) : view === "load" ? (
            <LoadView {...shared} />
          ) : view === "decisions" ? (
            <DecisionsView {...shared} />
          ) : (
            <MatrixView {...shared} />
          )}
        </>
      )}
    </div>
  );
}

