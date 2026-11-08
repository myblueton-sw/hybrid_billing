import { useEffect, useRef, type ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import { decimal } from "../../../sources/collected-focus/values";
import {
  fieldNames,
  type Data,
  type Charge,
  type Field,
  type Measure,
} from "./contract";
import {
  t,
  fieldLabel,
  stateLabel,
  basisLabel,
  unavailable,
  unknown,
} from "./copy";
export const money = (v: string, c: string, l: Locale) => {
  const n = BigInt(v);
  return `${n < 0n ? "−" : ""}${c} ${decimal((n < 0n ? -n : n).toString(), l)}`;
};
export function Money({
  value,
  currency,
  locale: l,
}: {
  value: string | null;
  currency: string;
  locale: Locale;
}) {
  return value === null ? (
    <span>{t(l, "No observed amount", "관측 금액 없음", "لا مبلغ مرصود")}</span>
  ) : (
    <bdi dir="ltr">{money(value, currency, l)}</bdi>
  );
}
function MeasureValue({
  measure: m,
  currency,
  locale: l,
}: {
  measure: Measure;
  currency: string;
  locale: Locale;
}) {
  return (
    <>
      <Money value={m.coefficient} currency={currency} locale={l} />
      <small>
        {t(l, "Present / missing", "값 있음 / 누락", "موجود / مفقود")}{" "}
        <bdi dir="ltr">
          {m.present} / {m.missing}
        </bdi>
      </small>
    </>
  );
}
function FieldValue({ field: f, locale: l }: { field: Field; locale: Locale }) {
  return (
    <>
      <span>
        {f.state === "value" || (f.state === "unmapped_version" && f.value) ? (
          <bdi dir="ltr">{f.value}</bdi>
        ) : (
          stateLabel(l, f.state)
        )}
      </span>
      {f.value !== null && <small>{stateLabel(l, f.state)}</small>}
    </>
  );
}
function ContractBoundary({ locale: l }: { locale: Locale }) {
  return (
    <section className="b07-boundary">
      <h2>{t(l, "Contractual decisions", "계약 판단", "قرارات تعاقدية")}</h2>
      <dl className="b07-facts">
        {[
          [
            t(
              l,
              "Commitment consumption recognition",
              "약정 소진 인정",
              "احتساب استهلاك الالتزام",
            ),
            unknown(l),
          ],
          [
            t(l, "Discount eligibility", "할인 적격성", "أهلية الخصم"),
            unknown(l),
          ],
          [
            t(
              l,
              "Credit to original transaction",
              "크레딧·원거래 연결",
              "ربط الرصيد بالمعاملة الأصلية",
            ),
            unavailable(l),
          ],
        ].map(([label, state]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{state}</dd>
          </div>
        ))}
      </dl>
      <p>
        {t(
          l,
          "Consumption recognition does not establish discount eligibility. Source Credit classification does not authorize a credit or match its original transaction. Marketplace fees are not automatically passed through.",
          "소진 인정은 할인 적격성을 뜻하지 않습니다. 원천 Credit 분류만으로 크레딧 승인·원거래 연결을 확인할 수 없으며 판매 수수료를 자동 전가하지 않습니다.",
          "احتساب الاستهلاك لا يثبت أهلية الخصم. تصنيف Credit من المصدر لا يجيز رصيداً ولا يربطه بمعاملته الأصلية. لا تُمرّر رسوم السوق تلقائياً.",
        )}
      </p>
    </section>
  );
}
function SourcePanel({
  locale: l,
  row: r,
  view,
  close,
}: {
  locale: Locale;
  row: Charge;
  view: string;
  close: () => void;
}) {
  return (
    <section
      className="b07-detail"
      aria-label={t(
        l,
        "Selected source evidence",
        "선택 원천 근거",
        "أدلة المصدر المحدد",
      )}
      onKeyDown={(e) => {
        if (
          e.key === "Escape" &&
          !(e.target as HTMLElement).closest("details[open]")
        ) {
          e.preventDefault();
          close();
        }
      }}
    >
      <header>
        <h2 tabIndex={-1} data-b07-detail-heading>
          {t(
            l,
            "Selected source evidence",
            "선택 원천 근거",
            "أدلة المصدر المحدد",
          )}
        </h2>
        <button type="button" onClick={close}>
          {t(
            l,
            "Close source details",
            "원천 상세 닫기",
            "إغلاق تفاصيل المصدر",
          )}
        </button>
      </header>
      {view === "credits" && <ContractBoundary locale={l} />}
      <p className="b07-meta">
        {t(
          l,
          "Technical source reference, not a transaction ID",
          "원천 기술 참조 · 거래 ID가 아닙니다",
          "مرجع تقني للمصدر، وليس معرّف معاملة",
        )}
        <bdi dir="ltr">{r.id}</bdi>
      </p>
      {view === "buyer" && (
        <dl className="b07-row-costs">
          {(["billed_cost", "effective_cost"] as const).map((b) => (
            <div key={b}>
              <dt>{basisLabel(l, b)}</dt>
              <dd>
                <Money value={r[b]} currency={r.currency} locale={l} />
              </dd>
            </div>
          ))}
        </dl>
      )}
      <dl className="b07-facts">
        {(view === "mapping"
          ? ["publisher", "issuer", "sku"]
          : view === "credits"
            ? ["category", "charge_class", "commitment"]
            : fieldNames
        ).map((k) => (
          <div key={k}>
            <dt>{fieldLabel(l, k as keyof Charge["fields"])}</dt>
            <dd>
              <FieldValue
                field={r.fields[k as keyof Charge["fields"]]}
                locale={l}
              />
            </dd>
          </div>
        ))}
      </dl>
      {view === "mapping" && (
        <>
          <h2>
            {t(l, "Contract counterpart", "계약 연결 근거", "المقابل التعاقدي")}
          </h2>
          <dl className="b07-facts">
            {[
              t(l, "Offer / agreement", "Offer / 계약", "العرض / الاتفاقية"),
              t(
                l,
                "Legal buyer responsibility",
                "법적 구매 책임",
                "مسؤولية المشتري القانونية",
              ),
              t(
                l,
                "Legal seller responsibility",
                "법적 판매 책임",
                "مسؤولية البائع القانونية",
              ),
              t(l, "Original transaction", "원거래", "المعاملة الأصلية"),
            ].map((x) => (
              <div key={x}>
                <dt>{x}</dt>
                <dd>{unavailable(l)}</dd>
              </div>
            ))}
          </dl>
          <p>
            {t(
              l,
              "A source SKU or account is not an offer, agreement or legal responsibility mapping.",
              "원천 SKU·계정은 offer·계약·법적 책임 연결을 대신하지 않습니다.",
              "SKU أو الحساب من المصدر ليس ربطاً بالعرض أو الاتفاقية أو المسؤولية القانونية.",
            )}
          </p>
        </>
      )}
      <details className="b07-source-context">
        <summary>
          {t(
            l,
            "Period and source references",
            "기간·원천 참조",
            "الفترة ومراجع المصدر",
          )}
        </summary>
        <dl className="b07-facts">
          {[
            [
              t(l, "Charge interval", "과금 기간", "فترة الرسوم"),
              `${r.charge_start} / ${r.charge_end}`,
            ],
            [
              t(l, "Billing interval", "청구 기간", "فترة الفوترة"),
              `${r.billing_start} / ${r.billing_end}`,
            ],
            [t(l, "FOCUS version", "FOCUS 버전", "إصدار FOCUS"), r.version],
            [
              t(
                l,
                "Billing account / sub-account",
                "청구 계정 / 하위 계정",
                "حساب الفوترة / الحساب الفرعي",
              ),
              `${r.billing_account} / ${r.sub_account ?? t(l, "None in source", "원천 값 없음", "لا قيمة في المصدر")}`,
            ],
            [
              t(
                l,
                "Canonical account reference",
                "원천 계정 참조",
                "مرجع الحساب",
              ),
              r.account,
            ],
            [
              t(l, "Tenant reference", "테넌트 참조", "مرجع المستأجر"),
              r.tenant,
            ],
            [
              t(
                l,
                "Source batch / ordinal",
                "원천 배치 / 행 순번",
                "دفعة المصدر / ترتيب الصف",
              ),
              `${r.batch} / ${r.ordinal}`,
            ],
            [
              t(
                l,
                "Source provenance class",
                "원천 출처 분류",
                "تصنيف منشأ المصدر",
              ),
              r.evidence_class,
            ],
          ].map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <bdi dir="ltr">{value}</bdi>
              </dd>
            </div>
          ))}
        </dl>
      </details>
    </section>
  );
}
export function Summary({
  locale: l,
  data: d,
}: {
  locale: Locale;
  data: Data;
}) {
  return (
    <section
      className="b07-summary"
      aria-label={t(l, "Source classification", "원천 분류", "تصنيف المصدر")}
    >
      <dl>
        {[
          [
            t(
              l,
              "Source rows in selection",
              "조회 범위 원천 행",
              "صفوف المصدر ضمن الاختيار",
            ),
            d.row_count,
          ],
          [
            t(
              l,
              "Marketplace candidate rows",
              "Marketplace 후보 행",
              "صفوف مرشّحة للسوق",
            ),
            d.counts.candidate,
          ],
          [
            t(
              l,
              "No name difference observed",
              "이름 차이 없음",
              "لم يُرصد اختلاف في الاسمين",
            ),
            d.counts.no_difference,
          ],
          [
            t(
              l,
              "Insufficient classification evidence",
              "분류 근거 부족",
              "أدلة غير كافية للتصنيف",
            ),
            d.counts.unknown,
          ],
        ].map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>
              <bdi dir="ltr">{value}</bdi>
            </dd>
          </div>
        ))}
      </dl>
      <p className="b07-meta">
        <bdi dir="ltr">
          {d.counts.candidate} + {d.counts.no_difference} + {d.counts.unknown} ={" "}
          {d.row_count}
        </bdi>{" "}
        ·{" "}
        {t(
          l,
          "Exact source-name comparison; no legal-party or direct-purchase conclusion",
          "원천 이름의 정확 일치 비교 · 법인 동일성·직접 구매를 판정하지 않습니다",
          "مقارنة دقيقة لأسماء المصدر؛ لا استنتاج بشأن الهوية القانونية أو الشراء المباشر",
        )}
      </p>
    </section>
  );
}
export function Totals({ locale: l, data: d }: { locale: Locale; data: Data }) {
  return (
    <section className="b07-totals">
      <h2>
        {t(
          l,
          "Costs across all candidate pages",
          "전체 후보 페이지의 비용",
          "تكاليف جميع صفحات المرشحين",
        )}
      </h2>
      <p className="b07-meta">
        {t(
          l,
          "Currencies and cost bases stay separate. Source costs do not certify customer invoices or collection.",
          "통화·비용 기준을 각각 표시합니다. 원천 비용은 고객 청구·수납 확정이 아닙니다.",
          "تُعرض العملات وأسس التكلفة منفصلة. تكاليف المصدر لا تثبت فوترة العملاء أو التحصيل.",
        )}
      </p>
      {d.candidate_amounts.length ? (
        <div className="b07-currency-totals">
          {d.candidate_amounts.map((a) => (
            <dl key={a.currency}>
              <div>
                <dt>
                  {t(
                    l,
                    "Candidate source rows",
                    "후보 원천 행",
                    "صفوف المصدر المرشّحة",
                  )}{" "}
                  · <bdi dir="ltr">{a.currency}</bdi>
                </dt>
                <dd>
                  <bdi dir="ltr">{a.rows}</bdi>
                </dd>
              </div>
              {(["billed_cost", "effective_cost"] as const).map((b) => (
                <div key={b}>
                  <dt>{basisLabel(l, b)}</dt>
                  <dd>
                    <MeasureValue
                      measure={a[b]}
                      currency={a.currency}
                      locale={l}
                    />
                  </dd>
                </div>
              ))}
            </dl>
          ))}
        </div>
      ) : (
        <p>
          {t(
            l,
            "No candidate cost observations in this selection",
            "이 조회 범위에서 후보 비용이 관측되지 않았습니다",
            "لا ملاحظات تكلفة للمرشحين ضمن هذا الاختيار",
          )}
        </p>
      )}
      <details className="b07-unclassified">
        <summary>
          {t(
            l,
            "Rows with insufficient classification evidence",
            "분류 근거가 부족한 행",
            "صفوف بأدلة تصنيف غير كافية",
          )}{" "}
          · <bdi dir="ltr">{d.counts.unknown}</bdi>
        </summary>
        <p>
          {t(
            l,
            "Null or unavailable issuer/publisher names remain unknown. Equal names alone do not establish direct purchase. These source costs are excluded from candidate totals.",
            "발행사·제공사 이름이 null·미확인이면 분류를 확정하지 않습니다. 같은 이름도 직접 구매를 뜻하지 않습니다. 이 원천 비용은 후보 합계에 포함하지 않습니다.",
            "الأسماء الفارغة أو غير المتاحة تبقى غير معروفة. تساوي الأسماء وحده لا يثبت الشراء المباشر. تُستبعد هذه التكاليف من إجماليات المرشحين.",
          )}
        </p>
        {d.unknown_amounts.map((a) => (
          <dl className="b07-row-costs" key={a.currency}>
            {(["billed_cost", "effective_cost"] as const).map((b) => (
              <div key={b}>
                <dt>{basisLabel(l, b)}</dt>
                <dd>
                  <MeasureValue
                    measure={a[b]}
                    currency={a.currency}
                    locale={l}
                  />
                </dd>
              </div>
            ))}
          </dl>
        ))}
      </details>
    </section>
  );
}
export function CandidateList({
  locale: l,
  data: d,
  view,
  select,
  close,
  page,
}: {
  locale: Locale;
  data: Data;
  view: string;
  select: (id: string) => void;
  close: () => void;
  page: (n: number) => void;
}) {
  const root = useRef<HTMLDivElement>(null),
    last = useRef<string | null>(null);
  useEffect(() => {
    if (d.selected && last.current !== d.selected.id) {
      const heading = [
        ...root.current!.querySelectorAll<HTMLElement>(
          "[data-b07-detail-heading]",
        ),
      ].find((h) => h.getClientRects().length);
      heading?.focus();
    }
    last.current = d.selected?.id ?? null;
  }, [d.selected]);
  const columns: [string, (r: Charge) => ReactNode][] = [
    [
      t(l, "Source charge", "원천 청구", "رسوم المصدر"),
      (r) => (
        <>
          <span>
            {r.service ??
              t(l, "Service unavailable", "서비스 값 없음", "الخدمة غير متاحة")}
          </span>
          <small>
            {fieldLabel(l, "publisher")}{" "}
            <bdi dir="ltr">{r.fields.publisher.value}</bdi>
          </small>
          <small>
            {fieldLabel(l, "issuer")}{" "}
            <bdi dir="ltr">{r.fields.issuer.value}</bdi>
          </small>
        </>
      ),
    ],
    [
      view === "credits"
        ? fieldLabel(l, "category")
        : t(
            l,
            "Account / charge start",
            "계정 / 과금 시작",
            "الحساب / بدء الرسوم",
          ),
      (r) =>
        view === "credits" ? (
          <>
            <FieldValue field={r.fields.category} locale={l} />
            <FieldValue field={r.fields.commitment} locale={l} />
          </>
        ) : (
          <>
            <bdi dir="ltr">{r.billing_account}</bdi>
            <small>
              <bdi className="b07-charge-date" dir="ltr">
                {r.charge_start.slice(0, 10)}
              </bdi>
              <bdi dir="ltr">{r.charge_start.slice(11)}</bdi>
            </small>
          </>
        ),
    ],
    [
      basisLabel(l, "billed_cost"),
      (r) => <Money value={r.billed_cost} currency={r.currency} locale={l} />,
    ],
    [
      basisLabel(l, "effective_cost"),
      (r) => (
        <Money value={r.effective_cost} currency={r.currency} locale={l} />
      ),
    ],
    [
      t(l, "Evidence", "근거", "الأدلة"),
      (r) => (
        <button
          type="button"
          data-b07-row={r.id}
          aria-pressed={d.selected?.id === r.id}
          aria-label={t(
            l,
            `Inspect ${r.fields.publisher.value} source charge`,
            ` ${r.fields.publisher.value} 원천 청구 근거 보기`,
            `فحص رسوم المصدر ${r.fields.publisher.value}`,
          )}
          onClick={() => select(r.id)}
        >
          {d.selected?.id === r.id
            ? t(l, "Selected", "선택됨", "محدد")
            : t(l, "Inspect source", "원천 근거 보기", "فحص المصدر")}
        </button>
      ),
    ],
  ];
  return (
    <div
      className={`b07-workspace ${d.selected ? "has-detail" : ""}`}
      ref={root}
    >
      <section className="b07-list">
        <header>
          <h2>
            {t(
              l,
              "Candidate source rows",
              "후보 원천 행",
              "صفوف المصدر المرشّحة",
            )}
          </h2>
          <bdi dir="ltr">
            {d.rows.length ? d.after + 1 : 0}–{d.after + d.rows.length} /{" "}
            {d.counts.candidate}
          </bdi>
        </header>
        {!d.rows.length ? (
          <p className="b07-empty">
            {t(
              l,
              "No candidate rows observed in this selection. This does not prove there were no Marketplace purchases. Change the source, period or account to inspect another scope.",
              "이 조회 범위에서 후보 행이 관측되지 않았습니다. Marketplace 구매가 없었다는 뜻은 아닙니다. 원천·기간·계정을 바꿔 다른 범위를 확인하세요.",
              "لم تُرصد صفوف مرشّحة ضمن هذا الاختيار. لا يثبت ذلك عدم وجود مشتريات سوق. غيّر المصدر أو الفترة أو الحساب لفحص نطاق آخر.",
            )}
          </p>
        ) : (
          <>
            <div className="b07-mobile-list">
              {d.rows.map((r) => (
                <article
                  key={r.id}
                  className={d.selected?.id === r.id ? "selected" : ""}
                >
                  <h3>{r.service}</h3>
                  <p>
                    <bdi dir="ltr">{r.fields.description.value}</bdi>
                  </p>
                  <dl className="b07-facts">
                    <div>
                      <dt>{fieldLabel(l, "publisher")}</dt>
                      <dd>
                        <bdi dir="ltr">{r.fields.publisher.value}</bdi>
                      </dd>
                    </div>
                    <div>
                      <dt>{fieldLabel(l, "issuer")}</dt>
                      <dd>
                        <bdi dir="ltr">{r.fields.issuer.value}</bdi>
                      </dd>
                    </div>
                  </dl>
                  <dl className="b07-row-costs">
                    {(["billed_cost", "effective_cost"] as const).map((b) => (
                      <div key={b}>
                        <dt>{basisLabel(l, b)}</dt>
                        <dd>
                          <Money
                            value={r[b]}
                            currency={r.currency}
                            locale={l}
                          />
                        </dd>
                      </div>
                    ))}
                  </dl>
                  {columns[4][1](r)}
                  {d.selected?.id === r.id && (
                    <SourcePanel locale={l} row={r} view={view} close={close} />
                  )}
                </article>
              ))}
            </div>
            <details className="b07-full-table" open={window.innerWidth >= 768}>
              <summary>
                {t(
                  l,
                  "Show full-column table",
                  "전체 열 표 보기",
                  "عرض جدول جميع الأعمدة",
                )}
              </summary>
              <div
                className="b07-table-region"
                tabIndex={0}
                role="region"
                aria-label={t(
                  l,
                  "Candidate source rows table",
                  "후보 원천 행 표",
                  "جدول صفوف المصدر المرشّحة",
                )}
              >
                <table>
                  <caption>
                    {t(
                      l,
                      "Marketplace candidate source evidence",
                      "Marketplace 후보 원천 근거",
                      "أدلة مصدر مرشّحة للسوق",
                    )}
                  </caption>
                  <thead>
                    <tr>
                      {columns.map(([label], i) => (
                        <th
                          scope="col"
                          key={label}
                          className={i === 2 || i === 3 ? "money" : ""}
                        >
                          {label}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {d.rows.map((r) => (
                      <tr
                        key={r.id}
                        className={d.selected?.id === r.id ? "selected" : ""}
                      >
                        {columns.map(([label, render], i) => (
                          <td
                            key={label}
                            className={i === 2 || i === 3 ? "money" : ""}
                          >
                            {render(r)}
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          </>
        )}
        <div className="b07-pager">
          <button
            id="b07-prev"
            disabled={d.after === 0}
            onClick={() => page(Math.max(0, d.after - 25))}
          >
            {t(l, "Previous rows", "이전 행", "الصفوف السابقة")}
          </button>
          <button
            id="b07-next"
            disabled={d.next_after === null}
            onClick={() => d.next_after !== null && page(d.next_after)}
          >
            {t(l, "Next rows", "다음 행", "الصفوف التالية")}
          </button>
        </div>
      </section>
      {d.selected ? (
        <aside className="b07-desktop-detail">
          <SourcePanel locale={l} row={d.selected} view={view} close={close} />
        </aside>
      ) : view === "credits" ? (
        <ContractBoundary locale={l} />
      ) : (
        <p className="b07-select-prompt">
          {t(
            l,
            "Select a candidate row to inspect its source evidence.",
            "후보 행을 선택하면 원천 근거를 확인할 수 있습니다.",
            "اختر صفاً مرشّحاً لفحص أدلة مصدره.",
          )}
        </p>
      )}
    </div>
  );
}
export function Seller({
  locale: l,
  count,
}: {
  locale: Locale;
  count: number;
}) {
  const rows = [
    [
      t(
        l,
        "Seller gross billed amount",
        "판매자 청구 총액",
        "إجمالي المبلغ المفوتر للبائع",
      ),
      t(
        l,
        "Seller billed-revenue records",
        "판매자 청구 수익 명세",
        "سجلات إيرادات البائع المفوترة",
      ),
    ],
    [
      t(l, "Marketplace fee", "판매 수수료", "رسوم السوق"),
      t(
        l,
        "Fee records and agreement terms",
        "수수료 명세·계약 조건",
        "سجلات الرسوم وشروط الاتفاقية",
      ),
    ],
    [
      t(l, "Refunds", "환불액", "المبالغ المستردة"),
      t(
        l,
        "Refund records with original charge references",
        "원청구 참조가 있는 환불 명세",
        "سجلات استرداد مرتبطة بالرسوم الأصلية",
      ),
    ],
    [
      t(l, "Taxes", "세금", "الضرائب"),
      t(
        l,
        "Applicable tax records and responsibilities",
        "적용 세금 명세·책임",
        "سجلات الضرائب ومسؤولياتها",
      ),
    ],
    [
      t(l, "Withheld amount", "보류액", "المبلغ المحتجز"),
      t(
        l,
        "Withholding and release evidence",
        "보류·해제 근거",
        "أدلة الاحتجاز والإفراج",
      ),
    ],
    [
      t(l, "Net settlement", "순정산액", "صافي التسوية"),
      t(
        l,
        "Authoritative seller settlement records",
        "판매자 확정 정산 명세",
        "سجلات تسوية البائع المعتمدة",
      ),
    ],
    [
      t(l, "Disbursement evidence", "지급 내역", "أدلة الصرف"),
      t(
        l,
        "Disbursement amount, date and status",
        "지급 금액·일자·상태",
        "مبلغ الصرف وتاريخه وحالته",
      ),
    ],
  ];
  return (
    <section className="b07-seller">
      <h2>{t(l, "Seller settlement", "판매자 정산", "تسوية البائع")}</h2>
      <p>
        {t(
          l,
          "Seller records are not connected. Buyer source costs cannot establish seller revenue, customer collection or disbursement.",
          "판매자 정산 명세가 연결되지 않았습니다. 구매 측 원천 비용으로 판매 수익·고객 수납·지급을 확정할 수 없습니다.",
          "سجلات البائع غير متصلة. تكاليف الشراء من المصدر لا تثبت إيرادات البائع أو تحصيل العملاء أو الصرف.",
        )}
      </p>
      <p className="b07-meta">
        {t(
          l,
          "Candidate buyer source rows in this scope",
          "현재 범위의 구매 측 후보 원천 행",
          "صفوف مصدر الشراء المرشّحة ضمن النطاق",
        )}{" "}
        <bdi dir="ltr">{count}</bdi>
      </p>
      <dl className="b07-settlement">
        {rows.map(([label, evidence]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>
              <strong>{unavailable(l)}</strong>
              <span>{evidence}</span>
            </dd>
          </div>
        ))}
      </dl>
      <p>
        {t(
          l,
          "These are separate evidence requirements, not a calculated settlement sequence. Settlement delays do not change customer documents or establish payment collection.",
          "각 항목은 별도 확인 근거이며 순정산 계산 순서가 아닙니다. 정산 지연을 이유로 고객 문서를 변경하거나 수납 완료로 처리하지 않습니다.",
          "هذه متطلبات أدلة منفصلة وليست تسلسلاً لحساب التسوية. تأخر التسوية لا يغير مستندات العملاء ولا يثبت التحصيل.",
        )}
      </p>
    </section>
  );
}
