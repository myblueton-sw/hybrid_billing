import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { reviewMoney as money } from "./values";
import type { Prepared } from "./model";
import { basisLabel, totalLabels } from "./copy";
export function SourceEvidence({
  locale: l,
  source,
}: {
  locale: Locale;
  source: Prepared;
}) {
  const d = source.data;
  return (
    <details className="allocation-evidence">
      <summary>
        {t(
          l,
          "Source and calculation evidence",
          "원천·계산 근거",
          "أدلة المصدر والحساب",
        )}
      </summary>
      <p>
        {t(
          l,
          "Review recipients are tenant access scopes returned under current authorization. They do not establish legal entities, departments or customer-contract ownership.",
          "검토 대상은 현재 권한으로 조회한 테넌트 접근 범위입니다. 법인·부서·고객 계약의 소유 관계를 뜻하지 않습니다.",
          "مستلمو المراجعة نطاقات وصول المستأجرين المُعادة بموجب التخويل الحالي. ولا تثبت كيانات قانونية أو أقساماً أو ملكية عقود العملاء.",
        )}
      </p>
      <p>
        {t(
          l,
          "Review calculations truncate toward zero at source precision (18 decimal places). This is not currency-specific accounting rounding or an approved allocation policy. Signed source credits are preserved. Precision remainders are not redistributed. Idle cost is unknown.",
          "검토용 계산은 원천 정밀도인 소수점 18자리에서 0 방향으로 절삭합니다. 통화별 회계 반올림이나 승인된 배분 정책을 적용한 결과가 아닙니다. 원천의 음수 크레딧을 유지합니다. 계산 정밀도 잔액을 자동 배분하지 않으며 유휴 원가는 확인 불가입니다.",
          "تُقتطع حسابات المراجعة نحو الصفر بدقة المصدر (18 خانة عشرية). هذا ليس تقريباً محاسبياً خاصاً بالعملة ولا سياسة توزيع معتمدة. تُحفظ الأرصدة السالبة للمصدر. لا يُعاد توزيع متبقي الدقة، وتكلفة الموارد الخاملة غير معروفة.",
        )}
      </p>
      <p>
        {t(
          l,
          "The month follows charge start in the tenant timezone without proration. Source coverage declarations are unverified; finalized collection is not financial closing.",
          "과금 시작 월은 테넌트 시간대로 구분하며 기간 안분하지 않습니다. 수집 범위 선언은 미검증이며 수집 확정은 재무 마감이 아닙니다.",
          "يُحدّد الشهر ببدء الرسوم في المنطقة الزمنية للمستأجر دون توزيع زمني. إقرارات تغطية المصدر غير متحقّقة، ونهائية الجمع ليست إقفالاً مالياً.",
        )}
      </p>
      <dl>
        <dt>
          {t(
            l,
            "Selection reference ID",
            "조회 기준 ID",
            "معرّف مرجع الاستعلام",
          )}
        </dt>
        <dd>
          <bdi dir="ltr">{d.snapshot}</bdi>
        </dd>
        <dt>
          {t(
            l,
            "Source data reference ID",
            "수집 데이터 기준 ID",
            "معرّف مرجع بيانات المصدر",
          )}
        </dt>
        <dd>
          <bdi dir="ltr">{d.source_snapshot}</bdi>
        </dd>
        <dt>
          {t(
            l,
            "Review context reference",
            "검토 범위 참조",
            "مرجع نطاق المراجعة",
          )}
        </dt>
        <dd>
          <bdi dir="ltr">{source.context}</bdi>
        </dd>
        <dt>{t(l, "Observed at", "조회 시각", "وقت الاستعلام")}</dt>
        <dd>
          <bdi dir="ltr">{d.observed_at}</bdi>
        </dd>
        <dt>{t(l, "Source finalization", "수집 확정", "نهائية جمع المصدر")}</dt>
        <dd>
          <bdi dir="ltr">
            {d.coverage.finalized_slices} / {d.coverage.slices}
          </bdi>
        </dd>
      </dl>
    </details>
  );
}
export default function SourceBasis({
  locale: l,
  source,
}: {
  locale: Locale;
  source: Prepared;
}) {
  const d = source.data;
  return (
    <section>
      <h2>{t(l, "Allocation basis", "배분 기준액", "مبلغ أساس التوزيع")}</h2>
      <p className="allocation-meta">
        {t(
          l,
          "Full server totals for this provider and charge-start month. Detail pages and recipient searches never change these amounts.",
          "공급사·과금 시작 월에 해당하는 서버 전체 집계입니다. 상세 페이지나 대상 검색으로 기준액을 바꾸지 않습니다.",
          "إجماليات الخادم الكاملة لهذا المزوّد وشهر بدء الرسوم. لا تغيّر صفحات التفاصيل أو عمليات البحث عن المستلمين هذه المبالغ.",
        )}
      </p>
      <div
        className="allocation-table"
        tabIndex={0}
        role="region"
        aria-label={t(
          l,
          "Source amount table",
          "원천 기준액 표",
          "جدول مبالغ المصدر",
        )}
      >
        <table>
          <thead>
            <tr>
              <th>{t(l, "Currency", "통화", "العملة")}</th>
              <th>{t(l, "Cost basis", "비용 기준", "أساس التكلفة")}</th>
              <th className="allocation-money">{totalLabels(l).source}</th>
              <th>{t(l, "Source rows", "원천 행", "صفوف المصدر")}</th>
              <th>
                {t(
                  l,
                  "Missing amount rows",
                  "금액 누락 행",
                  "صفوف بمبالغ مفقودة",
                )}
              </th>
              <th>{t(l, "Selection", "선택", "الاختيار")}</th>
            </tr>
          </thead>
          <tbody>
            {d.amounts.flatMap((a) =>
              (["billed_cost", "effective_cost"] as const).map((b) => (
                <tr
                  key={a.currency + b}
                  data-selected={
                    a.currency === source.currency && b === d.selection.basis
                  }
                >
                  <td data-label={t(l, "Currency", "통화", "العملة")}>
                    <bdi dir="ltr">{a.currency}</bdi>
                  </td>
                  <td
                    data-label={t(l, "Cost basis", "비용 기준", "أساس التكلفة")}
                  >
                    {basisLabel(l, b)}
                  </td>
                  <td
                    className="allocation-money"
                    data-label={totalLabels(l).source}
                  >
                    <bdi dir="ltr">
                      {a[b].coefficient === null
                        ? t(l, "Unknown", "알 수 없음", "غير معروف")
                        : money(a[b].coefficient, a.currency, l)}
                    </bdi>
                    {a[b].missing > 0 && (
                      <small>
                        {t(
                          l,
                          "Known subtotal only",
                          "확인된 금액의 부분 합계",
                          "مجموع فرعي للمبالغ المعلومة فقط",
                        )}
                      </small>
                    )}
                  </td>
                  <td
                    data-label={t(l, "Source rows", "원천 행", "صفوف المصدر")}
                  >
                    {a.rows}
                  </td>
                  <td
                    data-label={t(
                      l,
                      "Missing amount rows",
                      "금액 누락 행",
                      "صفوف بمبالغ مفقودة",
                    )}
                  >
                    {a[b].missing}
                  </td>
                  <td data-label={t(l, "Selection", "선택", "الاختيار")}>
                    {a.currency === source.currency && b === d.selection.basis
                      ? t(l, "Current basis", "현재 기준액", "الأساس الحالي")
                      : t(
                          l,
                          "Available source basis",
                          "조회된 원천 기준액",
                          "أساس مصدر متاح",
                        )}
                  </td>
                </tr>
              )),
            )}
          </tbody>
        </table>
      </div>
      {!d.amounts.length && (
        <p>
          {t(
            l,
            "No source amounts are available for this scope.",
            "선택 범위에 수집된 금액이 없습니다.",
            "لا تتوفر مبالغ مصدر لهذا النطاق.",
          )}
        </p>
      )}
      <p className="allocation-meta">
        {t(
          l,
          "Review recipient access scopes",
          "검토 대상 접근 범위",
          "نطاقات وصول مستلمي المراجعة",
        )}
        : {source.targets.length}
      </p>
      <SourceEvidence locale={l} source={source} />
    </section>
  );
}
