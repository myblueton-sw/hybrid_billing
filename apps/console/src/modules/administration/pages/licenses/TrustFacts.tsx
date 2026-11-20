import type { Locale } from "../../../../shared/types";
import type { TrustKey, TrustRecord } from "../../../../service/license-trust";
import { tk } from "./trust-copy";
export const utc=(value:number)=>new Date(value*1000).toISOString().replace(".000Z","Z");
export function KeyFacts({l,value}:{l:Locale;value:TrustKey}) {
 return <dl className="p07-facts">{([['keyId',value.id],['issuer',value.issuer],['fingerprint',value.fingerprint_sha256],['not_before',utc(value.not_before)],['not_after',utc(value.not_after)],['accept_until',utc(value.accept_until)]] as const).map(([name,v])=><div key={name}><dt>{tk(l,name)}</dt><dd><bdi dir="ltr">{v}</bdi></dd></div>)}</dl>;
}
export default function TrustFacts({l,record}:{l:Locale;record:TrustRecord}) {
 if(record.kind==="key")return <><KeyFacts l={l} value={record.key}/><dl className="p07-facts">{(['retired','compromised'] as const).map(name=><div key={name}><dt>{tk(l,name)}</dt><dd>{tk(l,record[name]?'yes':'no')}</dd></div>)}</dl></>;
 const p=record.proposal;
 return <><dl className="p07-facts"><div><dt>{tk(l,"number")}</dt><dd><bdi dir="ltr">{p.number}</bdi></dd></div><div><dt>{tk(l,"kind")}</dt><dd>{tk(l,p.kind==='key'?'newKeyValue':p.kind==='retire_key'?'retirementValue':'compromiseValue')}</dd></div><div><dt>{tk(l,"state")}</dt><dd>{tk(l,p.state)}</dd></div><div><dt>{tk(l,"proposer")}</dt><dd><bdi dir="ltr">{p.proposer}</bdi></dd></div><div><dt>{tk(l,"approver")}</dt><dd><bdi dir="ltr">{p.approver||'—'}</bdi></dd></div>{p.target_key&&<div><dt>{tk(l,"keyId")}</dt><dd><bdi dir="ltr">{p.target_key}</bdi></dd></div>}{p.retire_at&&<div><dt>{tk(l,"retirement")}</dt><dd><bdi dir="ltr">{utc(p.retire_at)}</bdi></dd></div>}</dl>{p.key&&<KeyFacts l={l} value={p.key}/>}</>;
}
