import type { Locale } from "../../../../shared/types";
import type { ClaimRecord } from "./contract";
import {
  t,
  unknown,
  synthetic,
  protectionLabel,
  basisLabel,
  dispositionLabel,
  externalKind,
  outcomeLabel,
} from "./copy";
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
    <span>{unknown(l)}</span>
  ) : (
    <bdi dir={direction}>{value}</bdi>
  );
}
export function CoverageValue({
  locale: l,
  value,
  unit,
}: {
  locale: Locale;
  value: string | null;
  unit: string | null;
}) {
  return value === null ? (
    <span>{unknown(l)}</span>
  ) : (
    <>
      <bdi dir="ltr" className="b10-coverage-value">
        {value}
        {unit !== null ? ` ${unit}` : ""}
      </bdi>
      {unit === null && (
        <>
          {" "}
          {t(l, "Unit not supplied", "단위가 제공되지 않음", "لم تُقدّم وحدة")}
        </>
      )}
    </>
  );
}
export function SyntheticNotice({ locale: l }: { locale: Locale }) {
  return <p className="b10-synthetic">{synthetic(l)}</p>;
}
function Facts({
  locale: l,
  rows,
}: {
  locale: Locale;
  rows: readonly (readonly [string, string | null])[];
}) {
  return (
    <dl className="b10-facts">
      {rows.map(([label, value]) => (
        <div key={label}>
          <dt>{label}</dt>
          <dd>
            <SourceValue locale={l} value={value} />
          </dd>
        </div>
      ))}
    </dl>
  );
}
export default function ClaimCard({
  locale: l,
  record: r,
  close,
}: {
  locale: Locale;
  record: ClaimRecord;
  close: () => void;
}) {
  const c = r.coverage;
  return (
    <section
      className="b10-result"
      aria-label={t(
        l,
        "Claim evidence card",
        "청구 적용 근거 카드",
        "بطاقة أدلة تطبيق الفوترة",
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
      <SyntheticNotice locale={l} />
      <header>
        <div>
          <h2 tabIndex={-1} data-b10-detail-heading>
            <bdi dir="ltr">{r.id}</bdi>
          </h2>
          <p>
            <bdi dir="auto">{r.target.label}</bdi>
          </p>
          <p className="b10-meta">
            {t(l, "Source revision", "원천 개정", "مراجعة المصدر")}{" "}
            <SourceValue locale={l} value={r.revision} />
          </p>
        </div>
        <button onClick={close}>
          {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
        </button>
      </header>
      <section>
        <h3>
          {t(
            l,
            "Coverage and protection",
            "적용 범위·보호",
            "التغطية والحماية",
          )}
        </h3>
        <p className="b10-protection">{protectionLabel(l, r.protection)}</p>
        <p>
          {t(
            l,
            "This source state applies only to the supplied application and coverage subset. It does not describe every interval of the obligation or establish current permission to bill.",
            "이 원천 상태는 제공된 적용 건과 해당 범위에만 해당합니다. 의무 전체의 상태나 현재 청구 권한을 뜻하지 않습니다.",
            "تنطبق حالة المصدر هذه على التطبيق والنطاق الجزئي المقدّمين فقط. لا تصف جميع فترات الالتزام ولا تثبت صلاحية الفوترة الحالية.",
          )}
        </p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Application reference", "적용 건 참조", "مرجع التطبيق"),
              r.application_id,
            ],
            [
              t(
                l,
                "Obligation reference",
                "경제적 의무 참조",
                "مرجع الالتزام الاقتصادي",
              ),
              r.obligation_id,
            ],
            [
              t(l, "Purpose reference", "적용 목적 참조", "مرجع غرض التطبيق"),
              r.purpose,
            ],
            [
              t(l, "Coverage reference", "적용 범위 참조", "مرجع التغطية"),
              c.identity,
            ],
            [
              t(
                l,
                "Coverage universe",
                "범위 전체 집합 참조",
                "مرجع مجموعة التغطية",
              ),
              c.universe,
            ],
            [t(l, "Unit", "단위", "الوحدة"), c.unit],
            [t(l, "Unit version", "단위 버전", "إصدار الوحدة"), c.unit_version],
          ]}
        />
        <p className="b10-source-description">
          <SourceValue locale={l} value={c.description} direction="auto" />
        </p>
        {r.external
          .filter((x) => ["pending", "unknown"].includes(x.outcome))
          .map((x) => (
            <div className="b10-external-warning" key={x.kind + ":" + x.id}>
              <strong>
                {t(
                  l,
                  "External outcome pending or unknown",
                  "외부 결과 대기·미확인",
                  "نتيجة خارجية معلّقة أو غير معروفة",
                )}
              </strong>
              <p>
                {externalKind(l, x.kind)} <bdi dir="ltr">{x.id}</bdi>{" "}
                {outcomeLabel(l, x.outcome)}
              </p>
            </div>
          ))}
        <h4>
          {t(
            l,
            "Source-declared coverage observations",
            "원천이 기록한 적용 범위 수치",
            "قيم التغطية المسجّلة في المصدر",
          )}
        </h4>
        <dl className="b10-values">
          {(
            [
              [
                t(
                  l,
                  "Reserved coverage",
                  "예약 범위 수치",
                  "قيمة التغطية المحجوزة",
                ),
                c.reserved,
              ],
              [
                t(
                  l,
                  "Applied coverage",
                  "적용 범위 수치",
                  "قيمة التغطية المطبّقة",
                ),
                c.applied,
              ],
              [
                t(
                  l,
                  "Reversed coverage",
                  "적용 취소 범위 수치",
                  "قيمة التغطية المعكوسة",
                ),
                c.reversed,
              ],
              [
                t(
                  l,
                  "Available coverage reported by source",
                  "가용 범위 수치 (원천 기록)",
                  "قيمة التغطية المتاحة في المصدر",
                ),
                c.available,
              ],
            ] as const
          ).map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <CoverageValue locale={l} value={value} unit={c.unit} />
              </dd>
            </div>
          ))}
        </dl>
        <p className="b10-meta">
          {t(
            l,
            "Source declarations only. Scalars do not establish overlap checks, available intervals or eligibility to rebill. Missing coverage is unknown, never zero. New source or submission IDs do not reset protected coverage.",
            "원천이 선언한 값입니다. 수치만으로 중복 검사·사용 가능 구간·재청구 가능 여부를 확정하지 않습니다. 누락은 0이 아닙니다. 새 원천·제출 ID로 보호 범위가 초기화되지 않습니다.",
            "هذه إقرارات مصدر فقط. لا تثبت القيم فحص التداخل أو الفترات المتاحة أو أهلية إعادة الفوترة. التغطية المفقودة ليست صفراً. لا تعيد معرّفات المصدر أو الإرسال الجديدة ضبط التغطية المحمية.",
          )}
        </p>
      </section>
      <section className="b10-amounts">
        <h3>
          {t(
            l,
            "Amounts by currency and basis",
            "통화·금액 기준별 관측값",
            "المبالغ حسب العملة والأساس",
          )}
        </h3>
        <p className="b10-meta">
          {t(
            l,
            "Remaining amount is neither available coverage nor a receivable balance. Source decimal strings are shown without rounding or padding. Different currencies and amount bases are not combined.",
            "남은 금액은 사용 가능 범위나 미수금 잔액이 아닙니다. 원천 소수 문자열을 반올림·자릿수 보충 없이 표시합니다. 다른 통화·금액 기준은 합산하지 않습니다.",
            "المبلغ المتبقي ليس تغطية متاحة ولا رصيد ذمم مدينة. تُعرض القيم العشرية الأصلية دون تقريب أو إضافة خانات. لا تُجمع العملات أو أسس المبالغ المختلفة.",
          )}
        </p>
        {!r.amounts.length ? (
          <p>
            {t(
              l,
              "No amount observations were supplied. This is not a zero amount.",
              "원천이 금액을 제공하지 않았습니다. 0원이라는 뜻이 아닙니다.",
              "لم تُقدّم قيم مبالغ. هذا لا يعني أن المبلغ صفر.",
            )}
          </p>
        ) : (
          r.amounts.map((a) => (
            <section
              className="b10-amount-block"
              key={a.currency + ":" + a.basis}
            >
              <h4>
                {basisLabel(l, a.basis)} <bdi dir="ltr">{a.currency}</bdi>
              </h4>
              <dl className="b10-values">
                {(
                  [
                    [
                      t(l, "Applied amount", "적용 금액", "المبلغ المطبّق"),
                      a.applied,
                    ],
                    [
                      t(
                        l,
                        "Remaining amount reported by source",
                        "남은 금액 (원천 기록)",
                        "المبلغ المتبقي في المصدر",
                      ),
                      a.remaining,
                    ],
                    [
                      t(
                        l,
                        "Reversed amount reported by source",
                        "적용 취소 금액 (원천 기록)",
                        "المبلغ المعكوس في المصدر",
                      ),
                      a.reversed,
                    ],
                  ] as const
                ).map(([label, value]) => (
                  <div key={label}>
                    <dt>{label}</dt>
                    <dd>
                      {value === null ? (
                        <span>{unknown(l)}</span>
                      ) : (
                        <div
                          className="b10-money-scroll"
                          tabIndex={0}
                          role="region"
                          aria-label={label}
                        >
                          <bdi dir="ltr" className="b10-money">
                            {a.currency} {value}
                          </bdi>
                        </div>
                      )}
                    </dd>
                  </div>
                ))}
              </dl>
            </section>
          ))
        )}
      </section>
      <section>
        <h3>
          {t(
            l,
            "Reversals and corrections",
            "적용 취소·정정",
            "عمليات العكس والتصحيح",
          )}
        </h3>
        <p>{dispositionLabel(l, r.correction.disposition)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(
                l,
                "Original Claim reference",
                "원래 청구 적용 기록 참조",
                "مرجع سجل التطبيق الأصلي",
              ),
              r.correction.original_claim_id,
            ],
          ]}
        />
        <h4>
          {t(
            l,
            "Linked correction references",
            "연결된 정정 기록 참조",
            "مراجع التصحيح المرتبطة",
          )}
        </h4>
        {r.correction.linked_claim_ids.length ? (
          <ul>
            {r.correction.linked_claim_ids.map((id) => (
              <li key={id}>
                <bdi dir="ltr">{id}</bdi>
              </li>
            ))}
          </ul>
        ) : (
          <p>
            {t(
              l,
              "No linked correction reference supplied.",
              "연결된 정정 참조를 제공하지 않았습니다.",
              "لم يُقدّم مرجع تصحيح مرتبط.",
            )}
          </p>
        )}
        <p>
          {t(
            l,
            "A money-only credit does not reopen quantity. Quantity reversal does not authorize replacement or rebilling. Explicit approved replacement linkage and ordinary approval/reservation remain separate requirements.",
            "금액만 정정해도 적용 수량은 다시 열리지 않습니다. 수량 적용 취소가 대체·재청구 권한을 뜻하지 않습니다. 승인된 대체 연결과 일반 승인·예약 절차는 별도로 필요합니다.",
            "لا يعيد التصحيح المالي وحده فتح الكمية. عكس تطبيق الكمية لا يأذن بالاستبدال أو إعادة الفوترة. تبقى صلة الاستبدال المعتمدة والموافقة والحجز المعتادان متطلبات مستقلة.",
          )}
        </p>
      </section>
      <section>
        <h3>{t(l, "Application context", "청구 적용 대상", "سياق التطبيق")}</h3>
        <dl className="b10-facts">
          {(
            [
              [
                t(l, "Legal entity", "법인", "الكيان القانوني"),
                r.target.entity,
              ],
              [t(l, "Customer", "고객", "العميل"), r.target.customer],
              [t(l, "Contract", "계약", "العقد"), r.target.contract],
              [t(l, "Component", "구성요소", "المكوّن"), r.target.component],
              [t(l, "Period", "기간", "الفترة"), r.target.period],
            ] as const
          ).map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <SourceValue
                  locale={l}
                  value={value}
                  direction={
                    label === t(l, "Period", "기간", "الفترة") ? "ltr" : "auto"
                  }
                />
              </dd>
            </div>
          ))}
        </dl>
        <p className="b10-meta">
          {t(l, "Partial application", "부분 적용", "تطبيق جزئي")}:{" "}
          {r.partial === null
            ? unknown(l)
            : r.partial
              ? t(
                  l,
                  "Reported partial",
                  "부분 적용으로 보고됨",
                  "مبلّغ عنه كجزئي",
                )
              : t(
                  l,
                  "Source reports not partial",
                  "부분 적용 아님 (원천 기록)",
                  "غير جزئي في المصدر",
                )}
        </p>
        <p className="b10-meta">
          {t(l, "Source updated time", "원천 갱신 시각", "وقت تحديث المصدر")}{" "}
          <bdi dir="ltr">{r.updated_at}</bdi>
        </p>
      </section>
      <section>
        <h3>{t(l, "Source lineage", "원천 연계", "تسلسل المصدر")}</h3>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Source reference", "원천 참조", "مرجع المصدر"),
              r.lineage.source,
            ],
            [
              t(l, "Allocation reference", "배분 참조", "مرجع التوزيع"),
              r.lineage.allocation,
            ],
            [
              t(l, "Sales reference", "판매 참조", "مرجع البيع"),
              r.lineage.sales,
            ],
            [
              t(
                l,
                "Charge-line reference",
                "청구 행 참조",
                "مرجع بند الفاتورة",
              ),
              r.lineage.charge_line,
            ],
            [
              t(l, "Submission reference", "제출 참조", "مرجع الإرسال"),
              r.lineage.submission,
            ],
          ]}
        />
      </section>
      <section>
        <h3>{t(l, "External references", "외부 참조", "المراجع الخارجية")}</h3>
        <p className="b10-meta">
          {t(
            l,
            "Source declarations do not prove document matching, reconciliation, nonapplication or release authority. Unknown outcomes retain protection until authoritative reconciliation.",
            "원천 기록이 문서 일치·대사·미적용·예약 해제 권한을 증명하지 않습니다. 외부 결과가 미확인이면 권한 있는 대사 전까지 보호를 유지해야 합니다.",
            "لا تثبت سجلات المصدر تطابق المستندات أو المطابقة أو عدم التطبيق أو صلاحية التحرير. تبقى النتائج غير المعروفة محمية إلى حين المطابقة الموثوقة.",
          )}
        </p>
        {!r.external.length ? (
          <p>
            {t(
              l,
              "No external references were supplied. This does not prove no external billing.",
              "외부 참조를 제공하지 않았습니다. 외부 청구가 없다는 증거는 아닙니다.",
              "لم تُقدّم مراجع خارجية. هذا لا يثبت عدم وجود فوترة خارجية.",
            )}
          </p>
        ) : (
          r.external.map((x) => (
            <article key={x.kind + ":" + x.id} className="b10-external-record">
              <h4>{externalKind(l, x.kind)}</h4>
              <p>{outcomeLabel(l, x.outcome)}</p>
              <Facts
                locale={l}
                rows={[
                  [t(l, "Reference", "참조", "المرجع"), x.id],
                  [
                    t(l, "External revision", "외부 개정", "المراجعة الخارجية"),
                    x.revision,
                  ],
                  [
                    t(
                      l,
                      "External observation time",
                      "외부 관측 시각",
                      "وقت الرصد الخارجي",
                    ),
                    x.observed_at,
                  ],
                ]}
              />
            </article>
          ))
        )}
      </section>
      <details className="b10-evidence">
        <summary>{t(l, "Source evidence", "원천 근거", "أدلة المصدر")}</summary>
        {r.blockers.length ? (
          <ul>
            {r.blockers.map((x) => (
              <li key={x.code}>
                <bdi dir="auto">{x.text}</bdi>
              </li>
            ))}
          </ul>
        ) : (
          <p>
            {t(
              l,
              "No blocker text supplied; this is not a readiness certificate.",
              "차단 사유를 제공하지 않았습니다. 처리 가능 판정은 아닙니다.",
              "لم تُقدّم أسباب حجب؛ هذا ليس شهادة جاهزية.",
            )}
          </p>
        )}
      </details>
      <p className="b10-readonly">
        {t(
          l,
          "This screen only reads application evidence. It cannot reserve, release, reverse, replace, submit, correct, pay or authorize a Claim.",
          "이 화면은 적용 근거를 조회만 합니다. 청구 적용의 예약·해제·적용 취소·대체·제출·정정·결제·권한 부여를 실행하지 않습니다.",
          "تقرأ هذه الشاشة أدلة التطبيق فقط. لا يمكنها الحجز أو التحرير أو العكس أو الاستبدال أو الإرسال أو التصحيح أو الدفع أو منح صلاحية التطبيق.",
        )}
      </p>
    </section>
  );
}
