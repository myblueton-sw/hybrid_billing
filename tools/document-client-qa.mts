import assert from "node:assert/strict";
import { writeFile } from "node:fs/promises";
import { documentTransport, type Intent } from "../apps/console/src/modules/issuance/pages/approval-inbox/documents/transport";
import { loadContext, loadHistory } from "../apps/console/src/modules/issuance/pages/approval-inbox/documents/reads";
import { emptyDraft, packetOf } from "../apps/console/src/modules/issuance/pages/approval-inbox/documents/packet";
import { attempt } from "../apps/console/src/service/admin";
import { serviceClient } from "../apps/console/src/service/api";
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith("--")?[[v.slice(2),a[i+1]]]:[]));
const proposerCookie=process.env.HB_DOCUMENT_CLIENT_PROPOSER,confirmerCookie=process.env.HB_DOCUMENT_CLIENT_CONFIRMER;
delete process.env.HB_DOCUMENT_CLIENT_PROPOSER;delete process.env.HB_DOCUMENT_CLIENT_CONFIRMER;
assert.ok(proposerCookie&&confirmerCookie&&args.api&&args.output&&args.request&&args.digest);
const evidence:{checks:string[];outcome?:string}={checks:[]};
function transport(cookie:string):typeof fetch{return async(input,init)=>fetch(new URL(String(input),args.api),{...init,headers:{...Object.fromEntries(new Headers(init?.headers)),Cookie:cookie,Origin:"http://127.0.0.1:8896"}});}
try{
 const proposerFetch=transport(proposerCookie),confirmerFetch=transport(confirmerCookie);
 const proposer=await serviceClient(proposerFetch).current(),confirmer=await serviceClient(confirmerFetch).current();
 assert.ok(proposer&&confirmer&&proposer.user!==confirmer.user);
 const p=documentTransport(proposer,proposerFetch),c=documentTransport(confirmer,confirmerFetch);
 const scope={request:args.request,digest:args.digest,workspace:proposer.workspace};
 const context=await loadContext(p.read,scope);
 assert.equal(context.header.total,"15");assert.equal(context.lines.length,2);assert.equal(context.permission.state,"allowed");
 evidence.checks.push("actual owned context and proposal permission parsed");
 const draft={...emptyDraft(context),selectedERP:true,unit:"document-unit",evidence:"client-evidence",net:"15",tax:"1",rationale:"Explicit synthetic client document",lines:Object.fromEntries(context.lines.map((l,i)=>[l.line,{quantity:i===0?"2":"3",tax:i===0?"0.25":"0.75",evidence:`client-line-${i}`}]))};
 const packet=packetOf(context,draft);
 const proposed=await p.submit({action:"propose",target:"client-document",fields:{packet:packet.text},rationale:packet.rationale,revision:context.revision,attempt:attempt()});
 assert.equal(proposed.state,"pending");
 const documents=await loadHistory(c.read,scope,"documents");
 assert.ok(documents.rows.some(r=>"document"in r&&r.document==="client-document"&&"digest"in r&&r.digest===proposed.digest));
 evidence.checks.push("strict packet accepted and independently discovered");
 const advice=await c.read("document_evidence.advice","client-document",{digest:proposed.digest});
 const confirmed=await c.submit({action:"confirm",target:"client-document",fields:{digest:proposed.digest},rationale:"Independent synthetic client confirmation",revision:advice.revision,attempt:attempt()});
 assert.equal(confirmed.state,"confirmed");
 evidence.checks.push("independent current advice and confirmation receipt parsed");
 const current=await p.read("document_evidence.advice","client-document",{digest:proposed.digest});
 const prepare:Intent={action:"prepare",target:"client-preparation",fields:{document:"client-document",digest:proposed.digest},rationale:"Explicit internal preparation",revision:current.revision,attempt:attempt()};
 const prepared=await p.submit(prepare),replayed=await p.submit(prepare);
 assert.equal(prepared.state,"prepared");assert.equal(replayed.replay,true);assert.equal(replayed.digest,prepared.digest);
 const history=await loadHistory(p.read,scope,"preparations");
 assert.ok(history.rows.some(r=>"preparation"in r&&r.preparation==="client-preparation"&&r.recorded_receipt===prepared.digest));
 const inspected=await p.read("erp_preparation.inspect","client-preparation",{});
 assert.equal(inspected.fields.ledger_observation,"matching_prepared");assert.equal(inspected.fields.recorded_receipt,prepared.digest);
 evidence.checks.push("real preparation, identical retry and owned receipt discovery parsed");
 const revoke=await c.read("document_evidence.advice","client-document",{digest:proposed.digest});
 assert.equal((await c.submit({action:"revoke",target:"client-document",fields:{digest:proposed.digest},rationale:"Synthetic client revocation",revision:revoke.revision,attempt:attempt()})).state,"revoked");
 assert.equal((await p.read("erp_preparation.inspect","client-preparation",{})).fields.recorded_receipt,prepared.digest);
 evidence.checks.push("revocation preserves preparation history");
 evidence.outcome="passed";
}catch{evidence.outcome="failed";process.exitCode=1;}
finally{await writeFile(args.output,JSON.stringify(evidence,null,2));}
