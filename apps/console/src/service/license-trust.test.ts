import { test } from "node:test";
import assert from "node:assert/strict";
import { publicKeyFile, proposalNumber, trustPageOf, trustOutcome, utcEpoch, readTrust } from "./license-trust";
import { AdminError } from "./admin";
const key={id:'issuer-key',issuer:'issuer',fingerprint_sha256:'a'.repeat(64),not_before:1,not_after:2,accept_until:3};
const proposal={number:'18446744073709551615',target:'license-change-18446744073709551615',kind:'key',state:'pending',proposer:'first',approver:'',key};
const page=(record:unknown)=>({state:'configured',revision:'r9',record});
const file={key_id:'issuer-key',issuer:'issuer',public_key:'a'.repeat(64),proof:'b'.repeat(128),not_before:1,not_after:2,accept_until:3};
test('proposal identity retains uint64 precision and canonical form',()=>{
 assert.equal(proposalNumber(proposal.number),proposal.number);
 for(const v of ['0','01','1.0','1e1','-1','18446744073709551616',1])assert.throws(()=>proposalNumber(v));
 assert.equal(trustPageOf(page(proposal),'proposal',proposal.number).state,'configured');
 assert.throws(()=>trustPageOf(page(proposal),'proposal','1'));
});
test('key reads bind exact identity and retain independent retirement and compromise flags',()=>{
 const p=trustPageOf(page({key,retired:true,compromised:true}),'key',key.id);assert.equal(p.state,'configured');
 assert.throws(()=>trustPageOf(page({key,retired:true,compromised:true}),'key','other'));
 for(const change of [{fingerprint_sha256:'A'.repeat(64)},{not_before:0},{not_after:1},{accept_until:1}])assert.throws(()=>trustPageOf(page({key:{...key,...change},retired:false,compromised:false}),'key',key.id));
});
test('proposal shape actor state and target mismatches are refused',()=>{
 for(const change of [{target:'license-change-1'},{kind:'reset_high_water'},{state:'withdrawn'},{state:'applied',approver:'first'},{state:'applied',approver:''},{approver:'second'},{target_key:'extra'},{retire_at:2},{key:null}])assert.throws(()=>trustPageOf(page({...proposal,...change}),'proposal',proposal.number));
 assert.equal(trustPageOf(page({...proposal,state:'applied',approver:'second'}),'proposal',proposal.number).state,'configured');
});
test('retirement and compromise proposal shapes cannot masquerade as one another',()=>{
 const {key:_,...base}=proposal;
 for(const kind of ['retire_key','compromise_key']){const value={...base,kind,target_key:'issuer-key',...(kind==='retire_key'?{retire_at:2}:{})};assert.equal(trustPageOf(page(value),'proposal',proposal.number).state,'configured');}
 assert.throws(()=>trustPageOf(page({...base,kind:'retire_key',target_key:'issuer-key'}),'proposal',proposal.number));
 assert.throws(()=>trustPageOf(page({...base,kind:'compromise_key',target_key:'issuer-key',retire_at:2}),'proposal',proposal.number));
});
test('absent or unconfigured reads discard injected prior records',()=>{
 for(const state of ['not_found','uninitialized','not_configured'])assert.deepEqual(trustPageOf({state,record:proposal},'proposal','1'),{state});
});
test('public-key file has exact bounded public schema and canonical bytes',()=>{
 assert.deepEqual(publicKeyFile(JSON.stringify(file)),Object.fromEntries(Object.entries(file).map(([k,v])=>[k,String(v)])));
 for(const change of [{private_key:'x'},{public_key:'a'.repeat(128)},{public_key:'A'.repeat(64)},{proof:'b'.repeat(126)},{not_before:1.5},{not_after:'2'},{not_after:1},{accept_until:1},{issuer:'invalid id'},{key_id:''}])assert.throws(()=>publicKeyFile(JSON.stringify({...file,...change})));
 assert.throws(()=>publicKeyFile(' '.repeat(4001)));assert.throws(()=>publicKeyFile('[]'));
});
test('retirement UTC values reject impossible dates local times and unsupported bounds',()=>{
 assert.equal(utcEpoch('2026-12-01T00:00:00Z'),String(Date.parse('2026-12-01T00:00:00Z')/1000));
 for(const v of ['2026-02-30T00:00:00Z','2026-12-01T00:00:00','2026-12-01T00:00:00+00:00','1970-01-01T00:00:00Z'])assert.throws(()=>utcEpoch(v));
});
test('command success must match operation and exact canonical proposal target',()=>{
 assert.equal(trustOutcome('licenses.propose_key','pending',{outcome:'pending',proposal:'1',target:'license-change-1'},'license-installation'),'1');
 assert.equal(trustOutcome('licenses.approve_key_change','applied',{outcome:'applied',proposal:'1',target:'license-change-1'},'license-change-1'),'1');
 assert.equal(trustOutcome('licenses.withdraw_key_change','withdrawn',{outcome:'withdrawn',proposal:'1',target:'license-change-1'},'license-change-1'),'1');
 assert.throws(()=>trustOutcome('licenses.propose_key','applied',{outcome:'applied',proposal:'1',target:'license-change-1'},'license-installation'));
 assert.throws(()=>trustOutcome('licenses.approve_key_change','applied',{outcome:'applied',proposal:'2',target:'license-change-2'},'license-change-1'));
 assert.throws(()=>trustOutcome('licenses.propose_key','pending',{outcome:'pending',proposal:'1',target:'license-change-01'},'license-installation'));
 assert.throws(()=>trustOutcome('licenses.propose_key','pending',{outcome:'pending'},'license-installation'));
});
test('settled domain refusal is distinct from malformed success',()=>{
 assert.equal(trustOutcome('licenses.propose_key','no_proof',{outcome:'no_proof'},'license-installation'),undefined);
 assert.equal(trustOutcome('licenses.withdraw_key_change','not_pending',{outcome:'not_pending'},'license-change-1'),undefined);
 assert.throws(()=>trustOutcome('licenses.propose_key','time_unavailable',{outcome:'time_unavailable'},'license-installation'));
 assert.throws(()=>trustOutcome('licenses.propose_key','no_proof',{outcome:'no_proof',proposal:'1'},'license-installation'));
});
test('read uses exact encoded endpoint and malformed response fails closed',async()=>{
 let endpoint='';const api={read:async(p:string)=>{endpoint=p;return page({...proposal,target:'license-change-1'});}};
 await assert.rejects(readTrust(api,'proposal',proposal.number),e=>e instanceof AdminError&&e.failure==='invalid');assert.equal(endpoint,'/licenses/key-changes/'+proposal.number);
});
