import type { Locale } from "../../../../shared/types";
import type {
  EvidenceRecord,
  BalanceRecord,
  ReservationRecord,
  CaseRecord,
  AdjustmentRecord,
  PaymentRecord,
} from "./contract";
import { t, reported } from "./copy";
import { Facts, SourceValue, Measure, MoneyValues, Holds } from "./Shared";
function Balance({ l, r }: { l: Locale; r: BalanceRecord }) {
  return (
    <>
      <section className="b11-domain">
        <h3>
          {t(
            l,
            "Credit balance observations",
            "크레딧 잔액 관측값",
            "قيم رصيد الائتمان",
          )}
        </h3>
        <p>
          {t(
            l,
            "A credit balance is not a receivable or cash. Source signs are preserved; no customer sign convention or current usability is inferred.",
            "크레딧 잔액은 미수금이나 현금이 아닙니다. 원천 부호를 유지하며 고객별 부호 규칙이나 현재 사용 가능 여부를 추론하지 않습니다.",
            "رصيد الائتمان ليس ذمماً مدينة ولا نقداً. تُحفظ إشارات المصدر دون افتراض دلالتها أو قابلية الاستخدام الحالية.",
          )}
        </p>
        <dl className="b11-values">
          {(
            [
              [
                t(
                  l,
                  "Available — source observation",
                  "사용 가능액·수량 (원천 관측)",
                  "المتاح كقيمة مصدر",
                ),
                r.available,
              ],
              [
                t(
                  l,
                  "Reserved — source observation",
                  "예약액·수량 (원천 관측)",
                  "المحجوز كقيمة مصدر",
                ),
                r.reserved,
              ],
            ] as const
          ).map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <Measure
                  locale={l}
                  value={value}
                  dimension={r.dimension}
                  label={label}
                />
              </dd>
            </div>
          ))}
        </dl>
      </section>
      <section>
        <h3>
          {t(
            l,
            "Source balance components",
            "원천 잔액 구성 항목",
            "مكوّنات الرصيد من المصدر",
          )}
        </h3>
        <p>
          {t(
            l,
            "Shown as supplied. This screen does not calculate or certify conservation, closing balance or availability.",
            "원천이 제공한 값입니다. 이 화면은 잔액 보존·마감 잔액·사용 가능액을 계산하거나 인증하지 않습니다.",
            "تُعرض كما قدّمها المصدر. لا تحسب الشاشة حفظ الرصيد أو الرصيد الختامي أو المتاح ولا تصدّق عليها.",
          )}
        </p>
        <dl className="b11-values">
          {(
            [
              [t(l, "Opening", "기초 잔액", "الرصيد الافتتاحي"), r.opening],
              [t(l, "Increases", "증가", "الزيادات"), r.increases],
              [t(l, "Applied", "적용", "المطبّق"), r.applied],
              [t(l, "Expired", "만료", "المنتهي"), r.expired],
              [t(l, "Reversals", "적용 취소", "العكس"), r.reversed],
              [t(l, "Closing", "마감 잔액", "الرصيد الختامي"), r.closing],
            ] as const
          ).map(([label, value]) => (
            <div key={label}>
              <dt>{label}</dt>
              <dd>
                <Measure
                  locale={l}
                  value={value}
                  dimension={r.dimension}
                  label={label}
                />
              </dd>
            </div>
          ))}
        </dl>
      </section>
      <section>
        <h3>
          {t(
            l,
            "Migration and history references",
            "이관·과거 이력 참조",
            "مراجع الترحيل والسجل السابق",
          )}
        </h3>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Migration ledger", "이관 원장", "سجل الترحيل"),
              r.migration_ref,
            ],
            [
              t(
                l,
                "Included historical coverage",
                "포함된 과거 범위",
                "التغطية السابقة المضمّنة",
              ),
              r.included_history_ref,
            ],
          ]}
        />
        <p>
          {t(
            l,
            "References do not prove duplicate prevention. Opening balance and reimported history must not be counted twice.",
            "참조만으로 중복 방지가 입증되지 않습니다. 기초 잔액과 재수집 과거 이력을 이중 계산해서는 안 됩니다.",
            "المراجع لا تثبت منع التكرار. يجب ألا يُحتسب الرصيد الافتتاحي والسجل المعاد استيراده مرتين.",
          )}
        </p>
      </section>
    </>
  );
}
function Reservation({ l, r }: { l: Locale; r: ReservationRecord }) {
  return (
    <section className="b11-domain">
      <h3>
        {t(
          l,
          "Reservation and application observation",
          "예약·적용 관측값",
          "قيمة الحجز والتطبيق",
        )}
      </h3>
      <p>{reported(l, r.state)}</p>
      <h4>
        {t(
          l,
          "Reservation or application value — source observation",
          "예약·적용 금액·수량 (원천 관측)",
          "قيمة الحجز أو التطبيق من المصدر",
        )}
      </h4>
      <Measure
        locale={l}
        value={r.value}
        dimension={r.dimension}
        label={t(
          l,
          "Reservation or application value",
          "예약·적용 금액·수량",
          "قيمة الحجز أو التطبيق",
        )}
      />
      <Facts
        locale={l}
        rows={[
          [
            t(l, "Credit reference", "크레딧 참조", "مرجع الائتمان"),
            r.credit_ref,
          ],
          [
            t(l, "Document reference", "문서 참조", "مرجع المستند"),
            r.document_ref,
          ],
          [
            t(l, "Original reference", "원래 기록 참조", "مرجع السجل الأصلي"),
            r.original_ref,
          ],
        ]}
      />
      <p>
        {t(
          l,
          "Source status is neither execution nor release authority. Unknown external results require reconciliation; no retry or expiry here changes the balance.",
          "원천 상태는 실행 결과나 해제 권한의 인증이 아닙니다. 외부 결과 미확인은 대사가 필요하며, 여기서 재조회하거나 시간이 지나도 잔액이 바뀌지 않습니다.",
          "حالة المصدر ليست تصديقاً على التنفيذ أو صلاحية التحرير. النتائج الخارجية المجهولة تتطلب مطابقة؛ الاستعلام أو مرور الوقت هنا لا يغيّر الرصيد.",
        )}
      </p>
    </section>
  );
}
function Case({ l, r }: { l: Locale; r: CaseRecord }) {
  return (
    <>
      <section className="b11-domain">
        <h3>
          {t(
            l,
            "Inquiry and dispute evidence",
            "문의·이의 근거",
            "أدلة الاستفسار والاعتراض",
          )}
        </h3>
        <p>{reported(l, r.state)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Original document", "원래 문서", "المستند الأصلي"),
              r.original_document_ref,
            ],
            [
              t(l, "Owner in source", "원천 담당자", "المسؤول في المصدر"),
              r.owner_label,
              "auto",
            ],
            [t(l, "Received time", "접수 시각", "وقت الاستلام"), r.received_at],
            [t(l, "Deadline", "처리 기한", "المهلة"), r.deadline],
          ]}
        />
        <h4>
          {t(
            l,
            "Disputed amounts — source observations",
            "이의 금액 (원천 관측)",
            "مبالغ الاعتراض كقيم مصدر",
          )}
        </h4>
        <MoneyValues
          locale={l}
          amounts={r.amounts}
          label={t(l, "Disputed amount", "이의 금액", "مبلغ الاعتراض")}
        />
        <p>
          {t(
            l,
            "An inquiry is not cancellation, a refund, payment exemption or a settled balance.",
            "문의 접수는 취소·환불·납부 의무 면제·잔액 정산이 아닙니다.",
            "الاستفسار ليس إلغاءً أو رداً للمبلغ أو إعفاءً من السداد أو تسوية للرصيد.",
          )}
        </p>
      </section>
      <details className="b11-note b11-internal" data-b11-private>
        <summary>
          {t(
            l,
            "Internal note — internal only",
            "내부 메모 · 내부 전용",
            "ملاحظة داخلية للاستخدام الداخلي فقط",
          )}
        </summary>
        <p>
          <SourceValue locale={l} value={r.internal_note} direction="auto" />
        </p>
      </details>
      <details className="b11-note b11-public" data-b11-public>
        <summary>
          {t(
            l,
            "Public reply — source record, no sending",
            "공개 답변 · 원천 기록, 전송 기능 없음",
            "رد علني مسجّل في المصدر دون إرسال",
          )}
        </summary>
        <p>
          <SourceValue locale={l} value={r.public_reply} direction="auto" />
        </p>
      </details>
      <section className="b11-consent">
        <h3>{t(l, "Customer consent", "고객 동의", "موافقة العميل")}</h3>
        <p>{reported(l, r.consent.state)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(
                l,
                "Consent evidence reference",
                "동의 근거 참조",
                "مرجع دليل الموافقة",
              ),
              r.consent.evidence_ref,
            ],
          ]}
        />
        <p>
          {t(
            l,
            "Consent is not inferred from a correction or explanation link.",
            "정정 문서나 차이 설명 연결로 고객 동의를 추론하지 않습니다.",
            "لا تُستنتج الموافقة من رابط تصحيح أو تفسير للفروقات.",
          )}
        </p>
      </section>
      <section>
        <h3>
          {t(l, "Resolution in source", "원천 해결 기록", "التسوية في المصدر")}
        </h3>
        <p>{reported(l, r.resolution)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Resolution reference", "해결 참조", "مرجع التسوية"),
              r.resolution_ref,
            ],
          ]}
        />
        <p>
          {t(
            l,
            "A recorded resolution does not itself alter a document, credit, receipt or payment obligation.",
            "해결 기록 자체가 문서·크레딧·수납·납부 의무를 변경하지 않습니다.",
            "سجل التسوية بحد ذاته لا يغيّر المستند أو الائتمان أو الإيصال أو التزام السداد.",
          )}
        </p>
      </section>
      <Holds locale={l} holds={r.holds} />
    </>
  );
}
function Adjustment({ l, r }: { l: Locale; r: AdjustmentRecord }) {
  return (
    <section className="b11-domain">
      <h3>
        {t(
          l,
          "Original-linked adjustment evidence",
          "원래 기록에 연결된 조정 근거",
          "دليل تسوية مرتبط بالأصل",
        )}
      </h3>
      <p>{reported(l, r.adjustment_kind)}</p>
      <p>{reported(l, r.state)}</p>
      <Facts
        locale={l}
        rows={[
          [
            t(l, "Original document", "원래 문서", "المستند الأصلي"),
            r.original_document_ref,
          ],
          [
            t(
              l,
              "Original Claim",
              "원래 청구 적용 기록",
              "سجل تطبيق الفوترة الأصلي",
            ),
            r.original_claim_ref,
          ],
          [
            t(l, "Correction reference", "정정 참조", "مرجع التصحيح"),
            r.correction_ref,
          ],
          [
            t(l, "Reason in source", "원천 사유", "السبب في المصدر"),
            r.reason,
            "auto",
          ],
          [
            t(l, "Evidence reference", "근거 참조", "مرجع الدليل"),
            r.evidence_ref,
          ],
          [
            t(l, "Approval reference", "승인 참조", "مرجع الاعتماد"),
            r.approval_ref,
          ],
        ]}
      />
      <h4>
        {t(
          l,
          "Adjustment amounts by currency — source observations",
          "통화별 조정 금액 (원천 관측)",
          "مبالغ التسوية حسب العملة من المصدر",
        )}
      </h4>
      <MoneyValues
        locale={l}
        amounts={r.amounts}
        label={t(l, "Adjustment amount", "조정 금액", "مبلغ التسوية")}
      />
      <p>
        {t(
          l,
          "These are monetary observations. Quantity-reversal evidence is not supplied by this projection. A money credit does not reopen quantity or authorize replacement/rebilling; an outage alone does not establish an SLA credit.",
          "금액 관측값입니다. 이 조회에는 수량 적용 취소 근거가 제공되지 않습니다. 금액 크레딧이 수량을 다시 열거나 대체·재청구를 허용하지 않으며, 장애 발생만으로 SLA 크레딧이 성립하지 않습니다.",
          "هذه قيم مالية. لا يقدّم هذا العرض دليل عكس تطبيق الكمية. الائتمان المالي لا يعيد فتح الكمية ولا يأذن بالاستبدال أو إعادة الفوترة، والعطل وحده لا يثبت ائتمان SLA.",
        )}
      </p>
    </section>
  );
}
function Payment({ l, r }: { l: Locale; r: PaymentRecord }) {
  return (
    <>
      <section className="b11-domain">
        <h3>
          {t(
            l,
            "Source issuance and due-date snapshot",
            "원천 발행·지급 기한 기록",
            "لقطة الإصدار والاستحقاق في المصدر",
          )}
        </h3>
        <p>{reported(l, r.issuance)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Original document", "원래 문서", "المستند الأصلي"),
              r.original_document_ref,
            ],
            [
              t(
                l,
                "Payment-term reference",
                "지급 조건 참조",
                "مرجع شروط السداد",
              ),
              r.term_ref,
            ],
            [
              t(l, "Payment-term label", "지급 조건 이름", "اسم شروط السداد"),
              r.term_label,
              "auto",
            ],
            [
              t(
                l,
                "Contract time zone",
                "계약 시간대",
                "المنطقة الزمنية للعقد",
              ),
              r.time_zone,
            ],
            [t(l, "Issue date", "발행일", "تاريخ الإصدار"), r.issue_date],
            [
              t(
                l,
                "Original due date",
                "원래 지급 기한",
                "موعد الاستحقاق الأصلي",
              ),
              r.original_due_date,
            ],
            [
              t(
                l,
                "Effective due date",
                "현재 유효 지급 기한",
                "موعد الاستحقاق الفعّال",
              ),
              r.effective_due_date,
            ],
          ]}
        />
        <p>
          {t(
            l,
            "Dates are supplied snapshots. This screen does not calculate Net terms, extensions or aging; redispatch and downloads do not restart the clock.",
            "원천이 제공한 날짜 기록입니다. 이 화면은 지급 조건·기한 연장·연체 일수를 계산하지 않습니다. 재전송·다운로드로 지급 기한이 다시 시작되지 않습니다.",
            "التواريخ لقطات مقدّمة من المصدر. لا تحسب الشاشة شروط السداد أو التمديد أو التقادم؛ إعادة الإرسال والتنزيل لا تعيدان بدء المدة.",
          )}
        </p>
      </section>
      <section className="b11-amounts">
        <h3>
          {t(
            l,
            "Remaining receivable — separate from credit",
            "남은 미수금 · 크레딧과 별도",
            "الذمم المتبقية منفصلة عن الائتمان",
          )}
        </h3>
        <MoneyValues
          locale={l}
          amounts={[r.receivable]}
          label={t(l, "Remaining receivable", "남은 미수금", "الذمم المتبقية")}
        />
      </section>
      <section className="b11-receipt">
        <h3>
          {t(
            l,
            "Receipt evidence and freshness",
            "수납 근거·최신성",
            "أدلة الإيصال وحداثتها",
          )}
        </h3>
        <p>{reported(l, r.receipt.state)}</p>
        <p>{reported(l, r.receipt.freshness)}</p>
        <Facts
          locale={l}
          rows={[
            [
              t(l, "Receipt reference", "수납 참조", "مرجع الإيصال"),
              r.receipt.reference,
            ],
            [
              t(
                l,
                "Receipt observation time",
                "수납 관측 시각",
                "وقت رصد الإيصال",
              ),
              r.receipt.observed_at,
            ],
          ]}
        />
        <p>
          {t(
            l,
            "A transfer claim, payment promise, public reply or unallocated cash is not a confirmed allocation. Missing or stale receipt evidence leaves overdue unknown.",
            "입금 주장·납부 약속·공개 답변·미배정 입금은 확정 배정이 아닙니다. 수납 근거가 없거나 오래됐으면 연체는 미확인입니다.",
            "ادّعاء التحويل أو وعد الدفع أو الرد العلني أو النقد غير المخصّص ليس تخصيصاً مؤكداً. نقص أدلة الإيصال أو تقادمها يترك التأخر غير معروف.",
          )}
        </p>
      </section>
      <section>
        <h3>
          {t(
            l,
            "Overdue state — source declaration",
            "연체 상태 (원천 기록)",
            "حالة التأخر المعلنة في المصدر",
          )}
        </h3>
        <p>{reported(l, r.overdue)}</p>
        <p>
          {t(
            l,
            "Only the source remaining balance after partial payment is shown. No confirmed overdue total, late fee, interest or debit is produced here.",
            "부분 수납 후 남은 잔액만 원천 관측값으로 표시합니다. 확정 연체 합계·연체료·이자·자동 출금을 만들지 않습니다.",
            "يُعرض فقط الرصيد المتبقي في المصدر بعد السداد الجزئي. لا تنتج الشاشة مجموعاً مؤكداً للتأخر أو رسوماً أو فوائد أو خصماً.",
          )}
        </p>
        <p>
          {t(
            l,
            "Dispute overlay",
            "이의 상태 (별도)",
            "حالة الاعتراض المنفصلة",
          )}
          : {reported(l, r.dispute)}
        </p>
      </section>
      <Holds locale={l} holds={r.holds} />
    </>
  );
}
export default function EvidencePanels({
  locale: l,
  record: r,
}: {
  locale: Locale;
  record: EvidenceRecord;
}) {
  switch (r.kind) {
    case "balance":
      return <Balance l={l} r={r} />;
    case "reservation":
      return <Reservation l={l} r={r} />;
    case "case":
      return <Case l={l} r={r} />;
    case "adjustment":
      return <Adjustment l={l} r={r} />;
    case "payment":
      return <Payment l={l} r={r} />;
  }
}
