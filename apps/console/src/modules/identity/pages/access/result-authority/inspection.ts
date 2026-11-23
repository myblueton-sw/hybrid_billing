import {exact,fail,frozen,grantOf,historyOf,id,instant,rationaleValid,resultDomain,type AuthorityDomain,type Envelope,type Read,type HistoryRow} from "./contract";
export type Inspection=Readonly<{namespace:string;row:HistoryRow;ceiling:string;notBefore:string;notAfter:string;rationale:string;decision?:Readonly<{at:string;actor:string;rationale:string;grant:string;recordedGrantStatus:string}>;revision:string;at:string}>;
export function inspectionOf(e:Envelope,proposal:string,expectedDigest?:string,d:AuthorityDomain=resultDomain):Inspection{
 const f=e.fields,decided=Object.hasOwn(f,"decided_at");
 exact(f,["capability","proposal","digest","state","mode","kind","recipient","ceiling","proposer","not_before","not_after","proposed_at","expires_at","rationale","receipt",...(decided?["decided_at","decided_by","decision_reason","grant","recorded_grant_status"]:[])]);
 if(f.mode!=="synthetic"||f.receipt!=="historical_decision"||f.proposal!==proposal||expectedDigest&&f.digest!==expectedDigest)return fail();
 const row=historyOf(JSON.stringify({proposal:f.proposal,digest:f.digest,kind:f.kind,capability:f.capability,recipient:f.recipient,state:f.state,proposer:f.proposer,proposed_at:f.proposed_at,expires_at:f.expires_at,decision_actor:decided?f.decided_by:""}),d);
 if(instant(f.not_before)===undefined||instant(f.not_after)===undefined||instant(f.not_before)!>=instant(f.not_after)!||!rationaleValid(f.rationale))return fail();
 if(row.kind==="console_ceiling"?f.ceiling!=="":!grantOf(d,f.ceiling))return fail();
 let decision:Inspection["decision"];
 if(decided){
  const at=instant(f.decided_at),proposedAt=instant(f.proposed_at)!,expiresAt=instant(f.expires_at)!;
  if(!id(f.decided_by)||at===undefined||at<proposedAt||at>=expiresAt||!rationaleValid(f.decision_reason))return fail();
  if(["rejected","withdrawn"].includes(row.state)?f.grant!==""||f.recorded_grant_status!=="":f.grant!==`${d.grantPrefix}${row.digest}`|| (row.state==="applied"?!["active","revoked","expired"].includes(f.recorded_grant_status):f.recorded_grant_status!==""))return fail();
  decision={at:f.decided_at,actor:f.decided_by,rationale:f.decision_reason,grant:f.grant,recordedGrantStatus:f.recorded_grant_status};
 }
 return frozen({namespace:d.namespace,row,ceiling:f.ceiling,notBefore:f.not_before,notAfter:f.not_after,rationale:f.rationale,decision,revision:e.revision,at:e.at});
}
export async function inspect(read:Read,row:HistoryRow,d:AuthorityDomain=resultDomain){return inspectionOf(await read(`${d.namespace}.inspect`,row.proposal,{}),row.proposal,row.digest,d);}
