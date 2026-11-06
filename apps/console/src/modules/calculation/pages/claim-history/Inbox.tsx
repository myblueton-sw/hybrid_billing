import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { ClaimRecord, Data } from "./contract";
import {
  t,
  protectionLabel,
  dispositionLabel,
  unknown,
  basisLabel,
  externalKind,
  outcomeLabel,
} from "./copy";
import ClaimCard, {
  SourceValue,
  SyntheticNotice,
  CoverageValue,
} from "./ClaimCard";
export default function Inbox({
  locale: l,
  data: d,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: Data;
  selected: ClaimRecord | undefined;
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
          "[data-b10-detail-heading]",
        ) ?? []),
      ]
        .find((e) => e.getClientRects().length)
        ?.focus();
    } else if (!selected && last.current) {
      const id = last.current;
      frame = requestAnimationFrame(() =>
        [
          ...(root.current?.querySelectorAll<HTMLButtonElement>(
            "[data-b10-record]",
          ) ?? []),
        ]
          .find((e) => e.dataset.b10Record === id && e.getClientRects().length)
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
      <p className="b10-meta b10-page-status" role="status">
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
              <bdi className="b10-count" dir="ltr">
                {d.total}
              </bdi>
            </>
          )}
        </span>
        <span>
          {t(l, "This page:", "현재 페이지:", "هذه الصفحة:")}{" "}
          <bdi className="b10-count" dir="ltr">
            {d.records.length}
          </bdi>
        </span>
      </p>
      {!d.records.length ? (
        <section className="b10-empty">
          <h2>
            {t(
              l,
              "No matching application records",
              "조건에 맞는 적용 기록 없음",
              "لا سجلات تطبيق مطابقة",
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
        <div className={`b10-workspace ${selected ? "has-detail" : ""}`}>
          <section className="b10-inbox">
            <h2>
              {t(l, "Application records", "청구 적용 기록", "سجلات التطبيق")}
            </h2>
            <ul className="b10-request-list">
              {d.records.map((r) => (
                <li key={r.id}>
                  <button
                    data-b10-record={r.id}
                    aria-expanded={selected?.id === r.id}
                    aria-current={selected?.id === r.id ? "true" : undefined}
                    className={selected?.id === r.id ? "selected" : ""}
                    onClick={() => select(selected?.id === r.id ? null : r.id)}
                    aria-label={t(
                      l,
                      `Inspect application ${r.id}`,
                      `청구 적용 ${r.id} 근거 보기`,
                      `فحص التطبيق ${r.id}`,
                    )}
                  >
                    <span className="b10-row-heading">
                      <bdi dir="ltr">{r.id}</bdi>
                      <span className="b10-synthetic-tag">
                        {t(l, "Synthetic", "합성", "تجريبي")}
                      </span>
                    </span>
                    <span>
                      <bdi dir="auto">{r.target.label}</bdi>
                    </span>
                    <span className="b10-meta">
                      {protectionLabel(l, r.protection)}
                    </span>
                    <span className="b10-meta">
                      {r.partial === null
                        ? t(
                            l,
                            "Partial status unconfirmed",
                            "부분 적용 여부 미확인",
                            "الحالة الجزئية غير مؤكدة",
                          )
                        : r.partial
                          ? t(
                              l,
                              "Reported partial application",
                              "부분 적용으로 보고됨",
                              "تطبيق جزئي مبلّغ عنه",
                            )
                          : null}{" "}
                      {r.correction.disposition !== "none" &&
                        dispositionLabel(l, r.correction.disposition)}
                    </span>
                    <span className="b10-meta">
                      <bdi dir="ltr">{r.updated_at}</bdi>{" "}
                      {selected?.id === r.id &&
                        t(l, "Selected", "선택됨", "محدد")}
                    </span>
                  </button>
                  {selected?.id === r.id && (
                    <div className="b10-mobile-detail">
                      <ClaimCard
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
          <aside className="b10-desktop-detail">
            {selected ? (
              <ClaimCard
                locale={l}
                record={selected}
                close={() => select(null)}
              />
            ) : (
              <p>
                {t(
                  l,
                  "Select an application to inspect its coverage, source amounts and external references.",
                  "적용 기록을 선택하면 범위·원천 금액·외부 참조를 확인할 수 있습니다.",
                  "اختر تطبيقاً لفحص تغطيته ومبالغ المصدر والمراجع الخارجية.",
                )}
              </p>
            )}
          </aside>
        </div>
      )}
      <div className="b10-pager">
        <button
          id="b10-first"
          disabled={d.cursor === null}
          onClick={() => page(null)}
        >
          {t(l, "First page", "처음으로", "الصفحة الأولى")}
        </button>
        <button
          id="b10-next"
          disabled={d.next_cursor === null}
          onClick={() => d.next_cursor !== null && page(d.next_cursor)}
        >
          {t(l, "Next page", "다음 페이지", "الصفحة التالية")}
        </button>
      </div>
      {d.records.length > 0 && (
        <details className="b10-full-table">
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
            className="b10-table-region"
            role="region"
            tabIndex={0}
            aria-label={t(
              l,
              "Application evidence table",
              "청구 적용 근거 표",
              "جدول أدلة التطبيق",
            )}
          >
            <table>
              <caption>
                {t(
                  l,
                  "Current page only. Currency and coverage dimensions remain separate.",
                  "현재 페이지만 표시합니다. 통화와 적용 범위 차원을 분리합니다.",
                  "الصفحة الحالية فقط. تبقى أبعاد العملة والتغطية منفصلة.",
                )}
              </caption>
              <thead>
                <tr>
                  {[
                    t(
                      l,
                      "Claim / revision",
                      "적용 기록·개정",
                      "السجل والمراجعة",
                    ),
                    t(
                      l,
                      "Contract / period / component",
                      "계약·기간·구성요소",
                      "العقد والفترة والمكوّن",
                    ),
                    t(l, "Source protection", "원천 보호 상태", "حماية المصدر"),
                    t(
                      l,
                      "Coverage observations",
                      "적용 범위 관측값",
                      "قيم التغطية",
                    ),
                    t(
                      l,
                      "Currency / basis / amounts",
                      "통화·기준·금액",
                      "العملة والأساس والمبالغ",
                    ),
                    t(
                      l,
                      "Submission / external references",
                      "제출·외부 참조",
                      "الإرسال والمراجع الخارجية",
                    ),
                    t(l, "Correction references", "정정 참조", "مراجع التصحيح"),
                    t(l, "Inspect", "근거 보기", "فحص"),
                  ].map((h) => (
                    <th key={h} scope="col">
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
                      <br />
                      <SourceValue locale={l} value={r.revision} />
                      <p>
                        <bdi dir="auto">{r.target.label}</bdi>
                      </p>
                    </th>
                    <td>
                      {[
                        r.target.contract,
                        r.target.period,
                        r.target.component,
                      ].map((v, i) => (
                        <div key={i}>
                          <SourceValue locale={l} value={v} />
                        </div>
                      ))}
                    </td>
                    <td>{protectionLabel(l, r.protection)}</td>
                    <td>
                      <SourceValue locale={l} value={r.coverage.identity} />
                      {(
                        [
                          "reserved",
                          "applied",
                          "reversed",
                          "available",
                        ] as const
                      ).map((k, i) => (
                        <div key={k}>
                          {
                            [
                              t(l, "Reserved", "예약", "محجوز"),
                              t(l, "Applied", "적용", "مطبّق"),
                              t(l, "Reversed", "적용 취소", "معكوس"),
                              t(
                                l,
                                "Available (source)",
                                "가용 (원천 기록)",
                                "متاح في المصدر",
                              ),
                            ][i]
                          }
                          :{" "}
                          <CoverageValue
                            locale={l}
                            value={r.coverage[k]}
                            unit={r.coverage.unit}
                          />
                        </div>
                      ))}
                    </td>
                    <td>
                      {r.amounts.length
                        ? r.amounts.map((a) => (
                            <div key={a.currency + ":" + a.basis}>
                              <p>
                                {basisLabel(l, a.basis)}{" "}
                                <bdi dir="ltr">{a.currency}</bdi>
                              </p>
                              {(
                                ["applied", "remaining", "reversed"] as const
                              ).map((k, i) => (
                                <div key={k}>
                                  {
                                    [
                                      t(l, "Applied", "적용", "مطبّق"),
                                      t(l, "Remaining", "남은 금액", "متبقٍّ"),
                                      t(l, "Reversed", "적용 취소", "معكوس"),
                                    ][i]
                                  }
                                  : <SourceValue locale={l} value={a[k]} />
                                </div>
                              ))}
                            </div>
                          ))
                        : unknown(l)}
                    </td>
                    <td>
                      <SourceValue locale={l} value={r.lineage.submission} />
                      {r.external.map((x) => (
                        <p key={x.kind + ":" + x.id}>
                          {externalKind(l, x.kind)} <bdi dir="ltr">{x.id}</bdi>{" "}
                          {outcomeLabel(l, x.outcome)}
                        </p>
                      ))}
                    </td>
                    <td>
                      {dispositionLabel(l, r.correction.disposition)}
                      <p>
                        <SourceValue
                          locale={l}
                          value={r.correction.original_claim_id}
                        />
                      </p>
                      {r.correction.linked_claim_ids.map((id) => (
                        <div key={id}>
                          <bdi dir="ltr">{id}</bdi>
                        </div>
                      ))}
                    </td>
                    <td>
                      <button
                        data-b10-record={r.id}
                        onClick={() => select(r.id)}
                      >
                        {t(
                          l,
                          "Inspect application",
                          "적용 근거 보기",
                          "فحص التطبيق",
                        )}
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
