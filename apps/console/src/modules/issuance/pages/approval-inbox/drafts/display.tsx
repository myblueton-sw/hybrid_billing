import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
const mono=(value:string|number)=><bdi dir="ltr">{value}</bdi>;
export function draftStateLabel(l:Locale,value:string) {
 const values:Record<string,[string,string,string]>={submitted:["Draft submitted","초안 제출됨","أُرسلت المسودة"],submission_unknown:["Submission result unconfirmed","제출 결과 미확인","نتيجة الإرسال غير مؤكدة"],submission_refused:["Submission refused","제출 거절","رُفض الإرسال"],none:["Review not saved","검토 미저장","لم تُحفظ المراجعة"],prepared:["Review saved","검토 저장됨","حُفظت المراجعة"],approval_unknown:["Approval result unconfirmed","승인 결과 미확인","نتيجة الاعتماد غير مؤكدة"],approved:["Draft approved","초안 승인됨","اعتُمدت المسودة"],refused:["Approval attempt refused","승인 시도 거절됨","رُفضت محاولة الاعتماد"]};
 const label=values[value];return label?t(l,...label):value;
}
export function DraftFeedback({locale:l,loading,error}:{locale:Locale;loading?:boolean;error?:string}) {
 if(loading)return <p role="status">{t(l,"Loading drafts…","초안을 불러오는 중입니다…","جارٍ تحميل المسودات…")}</p>;
 if(!error)return null;
 return <div role="alert"><p>{t(l,"The current data could not be verified. Check your session and permissions, then refresh.","현재 데이터를 확인하지 못했습니다. 세션과 권한을 확인한 뒤 새로 조회하세요.","تعذر التحقق من البيانات الحالية. تحقق من الجلسة والصلاحيات ثم أعد التحميل.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary>{mono(error)}</details></div>;
}
