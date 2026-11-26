import type {Locale} from "../../../../../shared/types";
export const t=(l:Locale,en:string,ko:string,ar:string)=>l==="ko"?ko:l==="ar"?ar:en;
export function label(l:Locale,value:string){
 const labels:Record<string,readonly [string,string,string]>={
  propose:["Propose result evidence","결과 근거 제안","اقتراح أدلة النتيجة"],confirm:["Confirm result evidence","결과 근거 확인","تأكيد أدلة النتيجة"],revoke:["Revoke result evidence","결과 근거 철회","إلغاء أدلة النتيجة"],apply:["Apply ERP result evidence internally","ERP 결과 근거 내부 반영","تطبيق أدلة نتائج ERP داخلياً"],inspect:["Inspect result evidence","결과 근거 조회","عرض أدلة النتيجة"],
  console_ceiling:["Console permission ceiling","콘솔 권한 상한","سقف صلاحيات وحدة التحكم"],human_grant:["Person permission","사용자 권한","صلاحية شخص"],
  pending:["Awaiting decision","결정 대기","بانتظار القرار"],applying:["Recording decision","결정 기록 중","جارٍ تسجيل القرار"],decision_unknown:["Outcome unknown","처리 결과 미확인","نتيجة العملية غير معروفة"],applied:["Grant recorded","권한 부여 기록됨","منح الصلاحية مسجل"],rejected:["Rejected","반려됨","مرفوض"],withdrawn:["Withdrawn","철회됨","مسحوب"],expired:["Proposal expired","제안 만료","انتهت صلاحية المقترح"],superseded:["Proposal no longer current","제안 기준 변경됨","لم يعد المقترح ساريًا"],
  approve:["Approve proposal","제안 승인","اعتماد المقترح"],reject:["Reject proposal","제안 반려","رفض المقترح"],withdraw:["Withdraw proposal","제안 철회","سحب المقترح"],recover:["Resolve unknown decision","미확정 결정 복구","تسوية القرار غير المعروف"],
  unknown:["Outcome unknown","처리 결과 미확인","نتيجة العملية غير معروفة"],refused:["Submission refused","제출 거절됨","رُفض الإرسال"],recorded:["Receipt recorded","처리 기록 수신됨","تم استلام سجل العملية"],
 };
 const words=labels[value];return words?t(l,...words):value;
}
