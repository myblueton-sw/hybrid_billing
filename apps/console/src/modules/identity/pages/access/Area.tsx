import type {ComponentProps} from "react";
import LiveAccess from "./Live";
import AuthorityScreen from "./result-authority/Screen";
import {t} from "./result-authority/copy";
import {areaOf,authorityAreas,words} from "./result-authority/areas";
import "./result-authority/areas.css";
export default function AccessArea(props:ComponentProps<typeof LiveAccess>&{params:URLSearchParams;onState:(state:string,items:string[])=>void;onDirty:(dirty:boolean)=>void}){
 const {locale:l,session}=props,area=props.view==="grants"?areaOf(props.params.get("area")):undefined;
 const refused=!!area&&props.params.has("client")&&props.params.get("client")!=="console";
 const entry=(href:string,current:boolean,text:string)=><li key={href}><a href={href} aria-current={current?"page":undefined} data-label={text}><span>{text}</span></a></li>;
 return <>
  {props.view==="grants"&&<nav aria-label={t(l,"Permission area","권한 영역","مجال الصلاحيات")} className="authority-areas"><ul>{entry(`#/access?view=grants&lang=${l}`,!area,t(l,"Workspace access","업무 공간 접근 권한","الوصول إلى مساحة العمل"))}{authorityAreas.map(item=>entry(`#/access?view=grants&area=${item.param}&lang=${l}`,item===area,words(l,item.title)))}</ul></nav>}
  {refused?<p role="alert">{words(l,area.refusal)}</p>:area?<AuthorityScreen key={JSON.stringify([session.user,session.workspace,session.csrf,area.param])} area={area} session={session} locale={l} proposal={props.params.get("proposal")??""} client={props.params.get("client")??""} onState={props.onState} onDirty={props.onDirty}/>:<LiveAccess {...props}/>}
 </>;
}
