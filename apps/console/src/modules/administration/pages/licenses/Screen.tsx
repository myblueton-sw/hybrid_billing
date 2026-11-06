import Notices from "./Notices";
import { Draft, Evidence, Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const reason: Words = [
  "서명된 설치 권리와 최종 지원 계약이 연결되지 않았습니다. 오픈소스 고지는 제품 사용 권리를 대신하지 않습니다.",
  "Signed installation entitlements and final support contracts are not connected. Open-source notices do not replace product-use rights.",
  "استحقاقات التثبيت الموقّعة وعقود الدعم النهائية غير متصلة. إشعارات المصادر المفتوحة لا تحل محل حقوق استخدام المنتج.",
];
export default function Licenses(p: Props) {
  const l = p.locale;
  return (
    <div className="ops-grid">
      <div>
        {p.view === "offline" ? (
          <Draft
            props={p}
            title={[
              "오프라인 권리 파일 범위 검토",
              "Review offline entitlement file scope",
              "مراجعة نطاق ملف الاستحقاق دون اتصال",
            ]}
            fields={[
              {
                id: "installation",
                label: [
                  "대상 설치 참조",
                  "Target installation reference",
                  "مرجع التثبيت المستهدف",
                ],
                required: true,
              },
              {
                id: "issuer",
                label: [
                  "발급 주체 참조",
                  "Issuer reference",
                  "مرجع جهة الإصدار",
                ],
                required: true,
              },
              {
                id: "reference",
                label: [
                  "파일 참조 · 원문 반입 없음",
                  "File reference · no content import",
                  "مرجع الملف · دون استيراد المحتوى",
                ],
                required: true,
              },
            ]}
            notice={[
              "파일 서명 검사와 적용은 미연결입니다. 파일을 업로드하거나 내용을 처리하지 않습니다.",
              "Signature verification and application are not connected. No file upload or content processing.",
              "التحقق من التوقيع والتطبيق غير متصلين. لا رفع ملفات أو معالجة محتوى.",
            ]}
          />
        ) : p.view === "notices" ? (
          <Notices locale={l} />
        ) : (
          <Evidence
            locale={l}
            title={
              p.view === "support"
                ? [
                    "계약·지원 범위",
                    "Contract and support scope",
                    "نطاق العقد والدعم",
                  ]
                : [
                    "설치 권리",
                    "Installation entitlements",
                    "استحقاقات التثبيت",
                  ]
            }
            fields={[
              [
                "설치·권리 참조",
                "Installation and entitlement reference",
                "مرجع التثبيت والاستحقاق",
              ],
              [
                "유효기간·상태",
                "Validity period and state",
                "فترة الصلاحية والحالة",
              ],
              [
                "운영 범위·계약 지표",
                "Operational scope and contract metrics",
                "نطاق التشغيل ومقاييس العقد",
              ],
              [
                "지원 주체·수준",
                "Support provider and level",
                "جهة الدعم ومستواه",
              ],
            ]}
            reason={reason}
          />
        )}
      </div>
      <aside>
        <Panel
          title={t(l, [
            "권리와 자료 접근",
            "Entitlements and evidence access",
            "الاستحقاقات والوصول إلى الأدلة",
          ])}
        >
          <p>
            {t(l, [
              "유효·예정·유예·만료·검증실패는 각각 다른 상태입니다. 권리 만료에 따른 신규 운영 제한과 보관 중인 문서·감사·반출 접근은 별도 계약으로 다룹니다.",
              "Valid, scheduled, grace, expired and verification failed are distinct states. New-operation restrictions and access to retained documents, audit and exports have separate contracts.",
              "الساري والمجدول وفترة السماح والمنتهي وفشل التحقق حالات منفصلة. تخضع قيود العمليات الجديدة والوصول إلى المستندات المحفوظة والتدقيق والتصدير لعقود مستقلة.",
            ])}
          </p>
        </Panel>
        <Panel
          title={t(l, [
            "Marketplace 연결",
            "Marketplace linkage",
            "الربط بالسوق",
          ])}
        >
          <p>
            {t(l, [
              "현재 미개통입니다. CSP별 권리·계약 정보의 후속 검토를 유지합니다.",
              "Not enabled. CSP-specific entitlement and contract review remains deferred.",
              "غير مفعّل. تظل مراجعة الاستحقاقات والعقود الخاصة بكل مزوّد مؤجلة.",
            ])}
          </p>
        </Panel>
      </aside>
    </div>
  );
}
