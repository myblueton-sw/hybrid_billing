import { useState } from "react";
import { text, type Props, type Words } from "./contracts";
import {
  Boundary,
  Facts,
  Label,
  Panel,
  ReviewForm,
  refInput,
  useReview,
} from "./ui";
type DraftField = {
  id: string;
  label: Words;
  kind?: "date" | "days" | "currency" | "text";
  choices?: readonly Words[];
  required?: boolean;
};
const contractDrafts: Record<
  string,
  { title: Words; help: Words; fields: DraftField[] }
> = {
  parties: {
    title: [
      "당사자와 거래 책임",
      "Parties and transaction responsibility",
      "الأطراف ومسؤولية المعاملة",
    ],
    help: [
      "직접 계약의 CSP 사용료와 서비스료 적격 범위를 구분합니다. 구매·지급·발행 주체는 서로 다른 참조입니다.",
      "Separate direct-contract CSP charges from eligible service fees. Buyer, payer and issuer are distinct references.",
      "افصل رسوم CSP للعقد المباشر عن رسوم الخدمة المؤهلة. المشتري والدافع والمصدر مراجع مستقلة.",
    ],
    fields: [
      {
        id: "transaction_path",
        label: ["거래 경로", "Transaction path", "مسار المعاملة"],
        choices: [
          ["직접 계약", "Direct", "مباشر"],
          ["재판매", "Resale", "إعادة بيع"],
        ],
        required: true,
      },
      {
        id: "buyer",
        label: ["구매 주체 참조", "Buyer reference", "مرجع المشتري"],
        required: true,
      },
      {
        id: "payer",
        label: ["지급 주체 참조", "Payer reference", "مرجع الدافع"],
        required: true,
      },
      {
        id: "issuer",
        label: ["발행 주체 참조", "Issuer reference", "مرجع المصدر"],
        required: true,
      },
      {
        id: "legal_entity",
        label: ["법인 참조", "Legal entity reference", "مرجع الكيان القانوني"],
        required: true,
      },
      {
        id: "currency",
        label: ["계약 통화", "Contract currency", "عملة العقد"],
        kind: "currency",
        required: true,
      },
      {
        id: "timezone",
        label: [
          "계약 시간대 참조",
          "Contract timezone reference",
          "مرجع المنطقة الزمنية للعقد",
        ],
        required: true,
      },
    ],
  },
  schedule: {
    title: [
      "사용기간과 청구 일정",
      "Service interval and billing schedule",
      "فترة الخدمة وجدول الفوترة",
    ],
    help: [
      "기간 종료일은 제외합니다. 일할·선후청구·FX 정책을 확인하기 전에는 청구액을 계산하지 않습니다.",
      "Interval end is exclusive. No invoice amount is calculated before proration, billing timing and FX policies are verified.",
      "نهاية الفترة غير مشمولة. لا يتم حساب مبلغ الفاتورة قبل التحقق من سياسات التقسيط وتوقيت الفوترة والصرف.",
    ],
    fields: [
      {
        id: "service_start",
        label: ["사용 시작일", "Service start", "بداية الخدمة"],
        kind: "date",
        required: true,
      },
      {
        id: "service_end",
        label: [
          "사용 종료일 · 제외",
          "Service end · exclusive",
          "نهاية الخدمة · غير مشمولة",
        ],
        kind: "date",
        required: true,
      },
      {
        id: "billing_start",
        label: ["청구 시작일", "Billing start", "بداية الفوترة"],
        kind: "date",
        required: true,
      },
      {
        id: "billing_end",
        label: [
          "청구 종료일 · 제외",
          "Billing end · exclusive",
          "نهاية الفوترة · غير مشمولة",
        ],
        kind: "date",
        required: true,
      },
      {
        id: "billing_cycle",
        label: [
          "청구 주기 정책 참조",
          "Billing cycle policy reference",
          "مرجع سياسة دورة الفوترة",
        ],
        required: true,
      },
      {
        id: "proration_policy",
        label: [
          "일할 정책 참조",
          "Proration policy reference",
          "مرجع سياسة التقسيط",
        ],
      },
      {
        id: "fx_policy",
        label: ["FX 정책 참조", "FX policy reference", "مرجع سياسة الصرف"],
      },
    ],
  },
  products: {
    title: [
      "상품 구성과 포함량",
      "Components and allowances",
      "المكونات والكميات المشمولة",
    ],
    help: [
      "정액·실사용·SLA 크레딧을 분리합니다. 정액에 포함된 전력이나 서비스는 추가 요금으로 중복 청구하지 않습니다.",
      "Keep fixed fees, consumption and SLA credits separate. Included power or services must not be billed twice.",
      "افصل الرسوم الثابتة والاستهلاك وائتمانات SLA. لا تفوتر الطاقة أو الخدمات المشمولة مرتين.",
    ],
    fields: [
      {
        id: "component",
        label: [
          "서비스 구성요소 참조",
          "Service component reference",
          "مرجع مكون الخدمة",
        ],
        required: true,
      },
      {
        id: "rate_revision",
        label: [
          "요율 개정 참조",
          "Rate revision reference",
          "مرجع مراجعة السعر",
        ],
        required: true,
      },
      {
        id: "allowance",
        label: [
          "포함량 정책 참조",
          "Allowance policy reference",
          "مرجع سياسة الكمية المشمولة",
        ],
      },
      {
        id: "service_fee_eligibility",
        label: [
          "판매 적격 범위 참조",
          "Sales eligibility scope reference",
          "مرجع نطاق أهلية البيع",
        ],
        required: true,
      },
    ],
  },
  documents: {
    title: [
      "문서와 지급 보호 참조",
      "Documents and protected payment references",
      "المستندات ومراجع الدفع المحمية",
    ],
    help: [
      "지급 정보·수신자·템플릿 변경은 기존 등록 채널의 확인과 별도 승인·효력 검증이 필요합니다. 계좌나 비밀 원문을 입력하지 마세요.",
      "Payment, recipient and template changes require verification through an existing registered channel and separate approval and validity checks. Do not enter bank details or secrets.",
      "تغييرات الدفع والمستلم والقالب تتطلب التحقق عبر قناة مسجلة وموافقة وفحص صلاحية منفصلين. لا تدخل تفاصيل مصرفية أو أسرارا.",
    ],
    fields: [
      {
        id: "erp_order",
        label: ["ERP 주문 참조", "ERP order reference", "مرجع طلب ERP"],
      },
      {
        id: "prepayment_reference",
        label: [
          "기존 선불 청구 참조",
          "Existing prepayment reference",
          "مرجع الفوترة المسبقة الحالية",
        ],
      },
    ],
  },
  termination: {
    title: [
      "종료와 최종 정산",
      "Termination and final settlement",
      "الإنهاء والتسوية النهائية",
    ],
    help: [
      "계약 종료와 최종 정산 완료는 별도 상태입니다. 자료 확보를 위해 긴급 보안 회수를 지연하지 않습니다. 자원·구독은 자동 취소되지 않습니다.",
      "Contract termination and final settlement are separate states. Evidence collection must not delay urgent access revocation. Resources and subscriptions are not automatically cancelled.",
      "إنهاء العقد والتسوية النهائية حالتان مستقلتان. جمع الأدلة لا يؤخر سحب الوصول العاجل. لا يتم إلغاء الموارد والاشتراكات تلقائيا.",
    ],
    fields: [
      {
        id: "termination_at",
        label: ["종료일 초안", "Termination date draft", "مسودة تاريخ الإنهاء"],
        kind: "date",
        required: true,
      },
      {
        id: "final_evidence",
        label: [
          "최종 증빙 확보 경로 참조",
          "Final evidence acquisition reference",
          "مرجع مسار الحصول على الأدلة النهائية",
        ],
        required: true,
      },
    ],
  },
  "payment-terms": {
    title: [
      "지급 조건 개정 초안",
      "Payment terms revision draft",
      "مسودة مراجعة شروط الدفع",
    ],
    help: [
      "발행일을 기준으로 한 일수와 달력·시간대를 별도 검토합니다. 기한을 계산하거나 기존 발행 문서의 snapshot을 변경하지 않습니다.",
      "Review days from issuance, calendar and timezone separately. No due date is calculated and existing issued snapshots are unchanged.",
      "راجع الأيام من الإصدار والتقويم والمنطقة الزمنية بشكل منفصل. لا يتم حساب الاستحقاق أو تعديل لقطات المستندات الصادرة.",
    ],
    fields: [
      {
        id: "term_revision",
        label: [
          "지급 조건 개정 참조",
          "Payment terms revision reference",
          "مرجع مراجعة شروط الدفع",
        ],
        required: true,
      },
      {
        id: "days_from_issue",
        label: [
          "발행일 기준 일수 · 0 이상 정수",
          "Days from issue · nonnegative integer",
          "الأيام من الإصدار · عدد صحيح غير سالب",
        ],
        kind: "days",
        required: true,
      },
      {
        id: "calendar_policy",
        label: [
          "달력 정책 참조",
          "Calendar policy reference",
          "مرجع سياسة التقويم",
        ],
        required: true,
      },
      {
        id: "timezone",
        label: [
          "기한 시간대 참조",
          "Due-date timezone reference",
          "مرجع المنطقة الزمنية للاستحقاق",
        ],
        required: true,
      },
      {
        id: "effective_start",
        label: ["효력 시작일", "Effective start", "بداية السريان"],
        kind: "date",
        required: true,
      },
      {
        id: "effective_end",
        label: [
          "효력 종료일 · 제외",
          "Effective end · exclusive",
          "نهاية السريان · غير مشمولة",
        ],
        kind: "date",
        required: true,
      },
    ],
  },
  recognition: {
    title: [
      "청구 주기와 월별 비용 인식",
      "Billing cadence and monthly expense recognition",
      "دورة الفوترة والاعتراف الشهري بالمصروفات",
    ],
    help: [
      "고객 청구와 내부 ERP 인식은 별개입니다. 미수신 월은 0이나 완료가 아닙니다. 납부기한은 월별 인식에서 시작하지 않습니다.",
      "Customer billing and internal ERP recognition are separate. Missing months are neither zero nor complete. Monthly recognition does not start a payment deadline.",
      "فوترة العميل والاعتراف الداخلي في ERP منفصلان. الأشهر المفقودة ليست صفرا أو مكتملة. الاعتراف الشهري لا يبدأ مهلة الدفع.",
    ],
    fields: [
      {
        id: "billing_cadence",
        label: [
          "고객 청구 주기 정책 참조",
          "Customer billing cadence policy",
          "سياسة دورة فوترة العميل",
        ],
        required: true,
      },
      {
        id: "recognition_cadence",
        label: [
          "ERP 인식 주기 정책 참조",
          "ERP recognition cadence policy",
          "سياسة دورة الاعتراف في ERP",
        ],
        required: true,
      },
      {
        id: "legal_entity",
        label: [
          "인식 법인 참조",
          "Recognition entity reference",
          "مرجع كيان الاعتراف",
        ],
        required: true,
      },
      {
        id: "amount_basis",
        label: [
          "인식 금액 기준 참조",
          "Recognition amount basis reference",
          "مرجع أساس مبلغ الاعتراف",
        ],
        required: true,
      },
    ],
  },
};
export default function Contracts(p: Props) {
  const l = p.locale,
    cfg = contractDrafts[p.view] ?? contractDrafts.parties;
  const [values, setValues] = useState<Record<string, string>>({});
  useReview(p, Object.values(values).some(Boolean));
  return (
    <>
      <Boundary l={l} />
      <div className="work-columns contract-columns">
        <Panel title={text(l, cfg.title)} id={`O03:data:${p.view}`}>
          <p>{text(l, cfg.help)}</p>
          <ReviewForm
            l={l}
            id={`O03:action:review-${p.view}`}
            onClear={() => setValues({})}
            validate={(form) => {
              for (const prefix of ["service", "billing", "effective"]) {
                const start = form.elements.namedItem(
                  prefix + "_start",
                ) as HTMLInputElement | null;
                const end = form.elements.namedItem(
                  prefix + "_end",
                ) as HTMLInputElement | null;
                if (start && end)
                  end.setCustomValidity(
                    start.value && end.value && start.value >= end.value
                      ? text(l, [
                          "종료일은 시작일 이후여야 합니다.",
                          "End must be after start.",
                          "يجب أن تكون النهاية بعد البداية.",
                        ])
                      : "",
                  );
              }
            }}
          >
            <div className="work-form-grid">
              {cfg.fields.map((f) => (
                <Label key={f.id} label={text(l, f.label)}>
                  {f.choices ? (
                    <select
                      name={f.id}
                      required={f.required}
                      value={values[f.id] ?? ""}
                      onChange={(e) =>
                        setValues({ ...values, [f.id]: e.target.value })
                      }
                    >
                      <option value="">
                        {text(l, ["선택하세요", "Choose", "اختر"])}
                      </option>
                      {f.choices.map((c, i) => (
                        <option key={c[1]} value={String(i)}>
                          {text(l, c)}
                        </option>
                      ))}
                    </select>
                  ) : (
                    <input
                      {...(f.kind === "date" ? {} : refInput)}
                      name={f.id}
                      required={f.required}
                      type={f.kind === "date" ? "date" : "text"}
                      pattern={
                        f.kind === "days"
                          ? "[0-9]+"
                          : f.kind === "currency"
                            ? "[A-Z]{3}"
                            : f.kind === "date"
                              ? undefined
                              : refInput.pattern
                      }
                      maxLength={
                        f.kind === "currency" ? 3 : f.kind === "days" ? 9 : 128
                      }
                      inputMode={f.kind === "days" ? "numeric" : undefined}
                      value={values[f.id] ?? ""}
                      onChange={(e) =>
                        setValues({ ...values, [f.id]: e.target.value })
                      }
                    />
                  )}
                </Label>
              ))}
            </div>
          </ReviewForm>
        </Panel>
        <Panel
          title={text(l, [
            "현재 계약 근거",
            "Current contract evidence",
            "أدلة العقد الحالية",
          ])}
        >
          <Facts
            l={l}
            labels={[
              [
                "서버 계약 개정",
                "Server contract revision",
                "مراجعة العقد على الخادم",
              ],
              ["계약 상태", "Contract status", "حالة العقد"],
              [
                "최종 정산 상태",
                "Final settlement status",
                "حالة التسوية النهائية",
              ],
              ...(p.view === "documents"
                ? ([
                    [
                      "보호된 지급 참조",
                      "Protected payment reference",
                      "مرجع الدفع المحمي",
                    ],
                    [
                      "수신자·템플릿 개정",
                      "Recipient and template revision",
                      "مراجعة المستلم والقالب",
                    ],
                  ] as Words[])
                : p.view === "recognition"
                  ? ([
                      [
                        "월별 실제 전기",
                        "Monthly actual posting",
                        "الترحيل الشهري الفعلي",
                      ],
                      [
                        "분기 청구 대사",
                        "Quarterly billing reconciliation",
                        "مطابقة الفوترة الفصلية",
                      ],
                    ] as Words[])
                  : []),
            ]}
          />
          <p className="work-meta">
            {text(l, [
              "초안 입력은 현재 계약이나 승인된 개정을 바꾸지 않습니다.",
              "Draft inputs do not change the current contract or an approved revision.",
              "مدخلات المسودة لا تغير العقد الحالي أو المراجعة المعتمدة.",
            ])}
          </p>
        </Panel>
      </div>
    </>
  );
}
