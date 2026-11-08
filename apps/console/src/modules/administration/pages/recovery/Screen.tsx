import { Draft, Evidence, Ledger, Panel, Steps } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Field, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const reason: Words = [
  "복원 세트·독립 결정 기록·외부 결과가 연결되지 않았습니다. 공개와 외부 쓰기의 재개는 판정되지 않았습니다.",
  "Restore sets, independent decision records and external outcomes are not connected. Publication and external writes have no resumption decision.",
  "مجموعات الاستعادة وسجلات القرارات المستقلة والنتائج الخارجية غير متصلة. لم يُتخذ قرار باستئناف النشر أو الكتابة الخارجية.",
];
const fields: Record<string, Field[]> = {
  sets: [
    {
      id: "set",
      label: [
        "복원 세트 참조",
        "Restore set reference",
        "مرجع مجموعة الاستعادة",
      ],
      required: true,
    },
    {
      id: "point",
      label: ["복원 기준 시각", "Recovery point", "نقطة الاستعادة"],
      kind: "datetime-local",
      required: true,
    },
    {
      id: "manifest",
      label: [
        "DB·객체·키·모듈 manifest 참조",
        "Database, object, key and module manifest reference",
        "مرجع بيان قاعدة البيانات والكائنات والمفاتيح والوحدات",
      ],
      required: true,
    },
  ],
  isolation: [
    {
      id: "generation",
      label: [
        "대상 설치 generation",
        "Target installation generation",
        "جيل التثبيت المستهدف",
      ],
      required: true,
    },
    {
      id: "fence",
      label: [
        "기존 worker·자격증명 격리 근거 참조",
        "Existing worker and credential isolation evidence reference",
        "مرجع دليل عزل العامل وبيانات الاعتماد الحالية",
      ],
      required: true,
    },
  ],
  reconciliation: [
    {
      id: "submission",
      label: [
        "원제출 참조",
        "Original submission reference",
        "مرجع الإرسال الأصلي",
      ],
      required: true,
    },
    {
      id: "external",
      label: [
        "ERP 결과 근거 참조",
        "ERP outcome evidence reference",
        "مرجع دليل نتيجة ERP",
      ],
      required: true,
    },
  ],
  migration: [
    {
      id: "customer",
      label: [
        "허용 고객 범위 참조",
        "Allowed customer scope reference",
        "مرجع نطاق العملاء المسموح",
      ],
      required: true,
    },
    {
      id: "manifest",
      label: [
        "고정 이관 manifest 참조",
        "Pinned migration manifest reference",
        "مرجع بيان الترحيل المثبّت",
      ],
      required: true,
    },
    {
      id: "cutoff",
      label: ["거래 기준일", "Transaction cutoff date", "تاريخ قطع المعاملات"],
      kind: "date",
      required: true,
    },
  ],
  resume: [
    {
      id: "watermark",
      label: [
        "최신 결정 watermark 참조",
        "Latest decision watermark reference",
        "مرجع آخر علامة مائية للقرار",
      ],
      required: true,
    },
    {
      id: "evidence",
      label: [
        "격리·대사·무결성 근거 참조",
        "Isolation, reconciliation and integrity evidence references",
        "مراجع أدلة العزل والمطابقة والسلامة",
      ],
      kind: "textarea",
      required: true,
    },
  ],
};
export default function Recovery(p: Props) {
  const l = p.locale;
  return (
    <>
      <Steps
        locale={l}
        steps={[
          ["복원 세트", "Restore set", "مجموعة الاستعادة"],
          [
            "격리·최신 결정",
            "Isolation and latest decisions",
            "العزل وأحدث القرارات",
          ],
          [
            "외부 결과 대사",
            "External reconciliation",
            "مطابقة النتائج الخارجية",
          ],
          [
            "별도 재개 판정",
            "Separate resumption decision",
            "قرار استئناف مستقل",
          ],
        ]}
      />
      <div className="ops-grid">
        <div>
          <Ledger
            locale={l}
            title={
              p.view === "migration"
                ? [
                    "고객 이관 범위",
                    "Customer migration scope",
                    "نطاق ترحيل العملاء",
                  ]
                : ["복구 근거", "Recovery evidence", "أدلة الاستعادة"]
            }
            columns={[
              ["참조", "Reference", "المرجع"],
              ["기준 시각", "As of", "الوقت المرجعي"],
              ["상태", "State", "الحالة"],
              ["차단 사유", "Blocking reason", "سبب المنع"],
            ]}
            reason={reason}
          />
          <Draft
            props={p}
            title={[
              "검토 대상 지정",
              "Define review scope",
              "تحديد نطاق المراجعة",
            ]}
            fields={fields[p.view] ?? fields.sets}
          />
        </div>
        <aside>
          <Evidence
            locale={l}
            title={[
              "재개 전 필수 근거",
              "Evidence before resumption",
              "الأدلة المطلوبة قبل الاستئناف",
            ]}
            fields={[
              [
                "설치 generation·격리",
                "Installation generation and isolation",
                "جيل التثبيت والعزل",
              ],
              [
                "삭제·회수 최신성",
                "Deletion and revocation freshness",
                "حداثة الحذف والإلغاء",
              ],
              [
                "문서·번호·outbox 대사",
                "Document, number and outbox reconciliation",
                "مطابقة المستندات والأرقام وصندوق الصادر",
              ],
              [
                "청구 적용 범위·기초잔액 보존",
                "Billing application coverage and opening balance preservation",
                "حفظ نطاق تطبيق الفوترة والأرصدة الافتتاحية",
              ],
            ]}
            reason={reason}
          />
          <Panel
            title={t(l, [
              "복원과 재개는 별도입니다",
              "Restoration and resumption are separate",
              "الاستعادة والاستئناف منفصلان",
            ])}
          >
            <p>
              {t(l, [
                "복원으로 문서를 재발행하거나 전송하지 않습니다. 외부 결과가 불명인 요청과 최신 내구 결정이 확인되기 전까지 재개를 보류합니다.",
                "Restoration does not reissue or resend documents. Resumption waits for unknown external outcomes and the latest durable decisions to be resolved.",
                "لا تعيد الاستعادة إصدار المستندات أو إرسالها. يُؤجّل الاستئناف حتى حسم النتائج الخارجية المجهولة والتحقق من أحدث القرارات الدائمة.",
              ])}
            </p>
          </Panel>
        </aside>
      </div>
    </>
  );
}
