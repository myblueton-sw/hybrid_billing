import { useEffect, useRef, useState } from "react";
import type { Locale } from "../shared/types";
import { text as t } from "../shared/copy";
const accounts = [
  { value: "sample-a", name: "ACC-000001" },
  { value: "sample-b", name: "ACC-000002" },
];
export default function AccountPicker({
  locale: l,
  value,
  onChange,
}: {
  locale: Locale;
  value: string;
  onChange: (value: string) => void;
}) {
  const [search, setSearch] = useState("");
  const searchInput = useRef<HTMLInputElement>(null);
  const [active, setActive] = useState(value);
  useEffect(() => {
    searchInput.current?.focus();
  }, []);
  const matches = accounts.filter((a) =>
    a.name.toLowerCase().includes(search.toLowerCase()),
  );
  return (
    <div className="header-account-picker">
      <label>
        {t(l, "Find an account", "계정 검색", "البحث عن حساب")}
        <input
          id="header-account-search"
          ref={searchInput}
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          autoComplete="off"
        />
      </label>
      <p className="muted">
        {t(
          l,
          "Two synthetic accounts only. Production account search is not connected.",
          "합성 계정 두 개만 표시합니다. 운영 계정 검색은 연결되지 않았습니다.",
          "حسابان اصطناعيان فقط. البحث في حسابات الإنتاج غير متصل.",
        )}
      </p>
      <div
        className="picker-list"
        role="listbox"
        aria-label={t(l, "Accounts", "계정 목록", "الحسابات")}
        onKeyDown={(e) => {
          if (["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) {
            e.preventDefault();
            const nodes = [
              ...e.currentTarget.querySelectorAll<HTMLButtonElement>(
                "[role=option]",
              ),
            ];
            const current = nodes.indexOf(
              document.activeElement as HTMLButtonElement,
            );
            const next =
              e.key === "Home"
                ? 0
                : e.key === "End"
                  ? nodes.length - 1
                  : (current +
                      (e.key === "ArrowDown" ? 1 : -1) +
                      nodes.length) %
                    nodes.length;
            nodes[next]?.focus();
          }
        }}
      >
        {matches.map((a) => (
          <button
            key={a.value}
            type="button"
            role="option"
            tabIndex={
              (matches.some((x) => x.value === active)
                ? active
                : matches[0]?.value) === a.value
                ? 0
                : -1
            }
            onFocus={() => setActive(a.value)}
            aria-selected={a.value === value}
            onClick={() => onChange(a.value)}
          >
            <bdi>{a.name}</bdi>
            <span>
              {a.value === value
                ? t(l, "Selected", "선택됨", "محدد")
                : t(l, "Choose", "선택", "اختيار")}
            </span>
          </button>
        ))}
      </div>
      {!matches.length && (
        <p role="status">
          {t(
            l,
            "No matching account",
            "일치하는 계정이 없습니다",
            "لا يوجد حساب مطابق",
          )}
        </p>
      )}
    </div>
  );
}
