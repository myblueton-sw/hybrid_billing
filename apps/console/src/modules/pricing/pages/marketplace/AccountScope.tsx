import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { Data } from "./contract";
import { fieldLabel as sourceLabel } from "../../../sources/collected-focus/labels";
import { SourceText } from "../../../sources/collected-focus/SourceValues";
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
    <details className="b07-account-scope" ref={detail}>
      <summary>
        {t(l, "Account scope", "계정 범위", "نطاق الحساب")} ·{" "}
        {data.selection.account
          ? t(
              l,
              "One account scope selected",
              "계정 범위 1개 선택",
              "نطاق حساب واحد محدد",
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
      <p className="b07-meta">
        {t(
          l,
          "Search matches the account list only. Selecting an account narrows the source rows. An account scope ID is derived by Hybrid Billing and does not establish legal ownership.",
          "검색은 계정 목록에만 적용됩니다. 계정을 선택하면 원천 행 범위가 좁아집니다. 계정 범위 ID는 Hybrid Billing 산출 값이며 법적 소유 관계가 아닙니다.",
          "يؤثر البحث في قائمة الحسابات فقط. يضيّق اختيار الحساب نطاق صفوف المصدر. معرّف نطاق الحساب يشتقه Hybrid Billing ولا يثبت ملكية قانونية.",
        )}
      </p>
      <button
        id="b07-all-accounts"
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
        <p className="b07-meta">
          {sourceLabel(l, "account")}:{" "}
          <bdi dir="ltr" data-b07-selected-account>
            {data.selection.account}
          </bdi>
        </p>
      )}
      {!data.account_matches ? (
        <p>
          {t(
            l,
            "No accounts match this account search. Candidate totals are not filtered by the search text.",
            "계정 검색 조건에 맞는 계정이 없습니다. 검색어가 후보 합계를 필터링하지 않습니다.",
            "لا تطابق حسابات هذا البحث. نص البحث لا يرشّح إجماليات المرشحين.",
          )}
        </p>
      ) : (
        <ul className="b07-account-list">
          {data.accounts.map((a) => (
            <li key={a.id}>
              <button
                data-b07-account={a.id}
                aria-pressed={data.selection.account === a.id}
                onClick={() => select(a.id)}
              >
                <span className="b07-account-field">
                  <span className="b07-account-label">
                    {sourceLabel(l, "billing_account")}
                  </span>{" "}
                  <span className="b07-account-value">
                    <bdi dir="ltr">{a.billing_account}</bdi>
                  </span>
                </span>{" "}
                <span className="b07-account-field">
                  <span className="b07-account-label">
                    {sourceLabel(l, "provider")}
                  </span>{" "}
                  <span className="b07-account-value">
                    <bdi dir="ltr">{a.provider}</bdi>
                  </span>
                </span>{" "}
                <span className="b07-account-field">
                  <span className="b07-account-label">
                    {sourceLabel(l, "sub_account")}
                  </span>{" "}
                  <span className="b07-account-value">
                    <SourceText value={a.sub_account} locale={l} />
                  </span>
                </span>{" "}
                <span className="b07-account-field">
                  <span className="b07-account-label">
                    {t(l, "Source rows", "원천 행", "صفوف المصدر")}
                  </span>{" "}
                  <span className="b07-account-value">
                    <bdi dir="ltr">{a.rows}</bdi>
                    {data.selection.account === a.id && (
                      <> · {t(l, "Selected", "선택됨", "محدد")}</>
                    )}
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="b07-pager">
        <button
          id="b07-account-prev"
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
          id="b07-account-next"
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
