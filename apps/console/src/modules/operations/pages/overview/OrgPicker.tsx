import { useState, useRef, useEffect } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  enterpriseLeaves,
  organizationResults,
  resolveOrganizations,
  encodeOrganizations,
  type OrganizationNode,
} from "./organization-scope";
import Paging, { usePage } from "./Paging";
function OrgRow({
  node,
  selected,
  onChange,
  drill,
  locale: l,
}: {
  node: OrganizationNode;
  selected: Set<string>;
  onChange: (ids: string[]) => void;
  drill: (id: string) => void;
  locale: Locale;
}) {
  const leaves = resolveOrganizations(true, node.id),
    count = leaves.filter((id) => selected.has(id)).length;
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current)
      ref.current.indeterminate = count > 0 && count < leaves.length;
  }, [count, leaves.length]);
  return (
    <div className="overview-org-row">
      <label>
        <input
          ref={ref}
          type="checkbox"
          checked={count === leaves.length}
          onChange={(e) => {
            const next = new Set(selected);
            for (const id of leaves) {
              if (e.target.checked) next.add(id);
              else next.delete(id);
            }
            onChange([...next]);
          }}
        />
        <span>
          <bdi>{node.id}</bdi>
          <small>
            {t(
              l,
              ...((node.depth === 1
                ? ["Company", "법인", "شركة"]
                : node.depth === 2
                  ? ["Division", "사업부", "قطاع"]
                  : ["Department", "부서", "قسم"]) as [string, string, string]),
            )}
          </small>
        </span>
      </label>
      <bdi dir="ltr">
        {count}/{leaves.length}
      </bdi>
      {node.depth < 3 && (
        <button
          onClick={() => drill(node.id)}
          aria-label={`${t(l, "Browse", "하위 보기", "تصفح")} ${node.id}`}
        >
          {t(l, "Browse", "하위 보기", "تصفح")}
        </button>
      )}
    </div>
  );
}
export default function OrgPicker({
  selected,
  onChange,
  locale: l,
}: {
  selected: string[];
  onChange: (value: string) => void;
  locale: Locale;
}) {
  const [parent, setParent] = useState(""),
    [query, setQuery] = useState("");
  const result = organizationResults(query, parent),
    page = usePage(
      result,
      parent + "|" + query + "|" + encodeOrganizations(selected),
    );
  const selectedSet = new Set(selected);
  const path = parent
    ? parent.split(".").map((_, i) =>
        parent
          .split(".")
          .slice(0, i + 1)
          .join("."),
      )
    : [];
  const browse = (id: string) => {
    setParent(id);
    setQuery("");
  };
  return (
    <details className="unified-picker overview-org-picker">
      <summary>
        {t(
          l,
          "Organizations · departments",
          "조직 · 부서",
          "المؤسسات · الأقسام",
        )}{" "}
        <bdi dir="ltr">{selected.length}/2400</bdi>
      </summary>
      <div className="overview-org-body">
        <div className="overview-org-actions">
          <button onClick={() => onChange("all")}>
            {t(
              l,
              "Select all organizations",
              "모든 조직 선택",
              "اختيار كل المؤسسات",
            )}
          </button>
          <button onClick={() => onChange("")}>
            {t(l, "Clear selection", "선택 해제", "مسح الاختيار")}
          </button>
        </div>
        <nav
          aria-label={t(l, "Organization path", "조직 경로", "مسار المؤسسة")}
        >
          <button onClick={() => browse("")}>
            {t(l, "Example group", "예제 그룹", "مجموعة تجريبية")}
          </button>
          {path.map((id) => (
            <button
              key={id}
              aria-current={id === parent ? "location" : undefined}
              onClick={() => browse(id)}
            >
              <bdi>{id}</bdi>
            </button>
          ))}
        </nav>
        <label>
          {t(
            l,
            "Search code within this path",
            "현재 경로에서 조직 코드 검색",
            "بحث بالرمز ضمن المسار",
          )}
          <input
            id="overview-org-search"
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </label>
        <p>
          {t(
            l,
            "Selecting a parent includes its departments. Search does not change selection.",
            "상위 조직 선택 시 하위 부서를 포함합니다. 검색은 선택 범위를 바꾸지 않습니다.",
            "اختيار الأصل يشمل أقسامه. البحث لا يغير الاختيار.",
          )}
        </p>
        <div className="overview-org-results">
          {page.rows.map((node) => (
            <OrgRow
              key={node.id}
              node={node}
              selected={selectedSet}
              onChange={(ids) => onChange(encodeOrganizations(ids))}
              drill={browse}
              locale={l}
            />
          ))}
        </div>
        {!result.length && (
          <p role="status">
            {t(
              l,
              "No matching organizations",
              "검색 결과가 없습니다",
              "لا توجد نتائج",
            )}
          </p>
        )}
        <Paging value={page} locale={l} />
      </div>
    </details>
  );
}
