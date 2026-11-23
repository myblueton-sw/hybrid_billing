import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { DraftError } from "./contract";
import type { Context } from "./context";
import type { Inspection, Recovery } from "./inspection";
import { draftIntent, type DraftIntent } from "./mutation";
function reject():never {throw new DraftError("draft_action_unavailable");}
function base(session:Session,revision:string,rationale:string) {
 return {actor:session.user,workspace:session.workspace,revision,rationale,attempt:attempt()};
}
function selected(context:Context,draft:Inspection) {
 if(context.revision!==draft.revision||context.final_request!==draft.final_request||context.final_digest!==draft.final_digest||context.existing_draft!==draft.draft||context.existing_digest!==draft.digest||draft.origin!=="configured")reject();
}
export function submissionIntent(session:Session,context:Context,language:string,rationale:string):DraftIntent {
 if(context.submit_action!=="ready"||context.submit_permission!=="allowed"||context.current_eligibility!=="eligible"||context.step_up!=="satisfied"||context.existing_draft||!context.configuration||!context.configuration_digest||!context.languages.includes(language))return reject();
 const target=attempt().request;
 return draftIntent({...base(session,context.revision,rationale),action:"submit",target,draft:target,draftDigest:"",arguments:{configuration:context.configuration,configuration_digest:context.configuration_digest,final_request:context.final_request,final_digest:context.final_digest,language}});
}
export function preparationIntent(session:Session,context:Context,draft:Inspection,bindingID:string,rationale:string):DraftIntent {
 selected(context,draft);
 const binding=context.rows.find(b=>b.binding===bindingID);
 if(context.prepare_action!=="ready"||context.approve_permission!=="allowed"||context.step_up!=="satisfied"||context.current_eligibility!=="eligible"||draft.submission_state!=="submitted"||session.user===draft.submitter||session.user===draft.final_requester||!binding)return reject();
 return draftIntent({...base(session,context.revision,rationale),action:"prepare_approval",target:attempt().request,draft:draft.draft,draftDigest:draft.digest,arguments:{draft:draft.draft,draft_digest:draft.digest,binding:binding.binding,binding_digest:binding.binding_digest}});
}
export function approvalIntent(session:Session,context:Context,draft:Inspection,rationale:string):DraftIntent {
 selected(context,draft);
 const review=draft.review;
 if(!["ready","recheck_required"].includes(context.approve_action)||context.approve_permission!=="allowed"||context.step_up!=="satisfied"||context.current_eligibility!=="eligible"||review?.state!=="prepared"||review.actor!==session.user)return reject();
 return draftIntent({...base(session,context.revision,rationale),action:"approve",target:review.id,draft:draft.draft,draftDigest:draft.digest,arguments:{digest:review.digest}});
}
export function recoveryIntent(session:Session,draft:Inspection,recovery:Recovery,rationale:string):DraftIntent {
 const retained=draft.recoveries.find(r=>r.operation===recovery.operation&&r.business_id===recovery.business_id);
 if(!retained||JSON.stringify(retained)!==JSON.stringify(recovery)||retained.original_actor!==session.user||retained.permission!=="allowed"||draft.origin!=="configured")return reject();
 return draftIntent({...base(session,draft.revision,rationale),action:retained.operation,target:retained.business_id,draft:draft.draft,draftDigest:retained.operation==="submit"?"":draft.digest,arguments:retained.arguments});
}
