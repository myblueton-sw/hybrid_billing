import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { EvidenceRecord, Data } from "./contract";
import { t, recordLabel } from "./copy";
import RecordCard from "./RecordCard";
import { SyntheticNotice, SourceValue } from "./Shared";
import {
  RowSummary,
  stateLabel,
  ReferenceSummary,
  TableValues,
} from "./RecordSummary";
export default function Inbox({
  locale: l,
  data: d,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: Data;
  selected: EvidenceRecord | undefined;
  select: (id: string | null) => void;
  page: (c: string | null) => void;
}) {
  const root = useRef<HTMLDivElement>(null),
    last = useRef<string | null>(null);
  useEffect(() => {
    let frame: number | undefined;
    if (selected && selected.id !== last.current) {
      [
        ...(root.current?.querySelectorAll<HTMLElement>(
          "[data-b11-detail-heading]",
        ) ?? []),
      ]
        .find((e) => e.getClientRects().length)
        ?.focus();
    } else if (!selected && last.current) {
      const id = last.current;
      frame = requestAnimationFrame(() =>
        [
          ...(root.current?.querySelectorAll<HTMLButtonElement>(
            "[data-b11-record]",
          ) ?? []),
        ]
          .find((e) => e.dataset.b11Record === id && e.getClientRects().length)
          ?.focus(),
      );
    }
    last.current = selected?.id ?? null;
    return () => {
      if (frame !== undefined) cancelAnimationFrame(frame);
    };
  }, [selected?.id]);
  return (
    <div ref={root}>
      <SyntheticNotice locale={l} />
      <p className="b11-meta b11-page-status" role="status">
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
              {t(l, "Exact total:", "전체 건수:", "الإجمالي الدقيق:")}{" "}
              <bdi className="b11-count" dir="ltr">
                {d.total}
              </bdi>
            </>
          )}
        </span>
        <span>
          {t(l, "This page:", "현재 페이지:", "هذه الصفحة:")}{" "}
          <bdi className="b11-count" dir="ltr">
            {d.records.length}
          </bdi>
        </span>
      </p>
      {!d.records.length ? (
        <section className="b11-empty">
          <h2>
            {t(
              l,
              "No matching source records",
              "조건에 맞는 원천 기록 없음",
              "لا سجلات مصدر مطابقة",
            )}
          </h2>
          <p>
            {t(
              l,
              "This is a validated empty response for the current view and filter, not an unconnected service.",
              "현재 보기·필터의 검증된 빈 응답입니다. 서비스 미연결 상태와는 다릅니다.",
              "هذه استجابة فارغة متحقق منها لهذا العرض والمرشّح، وليست خدمة غير متصلة.",
            )}
          </p>
        </section>
      ) : (
        <div className={`b11-workspace ${selected ? "has-detail" : ""}`}>
          <section className="b11-inbox">
            <h2>{t(l, "Source records", "원천 기록", "سجلات المصدر")}</h2>
            <ul className="b11-request-list">
              {d.records.map((r) => (
                <li key={r.id}>
                  <button
                    data-b11-record={r.id}
                    aria-expanded={selected?.id === r.id}
                    className={selected?.id === r.id ? "selected" : ""}
                    onClick={() => select(selected?.id === r.id ? null : r.id)}
                    aria-label={t(
                      l,
                      `Inspect source record ${r.id}`,
                      `원천 기록 ${r.id} 근거 보기`,
                      `فحص سجل المصدر ${r.id}`,
                    )}
                  >
                    <span className="b11-row-heading">
                      <bdi dir="ltr">{r.id}</bdi>
                      <span className="b11-synthetic-tag">
                        {t(l, "Synthetic", "합성", "تجريبي")}
                      </span>
                    </span>
                    <span>
                      <bdi dir="auto">{r.label}</bdi>
                    </span>
                    <span className="b11-meta">{stateLabel(l, r)}</span>
                    <span className="b11-row-summary">
                      <RowSummary locale={l} record={r} />
                    </span>
                    <span className="b11-meta">
                      <bdi dir="ltr">
                        {r.authority.observed_at ?? r.updated_at}
                      </bdi>{" "}
                      {selected?.id === r.id &&
                        t(l, "Selected", "선택됨", "محدد")}
                    </span>
                  </button>
                  {selected?.id === r.id && (
                    <div className="b11-mobile-detail">
                      <RecordCard
                        locale={l}
                        record={r}
                        close={() => select(null)}
                      />
                    </div>
                  )}
                </li>
              ))}
            </ul>
          </section>
          <aside className="b11-desktop-detail">
            {selected ? (
              <RecordCard
                locale={l}
                record={selected}
                close={() => select(null)}
              />
            ) : (
              <p>
                {t(
                  l,
                  "Select a source record to inspect its authority, values and separate evidence.",
                  "원천 기록을 선택하면 원장 기준·값·구분된 근거를 확인할 수 있습니다.",
                  "اختر سجل مصدر لفحص مرجعيته وقيمه وأدلته المنفصلة.",
                )}
              </p>
            )}
          </aside>
        </div>
      )}
      <div className="b11-pager">
        <button
          id="b11-first"
          disabled={d.cursor === null}
          onClick={() => page(null)}
        >
          {t(l, "First page", "처음으로", "الصفحة الأولى")}
        </button>
        <button
          id="b11-next"
          disabled={d.next_cursor === null}
          onClick={() => d.next_cursor !== null && page(d.next_cursor)}
        >
          {t(l, "Next page", "다음 페이지", "الصفحة التالية")}
        </button>
      </div>
      {d.records.length > 0 && (
        <details className="b11-full-table">
          <summary>
            {t(
              l,
              "Show full-column table",
              "전체 열 표 보기",
              "عرض جدول جميع الأعمدة",
            )}{" "}
            <bdi dir="ltr">({d.records.length})</bdi>
          </summary>
          <div
            className="b11-table-region"
            role="region"
            tabIndex={0}
            aria-label={t(
              l,
              "Credit and inquiry evidence table",
              "크레딧·문의 근거 표",
              "جدول أدلة الأرصدة والاستفسارات",
            )}
          >
            <table>
              <caption>
                {t(
                  l,
                  "Current page only. Credit, inquiry and receivable dimensions are kept separate. Internal notes and public replies are disclosed separately in the selected record.",
                  "현재 페이지만 표시합니다. 크레딧·문의·미수금 차원을 구분합니다. 내부 메모와 공개 답변은 선택한 기록의 별도 영역에서 확인합니다.",
                  "الصفحة الحالية فقط. تُفصل أبعاد الائتمان والاستفسار والذمم. تُعرض الملاحظات الداخلية والردود العلنية بشكل منفصل في السجل المحدد.",
                )}
              </caption>
              <thead>
                <tr>
                  {[
                    t(l, "Record / revision", "기록·개정", "السجل والمراجعة"),
                    t(
                      l,
                      "Customer / contract / period",
                      "고객·계약·기간",
                      "العميل والعقد والفترة",
                    ),
                    t(
                      l,
                      "Authority and as-of",
                      "원장 기준·관측 시각",
                      "المرجعية ووقت الرصد",
                    ),
                    t(l, "Source state", "원천 상태", "حالة المصدر"),
                    t(l, "Source values", "원천 값", "قيم المصدر"),
                    t(l, "Evidence references", "근거 참조", "مراجع الأدلة"),
                    t(l, "Inspect", "근거 보기", "فحص"),
                  ].map((h) => (
                    <th scope="col" key={h}>
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {d.records.map((r) => (
                  <tr key={r.id}>
                    <th scope="row">
                      <bdi dir="ltr">{r.id}</bdi>
                      <p>
                        <bdi dir="auto">{r.label}</bdi>
                      </p>
                      <SourceValue locale={l} value={r.revision} />
                      <p>{recordLabel(l, r.kind)}</p>
                    </th>
                    <td>
                      {[
                        r.target.customer,
                        r.target.contract,
                        r.target.period,
                      ].map((v, i) => (
                        <div key={i}>
                          <SourceValue
                            locale={l}
                            value={v}
                            direction={i === 2 ? "ltr" : "auto"}
                          />
                        </div>
                      ))}
                    </td>
                    <td>
                      <SourceValue locale={l} value={r.authority.reference} />
                      <p>
                        <SourceValue
                          locale={l}
                          value={r.authority.observed_at}
                        />
                      </p>
                    </td>
                    <td>{stateLabel(l, r)}</td>
                    <td>
                      <TableValues locale={l} record={r} />
                    </td>
                    <td>
                      <ReferenceSummary locale={l} record={r} />
                    </td>
                    <td>
                      <button
                        data-b11-record={r.id}
                        onClick={() => select(r.id)}
                      >
                        {t(l, "Inspect record", "기록 근거 보기", "فحص السجل")}
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
