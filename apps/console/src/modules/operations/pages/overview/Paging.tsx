import { useState, useEffect, useId } from "react";
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
    scope,
    rows: rows.slice(page * size, (page + 1) * size),
    page,
    total: rows.length,
    size,
    setPage: (p: number) =>
      setState({
        scope,
        page: Math.min(
          Math.max(0, Math.trunc(p)),
          Math.max(0, Math.ceil(rows.length / size) - 1),
        ),
      }),
  };
}
function PageJump({
  resetKey,
  page,
  count,
  onPage,
  locale: l,
}: {
  resetKey: string;
  page: number;
  count: number;
  onPage: (page: number) => void;
  locale: Locale;
}) {
  const [draft, setDraft] = useState(String(page + 1)),
    [error, setError] = useState(false);
  const errorId = useId();
  useEffect(() => {
    setDraft(String(page + 1));
    setError(false);
  }, [resetKey, page, count]);
  return (
    <form
      className="overview-page-jump"
      noValidate
      onSubmit={(e) => {
        e.preventDefault();
        const normalized = draft
          .trim()
          .replace(/[٠-٩]/g, (c) => String(c.charCodeAt(0) - 1632))
          .replace(/[۰-۹]/g, (c) => String(c.charCodeAt(0) - 1776));
        const n = Number(normalized);
        if (
          !/^\d+$/.test(normalized) ||
          !Number.isSafeInteger(n) ||
          n < 1 ||
          n > count
        ) {
          setError(true);
          return;
        }
        setError(false);
        onPage(n - 1);
      }}
    >
      <label>
        {t(l, "Page", "페이지", "الصفحة")}
        <input
          type="text"
          inputMode="numeric"
          dir="ltr"
          maxLength={9}
          value={draft}
          aria-invalid={error}
          aria-describedby={error ? errorId : undefined}
          onChange={(e) => {
            setDraft(e.target.value);
            setError(false);
          }}
        />
      </label>
      <bdi dir="ltr">/ {count}</bdi>
      <button type="submit">{t(l, "Go", "이동", "انتقال")}</button>
      {error && (
        <span id={errorId} role="alert">
          {t(
            l,
            "Enter a whole page number",
            "정수 페이지를 입력하세요",
            "أدخل رقم صفحة صحيحاً",
          )}{" "}
          <bdi dir="ltr">1–{count}</bdi>
        </span>
      )}
    </form>
  );
}
export default function Paging({
  value,
  locale: l,
  jump = true,
}: {
  jump?: boolean;
  value: {
    scope: string;
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
      {jump && total > size && (
        <PageJump
          resetKey={value.scope + "|" + total}
          page={page}
          count={Math.ceil(total / size)}
          onPage={setPage}
          locale={l}
        />
      )}
    </div>
  );
}
