import {instant,seconds,type Ceiling,type Policy} from "./contract";
export function validityProblems(before:string,after:string,policy:Policy,ceiling?:Ceiling):readonly string[]{
 const start=instant(before),end=instant(after),errors:string[]=[];
 if(start===undefined)errors.push("not_before");if(end===undefined)errors.push("not_after");
 if(!seconds(policy.max_grant_validity_seconds))return ["policy"];
 if(start!==undefined&&end!==undefined){
  if(start>=end||end-start>BigInt(policy.max_grant_validity_seconds)*1000000000n)errors.push("not_after");
  if(ceiling){const low=instant(ceiling.not_before),high=instant(ceiling.not_after);if(low===undefined||high===undefined||low>=high)return ["ceiling"];if(start<low)errors.push("not_before");if(end>high)errors.push("not_after");}
 }
 return [...new Set(errors)];
}
