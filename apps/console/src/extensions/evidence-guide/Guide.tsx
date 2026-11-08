import type { Locale } from "../../shared/types";
import { text as t } from "../../shared/copy";
export default function Guide({ locale: l }: { locale: Locale }) {
  return (
    <aside className="panel extension" data-extension="evidence-guide">
      <h2>
        {t(l, "Evidence checklist", "원본 확인 항목", "قائمة التحقق من الأدلة")}
      </h2>
      <p>
        {t(
          l,
          "Check collection period, account mapping and missing source references. This optional panel does not approve evidence.",
          "수집 기간, 계정 연결, 누락된 원본 참조를 확인하세요. 이 추가 패널은 원본을 승인하지 않습니다.",
          "تحقق من فترة الجمع وربط الحساب والمراجع الناقصة. هذه اللوحة الإضافية لا تعتمد الأدلة.",
        )}
      </p>
    </aside>
  );
}
