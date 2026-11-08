import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import type { Stage, Status } from "./model";
export const organizationName = (l: Locale, id: string) =>
  `${text(l, "Example organization", "예제 조직", "مؤسسة تجريبية")} ${id}`;
export const stageName = (l: Locale, stage: Stage) => {
  const labels: Record<Stage, [string, string, string]> = {
    collection: ["Collection", "수집", "الجمع"],
    reconciliation: ["Reconciliation", "대사", "المطابقة"],
    allocation: ["Allocation", "배분", "التخصيص"],
    calculation: ["Calculation", "계산", "الحساب"],
    approval: ["Approval", "승인", "الموافقة"],
    issuance: ["Issuance", "발행", "الإصدار"],
    accounting: ["Accounting", "회계", "المحاسبة"],
    delivery: ["Delivery", "전달", "التسليم"],
    receipt: ["Receipt", "수납", "التحصيل"],
  };
  return text(l, ...labels[stage]);
};
export const statusName = (l: Locale, status: Status) => {
  const labels: Record<Status, [string, string, string]> = {
    complete: ["Complete", "완료", "مكتمل"],
    pending: ["Pending", "대기", "قيد الانتظار"],
    blocked: ["Blocked", "차단", "محظور"],
    unknown: ["Unconfirmed", "미확인", "غير مؤكد"],
    na: ["Not applicable", "해당 없음", "غير منطبق"],
  };
  return text(l, ...labels[status]);
};
