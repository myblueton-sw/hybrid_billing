import { useCallback, useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { IntegrationsPage } from "../../../../service/integrations";
import type { Session } from "../../../../service/api";
import { c as sc, retryPhrase } from "../../../../service/copy";
import { k } from "./live-copy";
import { commandsAllowed, drop, forget, pruneRefusals, selectionOf, setScope, vanished, viewOf } from "./live-model";
import { Known, fsi, ltr, stamp } from "./live-format";
import type { Shared } from "./LiveCommand";
import { ClientsView } from "./LiveClients";
import { ModulesView } from "./LiveModules";
import { ChannelsView } from "./LiveChannels";
import { EventsView } from "./LiveEvents";
import { OwnPendingList } from "./LivePending";
import { forgetLost, heldRights, pruneLost, withheld } from "./live-rights";
import { r, rightList } from "./rights-copy";
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
    if (e.failure === "busy" || e.failure === "unavailable") return r.notResponding(l);
  }
  return "";
}

const sections = { modules: "modules", channels: "channels", events: "events" } as const;

export default function LiveIntegrations({ locale: l, view: rawView, params, session, onSignOut }: Props) {
  const api = useAdminClient();
  setScope(session.workspace, session.user, session.csrf);
  const view = viewOf(rawView);
  const [page, setPage] = useState<IntegrationsPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [loadError, setLoadError] = useState<unknown>(null);
  const [denied, setDenied] = useState(false);
  const [signedOut, setSignedOut] = useState(false);
  const [lostAccess, setLostAccess] = useState(false);
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [, setTick] = useState(0);
  const seq = useRef(0);
  const scrolled = useRef(false);
  const selection = selectionOf(params, page, view);
  const selectedId = selection.delivery ? "p02-delivery-" + selection.delivery : selection.proposal ? "p02-proposal-" + selection.proposal : selection.subscription && view === "channels" ? "p02-sub-" + selection.subscription : "";

  const load = useCallback(async (): Promise<IntegrationsPage | null> => {
    const n = ++seq.current;
    setBusy(true);
    try {
      const got = await api.integrations();
      if (n !== seq.current) return null;
      pruneRefusals(got, session.user);
      pruneLost(got);
      setPage(got);
      setStatus("ready");
      setLoadError(null);
      return got;
    } catch (e) {
      if (n !== seq.current) return null;
      if (e instanceof AdminError && e.failure === "denied") setDenied(true);
      if (e instanceof AdminError && e.failure === "unauthenticated") setSignedOut(true);
      setLoadError(e);
      setStatus("error");
      return null;
    } finally {
      if (n === seq.current) setBusy(false);
    }
  }, [api, session.user]);

  useEffect(() => { void load(); }, [load]);
  useEffect(() => {
    if (!selectedId || scrolled.current) return;
    scrolled.current = true;
    document.getElementById(selectedId)?.scrollIntoView({ block: "center" });
  }, [selectedId]);

  const showNotice = (text: string) => {
    setNotice(text);
    requestAnimationFrame(() => document.getElementById("p02-notice")?.focus());
  };

  if (denied) {
    return (
      <div className="p02-live">
        <section className="live-unavailable" aria-labelledby="p02-denied">
          <h2 id="p02-denied" tabIndex={-1} ref={(h) => { if (h && lostAccess) h.focus(); }}>{k.deniedTitle(l)}</h2>
          {lostAccess && <p className="p02-lost-access" role="status">{r.lastActionLost(l)}</p>}
          <p>{k.denied(l)}</p>
        </section>
      </div>
    );
  }
  if (signedOut) {
    return (
      <div className="p02-live">
        <div className="live-error" role="alert">
          <h2>{k.sessionEnded(l)}</h2>
          <button type="button" className="live-secondary" onClick={() => { forget(); forgetLost(); onSignOut(); }}>{k.signInAgain(l)}</button>
        </div>
      </div>
    );
  }

  const configured = page?.state === "configured";
  const shared: Shared | null = page && configured
    ? {
        l, page, api, me: session.user, selection, canCommand: commandsAllowed(page) && status !== "error",
        onDone: (text: string) => { showNotice(text); void load(); },
        onNotice: showNotice,
        onReload: load,
        onSignedOut: () => setSignedOut(true),
        onLostAccess: () => setLostAccess(true),
      }
    : null;
  const hidden = shared ? withheld(shared.page, view) : false;
  const held = page ? heldRights(page) : [];
  const sectionKey = view === "clients" ? "" : sections[view as keyof typeof sections];
  const sectionState = shared && sectionKey ? shared.page[sectionKey].state : "configured";
  const gone = shared ? vanished(shared.page, view) : [];
  const failure = failureText(l, loadError);

  return (
    <div className="p02-live" data-live={view} aria-busy={busy || undefined}>
      <div className="p02-scope">
        <dl className="p02-scope-facts">
          <div><dt>{k.workspace(l)}</dt><dd>{page ? <Known l={l} value={session.workspace} /> : "—"}</dd></div>
          <div><dt>{k.revision(l)}</dt><dd>{page ? <Known l={l} value={page.revision} /> : "—"}</dd></div>
          <div><dt>{k.now(l)}</dt><dd>{page ? fsi(stamp(l, page.now)) : "—"}</dd></div>
        </dl>
        <button type="button" className="live-secondary p02-reload" onClick={() => void load()}>{k.reload(l)}</button>
      </div>
      <p className="live-note-rule p02-statement"><strong>{k.statementLead(l)}</strong> {k.statement(l)}</p>
      {page && configured && (
        page.rights === null ? (
          <p className="p02-rights" id="p02-rights" data-rights="unknown">{r.rightsUnknown(l)}</p>
        ) : held.length === 0 ? (
          <p className="p02-rights" id="p02-rights" data-rights="none"><strong>{r.rightsNoneLead(l)}</strong> {r.rightsNoneRest(l)}</p>
        ) : (
          <p className="p02-rights" id="p02-rights" data-rights={held.join(" ")}><strong>{r.rightsLead(l)}</strong> {rightList(l, held)}.</p>
        )
      )}
      {notice && <div id="p02-notice" className="service-status" tabIndex={-1} role="status"><p>{notice}</p></div>}
      {status === "error" && (
        <div className="live-error p02-failure" role="alert">
          <h2>{r.loadFailed(l)}</h2>
          {failure && <p>{failure}</p>}
          <button type="button" className="live-secondary" onClick={() => void load()}>{sc.tryAgain(l)}</button>
        </div>
      )}
      <div className="live-toolbar"><h2 id="p02-view-title" tabIndex={-1}>{k.viewTitle(l, view)}</h2></div>
      {status === "loading" && !page && <p className="live-meta" role="status">{k.loading(l)}</p>}
      {page && page.state === "not_configured" && (
        <section className="live-note-rule p02-empty-site"><h2>{k.notConfigured(l)}</h2></section>
      )}
      {page && !configured && page.state !== "not_configured" && (
        <section className="live-note-rule p02-empty-site"><p>{k.stateUnknown(l, page.state)}</p></section>
      )}
      {shared && hidden && (
        <div className="p02-withheld-area" data-state="not_permitted">
          <p className="p02-area-withheld">{r.withheld(l, view)}</p>
          <OwnPendingList {...shared} view={view} />
        </div>
      )}
      {shared && !hidden && (
        <>
          <p className="live-note-rule p02-lead">
            {view === "modules" ? k.modulesLead(l) : view === "channels" ? k.channelsLead(l) : view === "events" ? k.eventsLead(l) : k.clientsLead(l)}
          </p>
          {!shared.canCommand && <p className="live-note-rule p02-readonly" role="status">{status === "error" ? r.readOnlyAfterFailure(l) : k.readOnly(l)}</p>}
          {gone.length > 0 && (
            <div className="p02-vanished" role="status">
              {gone.map((g) => (
                <div key={g.command + g.target} className="live-note-rule">
                  <p>
                    {g.command === "integrations.letter_approve_redeliver" || g.command === "integrations.letter_approve_dismiss"
                      ? k.vanishedLetter(l) + (g.command === "integrations.letter_approve_redeliver" ? " " + k.vanishedHint(l, g.context.event ?? "") : "")
                      : k.vanishedOther(l)}{" "}
                    {ltr(g.target)}
                  </p>
                  <button type="button" className="service-button-link" onClick={() => { drop(g.command, g.target); setTick((x) => x + 1); }}>{k.dismissNotice(l)}</button>
                </div>
              ))}
            </div>
          )}
          {sectionState !== "configured" && sectionState !== "not_configured" && (
            <p className="live-empty">{k.unknown(l)} ({ltr(sectionState)})</p>
          )}
          {view === "clients" ? (
            <ClientsView {...shared} />
          ) : view === "modules" ? (
            sectionState === "configured" ? <ModulesView {...shared} /> : sectionState === "not_configured" ? <p className="live-empty">{k.sectionEmpty(l, "modules")}</p> : null
          ) : view === "channels" ? (
            <ChannelsView {...shared} section={sectionState} />
          ) : sectionState === "configured" ? (
            <EventsView {...shared} />
          ) : sectionState === "not_configured" ? (
            <p className="live-empty">{k.sectionEmpty(l, "events")}</p>
          ) : null}
        </>
      )}
    </div>
  );
}
