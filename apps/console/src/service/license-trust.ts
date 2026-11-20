import { AdminError, type AdminClient } from "./admin";
import { boolean, choice, integer, ReadParseError, record, shape, text } from "./strict-read";
export const trustCommands = ["licenses.propose_key","licenses.propose_retirement","licenses.propose_compromise","licenses.approve_key_change","licenses.withdraw_key_change"] as const;
export type TrustCommand = typeof trustCommands[number];
const invalid = ():never => {throw new ReadParseError();};
export const trustId = (v:unknown) => {const s=text(v);return /^[A-Za-z0-9._:-]{1,128}$/.test(s)?s:invalid();};
export const proposalNumber = (v:unknown) => {const s=text(v);return /^[1-9][0-9]{0,19}$/.test(s)&&BigInt(s)<=18446744073709551615n?s:invalid();};
const epoch = (v:unknown) => {const n=integer(v);return n>0&&n<=253402300799?n:invalid();};
const hex = (v:unknown,n:number) => {const s=text(v);return s.length===n&&/^[0-9a-f]+$/.test(s)?s:invalid();};
const keyShape=shape({id:trustId,issuer:trustId,fingerprint_sha256:(v:unknown)=>hex(v,64),not_before:epoch,not_after:epoch,accept_until:epoch});
export type TrustKey = ReturnType<typeof keyShape>;
function keyOf(v:unknown):TrustKey {const k=keyShape(v);if(k.not_after<=k.not_before||k.accept_until<k.not_after)invalid();return k;}
export type TrustProposal = {number:string;target:string;kind:"key"|"retire_key"|"compromise_key";state:"pending"|"applied";proposer:string;approver:string;key?:TrustKey;target_key?:string;retire_at?:number};
export type TrustRecord = {kind:"key";key:TrustKey;retired:boolean;compromised:boolean}|{kind:"proposal";proposal:TrustProposal};
export type TrustPage = {state:"not_configured"|"uninitialized"|"not_found"}|{state:"configured";revision:string;record:TrustRecord};
export function trustPageOf(v:unknown,kind:"key"|"proposal",requested:string):TrustPage {
 const r=record(v),state=choice("not_configured","uninitialized","not_found","configured")(r.state);
 if(state!=="configured")return {state};
 const revision=text(r.revision);if(!/^r[0-9]+$/.test(revision))invalid();const row=record(r.record);
 if(kind==="key") {const key=keyOf(row.key);if(key.id!==requested)invalid();return {state,revision,record:{kind,key,retired:boolean(row.retired),compromised:boolean(row.compromised)}};}
 const number=proposalNumber(row.number),target=text(row.target),change=choice("key","retire_key","compromise_key")(row.kind),stage=choice("pending","applied")(row.state),proposer=trustId(row.proposer),approver=text(row.approver);
 if(number!==requested||target!=="license-change-"+number||stage==="pending"&&approver!==""||stage==="applied"&&(!approver||approver===proposer))invalid();if(approver)trustId(approver);
 const proposal:TrustProposal={number,target,kind:change,state:stage,proposer,approver};
 if(change==="key"){if(row.target_key!==undefined||row.retire_at!==undefined)invalid();proposal.key=keyOf(row.key);}
 else {if(row.key!==undefined)invalid();proposal.target_key=trustId(row.target_key);if(change==="retire_key")proposal.retire_at=epoch(row.retire_at);else if(row.retire_at!==undefined)invalid();}
 return {state,revision,record:{kind,proposal}};
}
export async function readTrust(api:Pick<AdminClient,"read">,kind:"key"|"proposal",id:string):Promise<TrustPage> {
 kind==="key"?trustId(id):proposalNumber(id);
 const value=await api.read(`/licenses/${kind==="key"?"keys":"key-changes"}/${encodeURIComponent(id)}`);
 try{return trustPageOf(value,kind,id);}catch{throw new AdminError("invalid");}
}
export function publicKeyFile(raw:string):Record<string,string> {
 if(new TextEncoder().encode(raw).length>4000)invalid();const r=record(JSON.parse(raw));
 const keys=["key_id","issuer","public_key","proof","not_before","not_after","accept_until"];
 if(Object.keys(r).length!==keys.length||Object.keys(r).some(k=>!keys.includes(k)))invalid();
 const start=epoch(r.not_before),end=epoch(r.not_after),until=epoch(r.accept_until);if(end<=start||until<end)invalid();
 return {key_id:trustId(r.key_id),issuer:trustId(r.issuer),public_key:hex(r.public_key,64),proof:hex(r.proof,128),not_before:String(start),not_after:String(end),accept_until:String(until)};
}
export function utcEpoch(raw:string):string {if(!/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/.test(raw))invalid();const n=Date.parse(raw);if(!Number.isFinite(n)||new Date(n).toISOString()!==raw.replace("Z",".000Z"))invalid();return String(epoch(n/1000));}
const reasons=["not_pending","ai_not_allowed","not_human","private_key","no_proof","compromised","retired","retire_past","time_unavailable","clock_anomaly","local_clock_behind"];
export function trustOutcome(command:TrustCommand,state:string,fields:Record<string,string>,target:string):string|undefined {
 const success=command.startsWith("licenses.propose_")?"pending":command==="licenses.approve_key_change"?"applied":"withdrawn";
 if(fields.outcome!==state)invalid();
 if(state===success){const number=proposalNumber(fields.proposal);if(fields.target!=="license-change-"+number||!command.startsWith("licenses.propose_")&&fields.target!==target)invalid();return number;}
 const allowed=command==="licenses.withdraw_key_change"?["ai_not_allowed","not_human","not_pending"]:command==="licenses.propose_key"?["ai_not_allowed","not_human","private_key","no_proof"]:command==="licenses.propose_compromise"?["ai_not_allowed","not_human","compromised"]:reasons;
 if(!allowed.includes(state)||fields.proposal!==undefined||fields.target!==undefined)invalid();
}
