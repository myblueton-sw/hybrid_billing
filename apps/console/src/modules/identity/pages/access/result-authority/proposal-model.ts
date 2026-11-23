import type {Attempt} from "../../../../../service/admin";
import {AuthorityError,capability,frozen,rationaleValid,type Capability,type Ceiling,type Recipient} from "./contract";
import type {ContextPage} from "./reads";
import {validityProblems} from "./validity";
import {freezeIntent,type Intent} from "./mutations";
export type ProposalKind="console_ceiling"|"human_grant";
export type ProposalDraft=Readonly<{recipient:string;ceiling:string;notBefore:string;notAfter:string;rationale:string}>;
export type ProposalContext=Readonly<{kind:ProposalKind;capability:Capability;ceilings:ContextPage;recipients?:ContextPage}>;
export type ProposalReview=Readonly<{context:ProposalContext;draft:ProposalDraft;intent:Intent}>;
export class ProposalError extends AuthorityError{constructor(readonly fields:readonly string[]){super("invalid_authority_proposal");}}
export function reviewProposal(context:ProposalContext,draft:ProposalDraft,target:string,attempt:Attempt):ProposalReview{
 const {kind,capability:selected,ceilings,recipients}=context,errors:string[]=[];
 if(!["console_ceiling","human_grant"].includes(kind)||!capability(selected)||ceilings.collection!=="ceilings"||ceilings.capability!==selected||!ceilings.policy.capabilities.includes(selected))throw new ProposalError(["context"]);
 if(kind==="human_grant"&&(!recipients||recipients.collection!=="recipients"||recipients.capability!==selected||recipients.revision!==ceilings.revision||JSON.stringify(recipients.policy)!==JSON.stringify(ceilings.policy)))throw new ProposalError(["context"]);
 const ceiling=ceilings.rows.find((row):row is Ceiling=>"ceiling" in row&&row.ceiling===draft.ceiling);
 const recipient=recipients?.rows.find((row):row is Recipient=>"user" in row&&row.user===draft.recipient);
 if(kind==="human_grant"&&!ceiling)errors.push("ceiling");
 if(kind==="human_grant"&&!recipient)errors.push("recipient");
 if(kind==="console_ceiling"&&(draft.recipient||draft.ceiling))errors.push("context");
 errors.push(...validityProblems(draft.notBefore,draft.notAfter,ceilings.policy,kind==="human_grant"?ceiling:undefined));
 if(!rationaleValid(draft.rationale))errors.push("rationale");
 if(errors.length)throw new ProposalError(frozen([...new Set(errors)]));
 const fields:Record<string,string>={capability:selected,not_before:draft.notBefore,not_after:draft.notAfter};
 if(kind==="human_grant"){fields.recipient=draft.recipient;fields.ceiling=draft.ceiling;}
 const intent=freezeIntent({action:kind==="console_ceiling"?"propose_console":"propose_human",target,fields,rationale:draft.rationale,revision:ceilings.revision,attempt});
 return frozen(structuredClone({context,draft,intent}));
}
