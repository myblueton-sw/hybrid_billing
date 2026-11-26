import {AuthorityError,frozen,resultDomain,type AuthorityDomain,type Capability,type Read} from "./contract";
import {contextPage,type Cursor} from "./reads";
import type {ProposalContext,ProposalKind} from "./proposal-model";
export async function proposalContext(read:Read,workspace:string,actor:string,kind:ProposalKind,capability:Capability,cursors:{ceilings?:Cursor;recipients?:Cursor}={},d:AuthorityDomain=resultDomain):Promise<ProposalContext>{
 const [ceilings,recipients]=await Promise.all([contextPage(read,workspace,actor,"ceilings",capability,cursors.ceilings,d),kind==="human_grant"?contextPage(read,workspace,actor,"recipients",capability,cursors.recipients,d):undefined]);
 if(recipients&&(recipients.revision!==ceilings.revision||JSON.stringify(recipients.policy)!==JSON.stringify(ceilings.policy)))throw new AuthorityError("evidence_stale");
 return frozen({kind,capability,ceilings,recipients});
}
