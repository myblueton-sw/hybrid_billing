import type { Locale } from "../../../../shared/types";
import type { Draft, Prepared, Result, Adjustment } from "./model";
import { t, notCalculated } from "./copy";
import { money } from "./values";
export default function Adjustments({
  locale: l,
  draft,
  source,
  result,
  adjustment,
  update,
  run,
  back,
}: {
  locale: Locale;
  draft: Draft;
  source: Prepared;
  result?: Result;
  adjustment?: Adjustment;
  update: (d: Draft) => void;
  run: () => void;
  back: () => void;
}) {
  return (
    <div className="pricing-workspace">
      <section>
        <h2>{t(l, "Adjustment review", "조정 검토", "مراجعة التعديل")}</h2>
        <div className="pricing-fields pricing-single">
          {(["target", "reason", "reference"] as const).map((k) => (
            <label key={k} htmlFor={`pricing-${k}`}>
              {k === "target"
                ? t(
                    l,
                    "Target review amount",
                    "목표 검토 금액",
                    "مبلغ المراجعة المستهدف",
                  )
                : k === "reason"
                  ? t(l, "Adjustment reason", "조정 사유", "سبب التعديل")
                  : t(l, "Evidence reference", "증빙 참조", "مرجع الأدلة")}
              {k === "reason" ? (
                <textarea
                  id={`pricing-${k}`}
                  value={draft[k]}
                  maxLength={300}
                  onChange={(e) => update({ ...draft, [k]: e.target.value })}
                />
              ) : (
                <span className={k === "target" ? "pricing-input" : undefined}>
                  <input
                    id={`pricing-${k}`}
                    dir={k === "target" ? "ltr" : undefined}
                    inputMode={k === "target" ? "decimal" : "text"}
                    value={draft[k]}
                    maxLength={k === "target" ? 40 : 300}
                    onChange={(e) => update({ ...draft, [k]: e.target.value })}
                  />
                  {k === "target" && <bdi dir="ltr">{source.row.currency}</bdi>}
                </span>
              )}
            </label>
          ))}
        </div>
        <button id="pricing-adjust" className="pricing-primary" onClick={run}>
          {t(l, "Preview difference", "차액 미리보기", "معاينة الفرق")}
        </button>
        {!result && (
          <p>
            {t(
              l,
              "A candidate preview is required.",
              "후보 금액이 없어 차액을 계산할 수 없습니다.",
              "تلزم معاينة مبلغ مرشح.",
            )}{" "}
            <button onClick={back}>
              {t(
                l,
                "Open pricing rules",
                "요금 규칙으로 이동",
                "فتح قواعد التسعير",
              )}
            </button>
          </p>
        )}
      </section>
      <aside className="pricing-rail">
        <h2>{t(l, "Review difference", "검토 차액", "فرق المراجعة")}</h2>
        <dl className="pricing-summary">
          {(["candidate", "target", "delta"] as const).map((k) => (
            <div key={k}>
              <dt>
                {k === "candidate"
                  ? t(
                      l,
                      "Candidate — unchanged",
                      "후보 금액 — 변경 없음",
                      "المبلغ المرشح — دون تغيير",
                    )
                  : k === "target"
                    ? t(
                        l,
                        "Target review amount",
                        "목표 검토 금액",
                        "مبلغ المراجعة المستهدف",
                      )
                    : t(
                        l,
                        "Target − candidate",
                        "목표 − 후보",
                        "المستهدف − المرشح",
                      )}
              </dt>
              <dd>
                <bdi dir="ltr" data-pricing-adjustment={k}>
                  {adjustment
                    ? money(
                        adjustment[k],
                        source.row.currency,
                        l,
                        k === "delta",
                      )
                    : notCalculated(l)}
                </bdi>
                {k === "delta" && adjustment && (
                  <small>
                    {BigInt(adjustment.delta) > 0n
                      ? t(l, "Increase", "증가", "زيادة")
                      : BigInt(adjustment.delta) < 0n
                        ? t(l, "Decrease", "감소", "انخفاض")
                        : t(l, "No change", "변동 없음", "دون تغيير")}
                  </small>
                )}
              </dd>
            </div>
          ))}
        </dl>
        <p className="pricing-meta">
          {t(
            l,
            "Local arithmetic review before FX and tax. No invoice application, issuance or approval is connected.",
            "환산·세금 적용 전 로컬 산술 검토입니다. 청구서 적용·발행·승인은 미연결입니다.",
            "مراجعة حسابية محلية قبل تحويل العملة والضريبة. تطبيق الفواتير وإصدارها واعتمادها غير متصل.",
          )}
        </p>
      </aside>
    </div>
  );
}
