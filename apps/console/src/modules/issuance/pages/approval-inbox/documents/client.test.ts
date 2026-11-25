import test from "node:test";
import assert from "node:assert/strict";
import { strictJSON } from "./json";
import { DocumentError, decimal, type FormContext, type ReadEnvelope } from "./contract";
import { loadContext, loadHistory, type ReadCommand } from "./reads";
import { emptyDraft, packetOf, PacketError, sum, type PacketDraft } from "./packet";
import { documentTransport, type Intent } from "./transport";
const hash = "a".repeat(64), scope = { request: "request", digest: hash, workspace: "workspace" };
const session = { user: "writer", workspace: "workspace", role: "member", login: "writer", csrf: "synthetic-session" };
function header(total = "11") { return { candidate: "candidate", candidate_digest: hash, profile: "synthetic_individual_net_v1", currency: "USD", total, customer: "customer", period: "2026-10", organization: { WorkspaceID: "workspace", IssuerLegalEntityID: "seller", BuyerLegalEntityID: "buyer", PayerID: "payer", Currency: "USD", Path: "external_invoice" }, erp: { connection: "erp", product: "product", tax_code: "tax-code", scale: 2 } }; }
const line = (i: number) => ({ line: `line-${String(i).padStart(2, "0")}`, rule: "rule", kind: "rated", amount: "1" });
function page(start = 0, end = 10, next = "line-09"): ReadEnvelope { return { revision: "r7", at: "2026-11-16T09:00:00Z", fields: { mode: "synthetic", request: "request", request_digest: hash, snapshot: "r7", next, header: JSON.stringify(header()), proposal_permission: JSON.stringify({ state: "allowed", code: "" }), ...Object.fromEntries(Array.from({ length: end-start }, (_, i) => [`row_${String(i).padStart(2,"0")}`, JSON.stringify(line(start+i))])) } }; }
async function context() { let n=0; return loadContext(async()=>page(n++===0?0:10,n===1?10:11,n===1?"line-09":""), scope); }
function draft(c: FormContext): PacketDraft { return { ...emptyDraft(c), selectedERP:true,unit:"unit",evidence:"evidence",net:"11",tax:"0",rationale:"Reviewed explicit synthetic facts",lines:Object.fromEntries(c.lines.map(l=>[l.line,{quantity:"1",tax:"0",evidence:"evidence"}])) }; }
const response=(v:unknown,status=200)=>new Response(JSON.stringify(v),{status});
function intent():Intent{return {action:"confirm",target:"document",fields:{digest:hash},rationale:"Independent review",revision:"r7",attempt:{request:"attempt",key:"key"}};}
function receipt(s:Intent){return {request_id:s.attempt.request,status:"completed",operation_id:"operation",target_id:s.target,result_revision:"r8",business_state:"confirmed",replay:false,as_of:"2026-11-16T09:00:00Z",fields:{mode:"synthetic",document:s.target,digest:hash,recorded_state:"confirmed",receipt:"historical_decision"}};}
function mock(command:(init:RequestInit)=>Response|Promise<Response>, current:()=>Response|Promise<Response>=()=>response({...session,csrf_token:session.csrf})):typeof fetch{return async(input,init)=>String(input).endsWith("/session")?current():command(init!);}
function deferred<T>(){let resolve!:(v:T)=>void;const promise=new Promise<T>(r=>{resolve=r;});return{promise,resolve};}

test("context joins all pages atomically with immutable exact identity",async()=>{
 const calls:Record<string,string>[]=[];
 const read:ReadCommand=async(_c,_t,fields)=>{calls.push(fields);return calls.length===1?page():page(10,11,"");};
 const c=await loadContext(read,scope);
 assert.equal(c.lines.length,11);assert.deepEqual(calls[1],{digest:hash,after:"line-09",snapshot:"r7"});assert.ok(Object.isFrozen(c.lines[0]));
});
for(const change of ["revision","header","permission","cursor","duplicate","gap","workspace","total","overflow"]){
 test(`context refuses ${change} rather than exposing partial lines`,async()=>{
  let calls=0;
  await assert.rejects(loadContext(async()=>{
   calls++;const p=structuredClone(calls===1?page():page(10,11,""));const f=p.fields as Record<string,string>;
   if(calls===2){
    if(change==="revision")f.snapshot="r8";
    if(change==="header")f.header=JSON.stringify({...header(),customer:"other"});
    if(change==="permission")f.proposal_permission=JSON.stringify({state:"denied",code:"permission_denied"});
    if(change==="cursor")f.next="line-10";
    if(change==="duplicate")f.row_00=JSON.stringify(line(9));
    if(change==="gap"){f.row_01=f.row_00;delete f.row_00;}
   }
   if(change==="workspace")f.header=JSON.stringify({...header(),organization:{...header().organization,WorkspaceID:"foreign"}});
   if(change==="total")f.header=JSON.stringify(header("12"));
   if(change==="overflow")f.row_10=JSON.stringify(line(10));
   return p;
  },scope),DocumentError);
 });
}
test("history is independently empty and preserves unresolved receipt",async()=>{
 const history:ReadEnvelope={revision:"r7",at:"2026-11-16T09:00:00Z",fields:{mode:"synthetic",request:"request",request_digest:hash,kind:"preparations",snapshot:"r7",next:"",current_eligibility:"unchecked"}};
 assert.equal((await loadHistory(async()=>history,scope,"preparations")).rows.length,0);
 const row={preparation:"owned",document:"document",document_digest:hash,actor:"writer",recorded_receipt:""};
 const got=await loadHistory(async()=>({...history,fields:{...history.fields,row_00:JSON.stringify(row)}}),scope,"preparations");
 assert.equal((got.rows[0] as typeof row).recorded_receipt,"");
 await assert.rejects(loadHistory(async()=>({...history,fields:{...history.fields,request_digest:"b".repeat(64)}}),scope,"preparations"));
});
for(const text of ['{"x":1,"x":2}','{"x":1,"\\u0078":2}','{"x":{"a":1,"a":2}}','[1,]','{"x":1,}','{"x":1} {}','"\\ud800"','1e999','01','['.repeat(18)+'0'+']'.repeat(18)])test(`strict JSON refuses ${text.slice(0,35)}`,()=>assert.throws(()=>strictJSON(text),DocumentError));
test("strict JSON retains harmless special keys without prototype writes",()=>{const v=strictJSON('{"__proto__":{"x":1},"x":[true,false,null,"😀"]}') as Record<string,unknown>;assert.equal(Object.getPrototypeOf(v),null);assert.deepEqual(v.x,[true,false,null,"😀"]);});
for(const value of ["01","1.0","0.00","+1","1e2","-0","-1",".1","1.","0.001"])test(`document decimals refuse ${value}`,()=>assert.equal(decimal(value,2),false));
test("exact addition does not round binary or large integers",()=>{assert.equal(sum(["0.1","0.2"]),"0.3");assert.equal(sum(["999999999999.99","0.01"]),"1000000000000");assert.equal(sum(["0","0"]),"0");});
test("packet requires explicit evidence and exact declared totals",async()=>{
 const c=await context();assert.throws(()=>packetOf(c,emptyDraft(c)),PacketError);
 const d=draft(c),p=packetOf(c,d);assert.equal(p.packet.Lines.length,11);assert.equal(p.packet.ExpectedTax,"0");assert.ok(Object.isFrozen(p.packet.Lines[0]));
 for(const change of [{net:"12"},{tax:"1"},{selectedERP:false},{unit:""},{rationale:" "},{rationale:"\ud800"}])assert.throws(()=>packetOf(c,{...d,...change}),DocumentError);
 assert.throws(()=>packetOf({...c,permission:{state:"denied",code:"permission_denied"}},d),DocumentError);
 assert.throws(()=>packetOf(c,{...d,lines:{...d.lines,extra:{quantity:"1",tax:"0",evidence:"extra"}}}),PacketError);
 const large={...d,lines:Object.fromEntries(c.lines.map(l=>[l.line,{quantity:"1",tax:"0",evidence:"x".repeat(128)}])),evidence:"x".repeat(128),unit:"x".repeat(128)};
 assert.throws(()=>packetOf(c,large),(e:unknown)=>e instanceof PacketError&&e.fields.includes("packet_size"));
});
test("transport freezes before session and response awaits",async()=>{
 const auth=deferred<Response>(),command=deferred<Response>();let sent:Record<string,unknown>|undefined;
 const s=intent(),original=structuredClone(s);
 const pending=documentTransport(session,mock(async init=>{sent=JSON.parse(String(init.body));return command.promise;},()=>auth.promise)).submit(s);
 (s.fields as Record<string,string>).digest="b".repeat(64);(s as {rationale:string}).rationale="changed";
 auth.resolve(response({...session,csrf_token:session.csrf}));
 for(let i=0;i<20&&!sent;i++)await new Promise(r=>setImmediate(r));
 assert.equal((sent?.body as Record<string,string>).digest,hash);assert.equal(sent?.rationale,original.rationale);
 (s.attempt as {request:string}).request="changed";command.resolve(response(receipt(original)));
 assert.equal((await pending).target,"document");
});
test("invalid intent cannot dispatch or select a command",async()=>{
 let calls=0;const api=documentTransport(session,mock(()=>{calls++;return response({});},()=>{calls++;return response({});}));
 for(const s of [{...intent(),action:"inject"},{...intent(),fields:{}},{...intent(),target:""},{...intent(),rationale:""},{...intent(),attempt:{request:"",key:"key"}}])await assert.rejects(api.submit(s as Intent),DocumentError);
 assert.equal(calls,0);
});
test("session replacement prevents mutation dispatch",async()=>{
 let calls=0;const api=documentTransport(session,mock(()=>{calls++;return response({});},()=>response({...session,csrf_token:"different"})));
 await assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.code==="session_changed"&&e.effect==="none");assert.equal(calls,0);
});
for(const mode of ["valid","wrong-target","wrong-digest","wrong-operation","extra","duplicate"]){test(`mutation response ${mode}`,async()=>{
 const s=intent(),r=receipt(s) as Record<string,unknown>;
 if(mode==="wrong-target")r.target_id="other";
 if(mode==="wrong-digest")(r.fields as Record<string,string>).digest="b".repeat(64);
 if(mode==="wrong-operation")r.operation_id="";
 if(mode==="extra")r.extra=true;
 const api=documentTransport(session,mock(()=>mode==="duplicate"?new Response(JSON.stringify(r).replace('{','{"status":"bad",')):response(r)));
 if(mode==="valid")assert.equal((await api.submit(s)).state,"confirmed");else await assert.rejects(api.submit(s),(e:unknown)=>e instanceof DocumentError&&e.effect==="unknown");
});}
for(const mode of ["clean","operation","identity","status","extra","incomplete","wrong-next","wrong-retry"]){test(`refusal ${mode} preserves effect distinction`,async()=>{
 const body:Record<string,unknown>={code:"permission_denied",request_id:"attempt",retry:false,next_action:"none"};
 if(mode==="operation")body.operation_id="operation";
 if(mode==="identity")body.request_id="different";
 if(mode==="extra")body.business_state="confirmed";
 if(mode==="incomplete")delete body.next_action;
 if(mode==="wrong-next")body.next_action="correct_request";
 if(mode==="wrong-retry")body.retry=true;
 const api=documentTransport(session,mock(()=>response(body,mode==="status"?500:403)));
 await assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.effect===(mode==="clean"?"none":"unknown")&&e.operation===(mode==="operation"?"operation":""));
});}

test("session lookup timeout has no dispatched effect",async t=>{
 t.mock.timers.enable({apis:["setTimeout"]});let commands=0;
 const api=documentTransport(session,mock(()=>{commands++;return response({});},()=>new Promise(()=>{})));
 const result=assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.effect==="none"&&e.code==="session_unavailable");
 t.mock.timers.tick(30001);await result;assert.equal(commands,0);
});
test("timeout after dispatch remains unknown",async t=>{
 t.mock.timers.enable({apis:["setTimeout"]});let dispatched=false;
 const api=documentTransport(session,mock(()=>{dispatched=true;return new Promise(()=>{});}));
 const result=assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.effect==="unknown"&&e.code==="request_timeout");
 for(let i=0;i<20&&!dispatched;i++)await new Promise(r=>setImmediate(r));
 assert.ok(dispatched);t.mock.timers.tick(30001);await result;
});
test("command response byte limit stops stream before unbounded buffering",async()=>{
 let cancelled=false;
 const stream=new ReadableStream<Uint8Array>({pull(c){c.enqueue(new Uint8Array(60000).fill(32));},cancel(){cancelled=true;}});
 const api=documentTransport(session,mock(()=>new Response(stream)));
 await assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.effect==="unknown");assert.ok(cancelled);
});
test("malformed UTF-8 cannot become a valid mutation receipt",async()=>{
 const api=documentTransport(session,mock(()=>new Response(new Uint8Array([0xff]))));
 await assert.rejects(api.submit(intent()),(e:unknown)=>e instanceof DocumentError&&e.effect==="unknown");
});
