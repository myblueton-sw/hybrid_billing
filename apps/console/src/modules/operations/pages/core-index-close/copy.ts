import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { CloseState, Stage } from "./model";
export function stateLabel(l: Locale, state: CloseState) {
  return t(
    l,
    ...(
      {
        incomplete: ["Incomplete", "미완료", "غير مكتمل"],
        blocked: ["Blocked", "차단", "محظور"],
        held: ["Held", "보류", "معلق"],
        exempt: ["Exempt", "면제", "معفى"],
        complete: ["Complete", "완료", "مكتمل"],
      } as Record<CloseState, [string, string, string]>
    )[state],
  );
}
export function stageLabel(l: Locale, stage: Stage) {
  return t(
    l,
    ...(
      {
        allocation: ["Allocation", "배분", "التخصيص"],
        collection: ["Collection", "수집", "الجمع"],
        calculation: ["Calculation", "계산", "الحساب"],
        approval: ["Approval", "승인", "الموافقة"],
        issuance: ["Issuance", "발행", "الإصدار"],
        delivery: ["Delivery", "전달", "التسليم"],
        accounting: ["Accounting", "회계", "المحاسبة"],
      } as Record<Stage, [string, string, string]>
    )[stage],
  );
}
