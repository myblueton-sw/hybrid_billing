import type {Attempt} from "../../../../../service/admin";
import {AuthorityError,frozen,id,instant,revision} from "./contract";
import type {Inspection} from "./inspection";
import {freezeIntent,type Intent} from "./mutations";
export type ReviewAction="approve"|"reject"|"withdraw"|"recover";
export type ActionReview=Readonly<{action:ReviewAction;detail:Inspection;intent:Intent}>;
export function reviewActions(detail:Inspection,actor:string):readonly ReviewAction[]{
 if(!id(actor))return frozen([]);
 const row=detail.row;
 if(row.state==="decision_unknown")return frozen(detail.decision?.actor===actor?["recover"]:[]);
 if(row.state!=="pending")return frozen([]);
 if(actor===row.proposer)return frozen(["withdraw"]);
 if(row.kind==="human_grant"&&actor===row.recipient)return frozen([]);
 return frozen(["approve","reject"]);
}
export function reviewDecision(detail:Inspection,actor:string,action:ReviewAction,rationale:string,attempt:Attempt):ActionReview{
 if(!revision(detail.revision)||instant(detail.at)===undefined||!reviewActions(detail,actor).includes(action))throw new AuthorityError("decision_review_unavailable");
 if(action==="recover"&&rationale!==detail.decision?.rationale)throw new AuthorityError("decision_review_unavailable");
 const intent=freezeIntent({action:action==="recover"?"approve":action,target:detail.row.proposal,fields:{digest:detail.row.digest},rationale,revision:detail.revision,attempt});
 return frozen({action,detail:structuredClone(detail),intent});
}
