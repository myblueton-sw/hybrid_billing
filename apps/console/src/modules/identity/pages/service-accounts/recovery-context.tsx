import { createContext, useContext, useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { attempt, type AdminClient } from "../../../../service/admin";
import type { Session } from "../../../../service/api";
import { issueAttempts, type IssueAttempts } from "./issue-attempts";

export type RecoverySession = Readonly<Pick<Session, "user" | "workspace" | "csrf">>;
export type Recovery = { issues: IssueAttempts; afterView: { focus: string; view: string } };
type Token = { valid: boolean; client: AdminClient; session: RecoverySession; store?: IssueAttempts };
type Lease = { token: Token; value: Recovery };
const Context = createContext<Recovery | null>(null);
const same = (a: RecoverySession, b: RecoverySession) => a.user === b.user && a.workspace === b.workspace && a.csrf === b.csrf;

export function ServiceAccountRecoveryProvider({ client, session, isCurrentSession, children }: {
  client: AdminClient; session: RecoverySession; isCurrentSession: (session: RecoverySession) => boolean; children: ReactNode;
}) {
  const tokenRef = useRef<Token | null>(null);
  const [lease, setLease] = useState<Lease | null>(null);
  const { user, workspace, csrf } = session;
  useLayoutEffect(() => {
    const token: Token = { valid: true, client, session: Object.freeze({ user, workspace, csrf }) };
    tokenRef.current = token;
    return () => {
      token.valid = false;
      token.store?.dispose();
      if (tokenRef.current === token) tokenRef.current = null;
    };
  }, [client, user, workspace, csrf, isCurrentSession]);
  useEffect(() => {
    const token = tokenRef.current;
    if (!token || !token.valid || token.client !== client || !same(token.session, { user, workspace, csrf }) || !isCurrentSession(token.session)) return;
    const store = issueAttempts(attempt, { mono: () => performance.now(), wall: () => Date.now() }, window,
      () => token.valid && isCurrentSession(token.session), () => document.visibilityState === "visible");
    token.store = store;
    const unsubscribe = client.subscribeAuthenticationLoss(() => {
      token.valid = false;
      store.dispose();
      setLease((prior) => prior?.value.issues === store ? null : prior);
    });
    if (store.activate()) setLease({ token, value: { issues: store, afterView: { focus: "", view: "" } } });
    return () => {
      unsubscribe();
      store.dispose();
      if (token.store === store) token.store = undefined;
    };
  }, [client, user, workspace, csrf, isCurrentSession]);
  const current = lease && lease.token.valid && lease.token.client === client && same(lease.token.session, session) && isCurrentSession(lease.token.session) && lease.value.issues.current();
  return <Context.Provider value={current ? lease.value : null}>{children}</Context.Provider>;
}

export function useServiceAccountRecovery(): Recovery | null {
  return useContext(Context);
}
