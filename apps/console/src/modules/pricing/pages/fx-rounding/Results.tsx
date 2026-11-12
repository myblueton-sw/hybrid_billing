import type { Locale } from "../../../../shared/types";
import {
  comparable,
  dateDays,
  draftKeys,
  profile,
  scale,
  type Draft,
  type Prepared,
  type Rational,
  type Result,
} from "./model";
import { t, label, modeLabel, missing, pending } from "./copy";
import { exact, money, subtract, asRational } from "./values";
export function ExactValue({
  locale: l,
  value,
  currency,
  signed = false,
}: {
  locale: Locale;
  value: Rational;
  currency: string;
  signed?: boolean;
}) {
  try {
    const v = exact(value, currency, signed);
    return (
      <>
        <bdi dir="ltr">{v.text}</bdi>
        {v.fraction && (
          <small>{t(l, "Exact fraction", "정확 분수", "كسر دقيق")}</small>
        )}
      </>
    );
  } catch {
    return <Unavailable locale={l} />;
  }
}
function Unavailable({ locale: l }: { locale: Locale }) {
  return (
    <span>
      {t(
        l,
        "Exact value unavailable — review display bounds.",
        "정확값 표시 불가 — 표시 범위를 확인하세요.",
        "القيمة الدقيقة غير متاحة — راجع حدود العرض.",
      )}
    </span>
  );
}
function Difference({
  locale: l,
  a,
  b,
  currency,
}: {
  locale: Locale;
  a: Rational;
  b: Rational;
  currency: string;
}) {
  try {
    return (
      <ExactValue
        locale={l}
        value={subtract(a, b)}
        currency={currency}
        signed
      />
    );
  } catch {
    return <Unavailable locale={l} />;
  }
}

export function Summary({
  locale: l,
  result,
}: {
  locale: Locale;
  result?: Result;
}) {
  return (
    <>
      <h2>
        {t(
          l,
          "Conversion review result",
          "환산 검토 결과",
          "نتيجة مراجعة التحويل",
        )}
      </h2>
      <p className="b05-meta" role="status">
        {result
          ? t(
              l,
              "Calculated for review only",
              "검토용 계산 완료",
              "حُسب للمراجعة فقط",
            )
          : pending(l)}
      </p>
      <dl
        className="b05-summary"
        data-fx-candidate={result ? "ready" : undefined}
      >
        <div>
          <dt>{t(l, "Rounded result", "처리 후 금액", "النتيجة المقربة")}</dt>
          <dd>
            {result ? (
              <bdi dir="ltr" data-fx-total="total">
                {money(result.total, result.currency, l)}
              </bdi>
            ) : (
              pending(l)
            )}
          </dd>
        </div>
        <div>
          <dt>
            {t(
              l,
              "Exact amount before rounding",
              "처리 전 정확값",
              "المبلغ الدقيق قبل التقريب",
            )}
          </dt>
          <dd>
            {result ? (
              <ExactValue
                locale={l}
                value={result.trace.withFee}
                currency={result.currency}
              />
            ) : (
              pending(l)
            )}
          </dd>
        </div>
        <div>
          <dt>
            {t(l, "Rounding difference", "금액 처리 차이", "فرق التقريب")}
          </dt>
          <dd>
            {result ? (
              <ExactValue
                locale={l}
                value={result.difference}
                currency={result.currency}
                signed
              />
            ) : (
              pending(l)
            )}
          </dd>
        </div>
      </dl>
      <p className="b05-meta">
        {t(l, "Calculation profile", "계산 프로필", "ملف الحساب")}:{" "}
        <bdi dir="ltr">{profile}</bdi>
      </p>
      <p className="b05-meta">
        {t(
          l,
          "Local source-cost review. No selling-price conversion or financial activation.",
          "로컬 원천 비용 검토입니다. 판매 금액 환산·재무 활성화는 미연결입니다.",
          "مراجعة محلية لتكلفة المصدر. تحويل مبلغ البيع والتفعيل المالي غير متصلين.",
        )}
      </p>
    </>
  );
}
export function ManualEvidence({
  locale: l,
  draft: d,
  source,
  edit,
}: {
  locale: Locale;
  draft: Draft;
  source?: Prepared;
  edit: () => void;
}) {
  const q = dateDays(d.quoteDate),
    r = dateDays(d.reviewDate);
  return (
    <section className="b05-imports">
      <h2>
        {t(
          l,
          "Manual quote evidence",
          "수동 환율 근거",
          "أدلة سعر الصرف اليدوية",
        )}
      </h2>
      {!draftKeys.some((k) => d[k]) && (
        <p>
          {t(
            l,
            "No manual quote evidence has been entered yet.",
            "아직 입력한 환율 근거가 없습니다.",
            "لم تُدخل أدلة سعر صرف يدوية بعد.",
          )}
        </p>
      )}
      <dl className="b05-evidence">
        <div>
          <dt>{label(l, "treatment")}</dt>
          <dd>
            {d.treatment === "unconverted"
              ? t(
                  l,
                  "User declared unconverted",
                  "사용자가 미환산으로 선언",
                  "أقرّ المستخدم بعدم التحويل",
                )
              : d.treatment
                ? t(
                    l,
                    "Already converted or unknown",
                    "이미 환산되었거나 확인 불가",
                    "محوّل بالفعل أو غير معروف",
                  )
                : missing(l)}
          </dd>
        </div>
        <div>
          <dt>
            {t(
              l,
              "Declared quote direction",
              "입력한 환율 방향",
              "اتجاه سعر الصرف المُدخل",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">
              {d.quoteUnit || "—"} {source?.row.currency ?? "—"} ={" "}
              {d.rate || "—"} {d.target || "—"}
            </bdi>
          </dd>
        </div>
        {(
          [
            "provider",
            "reference",
            "quoteDate",
            "reviewDate",
            "maxAge",
          ] as const
        ).map((k) => (
          <div key={k}>
            <dt>{label(l, k)}</dt>
            <dd>
              {k === "provider" || k === "reference" ? (
                d[k] || missing(l)
              ) : (
                <bdi
                  dir="ltr"
                  className={k.includes("Date") ? "b05-date" : undefined}
                >
                  {d[k] || missing(l)}
                  {d[k] && k.includes("Date") ? " UTC" : ""}
                </bdi>
              )}
            </dd>
          </div>
        ))}
        <div>
          <dt>
            {t(
              l,
              "Declared elapsed days",
              "선언 경과일",
              "الأيام المنقضية المصرّح بها",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">{q !== null && r !== null ? r - q : missing(l)}</bdi>
          </dd>
        </div>
      </dl>
      <button onClick={edit}>
        {t(
          l,
          "Edit manual evidence in conversion rules",
          "환산 규칙에서 수동 근거 입력",
          "تعديل الأدلة اليدوية في قواعد التحويل",
        )}
      </button>
      <h2>
        {t(
          l,
          "Official feeds are unconnected",
          "공식 고시 연동 없음",
          "مصادر الأسعار الرسمية غير متصلة",
        )}
      </h2>
      <p>
        {t(
          l,
          "No official quote feed, collection or CSV import is connected. Manual entries are reference evidence, not transaction authority. Dates and age are independently user declared, not certified market freshness.",
          "연결된 공식 고시·수집·CSV 반입이 없습니다. 수동 입력은 참고용 근거이며 거래 권한이 아닙니다. 날짜·경과일은 사용자 선언이며 검증된 시장 최신성이 아닙니다.",
          "لا توجد مصادر أسعار رسمية أو جمع أو استيراد CSV متصل. الإدخال اليدوي دليل مرجعي وليس صلاحية معاملة. التواريخ والعمر بإقرار مستقل من المستخدم وليست حداثة سوق معتمدة.",
        )}
      </p>
      <details>
        <summary>
          {t(
            l,
            "Unsupported by this review profile",
            "이 검토 프로필에서 지원하지 않음",
            "غير مدعوم في ملف المراجعة هذا",
          )}
        </summary>
        <p>
          {t(
            l,
            "Monthly averages, local timezones, holiday fallback, automatic latest quotes, customer-currency fee imports, supplier conversion history, contract precedence and server approvals are unconnected.",
            "월평균·현지 시간대·휴일 대체·자동 최신 환율·고객 통화 수수료 반입·공급사 환산 이력·계약 우선순위·서버 승인은 미연결입니다.",
            "المتوسطات الشهرية والمناطق الزمنية المحلية وبدائل العطلات وأحدث الأسعار التلقائية واستيراد رسوم عملة العميل وسجل تحويل المزوّد وأولوية العقود واعتمادات الخادم غير متصلة.",
          )}
        </p>
      </details>
    </section>
  );
}
export default function Results({
  locale: l,
  result,
  baseline,
  keep,
  clear,
}: {
  locale: Locale;
  result?: Result;
  baseline?: Result;
  keep: () => void;
  clear: () => void;
}) {
  const compatible = !!(result && baseline && comparable(result, baseline));
  const parameterKeys: (keyof Draft)[][] = [
    ["rate", "quoteUnit"],
    ["markup"],
    ["fee"],
    ["quantum"],
    ["mode"],
  ];
  const inputChanged = (index: number) =>
    !!(
      result &&
      baseline &&
      parameterKeys[index]?.some((k) => result.draft[k] !== baseline.draft[k])
    );
  const rows: [
    string,
    (r: Result) => React.ReactNode,
    ((r: Result) => Rational)?,
  ][] = [
    [
      t(l, "Quote equation", "입력 환율 방정식", "معادلة سعر الصرف"),
      (r) => (
        <bdi dir="ltr">
          {r.draft.quoteUnit} {r.sourceCurrency} = {r.draft.rate} {r.currency}
        </bdi>
      ),
    ],
    [label(l, "markup"), (r) => <bdi dir="ltr">{r.draft.markup}%</bdi>],
    [
      label(l, "fee"),
      (r) => (
        <bdi dir="ltr">
          {r.currency} {r.draft.fee}
        </bdi>
      ),
    ],
    [
      label(l, "quantum"),
      (r) => (
        <bdi dir="ltr">
          {r.currency} {r.draft.quantum}
        </bdi>
      ),
    ],
    [label(l, "mode"), (r) => modeLabel(l, r.draft.mode)],
    [
      t(l, "Converted — exact", "환산 후 — 정확값", "بعد التحويل — دقيق"),
      (r) => (
        <ExactValue locale={l} value={r.trace.quoted} currency={r.currency} />
      ),
      (r) => r.trace.quoted,
    ],
    [
      t(l, "After markup — exact", "가산 후 — 정확값", "بعد الزيادة — دقيق"),
      (r) => (
        <ExactValue locale={l} value={r.trace.marked} currency={r.currency} />
      ),
      (r) => r.trace.marked,
    ],
    [
      t(
        l,
        "Before rounding — including target fee",
        "처리 전 — 대상 통화 수수료 포함",
        "قبل التقريب — شاملاً رسوم العملة المستهدفة",
      ),
      (r) => (
        <ExactValue locale={l} value={r.trace.withFee} currency={r.currency} />
      ),
      (r) => r.trace.withFee,
    ],
    [
      t(l, "Rounded result", "처리 후 금액", "النتيجة المقربة"),
      (r) => <bdi dir="ltr">{money(r.total, r.currency, l)}</bdi>,
      (r) => asRational(r.total),
    ],
    [
      t(l, "Rounding difference", "금액 처리 차이", "فرق التقريب"),
      (r) => (
        <ExactValue
          locale={l}
          value={r.difference}
          currency={r.currency}
          signed
        />
      ),
      (r) => r.difference,
    ],
  ];
  return (
    <section
      data-fx-comparable={compatible ? "true" : "false"}
      data-fx-baseline={baseline ? "present" : undefined}
    >
      <h2>{t(l, "Conversion comparison", "환산 비교", "مقارنة التحويل")}</h2>
      <div className="b05-actions">
        <button id="fx-baseline" onClick={keep} disabled={!result}>
          {t(
            l,
            "Keep current candidate as session baseline",
            "현재 후보를 세션 기준으로 유지",
            "حفظ المرشح الحالي أساساً للجلسة",
          )}
        </button>
        <button id="fx-clear-baseline" onClick={clear} disabled={!baseline}>
          {t(l, "Clear baseline", "기준 지우기", "مسح الأساس")}
        </button>
      </div>
      <p className="b05-meta">
        {t(
          l,
          "Session only. Same source context and target currency are required for monetary differences.",
          "현재 세션에서만 유지합니다. 금액 차이는 같은 원천 범위·대상 통화에서만 계산합니다.",
          "لهذه الجلسة فقط. تتطلب فروق المبالغ سياق مصدر وعملة مستهدفة متطابقين.",
        )}
      </p>
      {!baseline && (
        <p>
          {t(
            l,
            "No session baseline. Keep a current candidate to compare.",
            "세션 기준이 없습니다. 현재 후보를 기준으로 유지하면 비교가 표시됩니다.",
            "لا يوجد أساس للجلسة. احتفظ بمرشح حالي للمقارنة.",
          )}
        </p>
      )}
      {result && baseline && (
        <p>
          {result.context === baseline.context
            ? t(
                l,
                "Same source context",
                "원천 범위 일치",
                "سياق المصدر متطابق",
              )
            : t(
                l,
                "Source context differs",
                "원천 범위 다름",
                "سياق المصدر مختلف",
              )}{" "}
          ·{" "}
          {result.currency === baseline.currency
            ? t(
                l,
                "Same target currency",
                "대상 통화 일치",
                "العملة المستهدفة متطابقة",
              )
            : t(
                l,
                "Target currency differs — no monetary subtraction",
                "대상 통화가 달라 금액 차이를 계산하지 않습니다",
                "العملة المستهدفة مختلفة — لا تُطرح المبالغ",
              )}
        </p>
      )}
      <div className="b05-comparison-table">
        <table>
          <thead>
            <tr>
              <th>{t(l, "Review item", "검토 항목", "عنصر المراجعة")}</th>
              <th>{t(l, "Session baseline", "세션 기준", "أساس الجلسة")}</th>
              <th>{t(l, "Current", "현재", "الحالي")}</th>
              <th>{t(l, "Change", "변화", "التغير")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([heading, format, amount], index) => (
              <tr key={heading}>
                <th scope="row">{heading}</th>
                <td
                  data-label={t(
                    l,
                    "Session baseline",
                    "세션 기준",
                    "أساس الجلسة",
                  )}
                >
                  {baseline ? format(baseline) : missing(l)}
                </td>
                <td data-label={t(l, "Current", "현재", "الحالي")}>
                  {result ? format(result) : pending(l)}
                </td>
                <td data-label={t(l, "Change", "변화", "التغير")}>
                  {compatible && amount ? (
                    <Difference
                      locale={l}
                      a={amount(result!)}
                      b={amount(baseline!)}
                      currency={result!.currency}
                    />
                  ) : compatible ? (
                    <span
                      data-fx-parameter-change={
                        inputChanged(index) ? "changed" : "unchanged"
                      }
                    >
                      {inputChanged(index)
                        ? t(l, "Changed input", "입력값 변경됨", "مدخل متغير")
                        : t(
                            l,
                            "Unchanged input",
                            "입력값 같음",
                            "مدخل دون تغيير",
                          )}
                    </span>
                  ) : (
                    t(l, "Not comparable", "비교 불가", "غير قابل للمقارنة")
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {result && (
        <>
          <h2 id="fx-trace-heading">
            {t(
              l,
              "Exact calculation trace",
              "정확 계산 추적",
              "أثر الحساب الدقيق",
            )}
          </h2>
          <p className="b05-meta">
            {t(
              l,
              "Scroll horizontally to inspect every exact integer. Stages are quoted conversion → markup → target fee → one quantum operation.",
              "가로로 스크롤하여 전체 정수를 확인하세요. 환산 → 가산 → 대상 통화 수수료 → 금액 처리 단위 한 번 적용 순서입니다.",
              "مرّر أفقياً لفحص كل عدد صحيح بدقة. الترتيب: تحويل السعر ثم الزيادة ثم رسوم العملة المستهدفة ثم عملية تقريب واحدة.",
            )}
          </p>
          <div
            className="b05-scroll b05-trace"
            dir="ltr"
            role="region"
            tabIndex={0}
            aria-labelledby="fx-trace-heading"
          >
            <table>
              <thead>
                <tr>
                  <th>{t(l, "Stage", "단계", "المرحلة")}</th>
                  <th>{t(l, "Numerator", "분자", "البسط")}</th>
                  <th>{t(l, "Denominator", "분모", "المقام")}</th>
                  <th>
                    {t(
                      l,
                      "Exact target amount",
                      "대상 통화 정확 금액",
                      "المبلغ الدقيق بالعملة المستهدفة",
                    )}
                  </th>
                </tr>
              </thead>
              <tbody>
                {(["quoted", "marked", "withFee"] as const).map((k) => (
                  <tr key={k}>
                    <th>
                      {k === "quoted"
                        ? t(
                            l,
                            "Quoted conversion",
                            "입력 환율 적용",
                            "تحويل سعر الصرف",
                          )
                        : k === "marked"
                          ? t(l, "Markup applied", "가산 적용", "تطبيق الزيادة")
                          : t(
                              l,
                              "Target fee added",
                              "대상 통화 수수료 추가",
                              "إضافة رسوم العملة المستهدفة",
                            )}
                    </th>
                    <td>
                      <bdi dir="ltr">{result.trace[k].numerator}</bdi>
                    </td>
                    <td>
                      <bdi dir="ltr">{result.trace[k].denominator}</bdi>
                    </td>
                    <td>
                      <ExactValue
                        locale={l}
                        value={result.trace[k]}
                        currency={result.currency}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <dl className="b05-evidence">
              {Object.entries(result.rounding).map(([k, v]) => (
                <div key={k}>
                  <dt>
                    {k === "numerator"
                      ? t(
                          l,
                          "Rounding numerator",
                          "금액 처리 분자",
                          "بسط التقريب",
                        )
                      : k === "denominator"
                        ? t(
                            l,
                            "Denominator × quantum coefficient",
                            "분모 × 처리 단위 계수",
                            "المقام × معامل وحدة التقريب",
                          )
                        : k === "quotient"
                          ? t(
                              l,
                              "Integer quotient before mode",
                              "방식 적용 전 정수 몫",
                              "خارج القسمة الصحيح قبل الطريقة",
                            )
                          : k === "remainder"
                            ? t(
                                l,
                                "Raw signed integer remainder",
                                "부호 있는 정수 나머지",
                                "الباقي الصحيح بإشارته",
                              )
                            : t(
                                l,
                                "Quantum count after mode",
                                "방식 적용 후 처리 단위 배수",
                                "عدد مضاعفات الوحدة بعد الطريقة",
                              )}
                  </dt>
                  <dd>
                    <bdi dir="ltr">{v}</bdi>
                  </dd>
                </div>
              ))}
            </dl>
          </div>
          <p className="b05-meta">
            {t(
              l,
              "Trace fractions are target coefficients at scale 18. Raw quotient and remainder are integer evidence, not monetary differences. Rounding difference is rounded amount minus the exact amount before rounding.",
              "계산 분수는 소수점 18자리 배율의 대상 통화 계수입니다. 정수 몫·나머지는 금액 차이가 아닙니다. 금액 처리 차이는 처리 후 금액에서 처리 전 정확값을 뺀 값입니다.",
              "كسور الأثر معاملات للعملة المستهدفة بمقياس 18. خارج القسمة والباقي أدلة صحيحة وليسا فروقاً نقدية. فرق التقريب هو المبلغ المقرب ناقص المبلغ الدقيق قبل التقريب.",
            )}
          </p>
          <details>
            <summary>
              {t(
                l,
                "Exact rate ratio — no decimal truncation",
                "정확 환율 비율 — 소수 절삭 없음",
                "نسبة الصرف الدقيقة — دون اقتطاع عشري",
              )}
            </summary>
            <ExactValue
              locale={l}
              value={{
                numerator: (
                  BigInt(result.normalized.numerator) * scale
                ).toString(),
                denominator: result.normalized.denominator,
              }}
              currency={`${result.currency}/${result.sourceCurrency}`}
            />
          </details>
        </>
      )}
    </section>
  );
}
