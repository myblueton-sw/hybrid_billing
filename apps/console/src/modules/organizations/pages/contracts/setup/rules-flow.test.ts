import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import type { ProtectedRules, ProtectedSummary } from "./protected";
import { rulesError } from "./rules-copy";
import { anyStepDirty, sessionStepStore } from "./step-store";
import { confirmRulesIntent, proposeRulesIntent, ruleOffers, secondsInput } from "./rules-flow";
const hash="a".repeat(64),session={user:"me",workspace:"workspace",csrf:"csrf"},attempt={request:"request-1",key:"key-1"};
const none:ProtectedRules={state:"none",observation:"absent",id:"",digest:"",approvals:"",validitySeconds:"",maxLeadSeconds:"",channelAgeSeconds:"",proposer:"",confirmer:"",pendingConfirmer:"",proposedAt:"",confirmedAt:""};
const proposed:ProtectedRules={state:"proposed",observation:"pending",id:"rules-1",digest:hash,approvals:"2",validitySeconds:"172800",maxLeadSeconds:"604800",channelAgeSeconds:"86400",proposer:"other",confirmer:"",pendingConfirmer:"",proposedAt:"2026-11-21T01:00:00Z",confirmedAt:""};
const field={subjectKind:"customer",changes:0,otherProvenance:0,unownedModule:0,latestScheduled:"",latestScheduledEffective:""};
const summary=(rules:ProtectedRules):ProtectedSummary=>({revision:"r5",at:"2026-11-24T09:00:00Z",scope:{contract:"contract-a",configuration:"configuration-a",digest:hash},rules,channels:0,channelsOtherProvenance:0,channelsUnownedModule:0,fields:{document_recipient:field,document_template:field,payment_account:field},currentEligibility:"unchecked"});
const refused=(run:()=>unknown)=>assert.throws(run,(e:unknown)=>e instanceof SetupError&&e.code==="invalid_setup_request");
const draft={approvals:"2",validity:"2",validityUnit:"days",lead:"7",leadUnit:"days",channelAge:"24",channelAgeUnit:"hours",reason:" Agreed control rules "};
test("proposing is offered only when no rules exist at all, and confirming only for a pending proposal of another person",()=>{
 assert.deepEqual(ruleOffers(summary(none),"me"),{propose:true,confirm:"none"});
 assert.deepEqual(ruleOffers(summary({...none,observation:"unowned"}),"me"),{propose:false,confirm:"none"});
 assert.deepEqual(ruleOffers(summary(proposed),"me"),{propose:false,confirm:"offer"});assert.equal(ruleOffers(summary({...proposed,proposer:"me"}),"me").confirm,"own");
 for(const rules of [{...proposed,state:"registration_unknown",observation:"absent",proposedAt:""},{...proposed,state:"confirmation_unknown",pendingConfirmer:"me"},{...proposed,state:"confirmed",observation:"active",confirmer:"third",confirmedAt:"2026-11-21T02:00:00Z"}])assert.deepEqual(ruleOffers(summary(rules),"me"),{propose:false,confirm:"none"});
});
test("a duration is a whole number of hours or days inside the bounds of its rule, with no default unit",()=>{
 assert.equal(secondsInput("24","hours","channelAge"),"86400");assert.equal(secondsInput(" 2 ","days","validity"),"172800");assert.equal(secondsInput("1","hours","validity"),"3600");assert.equal(secondsInput("366","days","lead"),"31622400");assert.equal(secondsInput("90","days","channelAge"),"7776000");
 for(const [value,unit,key] of [["23","hours","channelAge"],["91","days","channelAge"],["31","days","validity"],["0","hours","validity"],["367","days","lead"],["1.5","days","validity"],["","days","validity"],["2","","validity"],["2","weeks","validity"],["-1","days","validity"]] as const)refused(()=>secondsInput(value,unit,key));
});
test("a proposal carries the entered rules in seconds, the reason and the read revision",()=>{
 const intent=proposeRulesIntent(session,summary(none),"rules-new",draft,attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision,scope:intent.scope,actor:intent.actor},{operation:"propose_protected_rules",args:{rules:"rules-new",approvals:"2",validity_seconds:"172800",max_lead_seconds:"604800",channel_age_seconds:"86400"},rationale:"Agreed control rules",revision:"r5",scope:summary(none).scope,actor:"me"});
 for(const change of [{approvals:""},{approvals:"4"},{reason:" "},{validityUnit:""},{channelAge:"1"},{lead:""}])refused(()=>proposeRulesIntent(session,summary(none),"rules-new",{...draft,...change},attempt));
 refused(()=>proposeRulesIntent(session,summary(proposed),"rules-new",draft,attempt));refused(()=>proposeRulesIntent(session,summary({...none,observation:"unowned"}),"rules-new",draft,attempt));
});
test("a confirmation names the retained rules and digest, needs a reason and is never built for the proposer",()=>{
 const intent=confirmRulesIntent(session,summary(proposed)," Reviewed ",attempt);
 assert.deepEqual({operation:intent.operation,args:intent.args,rationale:intent.rationale,revision:intent.revision},{operation:"confirm_protected_rules",args:{rules:"rules-1",rules_digest:hash},rationale:"Reviewed",revision:"r5"});
 refused(()=>confirmRulesIntent(session,summary(proposed)," ",attempt));refused(()=>confirmRulesIntent(session,summary({...proposed,proposer:"me"}),"Reviewed",attempt));refused(()=>confirmRulesIntent(session,summary(none),"Reviewed",attempt));
});
test("non-canonical and boundary durations are normalised or refused, and both intents carry the session",()=>{
 assert.equal(secondsInput("030","days","validity"),"2592000");assert.equal(secondsInput("720","hours","validity"),"2592000");assert.equal(secondsInput("2160","hours","channelAge"),"7776000");
 refused(()=>secondsInput("721","hours","validity"));refused(()=>secondsInput("99999","days","lead"));refused(()=>secondsInput("123456","hours","lead"));
 for(const intent of [proposeRulesIntent(session,summary(none),"rules-new",draft,attempt),confirmRulesIntent(session,summary(proposed),"Reviewed",attempt)])assert.deepEqual({workspace:intent.workspace,csrf:intent.sessionCSRF,actor:intent.actor,attempt:intent.attempt,scope:intent.scope},{workspace:"workspace",csrf:"csrf",actor:"me",attempt,scope:summary(none).scope});
});
test("each refusal code of the rule steps has an open-ended sentence, and a server refusal of the request shape is not explained with the form sentence",()=>{
 const codes=["permission_denied","self_approval_denied","evidence_stale","prerequisite_failed","invalid_setup_request","invalid_confirm","unresolved_operation","reauthentication_required","recovery_read","session_capacity","dependency_unavailable"];
 for(const l of ["en","ko","ar"] as const)assert.equal(new Set(codes.map(code=>rulesError(l,code))).size,codes.length);
 assert.equal(rulesError("en","invalid_request"),rulesError("en","dependency_unavailable"));assert.equal(rulesError("en","revision_conflict"),rulesError("en","evidence_stale"));
 for(const code of ["permission_denied","evidence_stale","prerequisite_failed"])assert.ok(/may|Possible causes/.test(rulesError("en",code)),code);
});
test("unsent input or an unresolved request in any step store of the page counts for the unload guard",async()=>{
 const owner={user:"guard-user",workspace:"workspace",csrf:"guard-csrf"};
 const a=sessionStepStore("guard-a",owner,()=>async()=>"receipt"),b=sessionStepStore("guard-b",owner,()=>async()=>"receipt");
 assert.equal(anyStepDirty(),false);b.edit("form",{reason:"x"});assert.equal(anyStepDirty(),true);b.clear("form");assert.equal(anyStepDirty(),false);
 a.edit("form",{reason:"y"});assert.equal(anyStepDirty(),true);a.clear("form");
});
