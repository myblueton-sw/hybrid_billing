import type { ReactNode, Ref } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { FocusRow, OverviewData } from "../../collected-focus/api";
import {
  annotationColumn,
  chargeMonthBasis,
  chargeStartsInBillingPeriod,
  instantParts,
  missingFields,
  provenanceClass,
  rowVersion,
  valueState,
  type MappedField,
  type NumericField,
} from "../../collected-focus/fields";
import { decimal, money } from "../../collected-focus/values";
import { SourceColumnNames, VersionScope } from "./FieldReference";
import {
  fieldLabel,
  Instant,
  missingLabel,
  originLabel,
  originMeaning,
  provenanceLabel,
  unitMissingLabel,
  zeroLabel,
} from "../../collected-focus/labels";
const shownTimezones = 8;
export default function RowDetail({
  l,
  row,
  data,
  detailRef,
  onClose,
}: {
  l: Locale;
  row: FocusRow;
  data: Pick<OverviewData, "versions" | "tenants">;
  detailRef: Ref<HTMLElement>;
  onClose: () => void;
}) {
  const versions = data.versions,
    version = rowVersion(versions),
    basis = chargeMonthBasis(row, data.tenants),
    inBilling = chargeStartsInBillingPeriod(row),
    anyMissing = missingFields(row).length > 0;
  const noValue = t(
    l,
    "No value in the collected row",
    "수집된 행에 값이 없습니다",
    "لا قيمة في الصف المجمَّع",
  );
  const missing = (
    <strong className="collected-state">{missingLabel(l)}</strong>
  );
  const line = (
    field: MappedField,
    value: ReactNode,
    note?: ReactNode,
    unit?: { field: MappedField; value: string | null },
  ) => (
    <tr key={field} data-field={field} role="row">
      <th scope="row" role="rowheader">
        {fieldLabel(l, field)}
      </th>
      <td className="collected-names" role="cell">
        <span>
          <SourceColumnNames l={l} field={field} versions={versions} />
        </span>
        {unit && (
          <span>
            <SourceColumnNames l={l} field={unit.field} versions={versions} />
          </span>
        )}
      </td>
      <td className="collected-value" role="cell">
        <span>{value}</span>
        {unit &&
          (unit.value === null ? (
            <strong className="collected-state collected-unit">
              {unitMissingLabel(l)}
            </strong>
          ) : (
            <span className="collected-unit">
              <bdi>{unit.value}</bdi>
            </span>
          ))}
      </td>
      <td className="collected-note" role="cell">
        {note}
      </td>
    </tr>
  );
  const number = (
    field: NumericField,
    unit?: { field: MappedField; value: string | null },
    currency?: string,
  ) => {
    const state = valueState(row[field]);
    return line(
      field,
      state === "missing" ? (
        missing
      ) : (
        <bdi dir="ltr" className="collected-token">
          {currency ? money(row[field], currency, l) : decimal(row[field], l)}
        </bdi>
      ),
      state === "missing" ? (
        noValue
      ) : state === "zero" ? (
        <strong className="collected-state">{zeroLabel(l)}</strong>
      ) : null,
      unit,
    );
  };
  const textLine = (field: MappedField, value: string | null) =>
    line(
      field,
      value === null ? missing : <bdi>{value}</bdi>,
      value === null ? noValue : null,
    );
  const instant = (field: MappedField, value: string, end: boolean) =>
    line(
      field,
      <Instant value={value} full />,
      instantParts(value)
        ? end
          ? t(
              l,
              "End exclusive · UTC",
              "종료 시각 미포함 · UTC",
              "النهاية مستثناة · UTC",
            )
          : t(
              l,
              "Start inclusive · UTC",
              "시작 시각 포함 · UTC",
              "البداية مشمولة · UTC",
            )
        : t(
            l,
            "Time format not interpreted",
            "시각 형식을 해석하지 못했습니다",
            "تعذّر تفسير صيغة الوقت",
          ),
    );
  const group = (label: string) => (
    <tr className="collected-group" role="row">
      <th scope="rowgroup" colSpan={4} role="rowheader">
        {label}
      </th>
    </tr>
  );
  const pricingUnit = {
    field: "pricing_unit" as const,
    value: row.pricing_unit,
  };
  const close = (
    <button onClick={onClose}>
      {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
    </button>
  );
  const unsupplied = [
    t(
      l,
      "Original column text before normalization",
      "정규화 전 원본 열 문자열",
      "نص العمود الأصلي قبل التطبيع",
    ),
    t(l, "Reason a value is missing", "값이 누락된 사유", "سبب فقدان القيمة"),
    ...(version
      ? []
      : [
          t(
            l,
            "Mapping version of this row",
            "이 행의 매핑 버전",
            "إصدار الربط لهذا الصف",
          ),
        ]),
    ...(basis.rowTimezone
      ? []
      : [
          t(
            l,
            "Tenant and time zone of this row",
            "이 행의 테넌트와 시간대",
            "مستأجر هذا الصف ومنطقته الزمنية",
          ),
        ]),
    t(l, "Source delivery window", "원천 수집 구간", "نطاق تسليم المصدر"),
    t(
      l,
      "Row-level transformation history",
      "행별 변환 이력",
      "سجل التحويل على مستوى الصف",
    ),
    t(
      l,
      "Link to the original evidence",
      "원본 증빙 연결",
      "رابط إلى الدليل الأصلي",
    ),
  ];
  return (
    <section
      className="collected-detail"
      ref={detailRef}
      tabIndex={-1}
      data-item="S04:detail:field-origin"
      aria-label={t(
        l,
        "Selected row details",
        "선택 행 상세",
        "تفاصيل الصف المحدد",
      )}
    >
      <header>
        <div>
          <h2>
            {t(l, "Selected row details", "선택 행 상세", "تفاصيل الصف المحدد")}
          </h2>
          <p className="collected-row-line">
            <span>{row.provider}</span>
            <bdi>{row.billing_account}</bdi>
            <bdi dir="ltr">{row.charge_month}</bdi>
            <bdi dir="ltr" className="collected-code">
              {row.id.slice(0, 12)}
            </bdi>
          </p>
        </div>
        {close}
      </header>
      <div className="collected-detail-body">
        <div className="collected-detail-source">
          <h3>{originLabel(l, "source")}</h3>
          <p className="collected-caption">
            {originMeaning(l, "source")}{" "}
            <VersionScope l={l} versions={versions} />
          </p>
          <table className="collected-field-table" role="table">
            <thead role="rowgroup">
              <tr role="row">
                <th scope="col" role="columnheader">
                  {t(l, "Field", "필드", "الحقل")}
                </th>
                <th scope="col" role="columnheader">
                  {t(l, "FOCUS column", "FOCUS 열", "عمود FOCUS")}
                </th>
                <th scope="col" role="columnheader">
                  {t(l, "Value", "값", "القيمة")}
                </th>
                <th scope="col" role="columnheader">
                  {t(l, "Note", "비고", "ملاحظة")}
                </th>
              </tr>
            </thead>
            <tbody role="rowgroup">
              {group(t(l, "Costs", "비용", "التكاليف"))}
              {number("billed_cost", undefined, row.currency)}
              {number("effective_cost", undefined, row.currency)}
              {number("list_cost", undefined, row.currency)}
              {number("contracted_cost", undefined, row.currency)}
            </tbody>
            <tbody role="rowgroup">
              {group(
                t(
                  l,
                  "Quantities and unit prices",
                  "수량·단가",
                  "الكميات وأسعار الوحدة",
                ),
              )}
              {number("consumed_quantity", {
                field: "consumed_unit",
                value: row.consumed_unit,
              })}
              {number("pricing_quantity", pricingUnit)}
              {number("list_unit_price", pricingUnit, row.currency)}
              {number("contracted_unit_price", pricingUnit, row.currency)}
            </tbody>
            <tbody role="rowgroup">
              {group(t(l, "Periods", "기간", "الفترات"))}
              {instant("charge_start", row.charge_start, false)}
              {instant("charge_end", row.charge_end, true)}
              {instant("billing_start", row.billing_start, false)}
              {instant("billing_end", row.billing_end, true)}
              {inBilling === false && (
                <tr data-period-note role="row">
                  <td colSpan={4} role="cell">
                    {t(
                      l,
                      "The charge period starts outside the billing period in this row. Both are shown as collected.",
                      "이 행의 과금 기간 시작은 청구 기간 밖에 있습니다. 두 기간 모두 수집된 값 그대로 표시합니다.",
                      "تبدأ فترة الرسوم خارج فترة الفوترة في هذا الصف. تُعرض الفترتان كما جُمعتا.",
                    )}
                  </td>
                </tr>
              )}
            </tbody>
            <tbody role="rowgroup">
              {group(
                t(l, "Account and service", "계정·서비스", "الحساب والخدمة"),
              )}
              {textLine("provider", row.provider)}
              {textLine("service", row.service)}
              {textLine("billing_account", row.billing_account)}
              {textLine("sub_account", row.sub_account)}
              {line("currency", <bdi dir="ltr">{row.currency}</bdi>)}
            </tbody>
          </table>
          {anyMissing && (
            <p className="collected-caption" data-missing-reason>
              {t(
                l,
                "The row record does not carry the reason a value is missing.",
                "행 기록에는 값이 누락된 사유가 포함되어 있지 않습니다.",
                "سجل الصف لا يتضمن سبب فقدان القيمة.",
              )}
            </p>
          )}
        </div>
        <div className="collected-detail-side">
          <section data-origin="derived">
            <h3>{originLabel(l, "derived")}</h3>
            <dl>
              <dt>{fieldLabel(l, "charge_month")}</dt>
              <dd data-field="charge_month">
                <bdi dir="ltr">{row.charge_month}</bdi>
                <small>
                  {t(
                    l,
                    "Derived at query time from the charge period start in the tenant time zone, without proration.",
                    "조회 시점에 과금 기간 시작 시각을 테넌트 시간대로 바꿔 산출하며 기간 배분은 하지 않습니다.",
                    "يُشتق عند الاستعلام من بداية فترة الرسوم بحسب المنطقة الزمنية للمستأجر، دون توزيع زمني.",
                  )}
                </small>
              </dd>
              {basis.utcMonth && (
                <>
                  <dt>
                    {t(
                      l,
                      "UTC calendar month of the charge period start",
                      "과금 기간 시작의 UTC 기준 월",
                      "الشهر التقويمي بتوقيت UTC لبداية فترة الرسوم",
                    )}
                  </dt>
                  <dd data-field="utc_month">
                    <bdi dir="ltr">{basis.utcMonth}</bdi>
                    {basis.differs && (
                      <small>
                        {t(
                          l,
                          "The charge-start month differs because it uses the tenant time zone.",
                          "과금 시작 월은 테넌트 시간대를 사용하므로 UTC 기준 월과 다릅니다.",
                          "يختلف شهر بدء الرسوم لأنه يستخدم المنطقة الزمنية للمستأجر.",
                        )}
                      </small>
                    )}
                  </dd>
                </>
              )}
              <dt>
                {t(
                  l,
                  "Tenant time zone",
                  "테넌트 시간대",
                  "منطقة المستأجر الزمنية",
                )}
              </dt>
              <dd data-field="timezone">
                {basis.rowTimezone ? (
                  <bdi dir="ltr">{basis.rowTimezone}</bdi>
                ) : (
                  <>
                    {t(
                      l,
                      "The row record does not carry its tenant.",
                      "행 기록에는 테넌트가 포함되어 있지 않습니다.",
                      "سجل الصف لا يتضمن مستأجره.",
                    )}
                    {basis.timezones.length > 0 && (
                      <small>
                        {t(
                          l,
                          "Time zones in your authorized scope:",
                          "전체 허용 범위의 시간대:",
                          "المناطق الزمنية ضمن نطاقك المسموح:",
                        )}{" "}
                        <bdi dir="ltr">
                          {basis.timezones.slice(0, shownTimezones).join(", ")}
                          {basis.timezones.length > shownTimezones
                            ? ` +${basis.timezones.length - shownTimezones}`
                            : ""}
                        </bdi>
                      </small>
                    )}
                  </>
                )}
              </dd>
              <dt>{fieldLabel(l, "id")}</dt>
              <dd>
                <bdi dir="ltr" className="collected-code collected-token-pair">
                  {row.id.slice(0, 32)}
                  <wbr />
                  {row.id.slice(32)}
                </bdi>
              </dd>
              <dt>{fieldLabel(l, "account")}</dt>
              <dd>
                <bdi dir="ltr" className="collected-code">
                  {row.account.slice(0, 12)}
                </bdi>
              </dd>
            </dl>
          </section>
          <section data-origin="lineage">
            <h3>{originLabel(l, "lineage")}</h3>
            <dl>
              <dt>{fieldLabel(l, "batch")}</dt>
              <dd>
                <bdi dir="ltr" className="collected-code">
                  {row.batch}
                </bdi>
              </dd>
              <dt>{fieldLabel(l, "ordinal")}</dt>
              <dd>
                <bdi dir="ltr">{row.ordinal}</bdi>
              </dd>
            </dl>
            <h3>{originLabel(l, "annotation")}</h3>
            <dl>
              <dt>
                {fieldLabel(l, "evidence_class")}{" "}
                <bdi dir="ltr" className="collected-code">
                  {annotationColumn}
                </bdi>
              </dt>
              <dd data-field="evidence_class">
                {provenanceLabel(l, provenanceClass(row.evidence_class))}
                {provenanceClass(row.evidence_class) !== "unclassified" && (
                  <small>
                    <bdi dir="ltr" className="collected-code">
                      {row.evidence_class}
                    </bdi>
                  </small>
                )}
              </dd>
            </dl>
          </section>
          <section
            className="collected-unsupplied"
            data-item="S04:state:evidence-unavailable"
          >
            <h3>
              {t(
                l,
                "Not carried by the row record",
                "행 기록에 없는 정보",
                "غير متضمَّن في سجل الصف",
              )}
            </h3>
            <p>
              {t(
                l,
                "The row record in this query does not include the following, so the screen does not show or infer them.",
                "이 조회의 행 기록에는 다음 정보가 없으므로 화면에 표시하거나 추정하지 않습니다.",
                "سجل الصف في هذا الاستعلام لا يتضمن ما يلي، لذا لا تعرضه الشاشة ولا تستنتجه.",
              )}
            </p>
            <ul>
              {unsupplied.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </section>
        </div>
      </div>
      <footer>{close}</footer>
    </section>
  );
}
