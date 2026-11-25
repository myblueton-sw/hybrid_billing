import { useCallback, useEffect, useRef, useState } from "react";
import { diagnosticCopy } from "../../../../service/diagnostic-copy";
import type { Locale } from "../../../../shared/types";
import { AdminError, sequence, type Outcome, type PolicyPage, type TrustChange } from "../../../../service/admin";
import { useAdminClient } from "../../../../service/context";
import type { Session } from "../../../../service/api";
import { c } from "../../../../service/copy";
import { useWide } from "../users/Live";
import { w } from "../users/live-copy";
import { p } from "./live-copy";
import { policyFailure } from "./live-model";
import { ChangesHead, Conditions, CurrentRules, Detail, Lists, Revoked, ShortConditions } from "./TrustChanges";
import { ProposeForm, type Proposed } from "./TrustActions";
import { AfterActivation } from "./TrustAfter";
import { ConfirmDialog, type Confirming } from "./TrustConfirm";
import { ty } from "./trust-copy";
import type { ActionKey } from "./trust-model";
import "../users/live.css";
import "./trust.css";

type Form = { revise: TrustChange | null; initial: Proposed | null };

function Unavailable({ l, title, body, action }: { l: Locale; title: string; body: string; action?: { label: string; go: () => void } }) {
  return (
    <div className="live-users">
      <section className="live-unavailable" aria-labelledby="policy-unavailable">
        <h2 id="policy-unavailable">{title}</h2>
        <p>{body}</p>
        {action && <button type="button" className="live-secondary" onClick={action.go}>{action.label}</button>}
      </section>
    </div>
  );
}

function Providers({ l, onView }: { l: Locale; onView: (view: string) => void }) {
  const rows: [string, boolean, string][] = [
    [p.password(l), true, p.passwordNote(l)],
    ["SAML", false, ""],
    ["OIDC", false, ""],
    [p.ldap(l), false, ""],
  ];
  return (
    <div className="live-users" data-live="providers">
      <section className="live-list" aria-labelledby="policy-methods">
        <h2 id="policy-methods" className="live-title">{p.methodsTitle(l)}</h2>
        <p className="live-impact">{p.methodsLead(l)}</p>
        <div className="security-table-scroll live-table">
          <table role="table">
            <thead role="rowgroup"><tr role="row"><th scope="col" role="columnheader">{p.colMethod(l)}</th><th scope="col" role="columnheader">{p.colStatus(l)}</th><th scope="col" role="columnheader">{p.colNotes(l)}</th></tr></thead>
            <tbody role="rowgroup">
              {rows.map(([name, on, note]) => (
                <tr key={name} role="row">
                  <td role="cell" className="live-user"><bdi>{name}</bdi></td>
                  <td role="cell" data-label={p.colStatus(l)} className={on ? "live-strong" : "live-quiet"}>{on ? p.active(l) : p.notAvail(l)}</td>
                  <td role="cell" data-label={p.colNotes(l)} className="live-quiet">{note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="live-buttons"><button type="button" className="live-secondary" onClick={() => onView("policies")}>{p.goPolicy(l)}</button></div>
      </section>
    </div>
  );
}

export default function LivePolicy({ locale: l, view, session, onSignOut, onView, onUsers }: {
  locale: Locale; view: string; session: Session; onSignOut: () => void; onView: (view: string) => void; onUsers: () => void;
}) {
  const api = useAdminClient();
  const wide = useWide();
  const order = useRef(sequence());
  const [page, setPage] = useState<PolicyPage | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");
  const [error, setError] = useState<unknown>(null);
  const [revoked, setRevoked] = useState(false);
  const [selected, setSelected] = useState("");
  const [form, setForm] = useState<Form | null>(null);
  const [confirm, setConfirm] = useState<{ k: Confirming; opener: HTMLElement | null } | null>(null);
  const [after, setAfter] = useState<Outcome | null>(null);
  const [announce, setAnnounce] = useState("");
  const [focusTo, setFocusTo] = useState("");
  const had = useRef(false);
  useEffect(() => {
    if (!focusTo) return;
    const id = requestAnimationFrame(() => {
      (document.querySelector<HTMLElement>(focusTo) ?? document.querySelector<HTMLElement>("#m02-changes"))?.focus();
      setFocusTo("");
    });
    return () => cancelAnimationFrame(id);
  }, [focusTo, page, selected, form]);
  const say = (text: string) => { setAnnounce(""); setTimeout(() => setAnnounce(text), 50); };
  const clear = () => { setPage(null); setSelected(""); setForm(null); setConfirm(null); };
  const load = useCallback(() => {
    const n = order.current.next();
    setStatus("loading");
    return api.policy().then(
      (got) => {
        if (!order.current.latest(n)) return;
        had.current = true;
        setPage(got);
        setRevoked(false);
        setStatus("ready");
        setError(null);
      },
      (e) => {
        if (!order.current.latest(n)) return;
        if (had.current && e instanceof AdminError && (e.failure === "denied" || e.failure === "unauthenticated")) {
          setPage(null); setSelected(""); setForm(null); setConfirm(null);
          setRevoked(e.failure === "denied");
        }
        setError(e);
        setStatus("error");
      },
    );
  }, [api]);
  useEffect(() => { if (view === "policies") void load(); }, [load, view]);
  if (view === "providers") return <Providers l={l} onView={onView} />;
  if (view === "test") return <Unavailable l={l} title={p.testTitle(l)} body={p.testBody(l)} />;
  if (view === "keys") return <Unavailable l={l} title={p.keysTitle(l)} body={p.keysBody(l)} />;
  if (view === "sync") return <Unavailable l={l} title={p.syncTitle(l)} body={p.syncBody(l)} action={{ label: p.goUsers(l), go: onUsers }} />;
  if (view === "recovery") return <Unavailable l={l} title={p.recoveryTitle(l)} body={p.recoveryBody(l)} action={{ label: p.goUsers(l), go: onUsers }} />;
  if (after) {
    return (
      <div className="m02-live">
        <AfterActivation l={l} out={after} onSignIn={onSignOut} />
      </div>
    );
  }
  if (revoked) {
    return <Revoked l={l} onReload={() => void load()} />;
  }
  if (status === "error" && !page && error instanceof AdminError && error.failure === "denied") return <Unavailable l={l} title={p.deniedTitle(l)} body={p.denied(l)} />;
  const open = (id: string) => { setForm(null); setSelected(id); setFocusTo(".m02-detail-title"); };
  const closeDetail = () => { const back = selected ? `[data-change="${selected}"].m02-open` : ".m02-propose"; setSelected(""); setForm(null); setFocusTo(back); };
  const change = page && selected ? page.changes.find((x) => x.id === selected) ?? null : null;
  const onAction = (k: ActionKey, opener: HTMLElement) => {
    if (!change) return;
    if (k === "revise") { setForm({ revise: change, initial: null }); setFocusTo(".m02-form h2"); return; }
    setFocusTo("");
    setConfirm({ k: { kind: k, change, proposed: null }, opener });
  };
  const done = (k: Confirming, out: Outcome) => {
    setConfirm(null);
    if (k.kind === "activate") { setAfter(out); return; }
    const id = out.fields.change || k.change?.id || "";
    const n = k.change?.number ?? Number(id.split("-").pop() ?? 0);
    const text = k.kind === "propose" ? (k.change ? ty.done.revise(l, n) : ty.done.propose(l, n)) : ty.done[k.kind](l, n);
    say(text);
    setForm(null);
    setSelected(id);
    void load().then(() => setFocusTo(".m02-detail-title"));
  };
  const signedOut = () => { clear(); setError(new AdminError("unauthenticated")); setStatus("error"); };
  const panelOpen = Boolean(form || change);
  const stepping = !wide && panelOpen;
  const panel = page && (form ? (
    <ProposeForm key={(form.revise?.id ?? "new") + page.revision} l={l} page={page} revise={form.revise} initial={form.initial}
      onCancel={closeDetail}
      onReview={(v) => { setForm({ ...form, initial: v }); setConfirm({ k: { kind: "propose", change: form.revise, proposed: v }, opener: document.activeElement as HTMLElement | null }); }} />
  ) : change ? (
    <Detail l={l} page={page} c={change} split={wide} onClose={closeDetail} onAction={onAction} onOpen={open} />
  ) : null);
  const loadError = status === "error" && (
    <section className="live-error" role="alert">
      <h2>{p.loadFailed(l)}</h2>
      <p>{error instanceof AdminError && error.failure === "unauthenticated" ? policyFailure(l, error).body : ty.loadFailedBody(l)}</p>
      {page && <p>{diagnosticCopy.kept(l)}</p>}
      {error instanceof AdminError && error.failure === "unauthenticated"
        ? <button type="button" className="live-secondary" onClick={onSignOut}>{w.signInAgain(l)}</button>
        : <button type="button" className="live-secondary" onClick={() => void load()}>{c.tryAgain(l)}</button>}
    </section>
  );
  return (
    <div className="m02-live" data-live="policies" aria-busy={status === "loading" || undefined}>
      <p className="visually-hidden" role="status" aria-live="polite">{announce}</p>
      {stepping && page ? (
        <section className="m02-step" aria-label={form ? (form.revise ? ty.reviseTitle(l, form.revise.number) : ty.formTitle(l)) : ty.change(l, change?.number ?? 0)}>
          <button type="button" className="service-button-link live-back m02-back" onClick={closeDetail}>{ty.back(l)}</button>
          {loadError}
          <ShortConditions l={l} page={page} skipHold={change?.state === "activation_unknown"} />
          {panel}
        </section>
      ) : (
        <>
          {page && <Conditions l={l} page={page} onOpen={open} />}
          {!page && status === "loading" && (
            <section className="m02-current" aria-labelledby="m02-current">
              <h2 id="m02-current" tabIndex={-1}>{ty.currentTitle(l)}</h2>
              <p role="status" className="live-meta">{p.loading(l)}</p>
            </section>
          )}
          {loadError}
          {page && <CurrentRules l={l} page={page} loading={status === "loading"} onReload={() => void load()} />}
          {page && (
            <section className="m02-changes" aria-labelledby="m02-changes">
              <ChangesHead l={l} page={page} onPropose={() => { setSelected(""); setForm({ revise: null, initial: null }); setFocusTo(".m02-form h2"); }} />
              <div className="m02-split" data-mode={wide ? "split" : "step"} data-open={panelOpen ? "true" : "false"}>
                <Lists l={l} page={page} selected={selected} reduced={wide && panelOpen} onOpen={open} />
                {wide && panel && <aside className="m02-detail" tabIndex={0} aria-label={form ? ty.formTitle(l) : ty.change(l, change?.number ?? 0)}>{panel}</aside>}
              </div>
            </section>
          )}
        </>
      )}
      {page && confirm && (
        <ConfirmDialog l={l} page={page} api={api} k={confirm.k} opener={confirm.opener} workspace={session.workspace} onDone={done} onSignedOut={signedOut}
          onClose={(reload) => {
            setConfirm(null);
            if (reload) { setForm(null); void load().then(() => setFocusTo(".m02-detail-title")); }
          }} />
      )}
    </div>
  );
}
