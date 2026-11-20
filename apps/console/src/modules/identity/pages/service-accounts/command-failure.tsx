import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { ErrorDiagnostics } from "../../../../service/CommandDiagnostics";
import { FailureBox } from "../users/Live";
import { qcopy as t } from "./query-copy";

const terms = {
  unknown: [
    "The original request may already have completed. Keep its details and review the current target details. A failed retry does not prove the original request failed.",
    "원래 요청이 이미 완료되었을 수 있습니다. 요청 정보를 보존하고 현재 대상의 상세 정보를 검토하세요. 재시도 실패만으로 원래 요청의 실패를 확인할 수는 없습니다.",
    "ربما اكتمل الطلب الأصلي بالفعل. احتفظ بتفاصيله وراجع تفاصيل الهدف الحالية. فشل إعادة المحاولة لا يثبت فشل الطلب الأصلي.",
  ],
  revision: [
    "The request was refused because its expected revision no longer matches. Reload the current details and review the values before preparing a new request.",
    "예상 리비전이 현재 리비전과 일치하지 않아 요청이 거절되었습니다. 현재 상세 정보를 다시 조회하고 값을 검토한 후 새 요청을 준비하세요.",
    "رُفض الطلب لأن المراجعة المتوقعة لم تعد متطابقة. أعد تحميل التفاصيل الحالية وراجع القيم قبل إعداد طلب جديد.",
  ],
  denied: [
    "This request could not be authorized for this target in the current context. Review the current access and authentication requirements.",
    "현재 상황에서는 이 대상에 대한 요청을 허용할 수 없습니다. 현재 접근 권한과 인증 요건을 검토하세요.",
    "تعذّر التصريح بهذا الطلب لهذا الهدف في السياق الحالي. راجع متطلبات الوصول والمصادقة الحالية.",
  ],
  refused: [
    "The request was refused. Review its values, current prerequisites and effective limits before preparing a new request.",
    "요청이 거절되었습니다. 요청 값, 현재 선행 요건과 적용 한도를 검토한 후 새 요청을 준비하세요.",
    "رُفض الطلب. راجع قيمه والمتطلبات المسبقة الحالية والحدود السارية قبل إعداد طلب جديد.",
  ],
  authentication: [
    "Review the current authentication requirements before submitting again.",
    "현재 인증 요건을 검토한 후 다시 제출하세요.",
    "راجع متطلبات المصادقة الحالية قبل الإرسال مجدداً.",
  ],
  cancelled: [
    "Password confirmation was cancelled. Review the current authentication requirements before submitting again.",
    "비밀번호 재확인을 취소했습니다. 현재 인증 요건을 검토한 후 다시 제출하세요.",
    "أُلغي تأكيد كلمة المرور. راجع متطلبات المصادقة الحالية قبل الإرسال مجدداً.",
  ],
  confirmationTitle: [
    "Password confirmation could not be completed.",
    "비밀번호 재확인을 완료할 수 없습니다.",
    "تعذّر إكمال تأكيد كلمة المرور.",
  ],
  confirmation: [
    "Current password confirmation could not be verified. Review the authentication requirements and try confirmation again when available.",
    "현재 비밀번호 재확인 결과를 확인할 수 없습니다. 인증 요건을 검토하고 재확인이 가능해지면 다시 진행하세요.",
    "تعذّر التحقق من تأكيد كلمة المرور الحالي. راجع متطلبات المصادقة وحاول التأكيد مجدداً عندما يكون متاحاً.",
  ],
};
export type FailureOutcome = "unknown" | "refused" | "confirmation";
const text = (l: Locale, key: keyof typeof terms) =>
  terms[key][l === "en" ? 0 : l === "ko" ? 1 : 2];
function describe(l: Locale, error: unknown, outcome: FailureOutcome) {
  if (outcome === "unknown")
    return { title: t(l, "unknown"), body: text(l, "unknown") };
  const admin = error instanceof AdminError ? error : undefined;
  if (outcome === "confirmation")
    return {
      title: text(l, "confirmationTitle"),
      body: text(
        l,
        admin?.failure === "reauth_cancelled" ? "cancelled" : "confirmation",
      ),
    };
  return {
    title: t(l, "refused"),
    body: text(
      l,
      admin?.failure === "denied"
        ? "denied"
        : admin?.failure === "conflict" && admin.code === "revision_conflict"
          ? "revision"
          : admin?.failure === "reauth_cancelled"
            ? "cancelled"
            : admin?.failure === "reauth"
              ? "authentication"
              : "refused",
    ),
  };
}
export function CommandFailure({
  l,
  error,
  outcome,
  variant = "standalone",
}: {
  l: Locale;
  error: unknown;
  outcome: FailureOutcome;
  variant?: "standalone" | "inline";
}) {
  const shown = describe(l, error, outcome);
  return (
    <>
      {variant === "inline" ? (
        <div className="service-summary" role="alert">
          <p>{shown.body}</p>
        </div>
      ) : (
        <FailureBox l={l} error={error} focusKey={0} describe={() => shown} />
      )}
      <ErrorDiagnostics l={l} error={error} />
    </>
  );
}
