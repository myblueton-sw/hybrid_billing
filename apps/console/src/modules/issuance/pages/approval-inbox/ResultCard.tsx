import type { Locale } from "../../../../shared/types";
import type { ApprovalRecord, Data } from "./contract";
import {
  t,
  kindLabel,
  statusLabel,
  basisLabel,
  unknown,
  synthetic,
  sourceUnknown,
} from "./copy";
export function SourceValue({
  locale: l,
  value,
}: {
  locale: Locale;
  value: string | null;
}) {
  return value === null ? (
    <span>{unknown(l)}</span>
  ) : (
    <bdi dir="ltr">{value}</bdi>
  );
}
export function Money({
  locale: l,
  value,
  currency,
  label,
}: {
  locale: Locale;
  value: string | null;
  currency: string;
  label: string;
}) {
  return value === null ? (
    <span>{unknown(l)}</span>
  ) : (
    <div
      className="b09-money-scroll"
      role="region"
      tabIndex={value.length > 24 ? 0 : undefined}
      aria-label={label}
    >
      <bdi className="b09-money" dir="ltr">
        {currency} {value}
      </bdi>
    </div>
  );
}
export function SyntheticNotice({
  locale: l,
  compact = false,
}: {
  locale: Locale;
  compact?: boolean;
}) {
  return (
    <p
      className={compact ? "b09-synthetic b09-meta" : "b09-synthetic"}
      data-b09-synthetic
    >
      {synthetic(l)}
    </p>
  );
}
function KindBoundary({
  locale: l,
  kind,
}: {
  locale: Locale;
  kind: ApprovalRecord["kind"];
}) {
  return (
    <p>
      {
        {
          billing_result: t(
            l,
            "Billing-result approval is separate from final ERP amount approval and issuance.",
            "청구 계산 결과 승인은 최종 ERP 금액 승인·발행과 별개입니다.",
            "اعتماد نتيجة الفوترة منفصل عن اعتماد مبلغ ERP النهائي والإصدار.",
          ),
          final_erp_amount: t(
            l,
            "Final ERP amounts require external revision and reconciliation evidence. Unknown external values do not become not applicable.",
            "최종 ERP 금액에는 외부 개정·대사 근거가 필요합니다. 외부 값 미확인은 미적용이 아닙니다.",
            "مبالغ ERP النهائية تتطلب مراجعة خارجية وأدلة مطابقة. القيم غير المعروفة لا تصبح غير منطبقة.",
          ),
          adjustment: t(
            l,
            "An adjustment decision concerns the fixed difference computation only; application and final amount approval remain separate.",
            "조정 결정은 고정된 차액 계산에 한하며 적용·최종 금액 승인과 별개입니다.",
            "قرار التسوية يخص احتساب الفرق المثبت فقط؛ التطبيق واعتماد المبلغ النهائي منفصلان.",
          ),
          policy: t(
            l,
            "A policy decision concerns its exact revision, not customer billing or issuance.",
            "정책 결정은 해당 개정에 한하며 고객 청구·발행 승인이 아닙니다.",
            "قرار السياسة يخص مراجعتها المحددة، وليس اعتماد فوترة العميل أو الإصدار.",
          ),
          monthly_recognition: t(
            l,
            "Monthly expense recognition is separate from customer invoices, receivables, ERP posting confirmation and collection.",
            "월별 비용 인식은 고객 청구서·채권·ERP 전기 확인·수납과 별개입니다.",
            "الإثبات الشهري للمصروفات منفصل عن فواتير العملاء والذمم وتأكيد ترحيل ERP والتحصيل.",
          ),
        }[kind]
      }
    </p>
  );
}
export default function ResultCard({
  locale: l,
  record: r,
  data: d,
  close,
}: {
  locale: Locale;
  record: ApprovalRecord;
  data: Data;
  close: () => void;
}) {
  const sourceStates = [
    [
      t(
        l,
        "Dependency revision declaration",
        "원천 의존 개정 상태",
        "إقرار مراجعة الاعتماديات",
      ),
      {
        current: t(l, "Reported current", "현재로 보고됨", "مبلّغ عنها حالية"),
        stale: t(
          l,
          "Reported stale",
          "이전 개정으로 보고됨",
          "مبلّغ عنها قديمة",
        ),
        unknown: sourceUnknown(l),
      }[r.evidence.dependencies],
    ],
    [
      t(
        l,
        "Evidence completeness declaration",
        "원천 근거 제공 범위",
        "إقرار اكتمال الأدلة",
      ),
      {
        complete: t(
          l,
          "Reported complete evidence",
          "근거 전체 제공으로 보고됨",
          "مبلّغ عن اكتمال الأدلة",
        ),
        partial: t(
          l,
          "Partial evidence provided",
          "일부 근거만 제공됨",
          "قُدّمت أدلة جزئية",
        ),
        unknown: sourceUnknown(l),
      }[r.evidence.completeness],
    ],
    [
      t(l, "Validation declaration", "원천 검증 기록", "إقرار التحقق"),
      {
        passed: t(l, "Reported passed", "통과로 보고됨", "مبلّغ عن النجاح"),
        failed: t(l, "Reported failed", "실패로 보고됨", "مبلّغ عن الفشل"),
        unknown: sourceUnknown(l),
      }[r.evidence.validation],
    ],
    [
      t(
        l,
        "Separation-of-duties declaration",
        "원천 직무 분리 기록",
        "إقرار فصل المهام",
      ),
      {
        satisfied: t(
          l,
          "Reported satisfied",
          "충족으로 보고됨",
          "مبلّغ عن الاستيفاء",
        ),
        not_satisfied: t(
          l,
          "Reported not satisfied",
          "미충족으로 보고됨",
          "مبلّغ عن عدم الاستيفاء",
        ),
        unknown: sourceUnknown(l),
      }[r.authority.separation],
    ],
    [
      t(
        l,
        "Amount-limit declaration",
        "원천 금액 한도 기록",
        "إقرار حد المبلغ",
      ),
      {
        within: t(
          l,
          "Reported within limit",
          "한도 이내로 보고됨",
          "مبلّغ أنه ضمن الحد",
        ),
        exceeded: t(
          l,
          "Reported over limit",
          "한도 초과로 보고됨",
          "مبلّغ أنه يتجاوز الحد",
        ),
        unknown: sourceUnknown(l),
      }[r.authority.limit],
    ],
  ];
  return (
    <section
      className="b09-result"
      aria-label={t(
        l,
        "Result evidence card",
        "결과 검토 카드",
        "بطاقة أدلة النتيجة",
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
      <SyntheticNotice locale={l} compact />
      <header>
        <div>
          <h2 tabIndex={-1} data-b09-detail-heading>
            <bdi dir="auto">{r.target.label}</bdi>
          </h2>
          <p>{kindLabel(l, r.kind)}</p>
          <p className="b09-meta">{statusLabel(l, r.status)}</p>
        </div>
        <button onClick={close}>
          {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
        </button>
      </header>
      <KindBoundary locale={l} kind={r.kind} />
      <dl className="b09-facts">
        {[
          [t(l, "Request reference", "요청 참조", "مرجع الطلب"), r.id],
          [t(l, "Target reference", "대상 참조", "مرجع الهدف"), r.target.id],
          [t(l, "Legal entity", "법인", "الكيان القانوني"), r.target.entity],
          [t(l, "Customer", "고객", "العميل"), r.target.customer],
          [t(l, "Period", "기간", "الفترة"), r.target.period],
          [
            t(
              l,
              "Author in source record",
              "작성자 (원천 기록)",
              "المؤلف في سجل المصدر",
            ),
            r.author ? `${r.author.label} (${r.author.id})` : null,
          ],
          [
            t(
              l,
              "Actual decision actor in source",
              "결정자 (원천 기록)",
              "صاحب القرار الفعلي في المصدر",
            ),
            r.approver ? `${r.approver.label} (${r.approver.id})` : null,
          ],
          [t(l, "Decision time", "결정 시각", "وقت القرار"), r.decided_at],
          [t(l, "Deadline", "기한", "الموعد النهائي"), r.deadline],
          [t(l, "Updated at", "갱신 시각", "وقت التحديث"), r.updated_at],
        ].map(([label, value]) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>
              <SourceValue locale={l} value={value} />
            </dd>
          </div>
        ))}
      </dl>
      <section className="b09-amounts">
        <h3>
          {t(
            l,
            "Amounts by currency and basis",
            "통화·금액 기준별 관측값",
            "المبالغ حسب العملة والأساس",
          )}
        </h3>
        <p className="b09-meta">
          {t(
            l,
            "Source decimal strings are shown without rounding or a default currency precision. Different currencies and bases are never totaled here.",
            "원천 소수 문자열을 반올림·통화별 기본 자릿수 적용 없이 표시합니다. 다른 통화·금액 기준은 합산하지 않습니다.",
            "تُعرض السلاسل العشرية من المصدر دون تقريب أو دقة افتراضية للعملة. لا تُجمع العملات والأسس المختلفة هنا.",
          )}
        </p>
        {!r.amounts.length ? (
          <p>
            {t(
              l,
              "No amount observations were provided. This is not a zero amount.",
              "금액 관측값이 제공되지 않았습니다. 금액 0을 뜻하지 않습니다.",
              "لم تُقدّم ملاحظات مبالغ. هذا لا يعني مبلغاً صفرياً.",
            )}
          </p>
        ) : (
          r.amounts.map((a) => (
            <section
              className="b09-currency-basis"
              key={`${a.currency}:${a.basis}`}
            >
              <h4>
                <bdi dir="ltr">{a.currency}</bdi> {basisLabel(l, a.basis)}
              </h4>
              <dl>
                {(
                  [
                    [
                      t(l, "Base amount", "기준 금액", "المبلغ الأساسي"),
                      a.base,
                    ],
                    [
                      t(l, "Candidate amount", "후보 금액", "المبلغ المرشّح"),
                      a.candidate,
                    ],
                    [
                      t(
                        l,
                        "Declared difference",
                        "원천 기록 차이",
                        "الفرق المصرّح به",
                      ),
                      a.difference,
                    ],
                    [
                      t(
                        l,
                        "ERP observed amount",
                        "ERP 관측 금액",
                        "مبلغ ERP المرصود",
                      ),
                      a.erp,
                    ],
                  ] as const
                ).map(([label, value], i) => (
                  <div key={label}>
                    <dt>{label}</dt>
                    <dd
                      data-b09-amount={
                        i === 0
                          ? "base"
                          : i === 1
                            ? "candidate"
                            : i === 2
                              ? "difference"
                              : "erp"
                      }
                    >
                      {i === 3 && r.erp.applicability === "not_applicable" ? (
                        <span>
                          {t(
                            l,
                            "Recorded as not applicable",
                            "원천에 미적용으로 기록됨",
                            "مسجّل كغير منطبق",
                          )}
                        </span>
                      ) : (
                        <Money
                          locale={l}
                          value={value}
                          currency={a.currency}
                          label={label}
                        />
                      )}
                    </dd>
                    {i === 2 && (
                      <small>
                        {t(
                          l,
                          "Source-declared difference",
                          "원천에 기록된 차이",
                          "فرق مصرّح به من المصدر",
                        )}
                      </small>
                    )}
                    {i === 3 && (
                      <small>
                        {t(
                          l,
                          "External observation; not inferred",
                          "외부 관측값 · 추정하지 않음",
                          "ملاحظة خارجية؛ ليست مستنتجة",
                        )}
                      </small>
                    )}
                  </div>
                ))}
              </dl>
            </section>
          ))
        )}
      </section>
      <section>
        <h3>
          {t(l, "ERP observation scope", "ERP 관측 범위", "نطاق ملاحظات ERP")}
        </h3>
        <dl className="b09-facts">
          {[
            [
              t(l, "ERP applicability", "ERP 적용 여부", "انطباق ERP"),
              {
                applicable: t(
                  l,
                  "Applicable in source",
                  "원천 기록: 적용",
                  "المصدر: منطبق",
                ),
                not_applicable: t(
                  l,
                  "Not applicable in source",
                  "원천 기록: 미적용",
                  "المصدر: غير منطبق",
                ),
                unknown: t(
                  l,
                  "Applicability unverified",
                  "적용 여부 미확인",
                  "الانطباق غير متحقق منه",
                ),
              }[r.erp.applicability],
            ],
            [
              t(l, "External revision", "외부 개정", "المراجعة الخارجية"),
              r.erp.external_revision,
            ],
            [
              t(
                l,
                "External observation time",
                "외부 관측 시각",
                "وقت الملاحظة الخارجية",
              ),
              r.erp.observed_at,
            ],
          ].map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <SourceValue locale={l} value={value} />
              </dd>
            </div>
          ))}
        </dl>
      </section>
      <section>
        <h3>
          {t(
            l,
            "Source evidence declarations",
            "원천 근거 기록",
            "إقرارات أدلة المصدر",
          )}
        </h3>
        <p>
          {t(
            l,
            "These are source declarations. This screen does not certify current approval authority, limits, self-approval policy, billing readiness or input eligibility.",
            "원천의 기록 상태입니다. 이 화면은 현재 승인 권한·한도·자기 승인 정책·청구 준비·입력 적격성을 확정하지 않습니다.",
            "هذه إقرارات من المصدر. هذه الشاشة لا تثبت صلاحية الاعتماد الحالية أو الحدود أو سياسة الاعتماد الذاتي أو جاهزية الفوترة أو أهلية المدخلات.",
          )}
        </p>
        <dl className="b09-facts">
          {sourceStates.map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{value}</dd>
            </div>
          ))}
          <div>
            <dt>
              {t(
                l,
                "Declared validity deadline",
                "원천 기록의 유효 기한",
                "نهاية الصلاحية المعلنة",
              )}
            </dt>
            <dd>
              <SourceValue locale={l} value={r.authority.valid_until} />
            </dd>
          </div>
        </dl>
        {r.evidence.blockers.length > 0 && (
          <ul className="b09-blockers">
            {r.evidence.blockers.map((b) => (
              <li key={b.code}>
                <bdi dir="auto">{b.text}</bdi>
              </li>
            ))}
          </ul>
        )}
      </section>
      <details className="b09-evidence">
        <summary>
          {t(l, "Evidence references", "근거 참조", "مراجع الأدلة")}
        </summary>
        <dl className="b09-facts">
          {[
            [
              t(l, "Input digest", "입력 다이제스트", "تجزئة المدخلات"),
              r.evidence.input_digest,
            ],
            [
              t(l, "Result digest", "결과 다이제스트", "تجزئة النتيجة"),
              r.evidence.result_digest,
            ],
            [
              t(l, "Record revision", "기록 개정", "مراجعة السجل"),
              r.evidence.revision,
            ],
            [
              t(l, "Read snapshot", "조회 기준 ID", "معرّف مرجع القراءة"),
              d.snapshot,
            ],
            [
              t(l, "Permission revision", "권한 개정", "مراجعة الصلاحيات"),
              d.permission_revision,
            ],
          ].map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <SourceValue locale={l} value={value} />
              </dd>
            </div>
          ))}
        </dl>
      </details>
      <p className="b09-readonly">
        {t(
          l,
          "This screen only reads request evidence. Approve, reject, handoff, delegation and posting are not executable here. A recorded approval does not confirm issuance, delivery or current validity.",
          "이 화면은 요청 근거를 조회만 합니다. 승인·반려·인계·위임·전기는 실행하지 않습니다. 승인 기록은 발행·전달·현재 유효성의 확인이 아닙니다.",
          "تقرأ هذه الشاشة أدلة الطلبات فقط. لا تُنفّذ هنا الموافقة أو الرفض أو الإحالة أو التفويض أو الترحيل. سجل الاعتماد لا يؤكد الإصدار أو التسليم أو الصلاحية الحالية.",
        )}
      </p>
    </section>
  );
}
