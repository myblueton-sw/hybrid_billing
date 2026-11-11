import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { OverviewData as Data } from "./api";
import {
  fieldLabel,
  missingLabel,
} from "../../../sources/collected-focus/labels";
import { t } from "./copy";
const range = (after: number, length: number, total: number) =>
  `${length ? after + 1 : 0}–${after + length} / ${total}`;
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
    <details className="b08-account-scope" ref={detail}>
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
        <bdi className="b08-count" dir="ltr">
          ({data.account_matches} / {data.account_count})
        </bdi>
      </summary>
      <p className="b08-meta">
        {t(
          l,
          "Search matches the account list only. Selecting an account narrows the source rows. A FOCUS account reference is not legal ownership.",
          "검색은 계정 목록에만 적용됩니다. 계정을 선택하면 원천 행 범위가 좁아집니다. FOCUS 계정 참조는 법적 소유 관계가 아닙니다.",
          "يؤثر البحث في قائمة الحسابات فقط. يضيّق اختيار الحساب نطاق صفوف المصدر. مرجع حساب FOCUS ليس ملكية قانونية.",
        )}
      </p>
      <button
        id="b08-all-accounts"
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
        <p className="b08-meta">
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
            "No accounts match this account search. Source totals are not filtered by the search text.",
            "계정 검색 조건에 맞는 계정이 없습니다. 검색어가 원천 합계를 필터링하지 않습니다.",
            "لا تطابق حسابات هذا البحث. نص البحث لا يرشّح إجماليات المصدر.",
          )}
        </p>
      ) : (
        <ul className="b08-account-list">
          {data.accounts.map((a) => (
            <li key={a.id}>
              <button
                data-b08-account={a.id}
                aria-pressed={data.selection.account === a.id}
                onClick={() => select(a.id)}
              >
                <span>
                  {fieldLabel(l, "billing_account")}:{" "}
                  <bdi dir="ltr">{a.billing_account}</bdi>
                </span>
                <span>
                  {fieldLabel(l, "provider")}: <bdi dir="ltr">{a.provider}</bdi>
                </span>
                <span>
                  {fieldLabel(l, "sub_account")}:{" "}
                  {a.sub_account === null ? (
                    missingLabel(l)
                  ) : (
                    <bdi dir="ltr">{a.sub_account}</bdi>
                  )}
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
      <div className="b08-pager">
        <button
          id="b08-account-prev"
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
          id="b08-account-next"
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
