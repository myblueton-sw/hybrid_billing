import { useState, useEffect } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
export function usePage<T>(rows: T[], scope: string, size = 20) {
  const [state, setState] = useState({ scope, page: 0 });
  useEffect(() => {
    setState({ scope, page: 0 });
  }, [scope]);
  const page = Math.min(
    state.scope === scope ? state.page : 0,
    Math.max(0, Math.ceil(rows.length / size) - 1),
  );
  return {
    rows: rows.slice(page * size, (page + 1) * size),
    page,
    total: rows.length,
    size,
    setPage: (p: number) => setState({ scope, page: Math.max(0, p) }),
  };
}
export default function Paging({
  value,
  locale: l,
}: {
  value: {
    page: number;
    total: number;
    size: number;
    setPage: (p: number) => void;
  };
  locale: Locale;
}) {
  const { page, total, size, setPage } = value;
  return (
    <div className="overview-paging">
      <span aria-live="polite">
        <bdi dir="ltr">
          {total ? page * size + 1 : 0}–{Math.min((page + 1) * size, total)} /{" "}
          {total}
        </bdi>{" "}
        {t(l, "results", "건", "نتائج")}
      </span>
      <button disabled={page === 0} onClick={() => setPage(page - 1)}>
        {t(l, "Previous", "이전", "السابق")}
      </button>
      <button
        disabled={(page + 1) * size >= total}
        onClick={() => setPage(page + 1)}
      >
        {t(l, "Next", "다음", "التالي")}
      </button>
    </div>
  );
}
