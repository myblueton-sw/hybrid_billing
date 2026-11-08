import type { Locale } from "../../../../shared/types";
import type { OverviewData } from "../../../sources/collected-focus/api";
import { t } from "./copy";
export default function AccountScope({
  locale: l,
  data,
  select,
  page,
}: {
  locale: Locale;
  data: OverviewData;
  select: (id: string) => void;
  page: (after: number) => void;
}) {
  return (
    <details
      className="b05-account-scope"
      key={data.selection.q + data.selection.account}
      open={!!data.selection.q}
    >
      <summary>
        {t(l, "Account scope", "계정 범위", "نطاق الحساب")} ·{" "}
        <bdi dir="ltr">
          {data.selection.account
            ? (data.accounts.find((a) => a.id === data.selection.account)
                ?.billing_account ??
              t(l, "Selected account", "선택 계정", "الحساب المحدد"))
            : t(
                l,
                "All authorized accounts",
                "전체 허용 계정",
                "كل الحسابات المصرّح بها",
              )}
        </bdi>
      </summary>
      <p className="b05-meta">
        {t(
          l,
          "Search filters this account list only. Select an account to narrow the source rows; a search alone does not narrow them.",
          "검색은 이 계정 목록에만 적용됩니다. 계정을 선택하면 원천 행의 범위가 좁혀집니다. 검색만으로 행 범위가 바뀌지 않습니다.",
          "يصفّي البحث قائمة الحسابات فقط. اختر حساباً لتضييق صفوف المصدر؛ البحث وحده لا يغيّر نطاقها.",
        )}
      </p>
      <button id="fx-all-accounts" onClick={() => select("")}>
        {t(
          l,
          "All authorized accounts",
          "전체 허용 계정",
          "كل الحسابات المصرّح بها",
        )}
      </button>
      <p className="b05-meta">
        <bdi dir="ltr">{data.account_matches}</bdi>{" "}
        {t(
          l,
          "matching accounts · at most 25 per page",
          "검색된 계정 · 페이지당 최대 25개",
          "حساباً مطابقاً · حتى 25 لكل صفحة",
        )}
      </p>
      <div className="b05-account-list" data-fx-account-results="true">
        {data.accounts.length ? (
          data.accounts.map((a) => (
            <button
              key={a.id}
              data-fx-account={a.id}
              aria-pressed={data.selection.account === a.id}
              onClick={() => select(a.id)}
            >
              <bdi dir="ltr">
                {a.billing_account}
                {a.sub_account ? " / " + a.sub_account : ""} · {a.provider}
              </bdi>
            </button>
          ))
        ) : (
          <p>
            {t(
              l,
              "No accounts match this search. The source-row scope is unchanged.",
              "검색에 맞는 계정이 없습니다. 원천 행의 범위는 변경되지 않았습니다.",
              "لا توجد حسابات مطابقة. نطاق صفوف المصدر لم يتغيّر.",
            )}
          </p>
        )}
      </div>
      <div className="b05-pager">
        <button
          disabled={data.account_after === 0}
          onClick={() => page(Math.max(0, data.account_after - 25))}
        >
          {t(l, "Previous accounts", "이전 계정", "الحسابات السابقة")}
        </button>
        <bdi dir="ltr">
          {data.accounts.length ? data.account_after + 1 : 0}–
          {data.account_after + data.accounts.length} / {data.account_matches}
        </bdi>
        <button
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
