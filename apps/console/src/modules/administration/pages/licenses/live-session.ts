import type { Session } from "../../../../service/api";
export type LicenseResult = { outcome: string; request: string; operation: string; unknown: boolean; refusal?: boolean; access?: "denied" | "signedOut" };
type Snapshot = { pending: boolean; result: LicenseResult | null; settled: number };
export function createLicenseSession() {
 let snapshot: Snapshot = {pending:false,result:null,settled:0};
 const listeners = new Set<() => void>();
 const update = (next:Snapshot) => {snapshot=next;for(const listener of listeners)listener();};
 return {
  snapshot: () => snapshot,
  subscribe: (listener:()=>void) => {listeners.add(listener);return()=>{listeners.delete(listener);};},
  begin: () => {if(snapshot.pending || snapshot.result?.unknown)return false;update({...snapshot,pending:true,result:null});return true;},
  settle: (result:LicenseResult) => update({pending:false,result,settled:snapshot.settled+1}),
 };
}
const sessions = new WeakMap<Session, ReturnType<typeof createLicenseSession>>();
export function licenseSession(session:Session) {
 let state=sessions.get(session);
 if(!state){state=createLicenseSession();sessions.set(session,state);}
 return state;
}
