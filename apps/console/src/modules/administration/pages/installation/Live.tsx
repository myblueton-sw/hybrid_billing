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
import type { InstallationPage } from "../../../../service/installation";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { forget, pruneRefusals, reportPassed, setScope, viewOf } from "./live-model";
import { Code, Known, ltr } from "./live-format";
import { PlanView, now } from "./LivePlan";
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

function failureText(l: Locale, e: unknown): { title: string; body: string } {
  if (e instanceof AdminError) {
    if (e.failure === "limited") return { title: diagnosticCopy.load(l, "installation"), body: sc.limited(l, retryPhrase(l, e.retryAfter || 60)) };
    if (e.failure === "busy" || e.failure === "unavailable") return { title: diagnosticCopy.load(l, "installation"), body: diagnosticCopy.unavailable(l) };
  }
  return { title: diagnosticCopy.load(l, "installation"), body: "" };
}

export default function LiveInstallation({ locale: l, view: rawView, params, session, onSignOut }: Props) {
  const source = useAdminClient();
  const { api, detail } = useCommandDiagnostics(source, JSON.stringify([session.workspace, session.user, session.csrf, rawView]), () => { setNotice(""); setNoticeDetail(null); setTerminalDetail(null); });
  setScope(session.workspace, session.user, session.csrf);
  const view = viewOf(rawView);
  const [page, setPage] = useState<InstallationPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [denied, setDenied] = useState(false);
  const [signedOut, setSignedOut] = useState(false);
  const [notice, setNotice] = useState("");
  const [readComplete, setReadComplete] = useState(false);
  const [noticeDetail, setNoticeDetail] = useState<CommandDiagnostic | null>(null);
  const [terminalDetail, setTerminalDetail] = useState<CommandDiagnostic | null>(null);
  const [busy, setBusy] = useState(false);
  const seq = useRef(0);
  const scrolled = useRef(false);
  const asked = params.get("proposal") ?? "";
  const selected = /^[1-9][0-9]*$/.test(asked) && page?.proposals.some((p) => String(p.number) === asked) ? asked : "";

  const load = useCallback(async (userRefresh = false): Promise<boolean> => {
    const n = ++seq.current;
    setReadComplete(false);
    const generation = currentScope();
    setBusy(true);
    try {
      const got = await api.installation();
      if (n !== seq.current || generation !== currentScope()) return false;
      pruneRefusals(got, session.user);
      setPage(got);
      setStatus("ready");
      setReadComplete(userRefresh);
      setLoadError(null);
      return true;
    } catch (e) {
      if (n !== seq.current || generation !== currentScope()) return false;
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut(true);
      setLoadError(e);
      setStatus("error");
      return false;
    } finally {
      if (n === seq.current && generation === currentScope()) setBusy(false);
    }
  }, [api, session.user]);

  const refresh = (fromError = false) => {
    if (fromError) document.querySelector<HTMLButtonElement>(".p01-reload")?.focus();
    void load(true);
  };

  useLayoutEffect(() => { void load(); return () => { seq.current++; }; }, [load]);
  useEffect(() => {
    if (!selected || scrolled.current) return;
    scrolled.current = true;
    document.getElementById("p01-proposal-" + selected)?.scrollIntoView({ block: "center" });
  }, [selected]);

  const showNotice = (text: string, reference?: CommandDiagnostic) => {
    setNoticeDetail(reference ?? null);
    setNotice(text);
    requestAnimationFrame(() => document.getElementById("p01-notice")?.focus());
  };

  if (denied) {
    return (
      <div className="p01-live">
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
        {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
        <section className="live-unavailable" aria-labelledby="p01-denied">
          <h2 id="p01-denied">{k.deniedTitle(l)}</h2>
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="p01-live">
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
  const shared = page && configured
    ? {
        l, page, api, me: session.user, selected,
        onDone: (text: string) => { showNotice(text); void load(); },
        onNotice: showNotice,
        onReload: load,
        onDenied: () => setDenied(true),
        onSignedOut: (reference?: CommandDiagnostic) => { setTerminalDetail(reference ?? null); setSignedOut(true); },
      }
    : null;

  return (
    <div className="p01-live" data-live={view} aria-busy={busy || undefined}>
      <p className="visually-hidden p01-read-status" role="status" aria-live="polite">{readComplete ? sc.dataReloaded(l) : ""}</p>
        <CommandDiagnostics l={l} detail={terminalDetail ?? detail} />
      {loadError != null && <ErrorDiagnostics l={l} error={loadError} />}
      <div className="p01-scope">
        <dl className="p01-scope-facts">
          <div>
            <dt>{k.site(l)}</dt>
            <dd>{configured && page ? <><Known l={l} value={page.site.id} /> <span className="live-quiet">{page.site.platform ? ltr(page.site.platform) : k.unknown(l)}</span></> : "—"}</dd>
          </div>
          <div><dt>{k.revision(l)}</dt><dd>{page ? <Known l={l} value={page.revision} /> : "—"}</dd></div>
          <div><dt>{k.now(l)}</dt><dd>{now(l, page)}</dd></div>
        </dl>
        <button type="button" className="live-secondary p01-reload" onClick={() => refresh()}>{k.reload(l)}</button>
      </div>
      <p className="live-note-rule p01-statement"><strong>{k.statementLead(l)}</strong> {k.statement(l)}</p>
      {notice && <div id="p01-notice" className="service-status" tabIndex={-1} role="status"><p>{notice}</p>{noticeDetail && <DiagnosticDetails l={l} detail={noticeDetail} />}</div>}
      {status === "error" && (
        <div className="live-error p01-failure" role="alert">
          <h2>{failureText(l, loadError).title}</h2>
          {failureText(l, loadError).body && <p>{failureText(l, loadError).body}</p>}
          <button type="button" className="live-secondary" onClick={() => refresh(true)}>{sc.tryAgain(l)}</button>
        </div>
      )}
      <div className="live-toolbar"><h2 id="p01-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2></div>
      {status === "loading" && !page && <p className="live-meta" role="status">{k.loading(l)}</p>}
      {page && page.state === "not_configured" && (
        <section className="live-note-rule p01-empty-site"><h2>{k.notConfigured(l)}</h2></section>
      )}
      {page && !configured && page.state !== "not_configured" && (
        <section className="live-note-rule p01-empty-site"><p>{k.stateUnknown(l, page.state)}</p></section>
      )}
      {view === "support" || view === "access" ? (
        <Unconnected l={l} view={view} />
      ) : shared && view === "diagnostics" ? (
        <Diagnostics l={l} page={shared.page} />
      ) : shared ? (
        <PlanView {...shared} />
      ) : null}
    </div>
  );
}

function Unconnected({ l, view }: { l: Locale; view: string }) {
  return (
    <section className="p01-unconnected" aria-labelledby="p01-unconnected-title">
      <h3 id="p01-unconnected-title">{k.unconnected(l)}</h3>
      <p>{k.willHold(l)}</p>
      <ul>{(view === "support" ? k.supportItems(l) : k.accessItems(l)).map((x) => <li key={x}>{x}</li>)}</ul>
    </section>
  );
}

function Diagnostics({ l, page }: { l: Locale; page: InstallationPage }) {
  return (
    <>
      <section className="p01-destinations" aria-labelledby="p01-destinations-title">
        <h3 id="p01-destinations-title">{k.destinations(l)}</h3>
        {page.destinations.length === 0 ? (
          <p className="live-empty">{k.noDestinations(l)}</p>
        ) : (
          <div className="live-table">
            <table className="p01-table">
              <colgroup><col className="p01-c20" /><col className="p01-c30" /><col className="p01-c10" /><col className="p01-c40" /></colgroup>
              <thead>
                <tr><th scope="col">{k.colId(l)}</th><th scope="col">{k.colHost(l)}</th><th scope="col" className="p01-num">{k.colPort(l)}</th><th scope="col">{k.colRanges(l)}</th></tr>
              </thead>
              <tbody>
                {page.destinations.map((d, i) => (
                  <tr key={d.id || i}>
                    <td className="p01-first"><Known l={l} value={d.id} /></td>
                    <td data-label={k.colHost(l)} className="p01-wrap"><Known l={l} value={d.host} mono={false} /></td>
                    <td data-label={k.colPort(l)} className="p01-num">{d.port === null ? k.unknown(l) : <bdi dir="ltr">{d.port}</bdi>}</td>
                    <td data-label={k.colRanges(l)}>{d.ranges.length === 0 ? k.none(l) : d.ranges.map((r) => <span key={r} className="live-sub">{ltr(r)}</span>)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <section className="p01-reports" aria-labelledby="p01-reports-title">
        <h3 id="p01-reports-title">{k.reports(l)}</h3>
        {page.reports.length === 0 ? (
          <p className="live-empty">{k.noReports(l)}</p>
        ) : (
          <div className="live-table">
            <table className="p01-table">
              <colgroup>
                <col className="p01-c12" /><col className="p01-c18" /><col className="p01-c16" /><col className="p01-c10" /><col className="p01-c12" /><col className="p01-c32" />
              </colgroup>
              <thead>
                <tr>
                  <th scope="col">{k.colCheck(l)}</th><th scope="col">{k.colTarget(l)}</th><th scope="col">{k.colStatus(l)}</th>
                  <th scope="col">{k.colRequired(l)}</th><th scope="col">{k.colRoute(l)}</th><th scope="col">{k.colNext(l)}</th>
                </tr>
              </thead>
              <tbody>
                {page.reports.map((r, i) => (
                  <tr key={i}>
                    <td className="p01-first"><Code l={l} value={r.kind} say={k.check} /></td>
                    <td data-label={k.colTarget(l)}>
                      <Known l={l} value={r.destination} />
                      <span className="live-sub">{r.address ? <bdi dir="ltr">{r.address}</bdi> : k.notResolved(l)}</span>
                    </td>
                    <td data-label={k.colStatus(l)}><span className="p01-tag"><Code l={l} value={r.failure} say={k.failure} /></span></td>
                    <td data-label={k.colRequired(l)}>{reportPassed(r.failure) ? k.none(l) : k.required(l)}</td>
                    <td data-label={k.colRoute(l)}>{r.via ? <>{k.via(l)} {ltr(r.via)}</> : k.direct(l)}</td>
                    <td data-label={k.colNext(l)}>
                      <Code l={l} value={r.cause} say={k.cause} />
                      {r.steps.length > 0 && <ol className="p01-steps">{r.steps.map((s) => <li key={s}><Code l={l} value={s} say={k.step} /></li>)}</ol>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
      <section className="p01-unconnected" aria-labelledby="p01-requests-title">
        <h3 id="p01-requests-title">{k.requests(l)}</h3>
        <p>{k.requestsBody(l)}</p>
      </section>
    </>
  );
}
