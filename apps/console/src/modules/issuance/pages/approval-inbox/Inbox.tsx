import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { ApprovalRecord, Data } from "./contract";
import { t, kindLabel, statusLabel, unknown } from "./copy";
import ResultCard, { SyntheticNotice, SourceValue } from "./ResultCard";
export default function Inbox({
  locale: l,
  data: d,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: Data;
  selected: ApprovalRecord | undefined;
  select: (id: string | null) => void;
  page: (c: string | null) => void;
}) {
  const root = useRef<HTMLDivElement>(null),
    last = useRef<string | null>(null);
  useEffect(() => {
    if (selected && selected.id !== last.current) {
      [
        ...root.current!.querySelectorAll<HTMLElement>(
          "[data-b09-detail-heading]",
        ),
      ]
        .find((e) => e.getClientRects().length)
        ?.focus();
    } else if (!selected && last.current) {
      const id = last.current;
      requestAnimationFrame(() =>
        [
          ...(root.current?.querySelectorAll<HTMLButtonElement>(
            "[data-b09-record]",
          ) ?? []),
        ]
          .find((e) => e.dataset.b09Record === id && e.getClientRects().length)
          ?.focus(),
      );
    }
    last.current = selected?.id ?? null;
  }, [selected?.id]);
  return (
    <div ref={root}>
      <SyntheticNotice locale={l} />
      <p className="b09-meta b09-page-status" role="status">
        <span>
          {d.total === null ? (
            t(
              l,
              "Total count unconfirmed",
              "전체 건수 미확인",
              "العدد الإجمالي غير مؤكد",
            )
          ) : (
            <>
              {t(l, "Exact total", "전체 건수", "الإجمالي الدقيق")}{" "}
              <bdi className="b09-count" dir="ltr">
                {d.total}
              </bdi>
            </>
          )}
        </span>
        <span>
          {t(l, "This page", "현재 페이지", "هذه الصفحة")}{" "}
          <bdi className="b09-count" dir="ltr">
            {d.records.length}
          </bdi>
        </span>
        <span>
          {t(
            l,
            "Synthetic source reader",
            "합성 원천 조회자",
            "قارئ المصدر التجريبي",
          )}{" "}
          <bdi dir="ltr">{d.actor.label}</bdi>
        </span>
      </p>
      {!d.records.length ? (
        <section className="b09-empty">
          <h2>
            {t(
              l,
              "No matching requests in this read",
              "현재 조회 결과에 일치하는 요청 없음",
              "لا طلبات مطابقة في هذه القراءة",
            )}
          </h2>
          <p>
            {t(
              l,
              "This is the validated empty result for this view and filter, not an unconnected service.",
              "현재 보기·필터의 검증된 빈 응답입니다. 서비스 미연결 상태와는 다릅니다.",
              "هذه نتيجة فارغة متحقق منها لهذا العرض والمرشّح، وليست خدمة غير متصلة.",
            )}
          </p>
        </section>
      ) : (
        <div className={`b09-workspace ${selected ? "has-detail" : ""}`}>
          <section className="b09-inbox">
            <h2>{t(l, "Approval requests", "승인 요청", "طلبات الاعتماد")}</h2>
            <ul className="b09-request-list">
              {d.records.map((r) => (
                <li key={r.id}>
                  <button
                    className={selected?.id === r.id ? "selected" : ""}
                    data-b09-record={r.id}
                    aria-expanded={selected?.id === r.id}
                    onClick={() => select(selected?.id === r.id ? null : r.id)}
                    aria-label={t(
                      l,
                      `Inspect request ${r.id} ${r.target.label}`,
                      `요청 ${r.id} ${r.target.label} 근거 보기`,
                      `فحص الطلب ${r.id} ${r.target.label}`,
                    )}
                  >
                    <span className="b09-row-title">
                      <bdi dir="auto">{r.target.label}</bdi>
                    </span>
                    <span>{kindLabel(l, r.kind)}</span>
                    <span>{r.target.period ?? unknown(l)}</span>
                    <span className="b09-meta">
                      <bdi dir="ltr">{r.id}</bdi>
                    </span>
                    <span className="b09-meta">{statusLabel(l, r.status)}</span>
                    <span className="b09-meta">
                      {t(l, "Synthetic example", "합성 예시", "مثال تجريبي")}
                      {selected?.id === r.id && (
                        <> {t(l, "Selected", "선택됨", "محدد")}</>
                      )}
                    </span>
                  </button>
                  {selected?.id === r.id && (
                    <div className="b09-mobile-detail">
                      <ResultCard
                        locale={l}
                        data={d}
                        record={r}
                        close={() => select(null)}
                      />
                    </div>
                  )}
                </li>
              ))}
            </ul>
          </section>
          <aside className="b09-desktop-detail">
            {selected ? (
              <ResultCard
                locale={l}
                data={d}
                record={selected}
                close={() => select(null)}
              />
            ) : (
              <p>
                {t(
                  l,
                  "Select a request to inspect its target, source amounts and evidence.",
                  "요청을 선택하면 대상·원천 금액·근거를 확인할 수 있습니다.",
                  "اختر طلباً لفحص هدفه ومبالغ المصدر وأدلته.",
                )}
              </p>
            )}
          </aside>
        </div>
      )}
      <div className="b09-pager">
        <button
          id="b09-first"
          disabled={d.cursor === null}
          onClick={() => page(null)}
        >
          {t(l, "First page", "처음으로", "الصفحة الأولى")}
        </button>
        <button
          id="b09-next"
          disabled={d.next_cursor === null}
          onClick={() => d.next_cursor !== null && page(d.next_cursor)}
        >
          {t(l, "Next page", "다음 페이지", "الصفحة التالية")}
        </button>
      </div>
      {d.records.length > 0 && (
        <details className="b09-full-table">
          <summary>
            {t(
              l,
              "Show full-column table",
              "전체 열 표 보기",
              "عرض جدول جميع الأعمدة",
            )}{" "}
            <bdi className="b09-count" dir="ltr">
              ({d.records.length})
            </bdi>
          </summary>
          <div
            className="b09-table-region"
            role="region"
            tabIndex={0}
            aria-label={t(
              l,
              "Approval request evidence table",
              "승인 요청 근거 표",
              "جدول أدلة طلبات الاعتماد",
            )}
          >
            <table>
              <caption>
                {t(
                  l,
                  "Current-page request evidence. No amounts are flattened across currencies or bases.",
                  "현재 페이지 요청 근거입니다. 다른 통화·금액 기준을 하나의 금액으로 합치지 않습니다.",
                  "أدلة طلبات الصفحة الحالية. لا تُدمج العملات أو الأسس المختلفة في مبلغ واحد.",
                )}
              </caption>
              <thead>
                <tr>
                  {[
                    t(l, "Request reference", "요청 참조", "مرجع الطلب"),
                    t(l, "Approval type", "승인 유형", "نوع الاعتماد"),
                    t(l, "Target", "대상", "الهدف"),
                    t(l, "Entity / customer", "법인 / 고객", "الكيان / العميل"),
                    t(l, "Period", "기간", "الفترة"),
                    t(l, "Source status", "원천 상태", "حالة المصدر"),
                    t(
                      l,
                      "Author / decision actor",
                      "작성자 / 결정자",
                      "المؤلف / صاحب القرار",
                    ),
                    t(
                      l,
                      "Updated / deadline",
                      "갱신 / 기한",
                      "التحديث / الموعد",
                    ),
                    t(l, "Details", "상세", "التفاصيل"),
                  ].map((label) => (
                    <th scope="col" key={label}>
                      {label}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {d.records.map((r) => (
                  <tr key={r.id}>
                    <th scope="row">
                      <bdi dir="ltr">{r.id}</bdi>
                    </th>
                    <td>{kindLabel(l, r.kind)}</td>
                    <td>
                      <bdi dir="auto">{r.target.label}</bdi>
                    </td>
                    <td>
                      <SourceValue locale={l} value={r.target.entity} />
                      <br />
                      <SourceValue locale={l} value={r.target.customer} />
                    </td>
                    <td>
                      <SourceValue locale={l} value={r.target.period} />
                    </td>
                    <td>{statusLabel(l, r.status)}</td>
                    <td>
                      <bdi dir="auto">{r.author?.label ?? unknown(l)}</bdi>
                      <br />
                      <bdi dir="auto">{r.approver?.label ?? unknown(l)}</bdi>
                    </td>
                    <td>
                      <bdi dir="ltr">{r.updated_at}</bdi>
                      <br />
                      <SourceValue locale={l} value={r.deadline} />
                    </td>
                    <td>
                      <button
                        data-b09-record={r.id}
                        onClick={() => select(r.id)}
                        aria-expanded={selected?.id === r.id}
                      >
                        {t(l, "Inspect request", "요청 근거 보기", "فحص الطلب")}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </details>
      )}
    </div>
  );
}
