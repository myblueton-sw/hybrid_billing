import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import { text } from "../../../../shared/copy";
import type { UploadIntent } from "./review";
export default function UploadReview({
  intent,
  locale,
  onCancel,
  onConfirm,
}: {
  intent: UploadIntent;
  locale: Locale;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const copy = (en: string, ko: string, ar: string) => text(locale, en, ko, ar);
  useEffect(() => {
    const d = dialog.current;
    d?.showModal();
    return () => d?.close();
  }, []);
  return (
    <dialog
      ref={dialog}
      className="live-upload-review"
      aria-labelledby="upload-review-title"
      onCancel={(e) => {
        e.preventDefault();
        dialog.current?.close();
        onCancel();
      }}
    >
      <h2 id="upload-review-title">
        {copy(
          "Review file reception",
          "파일 수집 내용 확인",
          "مراجعة استلام الملف",
        )}
      </h2>
      <p>
        {copy(
          "Confirm the file and destination before validation and storage. This is not billing confirmation or publication.",
          "검증·저장 전에 파일과 수집 대상을 확인하세요. 청구 확정·공개가 아닙니다.",
          "أكد الملف والوجهة قبل التحقق والحفظ. هذا ليس تأكيداً للفوترة أو نشراً.",
        )}
      </p>
      <dl>
        <dt>{copy("File / bytes", "파일 / 바이트", "الملف / البايتات")}</dt>
        <dd>
          <bdi>{intent.file.name}</bdi> · <bdi>{intent.file.size}</bdi>
        </dd>
        <dt>
          {copy("Version / revision", "버전 / 리비전", "الإصدار / المراجعة")}
        </dt>
        <dd>
          <bdi>FOCUS {intent.version}</bdi> / <bdi>{intent.revision}</bdi>
        </dd>
        <dt>{copy("Account", "계정", "الحساب")}</dt>
        <dd>
          <bdi>{intent.profile.account}</bdi>
        </dd>
        <dt>
          {copy(
            "Workspace / connection",
            "업무 공간 / 연결",
            "مساحة العمل / الاتصال",
          )}
        </dt>
        <dd>
          <bdi>{intent.profile.workspace}</bdi>
          <br />
          <bdi>{intent.profile.connection}</bdi>
        </dd>
        <dt>
          {copy(
            "Period (end exclusive)",
            "기간 (종료시각 미포함)",
            "الفترة (النهاية غير مشمولة)",
          )}
        </dt>
        <dd>
          <bdi>{intent.profile.period_start}</bdi>
          <br />
          <bdi>{intent.profile.period_end}</bdi>
        </dd>
      </dl>
      <p>
        {copy(
          "Same revision and bytes return the existing receipt. Changed bytes require a new revision. Invalid input may be quarantined.",
          "같은 리비전·파일은 기존 결과를 반환합니다. 내용이 바뀌면 새 리비전이 필요합니다. 유효하지 않은 입력은 격리될 수 있습니다.",
          "تعيد المراجعة والبيانات المتطابقتان الاستلام السابق. تتطلب البيانات المعدلة مراجعة جديدة. قد تُعزل المدخلات غير الصالحة.",
        )}
      </p>
      <p>
        {copy(
          "After sending, closing this page does not cancel server processing. If the outcome is unknown, inspect saved receipts before retrying.",
          "전송 후 화면을 닫아도 서버 처리는 취소되지 않습니다. 결과가 불분명하면 재시도 전에 저장된 결과를 확인하세요.",
          "بعد الإرسال، إغلاق الصفحة لا يلغي معالجة الخادم. إذا كانت النتيجة مجهولة، افحص الاستلامات المحفوظة قبل إعادة المحاولة.",
        )}
      </p>
      <div className="live-review-actions">
        <button
          autoFocus
          onClick={() => {
            dialog.current?.close();
            onCancel();
          }}
        >
          {copy("Cancel", "취소", "إلغاء")}
        </button>
        <button
          className="primary"
          onClick={() => {
            dialog.current?.close();
            onConfirm();
          }}
        >
          {copy("Confirm reception", "수집 실행", "تأكيد الاستلام")}
        </button>
      </div>
    </dialog>
  );
}
