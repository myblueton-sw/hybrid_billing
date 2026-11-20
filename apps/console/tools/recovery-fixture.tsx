import { StrictMode, useLayoutEffect, useState } from "react";
import { createRoot } from "react-dom/client";
import { adminClient, type AdminClient, type Outcome } from "../src/service/admin";
import type { Locale } from "../src/shared/types";
import { Reveal } from "../src/modules/identity/pages/service-accounts/Live";
import { ServiceAccountRecoveryProvider, useServiceAccountRecovery, type Recovery, type RecoverySession } from "../src/modules/identity/pages/service-accounts/recovery-context";

const service = "svc-test";
const credential = "cred-0123456789abcdef";
const reference = "R".repeat(22);
let active: RecoverySession = { user: "test-user", workspace: "test-workspace", csrf: "synthetic-csrf" };
let visibility: DocumentVisibilityState = "visible";
Object.defineProperty(document, "visibilityState", { configurable: true, get: () => visibility });
const guard = (s: RecoverySession) => s.user === active.user && s.workspace === active.workspace && s.csrf === active.csrf;
let calls = 0;
let resolveRead: ((value: Response) => void) | null = null;
let api: AdminClient;
const makeClient = () => adminClient("synthetic-csrf", (async (path) => {
  if (String(path).startsWith("/api/v1/identity/services/auth-check")) return new Response(JSON.stringify({ code: "unauthenticated", request_id: "synthetic", next_action: "none" }), { status: 401 });
  if (!String(path).startsWith("/api/v1/identity/links/")) throw new Error("unexpected fixture transport");
  calls++;
  return new Promise<Response>((resolve) => { resolveRead = resolve; });
}) as typeof fetch);
const outcome: Outcome = { status: "completed", state: "credential_issued", job_id: "", fields: { credential, secret_ref: reference } };
let current: Recovery | null = null;
let prior: Recovery | null = null;
let setSession!: (s: RecoverySession) => void;
let setClient!: (c: AdminClient) => void;
let setShown!: (s: boolean) => void;
let setMounted!: (s: boolean) => void;
let setLocale!: (l: Locale) => void;
let authoritativeGapSafe = true;
let layoutGapSafe = true;
let lastRevoke: boolean | null = null;
const ids = new WeakMap<object, number>();
let nextId = 0;
const id = (store: object) => { if (!ids.has(store)) ids.set(store, ++nextId); return ids.get(store); };

function Probe({ show, locale, hide }: { show: boolean; locale: Locale; hide: () => void }) {
  const value = useServiceAccountRecovery();
  useLayoutEffect(() => { current = value; return () => { if (current === value) current = null; }; }, [value]);
  if (!value) return <p data-fixture="waiting">Waiting for active recovery</p>;
  return <div data-fixture="ready">{show && <Reveal l={locale} api={api} issues={value.issues} user={active.user} service={service} credential={credential} refId={reference} until="2027-01-01T00:00:00Z"
    onDone={() => { value.issues.unmark(active.user, credential); hide(); }} onLeave={hide} onRevoke={(notStored) => { lastRevoke = notStored; value.issues.revoked(active.user, credential); hide(); }} />}</div>;
}

function Fixture() {
  const [session, changeSession] = useState(active);
  const [client, changeClient] = useState(makeClient);
  const [show, changeShown] = useState(false);
  const [mounted, changeMounted] = useState(true);
  const [locale, changeLocale] = useState<Locale>("en");
  api = client;
  setSession = changeSession; setClient = changeClient; setShown = changeShown; setMounted = changeMounted; setLocale = changeLocale;
  useLayoutEffect(() => { if (prior) layoutGapSafe = layoutGapSafe && !prior.issues.current(); }, [client, session.user, session.workspace, session.csrf]);
  return <ServiceAccountRecoveryProvider client={client} session={session} isCurrentSession={guard}>
    {mounted && <Probe show={show} locale={locale} hide={() => changeShown(false)} />}
  </ServiceAccountRecoveryProvider>;
}

const root = createRoot(document.getElementById("root")!);
root.render(<StrictMode><Fixture /></StrictMode>);

const control = {
  snapshot: () => ({ ready: !!current?.issues.current(), id: current ? id(current.issues) : null, calls, phase: current?.issues.get(active.user, service)?.phase ?? "", markers: current?.issues.marks(active.user) ?? [], priorDisposed: prior?.issues.disposed() ?? null, gapSafe: authoritativeGapSafe, layoutGapSafe, lastRevoke }),
  async issue() {
    if (!current) throw new Error("fixture not ready");
    await current.issues.submit(active.user, service, { days: "1", reason: "Synthetic recovery check" }, "rev-1", 300, async () => outcome);
  },
  show: () => setShown(true),
  hide: () => setShown(false),
  unmountView: () => { prior = current; setMounted(false); },
  mountView: () => setMounted(true),
  locale: (l: Locale) => setLocale(l),
  visibility: (v: DocumentVisibilityState) => { visibility = v; },
  hideBeforeDispatch() {
    if (!current) throw new Error("fixture not ready");
    const issues = current.issues;
    const stop = issues.subscribe(() => { if (issues.collectionPhase(active.user, credential) === "pending") { visibility = "hidden"; stop(); } });
  },
  pagehide: () => { visibility = "hidden"; window.dispatchEvent(new PageTransitionEvent("pagehide", { persisted: true })); },
  switchScope(field: "user" | "workspace" | "csrf" | "client") {
    prior = current;
    setShown(false);
    if (field === "client") setClient(makeClient());
    else {
      active = { ...active, [field]: active[field] + "-changed" };
      authoritativeGapSafe = prior?.issues.open(prior ? (field === "user" ? "test-user" : active.user) : active.user, service) === null;
      setSession(active);
    }
  },
  async rejectAuthentication(alreadyLost = false) {
    const remembered = current ?? prior;
    if (!remembered) throw new Error("fixture not ready");
    prior = remembered;
    const target = alreadyLost ? makeClient() : api;
    await target.serviceAccount("auth-check", active.workspace).catch(() => undefined);
    if (alreadyLost) setClient(target);
  },
  resolve() {
    if (!resolveRead) throw new Error("no pending fixture read");
    const finish = resolveRead; resolveRead = null;
    finish(new Response(JSON.stringify({ credential, secret: "hbsk_" + "A".repeat(43) })));
  },
  rejectRead() {
    if (!resolveRead) throw new Error("no pending fixture read");
    const finish = resolveRead; resolveRead = null;
    finish(new Response(JSON.stringify({ code: "dependency_unavailable", request_id: "synthetic", next_action: "none" }), { status: 503 }));
  },
  unmountRoot: () => { if (current) prior = current; root.unmount(); },
};
Object.assign(window, { hbRecovery: control });
