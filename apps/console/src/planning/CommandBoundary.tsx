import type { Locale } from "../shared/types";
import { local, type Tr } from "./copy";
const labels: Record<string, Tr> = {
  validate: ["검증 재실행", "Recheck validation", "إعادة التحقق"],
  assign: ["담당 지정", "Assign owner", "تعيين مسؤول"],
  approve: ["승인", "Approve", "موافقة"],
  reject: ["반려", "Reject", "رفض"],
  handoff: ["인계", "Hand off", "تسليم"],
  change: ["변경 요청", "Request change", "طلب تغيير"],
  diagnose: ["진단 실행", "Run diagnostics", "تشغيل التشخيص"],
  activate: [
    "개통 검토 요청",
    "Request activation review",
    "طلب مراجعة التفعيل",
  ],
  publish: [
    "입력 확정·공개",
    "Confirm and publish input",
    "تأكيد المدخل ونشره",
  ],
  download: ["원본 다운로드", "Download original", "تنزيل الأصل"],
  simulate: ["영향 시뮬레이션", "Simulate impact", "محاكاة الأثر"],
  calculate: ["계산 요청", "Request calculation", "طلب الحساب"],
  adjust: ["조정 요청", "Request adjustment", "طلب تعديل"],
  issue: ["발행 요청", "Request issuance", "طلب الإصدار"],
  query: [
    "외부 결과 조회",
    "Query external outcome",
    "استعلام النتيجة الخارجية",
  ],
  submit: ["제출 요청", "Request submission", "طلب تقديم"],
  generate: ["패키지 생성", "Generate package", "إنشاء حزمة"],
  deliver: ["전달 요청", "Request delivery", "طلب تسليم"],
  retry: ["재시도 요청", "Request retry", "طلب إعادة المحاولة"],
  cancel: ["취소 요청", "Request cancellation", "طلب إلغاء"],
  ack: ["확인", "Acknowledge", "إقرار"],
  revoke: ["회수 요청", "Request revocation", "طلب إلغاء الوصول"],
  rotate: ["자격 교체", "Rotate credential", "تدوير الاعتماد"],
  test: ["시험 요청", "Request test", "طلب اختبار"],
  delete: ["삭제 계획 요청", "Request deletion plan", "طلب خطة حذف"],
  restore: ["복원 검토", "Review restore", "مراجعة الاسترداد"],
  run: ["시나리오 실행", "Run scenario", "تشغيل السيناريو"],
  reply: ["문의 제출", "Submit inquiry", "تقديم استفسار"],
  link: ["로그인 연결", "Link sign-in method", "ربط وسيلة الدخول"],
};
const actions: Record<string, string[]> = {
  U03: ["ack", "retry", "cancel"],
  U04: ["validate", "adjust", "assign"],
  O01: ["change"],
  O02: ["simulate", "approve"],
  O03: ["change", "simulate"],
  S01: ["diagnose", "rotate", "revoke"],
  S02: ["diagnose", "activate"],
  S03: ["validate", "retry"],
  S04: ["download"],
  S05: ["download", "restore"],
  S06: ["validate", "publish", "reject"],
  B01: ["validate", "approve"],
  B02: ["diagnose"],
  B03: ["simulate", "approve"],
  B04: ["simulate", "adjust", "approve"],
  B05: ["validate", "approve"],
  B06: ["validate"],
  B07: ["change", "validate"],
  B08: ["calculate", "retry", "approve"],
  B09: ["approve", "reject", "handoff"],
  B10: ["adjust"],
  B11: ["adjust", "deliver"],
  I01: ["issue", "download", "adjust"],
  I02: ["submit", "query"],
  I03: ["generate", "query"],
  I04: ["validate", "generate", "deliver"],
  R01: ["generate"],
  R02: ["download"],
  M01: ["change", "revoke"],
  M02: ["test", "approve", "activate"],
  M03: ["change", "revoke"],
  M04: ["rotate", "revoke"],
  M05: ["link", "revoke"],
  P01: ["diagnose", "generate"],
  P02: ["test", "retry"],
  P03: ["restore", "query"],
  P04: ["delete", "restore"],
  P05: ["test"],
  P06: ["run"],
  P07: ["validate"],
  C02: ["download"],
  C03: ["reply"],
};
export default function CommandBoundary({
  screen,
  locale: l,
  customer,
}: {
  screen: string;
  locale: Locale;
  customer: boolean;
}) {
  const keys = customer && screen === "U03" ? ["ack"] : (actions[screen] ?? []);
  if (!keys.length) return null;
  return (
    <section className="planning-command-boundary">
      <div>
        {keys.map((k) => (
          <button key={k} disabled aria-describedby="planning-command-reason">
            {local(l, labels[k])}
          </button>
        ))}
      </div>
      <p id="planning-command-reason">
        {local(l, [
          "정책·권한·서비스 미연결로 실행할 수 없습니다. 초안 작성과 조회만 가능한 시안입니다.",
          "Execution unavailable: policies, permissions and services are not connected. This candidate supports draft review and inspection only.",
          "التنفيذ غير متاح: السياسات والصلاحيات والخدمات غير متصلة. تدعم المعاينة مراجعة المسودات والفحص فقط.",
        ])}
      </p>
    </section>
  );
}
