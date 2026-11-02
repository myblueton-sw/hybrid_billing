import { useState } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { total, formatMoney, type Cost } from "./model";
import { organizationName } from "./copy";
import Paging, { usePage } from "./Paging";
export default function OrgBreakdown({
  rows,
  enterprise,
  scope,
  locale: l,
  onSelect,
}: {
  rows: Cost[];
  enterprise: boolean;
  scope: string;
  locale: Locale;
  onSelect: (id: string) => void;
}) {
  const [level, setLevel] = useState(1);
  const groups = new Map<string, Cost[]>();
  for (const row of rows) {
    const id = enterprise
      ? row.organization.split(".").slice(0, level).join(".")
      : row.organization;
    const group = groups.get(id) ?? [];
    group.push(row);
    groups.set(id, group);
  }
  const page = usePage([...groups], scope + "|" + level);
  return (
    <section className="unified-panel overview-org-breakdown">
      <header>
        <h2>
          {t(l, "Organization composition", "조직별 구성", "توزيع المؤسسات")}
        </h2>
        {enterprise && (
          <label>
            {t(l, "Level", "구분 수준", "المستوى")}
            <select
              id="overview-org-level"
              value={level}
              onChange={(e) => setLevel(Number(e.target.value))}
            >
              <option value="1">{t(l, "Company", "법인", "شركة")}</option>
              <option value="2">{t(l, "Division", "사업부", "قطاع")}</option>
              <option value="3">{t(l, "Department", "부서", "قسم")}</option>
            </select>
          </label>
        )}
      </header>
      <p className="unified-note">
        {t(
          l,
          "Selected leaf costs only · no consolidation or eliminations. Paging does not change totals.",
          "선택한 하위 조직 비용 합계 · 연결 조정·내부 거래 제거 없음. 페이지 이동은 합계를 바꾸지 않습니다.",
          "تكاليف الفروع المختارة فقط · دون توحيد أو استبعادات. الصفحات لا تغير الإجماليات.",
        )}
      </p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{t(l, "Organization", "조직", "المؤسسة")}</th>
              <th>
                {t(
                  l,
                  enterprise
                    ? "Included departments"
                    : "Included organizations",
                  enterprise ? "포함 부서" : "포함 조직",
                  enterprise ? "الأقسام المشمولة" : "المؤسسات المشمولة",
                )}
              </th>
              <th>USD</th>
              <th>KRW</th>
            </tr>
          </thead>
          <tbody>
            {page.rows.map(([id, items]) => (
              <tr key={id}>
                <th>
                  <button onClick={() => onSelect(id)}>
                    {organizationName(l, id)}
                  </button>
                </th>
                <td>{new Set(items.map((r) => r.organization)).size}</td>
                {(["USD", "KRW"] as const).map((c) => {
                  const sum = total(items, c);
                  return (
                    <td key={c}>
                      <bdi>{formatMoney(sum.value, c, l)}</bdi>
                      {sum.missing > 0 && (
                        <small>
                          {t(l, "Missing", "누락", "مفقود")} {sum.missing}
                        </small>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Paging value={page} locale={l} />
    </section>
  );
}
