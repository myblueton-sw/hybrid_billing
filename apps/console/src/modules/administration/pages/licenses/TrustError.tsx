import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import { k } from "./live-copy";
export type TrustProblem={field:string;text:string};
export default function TrustError({l,problem}:{l:Locale;problem:TrustProblem|null}) {
 const node=useRef<HTMLDivElement>(null);useEffect(()=>{if(problem)node.current?.focus();},[problem]);
 return problem?<div className="p07-errors" tabIndex={-1} ref={node} role="alert"><h4>{k(l,"errorTitle")}</h4><a href={'#'+problem.field} onClick={e=>{e.preventDefault();document.getElementById(problem.field)?.focus();}}>{problem.text}</a></div>:null;
}
