import { applicationContext, sameApplicationContext, type ApplicationContext } from "./relation";
import { basisOf } from "../documents/inspection";
import { decimal, timestamp } from "../documents/contract";
import { rationaleValid } from "../documents/packet";
import { bytes, digest, exact, fail, freeze, id, json, object, revision, type Preparation, type ReadCommand } from "./contract";
import { application, type HistoryRow } from "./history";
import { loadAdvice } from "./advice";
import type { ResultPacket } from "./packet";
export type Decision = Readonly<{ actor: string; reason: string; at: string }>;
export type Inspection = Readonly<{ result: string; digest: string; revision: string; state: HistoryRow["recorded_state"]; packet: ResultPacket; source: string; sourceDigest: string; packetDigest: string; comparison: readonly string[]; proposedBy: string; proposedAt: string; reason: string; confirmation?: Decision; revocation?: Decision; applicationState: HistoryRow["application_state"]; applicationReceipt: string; applicationContext: ApplicationContext }>;
function instant(v: string): bigint {
 const whole = v.slice(0,19)+"Z", fraction = v.slice(19,-1).replace(/^\./, "");
 return BigInt(Date.parse(whole))*1000000n+BigInt(fraction.padEnd(9,"0"));
}
function canonical(v: unknown): string {
 if(Array.isArray(v))return `[${v.map(canonical).join(",")}]`;
 if(object(v))return `{${Object.keys(v).sort().map(k=>`${JSON.stringify(k)}:${canonical(v[k])}`).join(",")}}`;
 return JSON.stringify(v);
}
function packet(raw: unknown, scope: Preparation): ResultPacket {
 exact(raw,["Mode","Profile","Evidence","Preparation","PreparationDigest","Connection","External","Currency","Customer","Period","Revision","Rows"]);
 if(raw.Mode!=="synthetic"||raw.Profile!=="synthetic_result_one_to_one_v1"||!id(raw.Evidence)||!id(raw.External)||raw.Preparation!==scope.preparation||raw.PreparationDigest!==scope.preparation_digest||raw.Connection!==scope.connection||raw.Currency!==scope.currency||raw.Customer!==scope.customer||raw.Period!==scope.period||!Number.isInteger(raw.Revision)||Number(raw.Revision)<1||Number(raw.Revision)>2147483647||!Array.isArray(raw.Rows)||!raw.Rows.length||raw.Rows.length>100)return fail();
 const seen=new Set<string>();
 for(const row of raw.Rows){exact(row,["ID","Line","Product","Unit","Net","Tax","Quantity"]);if(![row.ID,row.Line,row.Product,row.Unit].every(id)||!decimal(row.Net)||!decimal(row.Tax)||!decimal(row.Quantity,4,true)||seen.has(String(row.ID)))return fail();seen.add(String(row.ID));}
 return freeze(raw) as ResultPacket;
}
export async function inspectResult(read: ReadCommand, scope: Preparation, row: HistoryRow, workspace: string): Promise<Inspection> {
 const p=await read("result_evidence.inspect",row.result,{digest:row.digest}),f=p.fields;
 const keys=["mode","result","digest","recorded_state","receipt","application_state","application_receipt","current_eligibility","source","source_digest","packet_digest","document_digest","proposed_by","proposal_reason","proposed_at","packet","preparation","document","comparison","total","application_context"];
 for(const prefix of ["confirmation","revocation"])if(Object.keys(f).some(k=>k.startsWith(prefix+"_")))keys.push(prefix+"_by",prefix+"_reason",prefix+"_at");
 exact(f,keys);
 if(!revision(p.revision)||f.mode!=="synthetic"||f.result!==row.result||f.digest!==row.digest||f.document_digest!==scope.document_digest||f.receipt!=="historical_evidence"||f.current_eligibility!=="unchecked"||!["pending","confirmed","revoked"].includes(f.recorded_state)||!application(f.application_state,f.application_receipt)||f.proposed_by!==row.proposed_by||f.proposed_at!==row.proposed_at||!timestamp(f.proposed_at)||!rationaleValid(f.proposal_reason)||![f.source_digest,f.packet_digest].every(digest)||bytes(f.source)>4096)return fail();
 const reported=packet(json(f.packet),scope),original=packet(json(f.source),scope);
 if(canonical(reported)!==canonical(original))return fail();
 const sourceHash=Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256",new TextEncoder().encode(f.source))),b=>b.toString(16).padStart(2,"0")).join("");
 if(sourceHash!==f.source_digest)return fail();
 const document=json(f.document),preparation=json(f.preparation),total=json(f.total),comparison=json(f.comparison);
 if(!object(document)||document.ID!==scope.document||!object(preparation)||!object(preparation.Submission)||preparation.Submission.Key!==scope.preparation||preparation.Submission.Connection!==scope.connection||preparation.Submission.Customer!==scope.customer||preparation.Submission.Period!==scope.period)return fail();
 if(!id(workspace)||!id(document.Request)||!digest(document.RequestDigest)||preparation.Author!==scope.preparation_actor||!timestamp(preparation.Prepared))return fail();
 try { const basis=basisOf(f.document,{request:document.Request,digest:document.RequestDigest,workspace},scope.document);if(basis.Document.Currency!==scope.currency)return fail(); } catch { return fail(); }
 exact(total,["Net","Tax","Quantity"]);if(Object.values(total).some(v=>typeof v!=="string"||!/^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,4})?$/.test(v)))return fail();
 if(!Array.isArray(comparison)||comparison.some(v=>typeof v!=="string"||!["line_coverage","product_mismatch","unit_mismatch","net_mismatch","tax_mismatch","quantity_mismatch"].includes(v))||new Set(comparison).size!==comparison.length)return fail();
 const event=(prefix:string):Decision|undefined=>{if(!Object.hasOwn(f,prefix+"_by"))return undefined;const actor=f[prefix+"_by"],reason=f[prefix+"_reason"],at=f[prefix+"_at"];if(!id(actor)||!rationaleValid(reason)||!timestamp(at)||instant(at)<instant(f.proposed_at))return fail();return freeze({actor,reason,at});};
 const confirmation=event("confirmation"),revocation=event("revocation");
 if(f.application_state!=="not_started"&&!confirmation||confirmation?.actor===f.proposed_by||confirmation&&revocation&&instant(revocation.at)<instant(confirmation.at)||f.recorded_state==="pending"&&(confirmation||revocation||f.application_state!=="not_started")||f.recorded_state==="confirmed"&&(!confirmation||revocation)||f.recorded_state==="revoked"&&!revocation)return fail();
 const context=applicationContext(f.application_context,row.result,f.application_state as HistoryRow["application_state"]);
 if(context.position!=="not_started"&&context.kind==="replacement"&&context.predecessor_revision>=reported.Revision)return fail();
 return freeze({applicationContext:context,result:row.result,digest:row.digest,revision:p.revision,state:f.recorded_state as HistoryRow["recorded_state"],packet:reported,source:f.source,sourceDigest:f.source_digest,packetDigest:f.packet_digest,comparison:comparison as string[],proposedBy:f.proposed_by,proposedAt:f.proposed_at,reason:f.proposal_reason,confirmation,revocation,applicationState:f.application_state,applicationReceipt:f.application_receipt});
}
export async function inspectWithAdvice(read: ReadCommand, scope: Preparation, row: HistoryRow, workspace: string) {
 const inspection=await inspectResult(read,scope,row,workspace),advice=await loadAdvice(read,row.result,row.digest);
 if(!sameApplicationContext(inspection.applicationContext,advice.applicationContext)||inspection.revision!==advice.revision||inspection.applicationState!==advice.applicationState||inspection.applicationReceipt!==advice.applicationReceipt)return fail();
 return freeze({inspection,advice});
}
