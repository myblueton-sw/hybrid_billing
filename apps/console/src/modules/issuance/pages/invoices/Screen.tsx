import { useState } from "react";
import { t, viewFields, type Props } from "../../../../shared/domain-screen-ui/finance/contracts";
import {
  Criteria,
  EmptyTable,
  Evidence,
  Intro,
  Notice,
  Panel,
  PaymentEvidence,
  RecognitionEvidence,
  scopeInputs,
  useScreenState,
  type InputDef,
} from "../../../../shared/domain-screen-ui/finance/ui";

const inputs: readonly InputDef[] = [
  ...scopeInputs,
  {
    key: "kind",
    label: ["발행 책임 구분", "Issuance responsibility", "مسؤولية الإصدار"],
    required: true,
    options: [
      { value: "erp", label: ["ERP 발행", "ERP-issued", "إصدار ERP"] },
      { value: "self", label: ["자체 발행", "Self-issued", "إصدار ذاتي"] },
      {
        value: "internal",
        label: ["내부 정산", "Internal settlement", "تسوية داخلية"],
      },
    ],
  },
  {
    key: "currency",
    label: [
      "통화 코드 (ISO 4217)",
      "Currency code (ISO 4217)",
      "رمز العملة (ISO 4217)",
    ],
    required: true,
  },
  {
    key: "document",
    label: [
      "문서 참조 (선택)",
      "Document reference (optional)",
      "مرجع المستند (اختياري)",
    ],
    idOnly: true,
  },
];
export default function Invoices(props: Props) {
  const { locale: l, view, onDirty } = props;
  const [expanded, setExpanded] = useState(false);
  const [criteria, onReview] = useScreenState(props, "I01", { expanded });
  return (
    <div
      className="fin-screen fin-invoices"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="I01"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "발행 문서와 승인 근거를 확인하고, 원문을 유지한 정정과 전달 이력을 추적합니다.",
          "Inspect issuance responsibility and approval evidence, then trace corrections and delivery without replacing the original.",
          "راجع مسؤولية الإصدار وأدلة الموافقة وتتبع التصحيحات والتسليم مع الحفاظ على الأصل.",
        ]}
      />
      <Criteria
        l={l}
        screenId="I01"
        inputs={inputs}
        onDirty={onDirty}
        onReview={onReview}
      />
      {view === "payment-terms" ? (
        <PaymentEvidence l={l} id="I01" />
      ) : view === "recognition" ? (
        <RecognitionEvidence l={l} id="I01" />
      ) : view === "documents" ? (
        <div className="fin-split">
          <Panel
            title={t(l, ["발행 문서", "Issued documents", "المستندات الصادرة"])}
          >
            <EmptyTable
              l={l}
              screenId="I01"
              columns={viewFields.I01.documents}
              caption={["문서 목록", "Document list", "قائمة المستندات"]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "문서 책임",
              "Document responsibility",
              "مسؤولية المستند",
            ])}
          >
            <p>
              {t(l, [
                "조회 조건의 책임 구분은 검색 기준입니다. 실제 발행 법인·승인·번호가 확인되어야 문서를 발행할 수 있습니다.",
                "The responsibility criterion is a search choice. Issuance requires verified entity, approval and numbering evidence.",
                "تحديد المسؤولية معيار بحث. يتطلب الإصدار أدلة متحققة للكيان والموافقة والترقيم.",
              ])}
            </p>
            <Evidence
              l={l}
              screenId="I01"
              keys={["entity", "issued", "delivery", "approval"]}
            />
          </Panel>
        </div>
      ) : view === "body" ? (
        <div className="fin-split">
          <Panel
            className="fin-document-sheet"
            title={t(l, ["문서 본문", "Document body", "محتوى المستند"])}
          >
            <div className="fin-sheet-heading">
              <span>
                {t(l, ["문서 번호", "Document number", "رقم المستند"])}
              </span>
              <strong>
                {t(l, [
                  "발행 근거 미연결",
                  "Issuance evidence not connected",
                  "أدلة الإصدار غير متصلة",
                ])}
              </strong>
            </div>
            <Evidence
              l={l}
              screenId="I01"
              keys={[
                "document",
                "kind",
                "entity",
                "recipient",
                "period",
                "template",
              ]}
            />
            <div className="fin-sheet-amounts">
              <Evidence l={l} screenId="I01" keys={["net", "tax", "total"]} />
            </div>
            <Notice
              l={l}
              text={[
                "이는 발행된 문서나 원본 미리보기가 아닙니다. ERP가 결정할 세금·최종액도 아직 확인되지 않았습니다.",
                "This is not an issued document or original preview. ERP-determined tax and final amounts are not yet verified.",
                "هذا ليس مستنداً صادراً أو معاينة للأصل. الضرائب والمبالغ النهائية التي يحددها ERP غير متحققة.",
              ]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "발행·지급정보 확인",
              "Issuance and payment-information checks",
              "التحقق من الإصدار وبيانات الدفع",
            ])}
          >
            <Evidence
              l={l}
              screenId="I01"
              keys={["hash", "approval", "bank", "issued", "delivery"]}
            />
            <p className="fin-meta">
              {t(l, [
                "지급계좌·수신자·세금 식별·템플릿 변경은 기존 채널 확인과 별도 승인이 필요합니다.",
                "Payment-account, recipient, tax-identity and template changes require established-channel verification and separate approval.",
                "تغيير حساب الدفع أو المستلم أو الهوية الضريبية أو القالب يتطلب تحققاً عبر القناة المسجلة وموافقة منفصلة.",
              ])}
            </p>
          </Panel>
        </div>
      ) : view === "corrections" ? (
        <Panel
          title={t(l, [
            "원문을 유지한 정정",
            "Corrections preserving the original",
            "تصحيحات تحافظ على الأصل",
          ])}
        >
          <div className="fin-two">
            <section>
              <h3>
                {t(l, ["확정 원문", "Finalized original", "الأصل النهائي"])}
              </h3>
              <Evidence l={l} screenId="I01" keys={["original", "hash"]} />
            </section>
            <section>
              <h3>
                {t(l, ["정정 연결", "Correction linkage", "ربط التصحيح"])}
              </h3>
              <Evidence
                l={l}
                screenId="I01"
                keys={["correction", "approval"]}
              />
            </section>
          </div>
          <Notice
            l={l}
            text={[
              "정정 초안 생성은 원문·잔여 정정 범위·현재 권한을 확인한 연결 서비스가 필요합니다. 원문은 삭제하거나 덮어쓰지 않습니다.",
              "Creating a correction requires a connected service verifying the original, remaining scope and current authority. The original is not deleted or overwritten.",
              "إنشاء تصحيح يتطلب خدمة متصلة تتحقق من الأصل والنطاق المتبقي والصلاحيات الحالية. لا يُحذف الأصل ولا يُستبدل.",
            ]}
          />
        </Panel>
      ) : (
        <Panel
          title={t(l, [
            "승인과 전달의 별도 이력",
            "Separate approval and delivery history",
            "سجلان منفصلان للموافقة والتسليم",
          ])}
        >
          <Evidence l={l} screenId="I01" keys={viewFields.I01.history} />
          <Notice
            l={l}
            text={[
              "발행 실패와 전달 실패를 구분합니다. 전달 재시도가 문서를 다시 발행하지 않습니다.",
              "Issuance failure and delivery failure are distinct. A delivery retry must not reissue a document.",
              "فشل الإصدار يختلف عن فشل التسليم. إعادة محاولة التسليم لا تعيد إصدار المستند.",
            ]}
          />
        </Panel>
      )}
      <details
        className="fin-requirements"
        open={expanded}
        onToggle={(e) => setExpanded(e.currentTarget.open)}
      >
        <summary>
          {t(l, [
            "발행·다운로드·정정 연결 요건",
            "Issuance, download and correction requirements",
            "متطلبات ربط الإصدار والتنزيل والتصحيح",
          ])}
        </summary>
        <p>
          {t(l, [
            "발행 요청: 승인 해시·청구 적용 범위·번호·법정 조합 검증. 확정본 다운로드: 현재 반출 권한·동일 저장 바이트. 정정: 확정 원문과 잔여 범위. 지급 기한 연장: 사유·별도 승인·현재 개정. 연결 서비스가 없어 현재 요청하지 않습니다.",
            "Issuance: approved hash, billing application coverage, numbering and legal combination. Final download: current export authority and identical stored bytes. Correction: final original and remaining scope. Due-date extension: reason, separate approval and current revision. These commands are not submitted without a connected service.",
            "الإصدار: البصمة المعتمدة ونطاق تطبيق الفوترة والترقيم والتوافق القانوني. التنزيل النهائي: صلاحية التصدير والبايتات المخزنة نفسها. التصحيح: الأصل النهائي والنطاق المتبقي. تمديد الاستحقاق: السبب والموافقة المنفصلة والمراجعة الحالية. لا تُرسل هذه الأوامر دون خدمة متصلة.",
          ])}
        </p>
      </details>
      <p className="fin-meta">
        {criteria.reviewed
          ? t(l, [
              "위에서 입력 조건 검토를 확인할 수 있습니다.",
              "Criteria review is available above.",
              "مراجعة المعايير متاحة أعلاه.",
            ])
          : t(l, [
              "문서 선택·금액·발행 상태는 아직 확인되지 않았습니다.",
              "Document selection, amounts and issuance state are not verified.",
              "لم يتم التحقق من اختيار المستند والمبالغ وحالة الإصدار.",
            ])}
      </p>
    </div>
  );
}
