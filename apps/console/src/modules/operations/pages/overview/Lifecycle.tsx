import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { stages, type Closing, type Stage } from "./model";
import { organizationName, stageName, statusName } from "./copy";
export default function Lifecycle({
  rows,
  locale: l,
  selected,
  onSelect,
}: {
  rows: Closing[];
  locale: Locale;
  selected: string;
  onSelect: (row: Closing, stage: Stage) => void;
}) {
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
          {rows.length} {t(l, "units", "정산 건", "وحدات")}
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
            {rows.map((r) => (
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
    </section>
  );
}
