import { strictJSON } from "../../../../../service/strict-json";
export class AuthorityError extends Error {
 constructor(readonly code:string,readonly effect:"none"|"unknown"="none",readonly operation=""){super(code);}
}
export const fail=():never=>{throw new AuthorityError("invalid_authority_response");};
export const bytes=(value:string)=>new TextEncoder().encode(value).length;
export const id=(value:unknown):value is string=>typeof value==="string"&&/^[A-Za-z0-9._:-]{1,128}$/.test(value);
export const digest=(value:unknown):value is string=>typeof value==="string"&&/^[0-9a-f]{64}$/.test(value);
export const revision=(value:unknown):value is string=>typeof value==="string"&&/^r(?:0|[1-9][0-9]*)$/.test(value);
export const object=(value:unknown):value is Record<string,unknown>=>!!value&&typeof value==="object"&&!Array.isArray(value);
export function exact(value:unknown,keys:readonly string[]):asserts value is Record<string,unknown>{if(!object(value)||Object.keys(value).length!==keys.length||keys.some(key=>!Object.hasOwn(value,key)))fail();}
export function frozen<T>(value:T):Readonly<T>{if(value&&typeof value==="object"&&!Object.isFrozen(value)){for(const item of Object.values(value))frozen(item);Object.freeze(value);}return value;}
export function json(value:unknown):unknown{if(typeof value!=="string")return fail();try{return strictJSON(value,4096);}catch{return fail();}}
export const capabilities=["propose","confirm","revoke","apply","inspect"] as const;
export type Capability=typeof capabilities[number];
export const capability=(v:unknown):v is Capability=>typeof v==="string"&&capabilities.includes(v as Capability);
export const states=["pending","applying","decision_unknown","applied","rejected","withdrawn","expired","superseded"] as const;
export type RecordedState=typeof states[number];
export type Policy=Readonly<{capabilities:readonly Capability[];proposal_lifetime_seconds:string;max_grant_validity_seconds:string}>;
export type Recipient=Readonly<{user:string;login:string}>;
export type Ceiling=Readonly<{ceiling:string;capability:Capability;not_before:string;not_after:string}>;
export type HistoryRow=Readonly<{proposal:string;digest:string;kind:"console_ceiling"|"human_grant";capability:Capability;recipient:string;state:RecordedState;proposer:string;proposed_at:string;expires_at:string;decision_actor:string}>;
export type Envelope=Readonly<{revision:string;at:string;fields:Readonly<Record<string,string>>}>;
export type Read=(command:string,target:string,fields:Record<string,string>)=>Promise<Envelope>;
export function instant(value:unknown):bigint|undefined {
 if(typeof value!=="string"||!/^(?!0000)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/.test(value))return;
 const second=value.slice(0,19),millis=Date.parse(second+"Z");
 if(!Number.isFinite(millis)||new Date(millis).toISOString().slice(0,19)!==second)return;
 const fraction=value.includes(".")?value.slice(20,-1):"";
 return BigInt(millis)*1000000n+BigInt(fraction.padEnd(9,"0"));
}
export function seconds(value:unknown):value is string{return typeof value==="string"&&/^[1-9][0-9]{0,9}$/.test(value)&&BigInt(value)<=9223372036n;}
export function policyOf(value:unknown):Policy{
 const v=json(value);exact(v,["capabilities","proposal_lifetime_seconds","max_grant_validity_seconds"]);
 if(!Array.isArray(v.capabilities)||JSON.stringify(v.capabilities)!==JSON.stringify(capabilities)||!seconds(v.proposal_lifetime_seconds)||!seconds(v.max_grant_validity_seconds))return fail();
 return frozen(v) as Policy;
}
export function historyOf(value:string):HistoryRow{
 const v=json(value);exact(v,["proposal","digest","kind","capability","recipient","state","proposer","proposed_at","expires_at","decision_actor"]);
 if(!id(v.proposal)||!digest(v.digest)||!capability(v.capability)||!id(v.recipient)||!id(v.proposer)||typeof v.kind!=="string"||!["console_ceiling","human_grant"].includes(v.kind)||typeof v.state!=="string"||!states.includes(v.state as RecordedState)||instant(v.proposed_at)===undefined||instant(v.expires_at)===undefined||instant(v.proposed_at)!>=instant(v.expires_at)!||!(v.decision_actor===""||id(v.decision_actor)))return fail();
 if(v.kind==="console_ceiling"&&v.recipient!=="console"||v.kind==="human_grant"&&v.recipient===v.proposer)return fail();
 const decided=["applying","decision_unknown","applied","rejected","withdrawn"].includes(v.state);
 if(decided!==!!v.decision_actor)return fail();
 if(decided&&(v.state==="withdrawn"?v.decision_actor!==v.proposer:v.decision_actor===v.proposer||v.kind==="human_grant"&&v.decision_actor===v.recipient))return fail();
 return frozen(v) as HistoryRow;
}
export function recipientOf(value:string,actor:string):Recipient{
 const v=json(value);exact(v,["user","login"]);
 if(!id(v.user)||v.user===actor||typeof v.login!=="string"||bytes(v.login)>4096||/[\u0000-\u001f\u007f]/.test(v.login))return fail();
 return frozen(v) as Recipient;
}
export function ceilingOf(value:string,selected:Capability):Ceiling{
 const v=json(value);exact(v,["ceiling","capability","not_before","not_after"]);
 if(typeof v.ceiling!=="string"||!/^result-grant-[0-9a-f]{64}$/.test(v.ceiling)||v.capability!==selected||instant(v.not_before)===undefined||instant(v.not_after)===undefined||instant(v.not_before)!>=instant(v.not_after)!)return fail();
 return frozen(v) as Ceiling;
}
export function rationaleValid(value:unknown):value is string{return typeof value==="string"&&!!value.trim()&&bytes(value)<=1024&&!/[\u0000-\u001f\u007f]/.test(value)&&[...value].every(c=>c.length!==1||c.charCodeAt(0)<0xd800||c.charCodeAt(0)>0xdfff);}
