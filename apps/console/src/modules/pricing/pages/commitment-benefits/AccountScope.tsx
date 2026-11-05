import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { Data } from "./contract";
import { t } from "./copy";
import { range } from "./values";
export default function AccountScope({
  locale: l,
  data,
  select,
  page,
}: {
  locale: Locale;
  data: Data;
  select: (id: string) => void;
  page: (n: number) => void;
}) {
  const detail = useRef<HTMLDetailsElement>(null);
  useEffect(() => {
    if ((data.selection.q || data.account_after > 0) && detail.current)
      detail.current.open = true;
  }, [data.selection.q, data.account_after]);
  return (
    <details className="b06-account-scope" ref={detail}>
      <summary>
        {t(l, "Account scope", "계정 범위", "نطاق الحساب")} ·{" "}
        {data.selection.account
          ? t(
              l,
              "One canonical account selected",
              "원천 계정 1개 선택",
              "حساب مرجعي واحد محدد",
            )
          : t(
              l,
              "All authorized accounts",
              "전체 허용 계정",
              "كل الحسابات المصرّح بها",
            )}{" "}
        <bdi dir="ltr">
          ({data.account_matches} / {data.account_count})
        </bdi>
      </summary>
      <p className="b06-meta">
        {t(
          l,
          "Search matches the account list only. Selecting an account narrows the source rows and groups. A FOCUS account reference is not legal ownership.",
          "검색은 계정 목록에만 적용됩니다. 계정을 선택하면 원천 행·그룹 범위가 좁아집니다. FOCUS 계정 참조는 법적 소유 관계가 아닙니다.",
          "يؤثر البحث في قائمة الحسابات فقط. يضيّق اختيار الحساب نطاق صفوف المصدر والمجموعات. مرجع حساب FOCUS ليس ملكية قانونية.",
        )}
      </p>
      <button
        id="b06-all-accounts"
        aria-pressed={!data.selection.account}
        onClick={() => select("")}
      >
        {t(
          l,
          "All authorized accounts",
          "전체 허용 계정",
          "كل الحسابات المصرّح بها",
        )}
      </button>
      {data.selection.account && (
        <p className="b06-meta">
          {t(
            l,
            "Selected canonical account reference",
            "선택 원천 계정 참조",
            "مرجع الحساب المحدد",
          )}
          : <bdi dir="ltr">{data.selection.account}</bdi>
        </p>
      )}
      {!data.account_matches ? (
        <p>
          {t(
            l,
            "No accounts match this account search. Group totals are not filtered by the search text.",
            "계정 검색 조건에 맞는 계정이 없습니다. 검색어가 그룹 합계를 필터링하지 않습니다.",
            "لا تطابق حسابات هذا البحث. نص البحث لا يرشّح إجماليات المجموعات.",
          )}
        </p>
      ) : (
        <ul className="b06-account-list">
          {data.accounts.map((a) => (
            <li key={a.id}>
              <button
                data-b06-account={a.id}
                aria-pressed={data.selection.account === a.id}
                onClick={() => select(a.id)}
              >
                <span>
                  {t(l, "Billing account", "청구 계정", "حساب الفوترة")}:{" "}
                  <bdi dir="ltr">{a.billing_account}</bdi>
                </span>
                <span>
                  {t(l, "Provider", "공급자", "المزوّد")}:{" "}
                  <bdi dir="ltr">{a.provider}</bdi>
                </span>
                <span>
                  {t(l, "Sub-account", "하위 계정", "الحساب الفرعي")}:{" "}
                  <bdi dir="ltr">
                    {a.sub_account ??
                      t(
                        l,
                        "None in source",
                        "원천 값 없음",
                        "لا يوجد في المصدر",
                      )}
                  </bdi>
                </span>
                <span>
                  {t(l, "Source rows", "원천 행", "صفوف المصدر")}:{" "}
                  <bdi dir="ltr">{a.rows}</bdi>
                  {data.selection.account === a.id && (
                    <> · {t(l, "Selected", "선택됨", "محدد")}</>
                  )}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="b06-pager">
        <button
          id="b06-account-prev"
          disabled={data.account_after === 0}
          onClick={() => page(Math.max(0, data.account_after - 25))}
        >
          {t(l, "Previous accounts", "이전 계정", "الحسابات السابقة")}
        </button>
        <bdi dir="ltr">
          {range(
            data.account_after,
            data.accounts.length,
            data.account_matches,
          )}
        </bdi>
        <button
          id="b06-account-next"
          disabled={data.next_account_after === null}
          onClick={() =>
            data.next_account_after !== null && page(data.next_account_after)
          }
        >
          {t(l, "Next accounts", "다음 계정", "الحسابات التالية")}
        </button>
      </div>
    </details>
  );
}
