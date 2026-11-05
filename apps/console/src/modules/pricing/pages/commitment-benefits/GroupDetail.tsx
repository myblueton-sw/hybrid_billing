import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import {
  fieldNames,
  type Group,
  type Detail,
  type Charge,
  type Field,
} from "./contract";
import {
  t,
  costLabel,
  statusLabel,
  fieldLabel,
  stateLabel,
  noValue,
} from "./copy";
import { MeasureValue, MetadataValue } from "./GroupTable";
import { money, range, rowStatus } from "./values";
function FieldValue({ locale: l, field: f }: { locale: Locale; field: Field }) {
  return f.state === "value" ? (
    <bdi dir="ltr">{f.value}</bdi>
  ) : (
    <>
      <span>{stateLabel(l, f.state)}</span>
      {f.state === "unmapped_version" && f.value !== null && (
        <small>
          {t(l, "Unmapped raw value", "미매핑 원천 값", "قيمة خام غير مربوطة")}:{" "}
          <bdi dir="ltr">{f.value}</bdi>
        </small>
      )}
    </>
  );
}
function Intervals({
  locale: l,
  value: v,
}: {
  locale: Locale;
  value: Group["intervals"] | Charge;
}) {
  return (
    <dl className="b06-evidence">
      {(
        [
          [
            "charge_start",
            "charge_end",
            t(
              l,
              "Source charge interval envelope",
              "원천 과금 기간 범위",
              "فترة رسوم المصدر",
            ),
          ],
          [
            "billing_start",
            "billing_end",
            t(
              l,
              "Source billing interval envelope",
              "원천 청구 기간 범위",
              "فترة فوترة المصدر",
            ),
          ],
        ] as const
      ).map(([start, end, label]) => (
        <div key={start}>
          <dt>{label}</dt>
          <dd>
            <bdi dir="ltr" className="b06-date">
              {v[start]}
            </bdi>
            {" — "}
            <bdi dir="ltr" className="b06-date">
              {v[end]}
            </bdi>
          </dd>
        </div>
      ))}
    </dl>
  );
}
export default function GroupDetail({
  locale: l,
  group: g,
  detail: d,
  view,
  close,
  page,
}: {
  locale: Locale;
  group: Group;
  detail: Detail;
  view: string;
  close: () => void;
  page: (n: number) => void;
}) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    heading.current?.focus();
  }, [g.id, d.after]);
  return (
    <section
      className="b06-detail"
      data-b06-detail="true"
      onKeyDown={(e) => {
        if (
          e.key !== "Escape" ||
          e.defaultPrevented ||
          !(e.target instanceof HTMLElement)
        )
          return;
        const disclosure =
          e.target.closest<HTMLDetailsElement>("details[open]");
        if (disclosure) {
          disclosure.open = false;
          disclosure.querySelector("summary")?.focus();
        } else close();
        e.preventDefault();
        e.stopPropagation();
      }}
    >
      <div className="b06-detail-heading">
        <h2 ref={heading} tabIndex={-1}>
          {t(
            l,
            "Selected group evidence",
            "선택 그룹 근거",
            "أدلة المجموعة المحددة",
          )}
        </h2>
        <button id="b06-close" onClick={close}>
          {t(l, "Close detail", "상세 닫기", "إغلاق التفاصيل")}
        </button>
      </div>
      <p className="b06-source-id">
        <bdi dir="ltr">{g.source_id}</bdi>
      </p>
      <div className="b06-selected-costs">
        {(["billed_cost", "effective_cost"] as const).map((k) => (
          <dl key={k}>
            <dt>{costLabel(l, k)}</dt>
            <dd>
              <MeasureValue
                locale={l}
                value={g[k]}
                currency={g.currency}
                rows={g.rows}
                hook={k}
              />
            </dd>
          </dl>
        ))}
      </div>
      {view === "utilization" && (
        <>
          <h3>
            {t(
              l,
              "Source status partition",
              "원천 상태별 분류",
              "تقسيم حالات المصدر",
            )}
          </h3>
          <div
            className="b06-scroll b06-status-table"
            role="region"
            tabIndex={0}
            aria-label={t(
              l,
              "Source classifications and separate cost bases",
              "원천 분류 및 비용 기준별 금액",
              "تصنيفات المصدر وأسس التكلفة المنفصلة",
            )}
          >
            <table>
              <thead>
                <tr>
                  <th>
                    {t(l, "Source classification", "원천 분류", "تصنيف المصدر")}
                  </th>
                  <th>{t(l, "Rows", "행", "صفوف")}</th>
                  <th>{costLabel(l, "billed_cost")}</th>
                  <th>{costLabel(l, "effective_cost")}</th>
                </tr>
              </thead>
              <tbody>
                {(["used", "unused", "unknown"] as const).map((s) => (
                  <tr
                    key={s}
                    data-b06-status={s}
                    data-b06-status-rows={g.statuses[s].rows}
                  >
                    <th scope="row">{statusLabel(l, s)}</th>
                    <td data-label={t(l, "Rows", "행", "صفوف")}>
                      <bdi dir="ltr">{g.statuses[s].rows}</bdi>
                    </td>
                    {(["billed_cost", "effective_cost"] as const).map((k) => (
                      <td key={k} data-label={costLabel(l, k)}>
                        <MeasureValue
                          locale={l}
                          value={g.statuses[s][k]}
                          rows={g.statuses[s].rows}
                          currency={g.currency}
                        />
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="b06-meta">
            {t(
              l,
              "Source charge classifications, not VM utilization or commitment lifecycle status. Only the reviewed FOCUS 1.0 Used/Unused mapping is classified; other versions and values stay unknown.",
              "원천 청구 분류이며 VM 사용률·약정 활성 상태가 아닙니다. 검토된 FOCUS 1.0 Used/Unused 매핑만 분류하며 다른 버전·값은 미확인으로 유지합니다.",
              "تصنيفات رسوم المصدر وليست استخدام الأجهزة أو دورة حياة الالتزام. يُصنف ربط Used/Unused المراجع لإصدار FOCUS 1.0 فقط؛ تظل الإصدارات والقيم الأخرى غير معروفة.",
            )}
          </p>
          {g.statuses.unused.rows === 0 && (
            <p>
              {t(
                l,
                "No Unused-classified rows were observed. This does not mean unused cost is zero.",
                "미사용 분류 행이 관측되지 않았습니다. 미사용 비용 0을 뜻하지 않습니다.",
                "لم تُرصد صفوف مصنفة غير مستخدمة. لا يعني ذلك أن التكلفة غير المستخدمة تساوي صفراً.",
              )}
            </p>
          )}
          <p className="b06-meta">
            {t(
              l,
              "Source status values — lexical evidence only",
              "원천 상태 값 — 문자 근거이며 분류 판정 아님",
              "قيم حالة المصدر — أدلة نصية فقط",
            )}
          </p>
          <MetadataValue locale={l} value={g.metadata.status} />
        </>
      )}
      {view === "benefits" && (
        <>
          <h3>
            {t(
              l,
              "Benefit authority boundaries",
              "혜택 흐름의 권한 경계",
              "حدود صلاحيات تدفق المنافع",
            )}
          </h3>
          <ol className="b06-benefit-flow">
            <li>
              <h3>
                {t(
                  l,
                  "Provider source evidence",
                  "공급자 원천 근거",
                  "أدلة مصدر المزوّد",
                )}
              </h3>
              <dl className="b06-evidence">
                {(["type", "category", "status"] as const).map((k) => (
                  <div key={k}>
                    <dt>{fieldLabel(l, k)}</dt>
                    <dd>
                      <MetadataValue locale={l} value={g.metadata[k]} />
                    </dd>
                  </div>
                ))}
              </dl>
            </li>
            <li>
              <h3>
                {t(
                  l,
                  "Internal shared allocation",
                  "내부 공유 배분",
                  "التوزيع الداخلي المشترك",
                )}
              </h3>
              <p>
                {t(
                  l,
                  "Unconnected — no authorized B03 allocation evidence.",
                  "미연결 — 권한 있는 B03 배분 근거 없음.",
                  "غير متصل — لا توجد أدلة توزيع مخولة من B03.",
                )}
              </p>
            </li>
            <li>
              <h3>
                {t(
                  l,
                  "Customer selling discount and pass-through",
                  "고객 판매 할인·혜택 전달",
                  "خصم البيع للعميل وتمرير المنافع",
                )}
              </h3>
              <p>
                {t(
                  l,
                  "Unconnected — no authorized B04 selling-policy evidence.",
                  "미연결 — 권한 있는 B04 판매 정책 근거 없음.",
                  "غير متصل — لا توجد أدلة سياسة بيع مخولة من B04.",
                )}
              </p>
            </li>
          </ol>
          <p className="b06-meta">
            {t(
              l,
              "Unused-cost bearer, legal ownership, sales-contract prepayment recovery and prepayment ledger are unknown and unconnected. Source costs do not establish a customer discount.",
              "미사용 비용 부담 주체·법적 소유 관계·판매 계약 선납 회수·선납 원장은 미확인·미연결입니다. 원천 비용으로 고객 할인을 판단하지 않습니다.",
              "جهة تحمل التكلفة غير المستخدمة والملكية القانونية واسترداد الدفعات المقدمة بعقد البيع ودفترها غير معروفة وغير متصلة. تكاليف المصدر لا تثبت خصماً للعميل.",
            )}
          </p>
        </>
      )}
      <details className="b06-identity" open={view === "commitments"}>
        <summary>
          {t(
            l,
            "Group identity and source metadata",
            "그룹 식별·원천 메타데이터",
            "هوية المجموعة وبيانات المصدر",
          )}
        </summary>
        <dl className="b06-evidence">
          {[
            [t(l, "Provider", "공급자", "المزوّد"), g.provider],
            [
              t(l, "Billing account", "청구 계정", "حساب الفوترة"),
              g.billing_account,
            ],
            [
              t(l, "Sub-account", "하위 계정", "الحساب الفرعي"),
              g.sub_account ?? noValue(l),
            ],
            [
              t(
                l,
                "Canonical account reference",
                "원천 계정 참조",
                "مرجع الحساب",
              ),
              g.account,
            ],
            [
              t(
                l,
                "Authorized tenant scope reference",
                "허용 테넌트 범위 참조",
                "مرجع نطاق المستأجر المصرّح به",
              ),
              g.tenant,
            ],
            [t(l, "Group reference", "그룹 참조", "مرجع المجموعة"), g.id],
            [t(l, "Currency", "통화", "العملة"), g.currency],
          ].map(([k, v]) => (
            <div key={k}>
              <dt>{k}</dt>
              <dd>
                <bdi dir="ltr">{v}</bdi>
              </dd>
            </div>
          ))}
        </dl>
        <Intervals locale={l} value={g.intervals} />
        <p className="b06-meta">
          {t(
            l,
            "Source interval envelopes are earliest start through latest end, not continuous coverage or the commitment term. Start inclusive, end exclusive. Account references do not establish legal ownership.",
            "원천 기간 범위는 최초 시작부터 최종 종료까지이며 연속 완전성·약정 전체 기간을 뜻하지 않습니다. 시작 포함·끝 제외입니다. 계정 참조는 법적 소유 관계가 아닙니다.",
            "نطاقات فترات المصدر تمتد من أول بداية إلى آخر نهاية ولا تثبت تغطية مستمرة أو مدة الالتزام. تشمل البداية وتستثني النهاية. مراجع الحسابات لا تثبت ملكية قانونية.",
          )}
        </p>
        <dl className="b06-evidence">
          {fieldNames
            .filter((k) => k !== "id")
            .map((k) => (
              <div key={k}>
                <dt>{fieldLabel(l, k)}</dt>
                <dd>
                  <MetadataValue
                    locale={l}
                    value={g.metadata[k as Exclude<typeof k, "id">]}
                  />
                </dd>
              </div>
            ))}
        </dl>
        <p className="b06-meta">
          {t(
            l,
            "Different values are source variation, not an automatic conflict. Used and Unused rows may coexist. Missing-field states are retained per charge row below.",
            "서로 다른 값은 원천의 차이이며 자동 충돌 판정이 아닙니다. 사용·미사용 분류 행은 함께 존재할 수 있습니다. 누락 필드 상태는 아래 행별 근거에서 확인하세요.",
            "القيم المختلفة تنوع في المصدر وليست تعارضاً تلقائياً. قد تتواجد صفوف مستخدمة وغير مستخدمة معاً. تُحفظ حالات الحقول المفقودة لكل صف أدناه.",
          )}
        </p>
      </details>
      {view === "reconciliation" && (
        <section>
          <h3>
            {t(
              l,
              "Group mapping evidence",
              "그룹 매핑 근거",
              "أدلة ربط المجموعة",
            )}
          </h3>
          <p>
            {t(
              l,
              "Version-mapped / unmapped rows",
              "버전 매핑 / 미매핑 행",
              "صفوف بإصدار مربوط / غير مربوط",
            )}
            :{" "}
            <bdi dir="ltr">
              {g.mapping.version_mapped_rows} /{" "}
              {g.mapping.version_unmapped_rows}
            </bdi>
          </p>
          <p>
            {t(l, "Source versions", "원천 버전", "إصدارات المصدر")}:{" "}
            {g.versions.map((v) => (
              <bdi key={v} dir="ltr">
                {v}{" "}
              </bdi>
            ))}
          </p>
          <p className="b06-meta">
            {t(
              l,
              "Evidence review is not financial reconciliation complete. Fee, negation and amortization conservation across the full commitment term remain unverified.",
              "근거 검토는 재무 대사 완료가 아닙니다. 약정 전체 기간의 수수료·상계·상각 보존은 미검증입니다.",
              "مراجعة الأدلة ليست إكمالاً للتسوية المالية. لم يُتحقق من حفظ الرسوم والتعويضات والإطفاء طوال مدة الالتزام.",
            )}
          </p>
        </section>
      )}
      <h3>
        {t(l, "Source charge records", "원천 청구 행", "سجلات رسوم المصدر")}
      </h3>
      <p className="b06-meta">
        <bdi dir="ltr">{range(d.after, d.rows.length, d.total_rows)}</bdi> ·{" "}
        {t(
          l,
          "Up to 25 rows per page",
          "페이지당 최대 25개",
          "حتى 25 صفاً لكل صفحة",
        )}
      </p>
      <div className="b06-charge-records">
        {d.rows.map((r) => (
          <article key={r.id}>
            <h4>
              {r.service ??
                t(
                  l,
                  "Service unavailable",
                  "서비스명 없음",
                  "الخدمة غير متاحة",
                )}
            </h4>
            <p className="b06-meta">
              {statusLabel(l, rowStatus(r))} ·{" "}
              {t(l, "Source version", "원천 버전", "إصدار المصدر")}{" "}
              <bdi dir="ltr">{r.version}</bdi>
            </p>
            <div className="b06-charge-costs">
              {(["billed_cost", "effective_cost"] as const).map((k) => (
                <dl key={k}>
                  <dt>{costLabel(l, k)}</dt>
                  <dd>
                    <bdi dir="ltr">
                      {r[k] === null ? noValue(l) : money(r[k]!, r.currency, l)}
                    </bdi>
                  </dd>
                </dl>
              ))}
            </div>
            <details>
              <summary>
                {t(
                  l,
                  "Charge fields and delivery evidence",
                  "청구 필드·수집 근거",
                  "حقول الرسوم وأدلة التسليم",
                )}
              </summary>
              <dl className="b06-evidence">
                {fieldNames.map((k) => (
                  <div key={k}>
                    <dt>{fieldLabel(l, k)}</dt>
                    <dd>
                      <FieldValue locale={l} field={r.fields[k]} />
                    </dd>
                  </div>
                ))}
                {[
                  [
                    t(
                      l,
                      "Canonical row reference",
                      "원천 행 참조",
                      "مرجع الصف",
                    ),
                    r.id,
                  ],
                  [
                    t(
                      l,
                      "Evidence classification",
                      "근거 분류",
                      "تصنيف الأدلة",
                    ),
                    r.evidence_class,
                  ],
                  [
                    t(l, "Delivery batch", "수집 배치", "دفعة التسليم"),
                    r.batch,
                  ],
                  [
                    t(
                      l,
                      "Source row ordinal",
                      "원천 행 순번",
                      "الترتيب في المصدر",
                    ),
                    r.ordinal,
                  ],
                ].map(([k, v]) => (
                  <div key={k}>
                    <dt>{k}</dt>
                    <dd>
                      <bdi dir="ltr">{v}</bdi>
                    </dd>
                  </div>
                ))}
              </dl>
              <Intervals locale={l} value={r} />
            </details>
          </article>
        ))}
      </div>
      <div className="b06-pager">
        <button
          id="b06-row-prev"
          disabled={d.after === 0}
          onClick={() => page(Math.max(0, d.after - 25))}
        >
          {t(l, "Previous records", "이전 원천 행", "السجلات السابقة")}
        </button>
        <bdi dir="ltr">{range(d.after, d.rows.length, d.total_rows)}</bdi>
        <button
          id="b06-row-next"
          disabled={d.next_after === null}
          onClick={() => d.next_after !== null && page(d.next_after)}
        >
          {t(l, "Next records", "다음 원천 행", "السجلات التالية")}
        </button>
      </div>
    </section>
  );
}
