import { DraftActions } from "./Actions";
import { DraftEvidence } from "./Evidence";
import { DraftOutcomes } from "./Outcomes";
import { DraftFeedback,draftStateLabel } from "./display";
import { useEffect,useRef,useState } from "react";
import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
import { fixedMoney } from "./contract";
import { useDrafts } from "./use-drafts";
import "./screen.css";
const items=["list-scope","list-paging","list-row","inspect","action-state","term-choice","owned-recovery","flow"].map(id=>`B09:draft:${id}`);
const mono=(value:string|number)=><bdi dir="ltr">{value}</bdi>;
export default function DraftScreen({locale:l,onState,onDirty}:{locale:Locale;onState:(state:string,items:string[])=>void;onDirty?:(dirty:boolean)=>void}) {
 const binding=useDrafts(),{store,state}=binding;
 const [editing,setEditing]=useState(false);
 const [pages,setPages]=useState<string[]>([""]),[lines,setLines]=useState<string[]>([""]);
 const heading=useRef<HTMLHeadingElement>(null),page=state.list.value,draft=state.inspection.value;
 useEffect(()=>{if(store){setPages([""]);setLines([""]);const id=new URLSearchParams(location.hash.split("?")[1]).get("draft-id");let active=true;void store.list().then(()=>{if(active&&id)void store.open(id);});return()=>{active=false;};}},[store]);
 useEffect(()=>{onDirty?.(binding.unresolved||editing);return()=>onDirty?.(false);},[binding.unresolved,editing,onDirty]);
 useEffect(()=>{onState(JSON.stringify({view:"drafts",state:state.inspection.error||state.list.error?"error":state.inspection.loading||state.list.loading?"loading":draft?"draft-detail":"draft-list"}),items);},[state.inspection.error,state.list.error,state.inspection.loading,state.list.loading,draft,onState]);
 useEffect(()=>{if(draft)heading.current?.focus();},[draft]);
 function discard(){return !editing||window.confirm(t(l,"Discard the unsaved action form?","저장하지 않은 작업 입력을 버릴까요?","هل تريد تجاهل بيانات الإجراء غير المحفوظة؟"));}
 function inspect(id:string){if(discard()){setLines([""]);void store?.open(id);}}
 function refresh(){if(!discard())return;setPages([""]);setLines([""]);void store?.list();}
 function move(after:string,back=false){if(!page||!discard())return;setPages(back?pages.slice(0,-1):[...pages,after]);setLines([""]);void store?.list(after,after?page.snapshot:"");}
 function linePage(after:string,back=false){if(!draft||!discard())return;setLines(back?lines.slice(0,-1):[...lines,after]);void store?.open(draft.draft,after,after?draft.snapshot:"");}
 return <section className="b09-drafts" aria-label={t(l,"Issuance drafts","발행 초안","مسودات الإصدار")}>
  <header><h2>{t(l,"Issuance drafts","발행 초안","مسودات الإصدار")}</h2><p>{t(l,"Retained drafts and their latest review. This is not a complete decision history. Draft approval does not issue or deliver a document.","보관된 초안과 마지막 검토를 표시합니다. 전체 결정 이력은 아닙니다. 초안 승인은 문서 발행이나 전달을 수행하지 않습니다.","المسودات المحفوظة وآخر مراجعة لها. هذا ليس سجلًا كاملًا للقرارات. اعتماد المسودة لا يُصدر المستند ولا يسلّمه.")}</p><p>{t(l,"These are synthetic records. Historical approval does not establish current issuance eligibility.","합성 기록입니다. 과거 승인은 현재 발행 요건 충족을 보장하지 않습니다.","هذه سجلات اصطناعية. الاعتماد السابق لا يثبت أهلية الإصدار الحالية.")}</p><button onClick={refresh} disabled={!store||state.list.loading}>{t(l,"Refresh","새로 조회","تحديث")}</button></header>
  <DraftOutcomes locale={l} binding={binding} onInspect={inspect}/>
  <DraftFeedback locale={l} error={binding.error??state.list.error} loading={state.list.loading}/>
  {page&&<p role="status">{t(l,"Retained draft count","보관된 초안 수","عدد المسودات المحفوظة")}: {mono(page.count)}</p>}
  <div className={"b09-draft-workspace"+(!page?" b09-draft-single":"")}>
   <section aria-label={t(l,"Draft list","초안 목록","قائمة المسودات")}>
    {page?.count===0&&<p>{t(l,"No drafts are retained in this workspace.","이 업무 공간에 보관된 초안이 없습니다.","لا توجد مسودات محفوظة في مساحة العمل هذه.")}</p>}
    <ul className="b09-draft-list">{page?.rows.map(row=><li key={row.draft}><button aria-pressed={state.selected===row.draft} onClick={()=>inspect(row.draft)}><strong>{row.customer_name??t(l,"Customer ID","고객 ID","معرّف العميل")}{row.customer_name?null:<>: {mono(row.customer_id)}</>}</strong><span>{mono(row.period)}</span><span>{mono(`${row.currency} ${fixedMoney(row.total,row.scale)}`)}</span><span>{draftStateLabel(l,row.submission_state)} · {draftStateLabel(l,row.review_state)}</span><span>{t(l,"Draft ID","초안 ID","معرّف المسودة")}: {mono(row.draft)}</span></button></li>)}</ul>
    {page&&<nav className="b09-draft-pages" aria-label={t(l,"Draft pages","초안 페이지","صفحات المسودات")}><span>{t(l,"Page","페이지","الصفحة")} {mono(pages.length)}</span><button disabled={pages.length<2} onClick={()=>move(pages[pages.length-2],true)}>{t(l,"Previous","이전","السابق")}</button><button disabled={!page.next} onClick={()=>move(page.next)}>{t(l,"Next","다음","التالي")}</button></nav>}
   </section>
   <section className="b09-draft-detail" aria-label={t(l,"Draft details","초안 상세","تفاصيل المسودة")}>
    <DraftFeedback locale={l} loading={state.inspection.loading} error={state.inspection.error}/>
    {!state.selected&&<p>{t(l,"Select a draft to inspect its frozen evidence and latest review.","초안을 선택하여 고정된 근거와 마지막 검토를 확인하세요.","اختر مسودة للاطلاع على أدلتها المثبتة وآخر مراجعة لها.")}</p>}
    {draft&&<><h3 tabIndex={-1} ref={heading}>{t(l,"Draft details","초안 상세","تفاصيل المسودة")}</h3><dl>
     <dt>{t(l,"Draft ID","초안 ID","معرّف المسودة")}</dt><dd>{mono(draft.draft)}</dd>
     <dt>{t(l,"Customer ID","고객 ID","معرّف العميل")}</dt><dd>{mono(draft.customer_id)}</dd>
     <dt>{t(l,"Billing period","과금 기간","فترة الفوترة")}</dt><dd>{mono(draft.period)}</dd>
     <dt>{t(l,"Total","총액","الإجمالي")}</dt><dd>{mono(`${draft.currency} ${fixedMoney(draft.total,draft.scale)}`)}</dd>
     <dt>{t(l,"Submission","제출 상태","حالة الإرسال")}</dt><dd>{draftStateLabel(l,draft.submission_state)}</dd>
     <dt>{t(l,"Latest review","마지막 검토","آخر مراجعة")}</dt><dd>{draftStateLabel(l,draft.review?.state??"none")}</dd>
     <dt>{t(l,"Submitter ID","제출자 ID","معرّف المرسِل")}</dt><dd>{mono(draft.submitter)}</dd>
     {draft.review&&<><dt>{t(l,"Reviewer ID","검토자 ID","معرّف المراجع")}</dt><dd>{mono(draft.review.actor)}</dd></>}
    </dl>
    <h4>{t(l,"Draft lines","초안 항목","بنود المسودة")}</h4><ul className="b09-draft-line-cards">{draft.lines.map(line=><li key={line.index}><dl><dt>{t(l,"Description","항목 설명","الوصف")}</dt><dd>{line.text}</dd><dt>{t(l,"Amount","금액","المبلغ")}</dt><dd>{mono(`${draft.currency} ${fixedMoney(line.amount,draft.scale)}`)}</dd><dt>{t(l,"Tax line","세금 항목","بند ضريبي")}</dt><dd>{line.tax?t(l,"Yes","예","نعم"):t(l,"No","아니요","لا")}</dd></dl></li>)}</ul><div className="b09-draft-table" tabIndex={0} role="region" aria-label={t(l,"Draft lines","초안 항목","بنود المسودة")}><table><thead><tr><th>{t(l,"Description","항목 설명","الوصف")}</th><th>{t(l,"Amount","금액","المبلغ")}</th><th>{t(l,"Tax line","세금 항목","بند ضريبي")}</th></tr></thead><tbody>{draft.lines.map(line=><tr key={line.index}><td>{line.text}</td><td>{mono(`${draft.currency} ${fixedMoney(line.amount,draft.scale)}`)}</td><td>{line.tax?t(l,"Yes","예","نعم"):t(l,"No","아니요","لا")}</td></tr>)}</tbody></table></div>
    <nav aria-label={t(l,"Line pages","항목 페이지","صفحات البنود")}><button disabled={lines.length<2} onClick={()=>linePage(lines[lines.length-2],true)}>{t(l,"Previous","이전","السابق")}</button><span>{t(l,"Total lines","전체 항목","إجمالي البنود")}: {mono(draft.count)}</span><button disabled={!draft.next} onClick={()=>linePage(draft.next)}>{t(l,"Next","다음","التالي")}</button></nav>
    <DraftEvidence locale={l} draft={draft}/>
    <DraftActions key={draft.snapshot} locale={l} binding={binding} onDirty={setEditing}/>
    <DraftFeedback locale={l} loading={state.context.loading} error={state.context.error}/>
    </>}
   </section>
  </div>
 </section>;
}
