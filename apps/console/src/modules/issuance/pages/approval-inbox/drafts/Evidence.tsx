import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
import type { Inspection } from "./inspection";
const mono=(v:string|number)=><bdi dir="ltr">{v}</bdi>;
export function DraftEvidence({locale:l,draft}:{locale:Locale;draft:Inspection}) {
 const c=draft.coverage;
 if(!c)return null;
 const labels:Record<string,string>={payment_account:t(l,"Payment account reference","결제 계좌 참조","مرجع حساب الدفع"),document_recipient:t(l,"Document recipient reference","문서 수신자 참조","مرجع مستلم المستند"),document_template:t(l,"Document template reference","문서 서식 참조","مرجع قالب المستند")};
 return <section><h4>{t(l,"Saved review evidence","저장된 검토 근거","أدلة المراجعة المحفوظة")}</h4><p>{t(l,"These references and dates were frozen when the review was saved. Saving this review does not issue a document.","검토 저장 시 고정된 참조와 날짜입니다. 검토 저장은 문서를 발행하지 않습니다.","ثُبّتت هذه المراجع والتواريخ عند حفظ المراجعة. حفظ المراجعة لا يُصدر مستندًا.")}</p><dl>
  <dt>{t(l,"Reviewed issue-date basis","검토 시 발행 기준일","تاريخ الإصدار المرجعي عند المراجعة")}</dt><dd>{mono(c.issue_date)}</dd>
  <dt>{t(l,"Due date","납부 기한","تاريخ الاستحقاق")}</dt><dd>{mono(c.due_date)}</dd>
  <dt>{t(l,"Payment term","납부 조건","شرط الدفع")}</dt><dd>{mono(c.net_days)} {t(l,"calendar days","달력일","يوم تقويمي")} · {mono(c.zone)}</dd>
  <dt>{t(l,"Contract ID","계약 ID","معرّف العقد")}</dt><dd>{mono(c.contract)}</dd>
  <dt>{t(l,"Term ID / revision","조건 ID / 리비전","معرّف الشرط / المراجعة")}</dt><dd>{mono(c.term)} / {mono(c.revision)}</dd>
  {draft.references.map(r=><div key={r.field}><dt>{labels[r.field]}</dt><dd>{mono(r.ref_id)}<details><summary>{t(l,"Reference evidence","참조 근거","أدلة المرجع")}</summary><p>{t(l,"Change ID","변경 ID","معرّف التغيير")}: {mono(r.change)}</p><p>{t(l,"Subject ID","대상 ID","معرّف الموضوع")}: {mono(r.subject)}</p><p>{t(l,"Fingerprint","지문","البصمة")}: {mono(r.fingerprint)}</p><p>{t(l,"Effective at","효력 발생 시각","وقت بدء السريان")}: {mono(r.effective)}</p></details></dd></div>)}
 </dl></section>;
}
