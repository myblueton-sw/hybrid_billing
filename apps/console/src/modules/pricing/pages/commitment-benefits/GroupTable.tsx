import { fieldLabel as sourceLabel } from "../../../sources/collected-focus/labels";
import { SourceText } from "../../../sources/collected-focus/SourceValues";
import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { Data, Group, Measure, Metadata } from "./contract";
import {
  t,
  costLabel,
  statusLabel,
  noRows,
  noValuesPresent,
  fieldLabel,
} from "./copy";
import { money, range } from "./values";
export function MeasureValue({
  locale: l,
  value,
  currency,
  rows,
  hook,
}: {
  locale: Locale;
  value: Measure;
  currency: string;
  rows: number;
  hook?: string;
}) {
  return (
    <span className="b06-measure" data-b06-cost={hook}>
      <span>
        {rows === 0 ? (
          noRows(l)
        ) : value.coefficient === null ? (
          noValuesPresent(l)
        ) : (
          <bdi dir="ltr">{money(value.coefficient, currency, l)}</bdi>
        )}
      </span>
      {rows > 0 && (
        <small>
          {value.missing > 0 && value.present > 0 && (
            <span>
              {t(
                l,
                "Known partial sum",
                "확인된 값의 부분 합계",
                "مجموع جزئي للقيم المعلومة",
              )}{" "}
              ·{" "}
            </span>
          )}
          {t(l, "Present / missing", "값 있음 / 누락", "موجود / مفقود")}:{" "}
          <bdi dir="ltr">
            {value.present} / {value.missing}
          </bdi>
        </small>
      )}
    </span>
  );
}
export function MetadataValue({
  locale: l,
  value: m,
}: {
  locale: Locale;
  value: Metadata;
}) {
  return (
    <span className="b06-metadata">
      {m.values.length ? (
        <span className="b06-values">
          {m.values.map((v) => (
            <bdi key={v} dir="ltr">
              {v}
            </bdi>
          ))}
        </span>
      ) : (
        noValuesPresent(l)
      )}
      {m.distinct > 1 && (
        <small>
          {t(
            l,
            "Distinct observed values",
            "서로 다른 관측 값",
            "قيم مرصودة متميزة",
          )}
          : <bdi dir="ltr">{m.distinct}</bdi> ·{" "}
          {t(
            l,
            "Variation, not an automatic conflict",
            "값 차이이며 자동 충돌 판정 아님",
            "تنوع وليس تعارضاً تلقائياً",
          )}
        </small>
      )}
      {m.truncated && (
        <small>
          {t(
            l,
            "Display capped; full distinct count retained",
            "표시 개수 제한 · 전체 종류 수는 유지",
            "العرض محدود؛ عدد القيم المتميزة الكامل محفوظ",
          )}
        </small>
      )}
    </span>
  );
}
export function ScopeIdentity({
  locale: l,
  group: g,
}: {
  locale: Locale;
  group: Group;
}) {
  return (
    <span className="b06-scope-identity">
      <span>
        {sourceLabel(l, "provider")}: <bdi dir="ltr">{g.provider}</bdi>
      </span>
      <span>
        {sourceLabel(l, "billing_account")}:{" "}
        <bdi dir="ltr">{g.billing_account}</bdi>
      </span>
      <span>
        {sourceLabel(l, "sub_account")}:{" "}
        <SourceText value={g.sub_account} locale={l} />
      </span>
      <span>
        {t(l, "Currency", "통화", "العملة")}: <bdi dir="ltr">{g.currency}</bdi>
      </span>
    </span>
  );
}
export default function GroupTable({
  locale: l,
  data,
  view,
  select,
  page,
}: {
  locale: Locale;
  data: Data;
  view: string;
  select: (id: string) => void;
  page: (n: number) => void;
}) {
  const common: [string, (g: Group) => ReactNode][] = [
    [
      t(
        l,
        "Source commitment ID / scope",
        "원천 약정 ID / 범위",
        "معرّف الالتزام في المصدر / النطاق",
      ),
      (g) => (
        <>
          <bdi className="b06-source-id" dir="ltr">
            {g.source_id}
          </bdi>
          <ScopeIdentity locale={l} group={g} />
        </>
      ),
    ],
  ];
  const columns: [string, (g: Group) => ReactNode][] =
    view === "utilization"
      ? [
          ...common,
          ...(["used", "unused", "unknown"] as const).map(
            (s) =>
              [
                statusLabel(l, s),
                (g: Group) => <bdi dir="ltr">{g.statuses[s].rows}</bdi>,
              ] as [string, (g: Group) => ReactNode],
          ),
        ]
      : view === "benefits"
        ? [
            ...common,
            [
              fieldLabel(l, "type"),
              (g) => <MetadataValue locale={l} value={g.metadata.type} />,
            ],
            [
              fieldLabel(l, "category"),
              (g) => <MetadataValue locale={l} value={g.metadata.category} />,
            ],
            [
              t(l, "Source rows", "원천 행", "صفوف المصدر"),
              (g) => <bdi dir="ltr">{g.rows}</bdi>,
            ],
          ]
        : view === "reconciliation"
          ? [
              ...common,
              [
                t(
                  l,
                  "Version-mapped / unmapped rows",
                  "버전 매핑 / 미매핑 행",
                  "صفوف بإصدار مربوط / غير مربوط",
                ),
                (g) => (
                  <bdi dir="ltr">
                    {g.mapping.version_mapped_rows} /{" "}
                    {g.mapping.version_unmapped_rows}
                  </bdi>
                ),
              ],
              [
                t(l, "Source versions", "원천 버전", "إصدارات المصدر"),
                (g) => (
                  <span className="b06-values">
                    {g.versions.map((v) => (
                      <bdi key={v} dir="ltr">
                        {v}
                      </bdi>
                    ))}
                  </span>
                ),
              ],
              [
                fieldLabel(l, "charge_category"),
                (g) => (
                  <MetadataValue
                    locale={l}
                    value={g.metadata.charge_category}
                  />
                ),
              ],
            ]
          : [
              ...common,
              [
                t(l, "Source rows", "원천 행", "صفوف المصدر"),
                (g) => <bdi dir="ltr">{g.rows}</bdi>,
              ],
              ...(["billed_cost", "effective_cost"] as const).map(
                (k) =>
                  [
                    costLabel(l, k),
                    (g: Group) => (
                      <MeasureValue
                        locale={l}
                        value={g[k]}
                        rows={g.rows}
                        currency={g.currency}
                      />
                    ),
                  ] as [string, (g: Group) => ReactNode],
              ),
            ];
  const table = (selection: boolean) => (
    <table>
      <thead>
        <tr>
          {selection && <th>{t(l, "Inspect", "상세 보기", "فحص")}</th>}
          {columns.map(([h]) => (
            <th key={h}>{h}</th>
          ))}
        </tr>
      </thead>
      <tbody>
        {data.groups.map((g) => (
          <tr
            key={g.id}
            aria-current={data.selected?.group === g.id ? "true" : undefined}
          >
            {selection && (
              <td>
                <button
                  data-b06-group={g.id}
                  aria-pressed={data.selected?.group === g.id}
                  onClick={() => select(g.id)}
                >
                  {data.selected?.group === g.id
                    ? t(l, "Selected", "선택됨", "محدد")
                    : t(l, "Inspect group", "그룹 상세 보기", "فحص المجموعة")}
                </button>
              </td>
            )}
            {columns.map(([h, render]) => (
              <td key={h} data-label={h}>
                {render(g)}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
  return (
    <section className="b06-groups" id="b06-group-list" tabIndex={-1}>
      <h2>
        {t(
          l,
          "Observed commitment ID groups",
          "관측된 약정 ID 그룹",
          "مجموعات معرّفات الالتزام المرصودة",
        )}
      </h2>
      <p className="b06-meta">
        <bdi dir="ltr">{data.group_count}</bdi>{" "}
        {t(
          l,
          "scoped source groups · at most 25 per page",
          "범위 내 원천 그룹 · 페이지당 최대 25개",
          "مجموعات مصدر ضمن النطاق · حتى 25 لكل صفحة",
        )}
      </p>
      {data.group_count === 0 ? (
        <p>
          {data.row_count === 0
            ? t(
                l,
                "No collected rows in this selection. Source data may not have arrived.",
                "이 조회 범위에 수집된 원천 행이 없습니다. 자료 미도착일 수 있습니다.",
                "لا توجد صفوف مجمّعة في هذا الاختيار. قد لا تكون البيانات قد وصلت.",
              )
            : t(
                l,
                "No commitment identifiers were observed in these source rows. Unidentified evidence is shown separately.",
                "원천 행에서 약정 식별자가 관측되지 않았습니다. ID 미식별 근거는 별도로 표시합니다.",
                "لم تُرصد معرّفات التزام في صفوف المصدر هذه. تُعرض الأدلة غير المحددة منفصلة.",
              )}
        </p>
      ) : (
        <>
          <div
            className="b06-group-table b06-scroll"
            role="region"
            tabIndex={0}
            aria-label={t(
              l,
              "Commitment source groups",
              "약정 원천 그룹",
              "مجموعات مصدر الالتزام",
            )}
          >
            {table(true)}
          </div>
          <details className="b06-mobile-columns">
            <summary>
              {t(
                l,
                "All columns — scroll horizontally",
                "모든 열 보기 — 가로 스크롤",
                "كل الأعمدة — تمرير أفقي",
              )}
            </summary>
            <p className="b06-meta">
              {t(
                l,
                "Scroll horizontally to inspect all source columns.",
                "가로로 스크롤하여 전체 원천 열을 확인하세요.",
                "مرّر أفقياً لفحص كل أعمدة المصدر.",
              )}
            </p>
            <div
              className="b06-scroll"
              role="region"
              tabIndex={0}
              aria-label={t(
                l,
                "All commitment group columns",
                "약정 그룹 전체 열",
                "كل أعمدة مجموعات الالتزام",
              )}
            >
              {table(false)}
            </div>
          </details>
        </>
      )}
      <div className="b06-pager">
        <button
          id="b06-prev"
          disabled={data.after === 0}
          onClick={() => page(Math.max(0, data.after - 25))}
        >
          {t(l, "Previous", "이전", "السابق")}
        </button>
        <bdi dir="ltr">
          {range(data.after, data.groups.length, data.group_count)}
        </bdi>
        <button
          id="b06-next"
          disabled={data.next_after === null}
          onClick={() => data.next_after !== null && page(data.next_after)}
        >
          {t(l, "Next", "다음", "التالي")}
        </button>
      </div>
    </section>
  );
}
