import type { Locale } from "../shared/types";
import type { CandidateRow } from "./CandidateTable";
import { field, valueText } from "./fields";
import { local } from "./copy";
import CommandBoundary from "./CommandBoundary";
export default function Inbox({
  rows,
  current,
  keys,
  screen,
  locale: l,
  customer,
  onChoose,
}: {
  rows: CandidateRow[];
  current: CandidateRow;
  keys: string[];
  screen: string;
  locale: Locale;
  customer: boolean;
  onChoose: (id: string) => void;
}) {
  return (
    <div className="planning-inbox-layout">
      <ul className="planning-inbox">
        {rows.map((row) => (
          <li key={row.id}>
            <button
              aria-pressed={current.id === row.id}
              onClick={() => onChoose(row.id)}
            >
              <strong>
                <bdi>{row.id}</bdi> ·{" "}
                {valueText(
                  l,
                  row.values.kind ?? row.values.action ?? "pending",
                )}
              </strong>
              <span>
                {valueText(l, row.values.state ?? "pending")} ·{" "}
                <bdi>
                  {row.values.due ?? row.values.received ?? row.values.period}
                </bdi>
              </span>
              <small>
                {row.values.document ?? row.values.author ?? row.values.owner}
              </small>
            </button>
          </li>
        ))}
      </ul>
      <article className="planning-inbox-detail" aria-live="polite">
        <h3>
          <bdi>{current.id}</bdi>
        </h3>
        <p>
          {local(l, [
            "선택 항목 상세 · 합성 데이터",
            "Selected item detail · synthetic data",
            "تفاصيل العنصر المحدد · بيانات تجريبية",
          ])}
        </p>
        <dl>
          {[...new Set(keys)].map((k) => (
            <div key={k}>
              <dt>{field(l, k)}</dt>
              <dd>
                <bdi>
                  {valueText(
                    l,
                    k === "id"
                      ? current.id
                      : (current.values[k] ?? "unconfigured"),
                  )}
                </bdi>
              </dd>
            </div>
          ))}
        </dl>
        <CommandBoundary screen={screen} locale={l} customer={customer} />
      </article>
    </div>
  );
}
