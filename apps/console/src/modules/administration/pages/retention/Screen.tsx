import { Draft, Evidence, Ledger, Panel, Steps } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Field, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const reason: Words = [
  "보관 정책과 내구 결정 기록이 연결되지 않았습니다. 삭제 대상·잔존 자료·백업 만료를 확인할 수 없습니다.",
  "Retention policies and durable decision records are not connected. Deletion targets, remaining data and backup expiry are unknown.",
  "سياسات الاحتفاظ وسجلات القرارات الدائمة غير متصلة. أهداف الحذف والبيانات المتبقية وانتهاء النسخ الاحتياطية غير معروفة.",
];
const fields: Record<string, Field[]> = {
  policies: [
    {
      id: "scope",
      label: [
        "정책 범위·관할 참조",
        "Policy scope and jurisdiction reference",
        "مرجع نطاق السياسة والاختصاص",
      ],
      required: true,
    },
    {
      id: "class",
      label: ["자료 등급", "Data class", "فئة البيانات"],
      required: true,
    },
    {
      id: "origin",
      label: [
        "기산 사건 참조",
        "Retention origin event reference",
        "مرجع حدث بدء الاحتفاظ",
      ],
      required: true,
    },
    {
      id: "rule",
      label: [
        "기간·아카이브·삭제 기준",
        "Duration, archive and deletion criteria",
        "معايير المدة والأرشفة والحذف",
      ],
      kind: "textarea",
      required: true,
    },
  ],
  expiry: [
    {
      id: "class",
      label: ["자료 등급 필터", "Data class filter", "مرشح فئة البيانات"],
    },
    {
      id: "date",
      label: ["검토 기준일", "Review date", "تاريخ المراجعة"],
      kind: "date",
      required: true,
    },
  ],
  holds: [
    {
      id: "scope",
      label: ["보류 범위 참조", "Hold scope reference", "مرجع نطاق التعليق"],
      required: true,
    },
    {
      id: "owner",
      label: ["책임자 참조", "Accountable owner reference", "مرجع المسؤول"],
      required: true,
    },
    {
      id: "reason",
      label: [
        "보류·해제 검토 사유",
        "Hold or release review reason",
        "سبب مراجعة التعليق أو رفعه",
      ],
      kind: "textarea",
      required: true,
    },
  ],
  deletion: [
    {
      id: "manifest",
      label: [
        "고정 삭제 후보 manifest 참조",
        "Pinned deletion candidate manifest reference",
        "مرجع بيان مرشحي الحذف المثبّت",
      ],
      required: true,
    },
    {
      id: "revision",
      label: [
        "정책 revision 참조",
        "Policy revision reference",
        "مرجع مراجعة السياسة",
      ],
      required: true,
    },
  ],
  restore: [
    {
      id: "artifact",
      label: [
        "아카이브 원본 참조",
        "Archive original reference",
        "مرجع الأصل المؤرشف",
      ],
      required: true,
    },
    {
      id: "hash",
      label: ["원본 hash 참조", "Original hash reference", "مرجع بصمة الأصل"],
      required: true,
    },
    {
      id: "expiry",
      label: [
        "요청 임시 보관 만료",
        "Requested temporary storage expiry",
        "انتهاء التخزين المؤقت المطلوب",
      ],
      kind: "datetime-local",
      required: true,
    },
  ],
  decisions: [
    {
      id: "epoch",
      label: [
        "결정 epoch 참조",
        "Decision epoch reference",
        "مرجع حقبة القرار",
      ],
      required: true,
    },
    {
      id: "watermark",
      label: [
        "최신 watermark 참조",
        "Latest watermark reference",
        "مرجع آخر علامة مائية",
      ],
      required: true,
    },
  ],
};
export default function Retention(p: Props) {
  const l = p.locale;
  return (
    <>
      <div className="ops-grid">
        <div>
          <Ledger
            locale={l}
            title={
              p.view === "holds"
                ? [
                    "법적 보류·공유 참조",
                    "Legal holds and shared references",
                    "التعليق القانوني والمراجع المشتركة",
                  ]
                : p.view === "decisions"
                  ? [
                      "독립 결정·복원 최신성",
                      "Independent decisions and restoration freshness",
                      "القرارات المستقلة وحداثة الاستعادة",
                    ]
                  : [
                      "보관 정책·영향 범위",
                      "Retention policies and impact",
                      "سياسات الاحتفاظ والتأثير",
                    ]
            }
            columns={[
              [
                "자료·정책 참조",
                "Data / policy reference",
                "مرجع البيانات والسياسة",
              ],
              ["기준·revision", "Basis / revision", "الأساس والمراجعة"],
              ["보류·의존성", "Holds / dependencies", "التعليق والتبعيات"],
              ["잔존 상태", "Remaining state", "حالة البيانات المتبقية"],
            ]}
            reason={reason}
          />
          <Draft
            props={p}
            title={
              p.view === "policies"
                ? [
                    "보관 정책 초안",
                    "Retention policy draft",
                    "مسودة سياسة الاحتفاظ",
                  ]
                : [
                    "영향 검토 범위",
                    "Impact review scope",
                    "نطاق مراجعة التأثير",
                  ]
            }
            fields={fields[p.view] ?? fields.policies}
          />
        </div>
        <aside>
          <Evidence
            locale={l}
            title={[
              "자료 보호 근거",
              "Data protection evidence",
              "أدلة حماية البيانات",
            ]}
            fields={[
              [
                "법적 보류·유효 참조",
                "Legal holds and valid references",
                "التعليق القانوني والمراجع السارية",
              ],
              [
                "내구 결정 sequence·epoch",
                "Durable decision sequence / epoch",
                "تسلسل وحقبة القرار الدائم",
              ],
              [
                "저장소별 잔존 상태",
                "Remaining state per store",
                "الحالة المتبقية لكل مخزن",
              ],
              [
                "백업 만료·복원 창",
                "Backup expiry and restoration window",
                "انتهاء النسخ الاحتياطية ونافذة الاستعادة",
              ],
            ]}
            reason={reason}
          />
          <Panel
            title={t(l, [
              "삭제 상태의 구분",
              "Deletion state distinctions",
              "تمييز حالات الحذف",
            ])}
          >
            <Steps
              locale={l}
              steps={[
                ["접수", "Acceptance", "الاستلام"],
                [
                  "내구 결정·접근 차단",
                  "Durable decision and access block",
                  "قرار دائم ومنع الوصول",
                ],
                ["저장소 후속 정리", "Store cleanup", "تنظيف المخازن"],
                [
                  "백업 만료 확인",
                  "Backup expiry confirmation",
                  "تأكيد انتهاء النسخ الاحتياطية",
                ],
              ]}
            />
            <p>
              {t(l, [
                "접수만으로 삭제 완료를 표시하지 않습니다. 미확정·불명 결정과 남아 있는 백업을 별도로 확인합니다.",
                "Acceptance does not establish completed deletion. Unconfirmed decisions and remaining backups require separate evidence.",
                "الاستلام لا يثبت اكتمال الحذف. تتطلب القرارات غير المؤكدة والنسخ الاحتياطية المتبقية أدلة مستقلة.",
              ])}
            </p>
          </Panel>
        </aside>
      </div>
    </>
  );
}
