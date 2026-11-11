import type { Locale } from "../../../../shared/types";
import type { OverviewData, FocusRow } from "./api";
import { compare, type Baseline } from "./model";
import { t, basisLabel, unavailable, unknown } from "./copy";
import { Totals } from "./SourceInputs";
import { Money, RowAmount, AmountValue } from "./Amounts";
import {
  fieldLabel,
  missingLabel,
} from "../../../sources/collected-focus/labels";
function Authorities({ locale: l }: { locale: Locale }) {
  return (
    <section className="b08-authorities">
      <h2>
        {t(
          l,
          "Billing authorities",
          "청구 기준 연결",
          "مراجع الفوترة المعتمدة",
        )}
      </h2>
      <dl className="b08-facts b08-two-cols">
        {[
          t(
            l,
            "Contract and legal responsibility",
            "계약·법적 책임",
            "العقد والمسؤولية القانونية",
          ),
          t(
            l,
            "Target manifest and shared-pool dependencies",
            "대상 명세·공유 풀 의존성",
            "بيان النطاق واعتماديات المجمع المشترك",
          ),
          t(
            l,
            "Eligible components and claim scope",
            "청구 적격 구성요소·적용 범위",
            "المكونات المؤهلة ونطاق المطالبة",
          ),
          t(
            l,
            "Approved allocation, sales, FX and engine versions",
            "승인된 배분·판매·환율·계산 엔진 버전",
            "إصدارات التوزيع والمبيعات والصرف والمحرك المعتمدة",
          ),
        ].map((label) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{unavailable(l)}</dd>
          </div>
        ))}
      </dl>
      <p className="b08-meta">
        {t(
          l,
          "A temporary source baseline establishes none of these authorities.",
          "임시 원천 비교 기준은 위 청구 기준의 확정·승인을 대신하지 않습니다.",
          "مرجع المصدر المؤقت لا يثبت أياً من هذه الصلاحيات المعتمدة.",
        )}
      </p>
    </section>
  );
}
export { Authorities };
export function Runs({
  locale: l,
  data,
  go,
}: {
  locale: Locale;
  data: OverviewData;
  go: (v: string) => void;
}) {
  const stages = [
    [
      t(l, "Scope", "대상 범위", "النطاق"),
      t(
        l,
        "Source selection only",
        "원천 조회 범위만 선택됨",
        "اختيار مصدر فقط",
      ),
    ],
    [
      t(l, "Source readiness", "원천 준비", "جاهزية المصدر"),
      t(
        l,
        "Read; completeness unverified",
        "조회됨 · 완전성 미확인",
        "مقروء؛ الاكتمال غير متحقق",
      ),
    ],
    [t(l, "Allocation", "배분", "التوزيع"), unavailable(l)],
    [t(l, "Full calculation", "전체 계산", "الاحتساب الكامل"), unavailable(l)],
    [t(l, "Approval", "승인", "الاعتماد"), unavailable(l)],
    [
      t(l, "Issuance and delivery", "발행·전달", "الإصدار والتسليم"),
      unavailable(l),
    ],
    [t(l, "Closing", "마감", "الإقفال"), unavailable(l)],
  ];
  return (
    <>
      <div className="b08-run-grid">
        <Totals locale={l} data={data} />
        <section className="b08-empty-state">
          <h2>
            {t(l, "Billing runs", "청구 계산 실행", "عمليات احتساب الفواتير")}
          </h2>
          <p>
            {t(
              l,
              "Run records cannot be shown because the billing execution service is not connected.",
              "청구 계산 실행 서비스가 연결되지 않아 실행 이력을 표시할 수 없습니다.",
              "لا يمكن عرض سجلات العمليات لأن خدمة احتساب الفواتير غير متصلة.",
            )}
          </p>
          <div className="b08-actions">
            <button onClick={() => go("inputs")}>
              {t(
                l,
                "Inspect input rows",
                "입력 명세 확인",
                "فحص صفوف المدخلات",
              )}
            </button>
            <button onClick={() => go("comparison")}>
              {t(
                l,
                "Compare source baseline",
                "원천 비교 기준 보기",
                "عرض مقارنة مرجع المصدر",
              )}
            </button>
          </div>
        </section>
      </div>
      <section>
        <h2>{t(l, "Billing flow", "청구 업무 흐름", "مسار الفوترة")}</h2>
        <ol className="b08-flow">
          {stages.map(([label, status]) => (
            <li key={label}>
              <span>{label}</span>
              <small>{status}</small>
            </li>
          ))}
        </ol>
      </section>
    </>
  );
}
export function Trace({
  locale: l,
  row,
  go,
}: {
  locale: Locale;
  row: FocusRow | undefined;
  go: () => void;
}) {
  return (
    <div className="b08-two-cols">
      <section>
        <h2>
          {t(
            l,
            "Observed source evidence",
            "관측 원천 근거",
            "أدلة المصدر المرصودة",
          )}
        </h2>
        {row ? (
          <>
            <p>
              {row.service === null ? (
                <>
                  {fieldLabel(l, "service")}:{" "}
                  <strong className="b08-state">{missingLabel(l)}</strong>
                </>
              ) : (
                <bdi dir="ltr">{row.service}</bdi>
              )}
            </p>
            <dl className="b08-facts">
              {(["billed_cost", "effective_cost"] as const).map((b) => (
                <div key={b}>
                  <dt>{basisLabel(l, b)}</dt>
                  <dd>
                    <RowAmount
                      value={row[b]}
                      currency={row.currency}
                      locale={l}
                      note
                    />
                  </dd>
                </div>
              ))}
              <div>
                <dt>{fieldLabel(l, "id")}</dt>
                <dd>
                  <bdi dir="ltr" className="b08-code">
                    <span>{row.id.slice(0, 32)}</span>
                    <wbr />
                    <span>{row.id.slice(32)}</span>
                  </bdi>
                </dd>
              </div>
            </dl>
          </>
        ) : (
          <p>
            {t(
              l,
              "Inspect a source row in Inputs to view its evidence here.",
              "입력·정책에서 원천 행을 선택하면 이곳에서 근거를 볼 수 있습니다.",
              "افحص صف مصدر في المدخلات لعرض أدلته هنا.",
            )}
          </p>
        )}
        <button onClick={go}>
          {t(l, "Open input rows", "입력 명세로 이동", "فتح صفوف المدخلات")}
        </button>
      </section>
      <section>
        <h2>
          {t(l, "Results and rules", "계산 결과·규칙", "النتائج والقواعد")}
        </h2>
        <dl className="b08-facts">
          {[
            t(l, "Customer charge lines", "고객 청구 행", "بنود فوترة العملاء"),
            t(
              l,
              "Applied rule trace",
              "적용 규칙 근거",
              "أدلة القواعد المطبقة",
            ),
            t(
              l,
              "Separate adjustment lines",
              "별도 조정 행",
              "بنود التسوية المنفصلة",
            ),
            t(
              l,
              "Financial result reference",
              "청구 계산 결과 참조",
              "مرجع نتيجة الاحتساب",
            ),
          ].map((label) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>{unavailable(l)}</dd>
            </div>
          ))}
        </dl>
        <p>
          {t(
            l,
            "Source amounts are not selling amounts. A source query reference is not a billing-result hash.",
            "원천 금액은 판매 금액이 아닙니다. 조회 기준 ID는 청구 계산 결과 해시가 아닙니다.",
            "مبالغ المصدر ليست مبالغ البيع. مرجع الاستعلام ليس تجزئة نتيجة الفوترة.",
          )}
        </p>
      </section>
    </div>
  );
}
export function Comparison({
  locale: l,
  data: d,
  baseline: b,
}: {
  locale: Locale;
  data: OverviewData;
  baseline: Baseline | null;
}) {
  const values = b ? compare(b, d) : null,
    fresh = b && Date.parse(d.observed_at) > Date.parse(b.observed_at);
  return (
    <section className="b08-comparison">
      <h2>
        {t(l, "Compare source reads", "원천 조회 비교", "مقارنة قراءات المصدر")}
      </h2>
      <p>
        {t(
          l,
          "This compares source observations only, not billing runs or customer charges. Changed source references require a new baseline because historical visibility cannot be reauthorized here.",
          "원천 관측만 비교하며 청구 실행·고객 청구 비교가 아닙니다. 원천 기준이 바뀌면 과거 자료의 열람 권한을 확인할 수 없어 비교 기준을 다시 선택해야 합니다.",
          "تُقارن ملاحظات المصدر فقط، لا عمليات الفوترة أو رسوم العملاء. تغيّر مرجع المصدر يتطلب مرجعاً جديداً لأن صلاحية البيانات السابقة لا تُثبت هنا.",
        )}
      </p>
      {!b ? (
        <p className="b08-empty-state">
          {t(
            l,
            "Temporarily keep a source baseline, then use New query on the same source and scope.",
            "원천 비교 기준을 잠시 보관한 뒤 같은 원천·범위에서 새로 조회하세요.",
            "احتفظ مؤقتاً بمرجع المصدر ثم استخدم استعلاماً جديداً للمصدر والنطاق نفسيهما.",
          )}
        </p>
      ) : (
        <>
          <div className="b08-observations">
            {[
              [
                t(
                  l,
                  "Temporary baseline",
                  "임시 비교 기준",
                  "مرجع المقارنة المؤقت",
                ),
                b.observed_at,
                b.row_count,
              ],
              [
                t(l, "Current read", "현재 조회", "القراءة الحالية"),
                d.observed_at,
                d.row_count,
              ],
            ].map(([label, time, count]) => (
              <section key={label}>
                <h3>{label}</h3>
                <dl className="b08-facts">
                  <div>
                    <dt>{t(l, "Read time", "조회 시각", "وقت القراءة")}</dt>
                    <dd>
                      <bdi dir="ltr">{time}</bdi>
                    </dd>
                  </div>
                  <div>
                    <dt>{t(l, "Source rows", "원천 행", "صفوف المصدر")}</dt>
                    <dd>
                      <bdi dir="ltr">{count}</bdi>
                    </dd>
                  </div>
                </dl>
              </section>
            ))}
          </div>
          {!fresh ? (
            <p role="status">
              {t(
                l,
                "Read the source again with New query before comparing. Keeping this read does not create a comparison result.",
                "새 조회로 원천을 다시 읽은 뒤 비교합니다. 현재 조회를 보관한 것만으로 비교 결과가 생기지 않습니다.",
                "أعد قراءة المصدر باستعلام جديد قبل المقارنة. الاحتفاظ بالقراءة الحالية لا يُنشئ نتيجة مقارنة.",
              )}
            </p>
          ) : (
            values?.map((a) => (
              <section className="b08-currency-comparison" key={a.currency}>
                <h3>
                  <bdi dir="ltr">{a.currency}</bdi>
                </h3>
                {(["billed_cost", "effective_cost"] as const).map((basis) => (
                  <div className="b08-measure-comparison" key={basis}>
                    <h4>{basisLabel(l, basis)}</h4>
                    <dl>
                      <div>
                        <dt>
                          {t(
                            l,
                            "Baseline source amount",
                            "비교 기준 원천 금액",
                            "مبلغ المصدر المرجعي",
                          )}
                        </dt>
                        <dd>
                          {a.baseline ? (
                            <AmountValue
                              value={a.baseline[basis]}
                              currency={a.currency}
                              locale={l}
                            />
                          ) : (
                            <span>
                              {t(
                                l,
                                "No matching currency observation",
                                "대응 통화 관측 없음",
                                "لا ملاحظة مطابقة للعملة",
                              )}
                            </span>
                          )}
                        </dd>
                      </div>
                      <div>
                        <dt>
                          {t(
                            l,
                            "Current source amount",
                            "현재 원천 금액",
                            "مبلغ المصدر الحالي",
                          )}
                        </dt>
                        <dd>
                          {a.current ? (
                            <AmountValue
                              value={a.current[basis]}
                              currency={a.currency}
                              locale={l}
                            />
                          ) : (
                            <span>
                              {t(
                                l,
                                "No matching currency observation",
                                "대응 통화 관측 없음",
                                "لا ملاحظة مطابقة للعملة",
                              )}
                            </span>
                          )}
                        </dd>
                      </div>
                      <div>
                        <dt>
                          {t(
                            l,
                            "Current minus baseline",
                            "현재 − 비교 기준",
                            "الحالي ناقص المرجع",
                          )}
                        </dt>
                        <dd data-b08-delta={basis}>
                          {a[basis] === null ? (
                            t(
                              l,
                              "Unavailable: complete matching observations required",
                              "비교 불가 · 누락 없는 대응 관측이 필요합니다",
                              "غير متاح: تلزم ملاحظات متطابقة بلا قيم مفقودة",
                            )
                          ) : (
                            <Money
                              value={a[basis]}
                              currency={a.currency}
                              locale={l}
                            />
                          )}
                        </dd>
                      </div>
                    </dl>
                  </div>
                ))}
              </section>
            ))
          )}
          <p>
            {t(
              l,
              "Equal totals or a zero difference do not establish unchanged rows, policies, eligibility or completed financial validation. No change-cause attribution is connected.",
              "합계 일치·차이 0은 행·정책 불변, 청구 적격성, 재무 검증 완료를 뜻하지 않습니다. 변경 원인 판정은 미연결입니다.",
              "تساوي الإجماليات أو الفرق الصفري لا يثبت عدم تغيّر الصفوف أو السياسات أو الأهلية أو اكتمال التحقق المالي. تحديد سبب التغيّر غير متصل.",
            )}
          </p>
        </>
      )}
    </section>
  );
}
export function Validation({
  locale: l,
  data: d,
}: {
  locale: Locale;
  data: OverviewData;
}) {
  const groups = [
    [
      t(l, "Observed", "관측된 내용", "المرصود"),
      [
        [
          t(l, "Source query", "원천 조회", "استعلام المصدر"),
          t(
            l,
            "Read successfully; not financial validation",
            "조회 성공 · 재무 검증이 아닙니다",
            "قُرئ بنجاح؛ ليس تحققاً مالياً",
          ),
        ],
        [
          t(l, "Source row count", "원천 행 수", "عدد صفوف المصدر"),
          String(d.row_count),
        ],
      ],
    ],
    [
      t(l, "Unverified", "미확인", "غير متحقق منه"),
      [
        [
          t(
            l,
            "Source completeness for this selection",
            "현재 범위 원천 완전성",
            "اكتمال المصدر ضمن الاختيار",
          ),
          unknown(l),
        ],
        [
          t(
            l,
            "Contract eligibility",
            "계약상 청구 적격성",
            "الأهلية التعاقدية للفوترة",
          ),
          unknown(l),
        ],
        [
          t(
            l,
            "Pool and claim checks",
            "공유 풀·청구 적용 검증",
            "فحوص المجمع والمطالبة",
          ),
          unknown(l),
        ],
      ],
    ],
    [
      t(l, "Not connected", "미연결", "غير متصل"),
      [
        [
          t(
            l,
            "Approved policy, FX and engine",
            "승인된 정책·환율·계산 엔진",
            "السياسة والصرف والمحرك المعتمد",
          ),
          unavailable(l),
        ],
        [
          t(
            l,
            "Financial result validation",
            "청구 계산 결과 검증",
            "التحقق من نتيجة الفوترة",
          ),
          unavailable(l),
        ],
        [t(l, "Calculation lock", "계산 잠금", "قفل الاحتساب"), unavailable(l)],
        [t(l, "Issuance lock", "발행 잠금", "قفل الإصدار"), unavailable(l)],
      ],
    ],
  ] as [string, [string, string][]][];
  return (
    <section>
      <h2>
        {t(l, "Pre-billing checks", "청구 전 확인", "فحوص ما قبل الفوترة")}
      </h2>
      <p>
        {t(
          l,
          "A successful source read does not mean this scope is ready to bill.",
          "원천 조회 성공은 청구 준비 완료가 아닙니다.",
          "نجاح قراءة المصدر لا يعني جاهزية هذا النطاق للفوترة.",
        )}
      </p>
      <div className="b08-check-groups">
        {groups.map(([label, rows]) => (
          <section key={label}>
            <h3>{label}</h3>
            <dl className="b08-facts">
              {rows.map(([term, value]) => (
                <div key={term}>
                  <dt>{term}</dt>
                  <dd>
                    {/^\d+$/.test(value) ? <bdi dir="ltr">{value}</bdi> : value}
                  </dd>
                </div>
              ))}
            </dl>
          </section>
        ))}
      </div>
    </section>
  );
}
export function Recognition({ locale: l }: { locale: Locale }) {
  return (
    <section>
      <h2>
        {t(
          l,
          "Recognition and billing remain separate",
          "비용 인식과 고객 청구의 분리",
          "فصل إثبات المصروفات عن فوترة العملاء",
        )}
      </h2>
      <div className="b08-two-cols">
        {[
          [
            t(
              l,
              "Monthly expense recognition (ERP)",
              "월별 비용 인식 (ERP)",
              "الإثبات الشهري للمصروفات (ERP)",
            ),
            [
              t(
                l,
                "Legal entity and book",
                "법인·장부",
                "الكيان القانوني والدفتر",
              ),
              t(
                l,
                "Amount basis and cadence",
                "금액 기준·주기",
                "أساس المبلغ والدورية",
              ),
              t(
                l,
                "Monthly recognition batches",
                "월별 인식 배치",
                "دفعات الإثبات الشهرية",
              ),
              t(
                l,
                "Actual ERP posting",
                "실제 ERP 전기",
                "الترحيل الفعلي في ERP",
              ),
            ],
          ],
          [
            t(
              l,
              "Quarterly customer billing",
              "분기 고객 청구",
              "فوترة العملاء ربع السنوية",
            ),
            [
              t(
                l,
                "Linked recognition months",
                "연결 인식 월",
                "أشهر الإثبات المرتبطة",
              ),
              t(l, "Approved adjustments", "승인된 조정", "التسويات المعتمدة"),
              t(
                l,
                "Quarterly reconciliation",
                "분기 대사",
                "المطابقة ربع السنوية",
              ),
            ],
          ],
        ].map(([title, labels]) => (
          <section key={title as string}>
            <h3>{title}</h3>
            <dl className="b08-facts">
              {(labels as string[]).map((label) => (
                <div key={label}>
                  <dt>{label}</dt>
                  <dd>{unavailable(l)}</dd>
                </div>
              ))}
            </dl>
          </section>
        ))}
      </div>
      <p>
        {t(
          l,
          "No invoice, posting, recognition approval, receivable, collection or payment-term clock is created here. Missing months and unknown external states are not zero or reconciled.",
          "고객 청구서·전기·인식 승인·채권·수납을 만들거나 납부기한을 시작하지 않습니다. 누락 월과 외부 미확인 상태는 0·대사 완료가 아닙니다.",
          "لا تُنشأ هنا فاتورة أو ترحيل أو اعتماد إثبات أو ذمم أو تحصيل ولا يبدأ أجل دفع. الأشهر المفقودة والحالات الخارجية غير المعروفة ليست صفراً أو مطابقة مكتملة.",
        )}
      </p>
    </section>
  );
}
