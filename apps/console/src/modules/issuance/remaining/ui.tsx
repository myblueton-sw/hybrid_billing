import { useEffect, useRef, useState, type ReactNode } from "react";
import type { Locale } from "../../../shared/types";
import {
  fields,
  t,
  type FieldKey,
  type FinanceId,
  type Props,
  type Text,
  viewFields,
} from "./contracts";

export const words = {
  unknown: ["미확인", "Unknown", "غير معروف"],
  unavailable: [
    "조회 서비스 미연결",
    "Read service not connected",
    "خدمة القراءة غير متصلة",
  ],
  noCount: [
    "전체 건수 미확인",
    "Total count unknown",
    "العدد الإجمالي غير معروف",
  ],
  criteria: ["조회 조건", "Query criteria", "معايير الاستعلام"],
  review: ["조건 검토", "Review criteria", "مراجعة المعايير"],
  reset: ["입력 지우기", "Clear inputs", "مسح المدخلات"],
  local: [
    "현재 브라우저의 입력 검토입니다. 조회·저장·업무 실행은 요청하지 않습니다.",
    "Review inputs in this browser. No query, save or business command is submitted.",
    "مراجعة المدخلات في هذا المتصفح. لا يتم إرسال استعلام أو حفظ أو أمر عمل.",
  ],
  source: [
    "조회 연결 후 현재 권한과 같은 범위의 근거를 표시합니다. 미확인 값은 0이나 완료를 뜻하지 않습니다.",
    "Connected evidence must match current authority and scope. Unknown values do not mean zero or completed.",
    "يجب أن تطابق الأدلة المتصلة الصلاحيات والنطاق الحاليين. القيم غير المعروفة لا تعني صفراً أو اكتمالاً.",
  ],
  noRows: [
    "조회 서비스가 연결되지 않아 행을 표시할 수 없습니다.",
    "Rows are unavailable until the read service is connected.",
    "الصفوف غير متاحة حتى اتصال خدمة القراءة.",
  ],
  missing: [
    "필수 입력을 확인하세요.",
    "Check the required inputs.",
    "تحقق من المدخلات المطلوبة.",
  ],
  invalid: [
    "입력 형식이나 기간을 확인하세요.",
    "Check the input format or period.",
    "تحقق من صيغة الإدخال أو الفترة.",
  ],
} satisfies Record<string, Text>;

export function Panel({
  title,
  children,
  className = "",
  id,
}: {
  title: string;
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  return (
    <section className={`fin-panel ${className}`} data-item={id}>
      <h2>{title}</h2>
      {children}
    </section>
  );
}
export function Intro({ l, purpose }: { l: Locale; purpose: Text }) {
  return (
    <div className="fin-intro">
      <p>{t(l, purpose)}</p>
      <div className="fin-source">
        <strong>{t(l, words.unavailable)}</strong>
        <span>
          {t(l, [
            "기준시각·조회 범위: 미확인",
            "As-of and authorized scope: unknown",
            "الوقت المرجعي والنطاق المصرح: غير معروفين",
          ])}
        </span>
      </div>
      <p className="fin-meta">{t(l, words.source)}</p>
    </div>
  );
}
export function Evidence({
  l,
  keys,
  screenId,
}: {
  l: Locale;
  keys: readonly FieldKey[];
  screenId: string;
}) {
  return (
    <dl className="fin-evidence">
      {keys.map((key) => (
        <div key={key} data-item={`${screenId}:data:${fields[key].id}`}>
          <dt>{t(l, fields[key].label)}</dt>
          <dd>
            {t(l, words.unknown)}
            <small>{t(l, words.unavailable)}</small>
          </dd>
        </div>
      ))}
    </dl>
  );
}
export function EmptyTable({
  l,
  columns,
  screenId,
  caption,
}: {
  l: Locale;
  columns: readonly FieldKey[];
  screenId: string;
  caption: Text;
}) {
  return (
    <>
      <p className="fin-meta">
        {t(l, words.noCount)}.{" "}
        {t(l, [
          "연결 후 한 페이지 최대 25행",
          "Up to 25 rows per page after connection",
          "حتى 25 صفاً للصفحة بعد الاتصال",
        ])}
      </p>
      <div
        className="fin-table-wrap"
        tabIndex={0}
        role="region"
        aria-label={t(l, caption)}
      >
        <table>
          <caption>{t(l, caption)}</caption>
          <thead>
            <tr>
              {columns.map((k) => (
                <th
                  key={k}
                  scope="col"
                  data-item={`${screenId}:data:${fields[k].id}`}
                >
                  {t(l, fields[k].label)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td colSpan={columns.length}>{t(l, words.noRows)}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </>
  );
}
export function Notice({ l, text }: { l: Locale; text: Text }) {
  return <p className="fin-note">{t(l, text)}</p>;
}
export type InputDef = {
  key: string;
  label: Text;
  type?: "text" | "month" | "date";
  required?: boolean;
  options?: readonly { value: string; label: Text }[];
  idOnly?: boolean;
};
export type CriteriaState = {
  values: Record<string, string>;
  reviewed: boolean;
  revision: number;
};
export function Criteria({
  l,
  screenId,
  inputs,
  onDirty,
  onReview,
  title = words.criteria,
}: {
  l: Locale;
  screenId: string;
  inputs: readonly InputDef[];
  onDirty: Props["onDirty"];
  onReview: (s: CriteriaState) => void;
  title?: Text;
}) {
  const [values, setValues] = useState<Record<string, string>>({}),
    [reviewed, setReviewed] = useState(false),
    [errors, setErrors] = useState<string[]>([]),
    [revision, setRevision] = useState(0);
  const first = useRef<HTMLDivElement>(null);
  useEffect(() => {
    onDirty(Object.values(values).some(Boolean));
    onReview({ values, reviewed, revision });
  }, [values, reviewed, revision, onDirty, onReview]);
  useEffect(() => () => onDirty(false), [onDirty]);
  const validate = () => {
    const bad = inputs
      .filter(
        (input) =>
          (input.required && !values[input.key]?.trim()) ||
          (values[input.key] &&
            input.idOnly &&
            !/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(values[input.key])) ||
          (values[input.key] &&
            input.key === "currency" &&
            !/^[A-Z]{3}$/.test(values[input.key])),
      )
      .map((x) => x.key);
    const start = values.start,
      end = values.end;
    if (start && end && start >= end) bad.push("end");
    if (
      values.compareStart &&
      values.compareEnd &&
      values.compareStart >= values.compareEnd
    )
      bad.push("compareEnd");
    if (bad.length) {
      setErrors([...new Set(bad)]);
      setReviewed(false);
      requestAnimationFrame(() =>
        first.current
          ?.querySelector<HTMLElement>(`[name="${bad[0]}"]`)
          ?.focus(),
      );
    } else {
      setErrors([]);
      setReviewed(true);
    }
  };
  return (
    <section
      className="fin-criteria"
      data-item={`${screenId}:input:criteria`}
      ref={first}
    >
      <h2>{t(l, title)}</h2>
      <p className="fin-meta">{t(l, words.local)}</p>
      <form
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          validate();
        }}
      >
        <div className="fin-fields">
          {inputs.map((input) => (
            <label key={input.key} htmlFor={`${screenId}-${input.key}`}>
              <span>
                {t(l, input.label)}
                {input.required
                  ? t(l, [" (필수)", " (required)", " (مطلوب)"])
                  : ""}
              </span>
              {input.options ? (
                <select
                  id={`${screenId}-${input.key}`}
                  name={input.key}
                  value={values[input.key] ?? ""}
                  aria-invalid={errors.includes(input.key)}
                  aria-required={input.required}
                  onChange={(e) => {
                    setValues((v) => ({ ...v, [input.key]: e.target.value }));
                    setReviewed(false);
                    setErrors([]);
                    setRevision((n) => n + 1);
                  }}
                >
                  <option value="">
                    {t(l, ["선택하세요", "Choose an option", "اختر خياراً"])}
                  </option>
                  {input.options.map((option) => (
                    <option key={option.value} value={option.value}>
                      {t(l, option.label)}
                    </option>
                  ))}
                </select>
              ) : (
                <input
                  id={`${screenId}-${input.key}`}
                  name={input.key}
                  type={input.type ?? "text"}
                  dir="ltr"
                  autoComplete="off"
                  spellCheck={false}
                  maxLength={128}
                  value={values[input.key] ?? ""}
                  aria-invalid={errors.includes(input.key)}
                  aria-required={input.required}
                  onChange={(e) => {
                    setValues((v) => ({ ...v, [input.key]: e.target.value }));
                    setReviewed(false);
                    setErrors([]);
                    setRevision((n) => n + 1);
                  }}
                />
              )}{" "}
              {errors.includes(input.key) && (
                <small>
                  {t(l, !values[input.key] ? words.missing : words.invalid)}
                </small>
              )}
            </label>
          ))}
        </div>
        <div className="fin-actions">
          <button
            type="submit"
            data-item={`${screenId}:action:review-criteria`}
          >
            {t(l, words.review)}
          </button>
          <button
            type="button"
            onClick={() => {
              setValues({});
              setReviewed(false);
              setErrors([]);
              setRevision((n) => n + 1);
            }}
            data-item={`${screenId}:action:clear-inputs`}
          >
            {t(l, words.reset)}
          </button>
        </div>
        <p className="fin-feedback" role="status">
          {errors.length
            ? t(l, words.invalid)
            : reviewed
              ? t(l, [
                  "입력 형식을 검토했습니다. 권한·대상 존재·정책 적격성은 확인되지 않았으며 서버 조회도 실행되지 않았습니다.",
                  "Input format reviewed. Authority, target existence and policy eligibility remain unverified. No server query was executed.",
                  "تمت مراجعة صيغة الإدخال. الصلاحيات ووجود الأهداف والأهلية غير متحققة. لم يُنفذ استعلام للخادم.",
                ])
              : t(l, [
                  "조건을 입력하면 이 화면에서 검토할 수 있습니다.",
                  "Enter criteria to review them on this screen.",
                  "أدخل المعايير لمراجعتها في هذه الشاشة.",
                ])}
        </p>
      </form>
    </section>
  );
}
export const scopeInputs: readonly InputDef[] = [
  {
    key: "entity",
    label: ["법인 참조", "Legal entity reference", "مرجع الكيان القانوني"],
    required: true,
    idOnly: true,
  },
  {
    key: "start",
    label: ["시작일 (포함)", "Start date (inclusive)", "تاريخ البداية (مشمول)"],
    type: "date",
    required: true,
  },
  {
    key: "end",
    label: [
      "종료일 (미포함)",
      "End date (exclusive)",
      "تاريخ النهاية (غير مشمول)",
    ],
    type: "date",
    required: true,
  },
];
export function useScreenState(
  props: Props,
  id: FinanceId,
  viewState: unknown = null,
) {
  const [state, setState] = useState<CriteriaState>({
    values: {},
    reviewed: false,
    revision: 0,
  });
  const signature = JSON.stringify([
    id,
    props.view,
    state.reviewed,
    state.revision,
    viewState,
    Object.keys(state.values)
      .filter((k) => state.values[k])
      .sort(),
  ]);
  useEffect(() => {
    props.onState(signature, [
      ...new Set([
        `${id}:state:unconnected`,
        `${id}:input:criteria`,
        `${id}:action:review-criteria`,
        `${id}:action:clear-inputs`,
        ...Object.keys(viewFieldsFor(id, props.view)).map(
          (k) => `${id}:data:${k}`,
        ),
      ]),
    ]);
  }, [signature, id, props.view, props.onState]);
  return [state, setState] as const;
}
function viewFieldsFor(id: FinanceId, view: string) {
  return Object.fromEntries(
    (viewFields[id][view] ?? []).map((k) => [fields[k].id, null]),
  );
}
export function PaymentEvidence({ l, id }: { l: Locale; id: string }) {
  return (
    <Panel
      title={t(l, [
        "지급 기한·수납 근거",
        "Payment due dates and receipt evidence",
        "تواريخ استحقاق الدفع وأدلة التحصيل",
      ])}
    >
      <Notice
        l={l}
        text={[
          "수납 권위·배정·효력일·확인시각이 연결되지 않았습니다. 기한 내·당일·연체·완납·해당 없음을 판정하지 않습니다. 분쟁·독촉 보류는 기한과 잔액을 바꾸지 않습니다.",
          "Receipt authority, allocation, effective date and freshness are not connected. Within term, due today, overdue, paid and not applicable are not determined. Disputes and collections holds do not alter due dates or balances.",
          "مرجع التحصيل وتخصيصه وتاريخ سريانه وحداثته غير متصلة. لم تُحدد حالات الاستحقاق أو التأخر أو السداد أو عدم الانطباق. النزاعات وتعليق التحصيل لا يغيران المواعيد أو الأرصدة.",
        ]}
      />
      <Evidence l={l} screenId={id} keys={viewFields.I01["payment-terms"]} />
    </Panel>
  );
}
export function RecognitionEvidence({ l, id }: { l: Locale; id: string }) {
  return (
    <Panel
      title={t(l, [
        "월별 비용 인식과 분기 청구",
        "Monthly expense recognition and quarterly billing",
        "الاعتراف الشهري بالمصروفات والفوترة الربع سنوية",
      ])}
    >
      <Notice
        l={l}
        text={[
          "월별 인식 전송·실제 전기·분기 청구는 각각 별도 근거가 필요합니다. 미수신 월을 0으로 채우지 않으며 인식 전송에서 지급 기한을 시작하지 않습니다.",
          "Monthly recognition submission, actual posting and quarterly billing require separate evidence. Missing months are not filled with zero, and recognition submission does not start a payment term.",
          "إرسال الاعتراف الشهري والترحيل الفعلي والفوترة الربع سنوية تتطلب أدلة منفصلة. الأشهر المفقودة ليست صفراً وإرسال الاعتراف لا يبدأ مدة الدفع.",
        ]}
      />
      <Evidence
        l={l}
        screenId={id}
        keys={viewFields[id as FinanceId]?.recognition ?? []}
      />
    </Panel>
  );
}
