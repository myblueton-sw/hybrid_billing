import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { AdminError, attempt } from "../../../../service/admin";
import { useAdminClient, useService } from "../../../../service/context";
import { licenseOutcomes, readLicense, type LicensePage } from "../../../../service/licenses";
import type { Props } from "../../../../shared/domain-screen-ui/operations/types";
import { Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import Notices from "./Notices";
import LiveForm, { type SubmitLicense } from "./LiveForm";
import { k, label } from "./live-copy";
import { licenseSession } from "./live-session";
import "./live.css";

export default function LiveLicenses(p: Props & { onSignOut: () => void }) {
 const l=p.locale, api=useAdminClient(), service=useService()!;
 const controller=licenseSession(service.session), snapshot=useSyncExternalStore(controller.subscribe,controller.snapshot);
 const busy=snapshot.pending, result=snapshot.result;
 const [page,setPage]=useState<LicensePage|null>(null), [phase,setPhase]=useState<"loading"|"ready"|"error"|"denied"|"signedOut">("loading");
 const alive=useRef(true), sequence=useRef(0), resultNode=useRef<HTMLDivElement>(null);
 const load=useCallback(async()=>{
  const n=++sequence.current;setPage(null);setPhase("loading");
  try {const value=await readLicense(api);if(!alive.current || n!==sequence.current)return;setPage(value);setPhase("ready");}
  catch(e){if(!alive.current || n!==sequence.current)return;setPage(null);setPhase(e instanceof AdminError && e.failure==="denied"?"denied":e instanceof AdminError && e.failure==="unauthenticated"?"signedOut":"error");}
 },[api]);
 useEffect(()=>{alive.current=true;if(snapshot.result?.access){setPage(null);setPhase(snapshot.result.access);}else{void load();}return()=>{alive.current=false;sequence.current++;};},[load,snapshot.settled]);
 useEffect(()=>{p.onState(JSON.stringify({screen:"P07",view:p.view,source:phase,state:page?.state??"",outcome:result?.outcome??"",unknown:!!result?.unknown}),["P07:data:license-status",`P07:state:${phase}`,`P07:data:${p.view}`,"P07:action:reload","P07:action:initialize","P07:action:select-mode","P07:action:register","P07:action:activate","P07:field:rationale","P07:field:signed-envelope","P07:field:terms-acceptance"]);},[phase,page,p.view,p.onState,result]);
 useEffect(()=>{if(result)resultNode.current?.focus();},[result]);
 const submit:SubmitLicense=async(command,fields,rationale)=>{
  if(!page || phase!=="ready" || !controller.begin())return;
  const tried=attempt();
  try {
   const outcome=await api.command(command,"license-installation",page.revision,rationale,fields,tried);
   const name=outcome.fields.outcome;
   if(outcome.status!=="completed" || !licenseOutcomes.includes(name as typeof licenseOutcomes[number]) || outcome.state!==name || !/^r[0-9]+$/.test(outcome.revision??""))throw new AdminError("partial");
   controller.settle({outcome:name,request:tried.request,operation:"",unknown:false});
  } catch(e) {
   if(!(e instanceof AdminError) || ["partial","unavailable","invalid"].includes(e.failure)) {
    controller.settle({outcome:"",request:tried.request,operation:e instanceof AdminError?e.operation:"",unknown:true});
   } else {
    controller.settle({outcome:"",request:tried.request,operation:e.operation,unknown:false,refusal:true,...(e.failure==="denied"?{access:"denied" as const}:e.failure==="unauthenticated"?{access:"signedOut" as const}:{})});
   }
  }
 };
 const configured=page?.state==="configured", status=configured?page.status:null;
 const facts: [string, React.ReactNode][] = page && page.state!=="not_configured" ? [[k(l,"installation"),<bdi dir="ltr">{page.installation}</bdi>],[k(l,"observed"),<bdi dir="ltr">{page.observed_at}</bdi>],[k(l,"authority"),k(l,page.time_authority)]]:[];
 if(status)facts.push([k(l,"mode"),label(l,status.mode)],[k(l,"state"),label(l,status.state)],[k(l,"reason"),label(l,status.reason)],...(["evaluation_active","evaluation_expired"].includes(status.state)?[[k(l,"remaining"),status.remaining_seconds] as [string,React.ReactNode]]:[]),[k(l,"lastVerified"),label(l,status.last_verified)],[k(l,"lastVerifiedAt"),<bdi dir="ltr">{status.last_verified_at||"—"}</bdi>]);
 return <div className="ops-grid p07-live" data-live-state={phase} data-live-view={p.view} aria-busy={busy || phase==="loading" || undefined}>
  <div>
   {p.view==="notices"?<Notices locale={l}/>:p.view==="support"?<Panel title={k(l,"supportTitle")}><p>{k(l,"support")}</p></Panel>:<>
    <div className="p07-head"><h2>{k(l,p.view==="offline"?"offline":"title")}</h2><button type="button" onClick={()=>void load()} disabled={busy}>{k(l,"reload")}</button></div>
    {phase!=="ready" && <div className="p07-state" role={phase==="loading"?"status":"alert"}><p>{k(l,phase==="loading"?"loading":phase==="denied"?"denied":phase==="signedOut"?"signedOut":"failed")}</p>{phase==="signedOut"&&<button type="button" onClick={p.onSignOut}>{k(l,"signIn")}</button>}</div>}
    {result && <div className="p07-state" ref={resultNode} tabIndex={-1} role={result.unknown?"alert":"status"}><strong>{k(l,result.unknown?"unknown":result.refusal?"refused":"outcome")}</strong>{!result.unknown&&!result.refusal&&<><p>{label(l,result.outcome)}</p><p>{k(l,"resultNote")}</p></>}<p>{k(l,"request")}: <bdi dir="ltr">{result.request}</bdi></p>{result.operation&&<p>{k(l,"operation")}: <bdi dir="ltr">{result.operation}</bdi></p>}</div>}
    {page?.state==="not_configured"&&<p className="p07-state">{k(l,"notConfigured")}</p>}
    {page?.state==="uninitialized"&&<p className="p07-state">{k(l,"uninitialized")}</p>}
    {status&&<div className="p07-state"><strong>{label(l,status.state)}</strong>{status.reason&&<p>{label(l,status.reason)}</p>}</div>}
    {!!facts.length&&<dl className="p07-facts">{facts.map(([name,value])=><div key={name}><dt>{name}</dt><dd>{value}</dd></div>)}</dl>}
    {status?.entitlement&&<Panel title={k(l,"entitlement")}><dl className="p07-facts"><div><dt>{k(l,"record")}</dt><dd><bdi dir="ltr">{status.entitlement.id}</bdi></dd></div><div><dt>{k(l,"channel")}</dt><dd>{label(l,status.entitlement.channel)}</dd></div><div><dt>{k(l,"metric")}</dt><dd>{label(l,status.entitlement.metric)}</dd></div><div><dt>{k(l,"subject")}</dt><dd>{status.entitlement.subject}</dd></div><div><dt>{k(l,"agreement")}</dt><dd><bdi dir="ltr">{status.entitlement.agreement}</bdi></dd></div><div><dt>{k(l,"scope")}</dt><dd>{status.entitlement.scope.join(", ")}</dd></div><div><dt>{k(l,"validity")}</dt><dd><bdi dir="ltr">{new Date(status.entitlement.start*1000).toISOString()} – {status.entitlement.perpetual?k(l,"perpetual"):new Date(status.entitlement.end*1000).toISOString()}</bdi></dd></div></dl></Panel>}
    {status?.evaluation&&<Panel title={k(l,"evaluation")}><dl className="p07-facts"><div><dt>{k(l,"record")}</dt><dd><bdi dir="ltr">{status.evaluation.grant}</bdi></dd></div><div><dt>{k(l,"scope")}</dt><dd>{status.evaluation.scope.join(", ")}</dd></div><div><dt>{k(l,"subject")}</dt><dd>{status.evaluation.subject}</dd></div><div><dt>{k(l,"terms")}</dt><dd><bdi dir="auto">{status.evaluation.terms}</bdi></dd></div><div><dt>{k(l,"validity")}</dt><dd><bdi dir="ltr">{status.evaluation.start} – {status.evaluation.end}</bdi></dd></div></dl></Panel>}
    {page&&phase==="ready"&&!result?.unknown&&<LiveForm key={p.view+":"+page.revision} l={l} page={page} offline={p.view==="offline"} busy={busy} submit={submit}/>}
   </>}
  </div>
  <aside><Panel title={k(l,"rights")}><p>{k(l,"rightsNote")}</p></Panel><Panel title="Marketplace"><p>{k(l,"support")}</p></Panel></aside>
 </div>;
}
