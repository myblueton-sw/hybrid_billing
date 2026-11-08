import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { stages, type Closing, type Stage, type Status } from "./model";
import { organizationName, stageName, statusName } from "./copy";
import Paging, { usePage } from "./Paging";
import {
  filterClosings,
  emptyClosingFilter,
  type ClosingFilter,
} from "./closing-filter";
export default function Lifecycle({
  rows,
  scope,
  locale: l,
  selected,
  onSelect,
  onFilterChange,
}: {
  rows: Closing[];
  scope: string;
  locale: Locale;
  selected: string;
  onFilterChange: (filter: ClosingFilter) => void;
  onSelect: (row: Closing, stage: Stage) => void;
}) {
  const [filter, setFilter] = useState(emptyClosingFilter);
  const filtered = filterClosings(rows, filter);
  const page = usePage(filtered, scope + "|" + JSON.stringify(filter));
  const change = (value: ClosingFilter) => {
    setFilter(value);
    onFilterChange(value);
  };
  return (
    <section className="unified-panel unified-lifecycle">
      <header>
        <div>
          <h2>
            {t(
              l,
              "Selected month · closing status",
              "선택한 월의 정산 처리 현황",
              "الشهر المحدد · حالة الإقفال",
            )}
          </h2>
          <p>
            {t(
              l,
              "Whole closing units intersecting selected sources. Each row covers all its sources, not just the filtered source.",
              "선택 원천을 포함하는 정산 단위입니다. 상태는 선택 원천만이 아닌 각 정산 건의 전체 원천 기준입니다.",
              "وحدات إقفال تشمل المصادر المختارة. تغطي حالة كل صف جميع مصادره وليس المصدر المصفى فقط.",
            )}
          </p>
        </div>
        <span>
          <bdi dir="ltr" data-list-count>
            {filtered.length} / {rows.length}
          </bdi>{" "}
          {t(l, "units", "정산 건", "وحدات")}
        </span>
      </header>
      <p className="unified-note">
        {t(
          l,
          "Stages are independent. Accounting, delivery and receipt may finish in different periods. Scroll horizontally for all stages.",
          "단계는 독립 상태입니다. 회계·전달·수납은 다른 기간에 끝날 수 있습니다. 전체 단계는 가로로 스크롤하세요.",
          "المراحل مستقلة وقد تكتمل المحاسبة والتسليم والتحصيل في فترات مختلفة. مرر أفقياً لعرض جميع المراحل.",
        )}
      </p>
      <form
        className="overview-closing-filters"
        role="search"
        aria-label={t(
          l,
          "Find closing units",
          "정산 건 찾기",
          "البحث عن وحدات الإقفال",
        )}
        onSubmit={(e) => e.preventDefault()}
      >
        <label>
          {t(l, "Organization code", "조직 코드", "رمز المؤسسة")}
          <input
            id="closing-search"
            type="search"
            value={filter.query}
            onChange={(e) => change({ ...filter, query: e.target.value })}
          />
        </label>
        <label>
          {t(l, "Stage to check", "확인 단계", "المرحلة المطلوبة")}
          <select
            id="closing-stage"
            value={filter.stage}
            onChange={(e) =>
              change({ ...filter, stage: e.target.value as Stage | "" })
            }
          >
            <option value="">
              {t(
                l,
                "Any matching stage",
                "일치하는 단계 하나 이상",
                "أي مرحلة مطابقة",
              )}
            </option>
            {stages.map((stage) => (
              <option key={stage} value={stage}>
                {stageName(l, stage)}
              </option>
            ))}
          </select>
        </label>
        <label>
          {t(l, "Stage status", "단계 상태", "حالة المرحلة")}
          <select
            id="closing-status"
            value={filter.status}
            onChange={(e) =>
              change({ ...filter, status: e.target.value as Status | "" })
            }
          >
            <option value="">
              {t(l, "All statuses", "모든 상태", "كل الحالات")}
            </option>
            {(["complete", "pending", "blocked", "unknown", "na"] as const).map(
              (status) => (
                <option key={status} value={status}>
                  {statusName(l, status)}
                </option>
              ),
            )}
          </select>
        </label>
        <button type="button" onClick={() => change(emptyClosingFilter)}>
          {t(
            l,
            "Reset list filters",
            "목록 필터 초기화",
            "إعادة تعيين عوامل التصفية",
          )}
        </button>
      </form>
      <p className="unified-note">
        {t(
          l,
          "Filters narrow this list only; financial totals are unchanged. A matching stage does not mean the entire closing unit is complete.",
          "목록만 필터링하며 통합 금액은 바뀌지 않습니다. 한 단계의 완료는 정산 전체 완료를 뜻하지 않습니다.",
          "تصفية القائمة فقط دون تغيير الإجماليات المالية. اكتمال مرحلة لا يعني اكتمال وحدة الإقفال.",
        )}
      </p>
      {!filtered.length && (
        <p role="status">
          {t(
            l,
            "No units match these filters",
            "조건에 맞는 정산 건이 없습니다",
            "لا توجد وحدات مطابقة",
          )}
        </p>
      )}
      <div
        className="unified-matrix"
        role="region"
        tabIndex={0}
        aria-label={t(
          l,
          "Closing stages, horizontally scrollable",
          "정산 단계, 가로 스크롤",
          "مراحل الإقفال، تمرير أفقي",
        )}
      >
        <table>
          <thead>
            <tr>
              <th>
                {t(
                  l,
                  "Organization / contract path",
                  "조직 / 계약 경로",
                  "المؤسسة / مسار العقد",
                )}
              </th>
              {stages.map((s) => (
                <th key={s}>{stageName(l, s)}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {page.rows.map((r) => (
              <tr key={r.id}>
                <th>
                  <strong>{organizationName(l, r.organization)}</strong>
                  <span>
                    <bdi>{r.currency}</bdi> ·{" "}
                    {r.internal
                      ? t(l, "Internal", "내부 정산", "داخلي")
                      : t(l, "External", "외부 청구", "خارجي")}
                  </span>
                  <small>
                    <bdi>{r.sources.join(" · ")}</bdi>
                  </small>
                </th>
                {stages.map((stage) => (
                  <td key={stage}>
                    <button
                      className={`stage-${r.status[stage]}`}
                      aria-pressed={selected === r.id + ":" + stage}
                      onClick={() => onSelect(r, stage)}
                    >
                      {statusName(l, r.status[stage])}
                    </button>
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Paging value={page} locale={l} />
    </section>
  );
}
