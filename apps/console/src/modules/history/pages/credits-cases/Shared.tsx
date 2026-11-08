import type { Locale } from "../../../../shared/types";
import type { Dimension, Money, Hold } from "./contract";
import { t, unknown, synthetic, reported } from "./copy";
export function SourceValue({
  locale: l,
  value,
  direction = "ltr",
}: {
  locale: Locale;
  value: string | null;
  direction?: "auto" | "ltr";
}) {
  return value === null ? (
    <span className="b11-missing">{unknown(l)}</span>
  ) : (
    <bdi dir={direction}>{value}</bdi>
  );
}
export function Facts({
  locale: l,
  rows,
}: {
  locale: Locale;
  rows: readonly (readonly [string, string | null, ("auto" | "ltr")?])[];
}) {
  return (
    <dl className="b11-facts">
      {rows.map(([label, value, direction]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>
            <SourceValue locale={l} value={value} direction={direction} />
          </dd>
        </div>
      ))}
    </dl>
  );
}
export function Measure({
  locale: l,
  value,
  dimension,
  label,
}: {
  locale: Locale;
  value: string | null;
  dimension: Dimension;
  label?: string;
}) {
  return value === null ? (
    <span className="b11-missing">{unknown(l)}</span>
  ) : (
    <span
      className="b11-money-scroll"
      role="region"
      tabIndex={0}
      aria-label={
        label ??
        t(l, "Exact source value", "정확한 원천 값", "قيمة المصدر الدقيقة")
      }
    >
      <bdi dir="ltr" className="b11-money">
        {dimension.kind === "money"
          ? `${dimension.currency} ${value}`
          : `${value} ${dimension.unit}`}
      </bdi>
    </span>
  );
}
export function MoneyValues({
  locale: l,
  amounts,
  label,
}: {
  locale: Locale;
  amounts: Money[];
  label?: string;
}) {
  return amounts.length ? (
    <div className="b11-money-values">
      {amounts.map((a) => (
        <Measure
          key={a.currency}
          locale={l}
          value={a.value}
          dimension={{ kind: "money", currency: a.currency, unit: null }}
          label={label ? `${label} ${a.currency}` : undefined}
        />
      ))}
    </div>
  ) : (
    <p>
      {t(
        l,
        "No amount observations were supplied. This does not mean zero.",
        "금액 관측값이 제공되지 않았습니다. 0이라는 뜻은 아닙니다.",
        "لم تُقدّم قيم مبالغ. هذا لا يعني صفراً.",
      )}
    </p>
  );
}
export function SyntheticNotice({ locale: l }: { locale: Locale }) {
  return <p className="b11-synthetic">{synthetic(l)}</p>;
}
export function Holds({ locale: l, holds }: { locale: Locale; holds: Hold[] }) {
  return (
    <section className="b11-holds">
      <h3>
        {t(
          l,
          "Related holds — separate from dates and balances",
          "관련 보류 · 지급 기한·잔액과 별도",
          "إجراءات التعليق المرتبطة منفصلة عن المواعيد والأرصدة",
        )}
      </h3>
      <p>
        {t(
          l,
          "Requested is not effective. Unknown is not released. Holds and inquiries do not change due dates, payment obligations or balances; release does not automatically resume work.",
          "요청은 효력 발생이 아니며 미확인은 해제가 아닙니다. 보류·문의가 지급 기한·납부 의무·잔액을 변경하지 않습니다. 해제해도 작업이 자동 재개되지 않습니다.",
          "الطلب ليس أثراً نافذاً والمجهول ليس تحريراً. إجراءات التعليق والاستفسارات لا تغيّر مواعيد السداد أو الالتزامات أو الأرصدة. التحرير لا يستأنف العمل تلقائياً.",
        )}
      </p>
      {holds.length ? (
        holds.map((h, i) => (
          <article key={h.reference ?? i} className="b11-hold-record">
            <h4>{reported(l, h.type)}</h4>
            <p>{reported(l, h.effect)}</p>
            <Facts
              locale={l}
              rows={[
                [
                  t(l, "Hold reference", "보류 참조", "مرجع إجراء التعليق"),
                  h.reference,
                ],
                [
                  t(l, "Hold revision", "보류 개정", "مراجعة إجراء التعليق"),
                  h.revision,
                ],
                [
                  t(l, "Target reference", "대상 참조", "مرجع الهدف"),
                  h.target_ref,
                ],
                [
                  t(l, "Observation time", "관측 시각", "وقت الرصد"),
                  h.observed_at,
                ],
              ]}
            />
            <p>
              {t(l, "External effect", "외부 효력", "الأثر الخارجي")}:{" "}
              {reported(l, h.external_effect)}
            </p>
          </article>
        ))
      ) : (
        <p>
          {t(
            l,
            "No hold observations supplied; this does not prove there are no holds.",
            "보류 관측값을 제공하지 않았습니다. 보류가 없다는 증거는 아닙니다.",
            "لم تُقدّم معلومات عن إجراءات التعليق؛ لا يثبت هذا عدم وجودها.",
          )}
        </p>
      )}
    </section>
  );
}
