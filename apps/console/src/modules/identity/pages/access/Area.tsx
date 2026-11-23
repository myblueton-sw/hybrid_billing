import type {ComponentProps} from "react";
import LiveAccess from "./Live";
import AuthorityScreen from "./result-authority/Screen";
import {t} from "./result-authority/copy";
export default function AccessArea(props:ComponentProps<typeof LiveAccess>&{params:URLSearchParams;onState:(state:string,items:string[])=>void;onDirty:(dirty:boolean)=>void}){
 const {locale:l,session}=props,result=props.view==="grants"&&props.params.get("area")==="result";
 if(result&&props.params.has("client")&&props.params.get("client")!=="console")return <p role="alert">{t(l,"This client cannot be provisioned through console result permissions.","이 클라이언트는 콘솔 결과 권한 부여 대상이 아닙니다.","لا يمكن منح هذا العميل صلاحيات النتائج الخاصة بوحدة التحكم.")}</p>;
 return <>
  {props.view==="grants"&&<nav aria-label={t(l,"Permission area","권한 영역","مجال الصلاحيات")} className="authority-actions"><a href={`#/access?view=grants&lang=${l}`} aria-current={!result?"page":undefined}>{t(l,"Workspace access","업무 공간 접근 권한","الوصول إلى مساحة العمل")}</a><a href={`#/access?view=grants&area=result&lang=${l}`} aria-current={result?"page":undefined}>{t(l,"ERP result permissions","ERP 결과 권한","صلاحيات نتائج ERP")}</a></nav>}
  {result?<AuthorityScreen key={JSON.stringify([session.user,session.workspace,session.csrf])} session={session} locale={l} proposal={props.params.get("proposal")??""} client={props.params.get("client")??""} onState={props.onState} onDirty={props.onDirty}/>:<LiveAccess {...props}/>}
 </>;
}
