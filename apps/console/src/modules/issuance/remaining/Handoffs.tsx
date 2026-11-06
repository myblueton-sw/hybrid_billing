import { useState } from "react";
import { t, viewFields, type Props, type Text } from "./contracts";
import {
  Criteria,
  EmptyTable,
  Evidence,
  Intro,
  Notice,
  Panel,
  RecognitionEvidence,
  scopeInputs,
  useScreenState,
  type InputDef,
} from "./ui";
const inputs: readonly InputDef[] = [
  ...scopeInputs,
  {
    key: "submission",
    label: ["제출 참조", "Submission reference", "مرجع الإرسال"],
    idOnly: true,
    required: true,
  },
  {
    key: "template",
    label: [
      "템플릿 개정 참조",
      "Template revision reference",
      "مرجع مراجعة القالب",
    ],
    idOnly: true,
  },
];
const sheets: readonly Text[] = [
  ["가이드", "Guide", "الدليل"],
  ["헤더", "Header", "الرأس"],
  ["상세", "Detail", "التفاصيل"],
  ["세금·조정", "Tax and adjustment", "الضريبة والتعديل"],
  ["매핑", "Mapping", "الربط"],
  ["대사", "Reconciliation", "المطابقة"],
  ["반영 결과", "Application result", "نتيجة المعالجة"],
];
export default function Handoffs(props: Props) {
  const { locale: l, view, onDirty } = props;
  const [purpose, setPurpose] = useState("review");
  const [, onReview] = useScreenState(props, "I03", { purpose });
  return (
    <div
      className="fin-screen fin-handoffs"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="I03"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "승인 명세의 고정 파일과 외부 반영 근거를 같은 제출 ID로 추적합니다. 파일 다운로드와 외부 반영을 구분합니다.",
          "Trace immutable files and external application evidence under the same submission ID. A download is distinct from external application.",
          "تتبع الملفات الثابتة وأدلة المعالجة الخارجية بمعرف إرسال مشترك. التنزيل يختلف عن المعالجة الخارجية.",
        ]}
      />
      <Criteria
        l={l}
        screenId="I03"
        inputs={inputs}
        onDirty={onDirty}
        onReview={onReview}
      />
      {view === "recognition" ? (
        <RecognitionEvidence l={l} id="I03" />
      ) : view === "files" ? (
        <div className="fin-split">
          <Panel
            title={t(l, [
              "파일 목적과 구성",
              "File purpose and composition",
              "غرض الملف ومكوناته",
            ])}
          >
            <label className="fin-purpose">
              {t(l, [
                "검토할 구성",
                "Composition to inspect",
                "المكونات المراد فحصها",
              ])}
              <select
                value={purpose}
                onChange={(e) => setPurpose(e.target.value)}
                data-item="I03:input:file-purpose"
              >
                <option value="review">
                  {t(l, [
                    "내부 정산 검토용",
                    "Internal settlement review",
                    "مراجعة التسوية الداخلية",
                  ])}
                </option>
                <option value="upload">
                  {t(l, [
                    "ERP 재입력용",
                    "ERP re-entry upload",
                    "تحميل لإعادة الإدخال إلى ERP",
                  ])}
                </option>
                <option value="monthly">
                  {t(l, [
                    "월별 비용 인식용",
                    "Monthly expense recognition",
                    "الاعتراف الشهري بالمصروفات",
                  ])}
                </option>
              </select>
            </label>
            <Notice
              l={l}
              text={
                purpose === "review"
                  ? [
                      "내부 표준 파일의 예정 구성입니다. 실제 파일·행 수·금액은 아직 생성되지 않았습니다. 고객용은 별도의 새 공개 파일로 생성해야 합니다.",
                      "Planned composition of the internal standard workbook. No file, row count or amount has been generated. Customer workbooks must be newly generated public copies.",
                      "هذه المكونات المخططة للملف الداخلي القياسي. لم يُنشأ ملف أو عدد صفوف أو مبالغ. ملفات العملاء نسخ إفصاح جديدة منفصلة.",
                    ]
                  : purpose === "upload"
                    ? [
                        "ERP 업로드는 제품·버전별 검증 양식과 미반영 범위의 고정 승인 명세가 필요합니다. 현재 지원 조합은 미확인입니다.",
                        "ERP upload requires a product/version-verified template and fixed approval for unapplied scope. The supported combination is currently unverified.",
                        "تحميل ERP يتطلب قالباً متحققاً للمنتج والإصدار وموافقة ثابتة للنطاق غير المعالج. التوافق المدعوم غير متحقق حالياً.",
                      ]
                    : [
                        "월별 비용 인식 파일은 목적·귀속월·승인 배치를 고정합니다. 인보이스 발행이나 실제 전기 완료를 의미하지 않습니다.",
                        "Monthly recognition files bind purpose, recognition month and approved batch. They do not imply invoice issuance or actual posting.",
                        "ملفات الاعتراف الشهري تثبت الغرض والشهر والدفعة المعتمدة. لا تعني إصدار فاتورة أو اكتمال الترحيل الفعلي.",
                      ]
              }
            />
            {purpose === "review" ? (
              <details className="fin-manifest" open>
                <summary>
                  {t(l, [
                    "표준 정산 파일의 시트 구조",
                    "Standard settlement workbook structure",
                    "بنية أوراق ملف التسوية القياسي",
                  ])}
                </summary>
                <ul>
                  {sheets.map((sheet, i) => (
                    <li key={i}>
                      <span>{t(l, sheet)}</span>
                      <small>
                        {t(l, [
                          "생성·행 수 미확인",
                          "Generation and row count unknown",
                          "الإنشاء وعدد الصفوف غير معروفين",
                        ])}
                      </small>
                    </li>
                  ))}
                </ul>
              </details>
            ) : (
              <Evidence
                l={l}
                screenId="I03"
                keys={
                  purpose === "upload"
                    ? ["template", "generator", "hash", "rowIds"]
                    : ["purpose", "month", "batch", "manifest"]
                }
              />
            )}
          </Panel>
          <Panel
            title={t(l, [
              "파일 원본과 검증",
              "Artifact identity and validation",
              "هوية الملف والتحقق منه",
            ])}
          >
            <Evidence
              l={l}
              screenId="I03"
              keys={[
                "filePurpose",
                "template",
                "generator",
                "hash",
                "rowIds",
                "fileHash",
                "manifest",
                "total",
              ]}
            />
            <p>
              {t(l, [
                "매크로·외부 연결을 포함하지 않는 새 파일이어야 합니다. 긴 ID와 decimal 원값은 텍스트로 보존하고, 안전 열람본을 원본으로 표시하지 않습니다.",
                "New files must exclude macros and external connections. Preserve long IDs and raw decimals as text, and distinguish safe viewing copies from originals.",
                "يجب أن تخلو الملفات الجديدة من وحدات الماكرو والاتصالات الخارجية. تُحفظ المعرفات الطويلة والأرقام العشرية كنص وتُميز نسخ العرض الآمنة عن الأصول.",
              ])}
            </p>
          </Panel>
        </div>
      ) : view === "outcomes" ? (
        <div className="fin-two">
          <Panel
            title={t(l, [
              "외부 반영 결과",
              "External application result",
              "نتيجة المعالجة الخارجية",
            ])}
          >
            <Evidence l={l} screenId="I03" keys={viewFields.I03.outcomes} />
          </Panel>
          <Panel
            title={t(l, [
              "인계와 반영의 경계",
              "Handoff versus application",
              "الفصل بين التسليم والمعالجة",
            ])}
          >
            <ol className="fin-workflow">
              <li>
                <strong>
                  {t(l, [
                    "파일 생성·다운로드",
                    "File generation and download",
                    "إنشاء الملف وتنزيله",
                  ])}
                </strong>
                <p>
                  {t(l, [
                    "고정 파일의 생성·반출 근거",
                    "Evidence of immutable file generation and export",
                    "دليل إنشاء الملف الثابت وتصديره",
                  ])}
                </p>
              </li>
              <li>
                <strong>
                  {t(l, [
                    "외부 결과 등록",
                    "External-result registration",
                    "تسجيل النتيجة الخارجية",
                  ])}
                </strong>
                <p>
                  {t(l, [
                    "같은 제출·행 ID와 외부 참조 필요",
                    "Requires the same submission/row IDs and external reference",
                    "يتطلب معرفات الإرسال والصفوف نفسها ومرجعاً خارجياً",
                  ])}
                </p>
              </li>
              <li>
                <strong>
                  {t(l, [
                    "행별 대사",
                    "Per-row reconciliation",
                    "المطابقة لكل صف",
                  ])}
                </strong>
                <p>
                  {t(l, [
                    "부분 반영·불명·실패를 구분",
                    "Keep partial, unknown and failed application distinct",
                    "تمييز المعالجة الجزئية وغير المعروفة والفاشلة",
                  ])}
                </p>
              </li>
            </ol>
            <Notice
              l={l}
              text={[
                "현재 단계의 처리 여부는 모두 미확인입니다. 이 안내 순서는 실행 이력이 아닙니다.",
                "All processing outcomes are unknown. This sequence is guidance, not an execution history.",
                "جميع نتائج المعالجة غير معروفة. هذا التسلسل إرشادي وليس سجلاً للتنفيذ.",
              ]}
            />
          </Panel>
        </div>
      ) : view === "reconciliation" ? (
        <Panel
          title={t(l, [
            "동일 제출의 행별 대사",
            "Per-row reconciliation for the same submission",
            "مطابقة الصفوف للإرسال نفسه",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="I03"
            columns={viewFields.I03.reconciliation}
            caption={[
              "제출 행과 외부 반영",
              "Submission rows and external application",
              "صفوف الإرسال والمعالجة الخارجية",
            ]}
          />
          <Notice
            l={l}
            text={[
              "반영 결과는 금액을 덮어쓰지 않습니다. 불명 범위를 실패로 간주해 재인계하거나 이미 반영된 행을 다시 제출하지 않습니다.",
              "Application results do not overwrite amounts. Unknown scope cannot be treated as failed for redelivery, and applied rows cannot be submitted again.",
              "نتائج المعالجة لا تستبدل المبالغ. لا يُعامل النطاق غير المعروف كفاشل لإعادة التسليم ولا تُعاد الصفوف المعالجة.",
            ]}
          />
        </Panel>
      ) : (
        <Panel
          title={t(l, [
            "파일 인계 원장",
            "File handoff ledger",
            "سجل تسليم الملفات",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="I03"
            columns={viewFields.I03.handoffs}
            caption={[
              "파일 인계 목록",
              "File handoff list",
              "قائمة تسليم الملفات",
            ]}
          />
        </Panel>
      )}
      <details className="fin-requirements">
        <summary>
          {t(l, [
            "파일 생성·결과 등록·재인계 요건",
            "File generation, result registration and redelivery requirements",
            "متطلبات إنشاء الملف وتسجيل النتيجة وإعادة التسليم",
          ])}
        </summary>
        <p>
          {t(l, [
            "표준 XLSX·고객 파일과 검증된 ERP 업로드 파일은 목적을 구분합니다. 생성·결과 등록·실패 범위 재인계에는 현재 권한·승인 개정·고정 제출/행 ID·외부 근거가 필요합니다. 생성·반출·등록 서비스가 없어 실제 작업은 실행하지 않습니다.",
            "Standard XLSX, customer files and verified ERP upload files have separate purposes. Generation, result registration and redelivery require current authority, approval revision, fixed submission/row IDs and external evidence. No actual work executes without connected generation, export and registration services.",
            "للملفات القياسية وملفات العملاء وملفات تحميل ERP المتحققة أغراض منفصلة. الإنشاء وتسجيل النتائج وإعادة التسليم تتطلب صلاحيات حديثة ومراجعة الموافقة ومعرفات ثابتة وأدلة خارجية. لا يُنفذ عمل فعلي دون خدمات متصلة.",
          ])}
        </p>
      </details>
    </div>
  );
}
