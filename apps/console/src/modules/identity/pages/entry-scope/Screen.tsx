import { useEffect, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { entryUrl } from "../../entry-ui";
export default function Screen({ locale: l, mode, onState }: ScreenProps) {
  const [instance] = useState(() => crypto.randomUUID());
  const [search, setSearch] = useState(""),
    [selected, setSelected] = useState(""),
    [revision, setRevision] = useState(0);
  const choices = [
    {
      id: "sample-a",
      name: "DEMO Workspace A",
      entity: "DEMO Organization A",
      role: t(l, "Internal user", "내부 사용자", "مستخدم داخلي"),
    },
    {
      id: "sample-b",
      name: "DEMO Workspace B",
      entity: "DEMO Organization B",
      role: t(l, "Customer user", "고객 사용자", "مستخدم عميل"),
    },
  ];
  const rows =
    mode === "synthetic"
      ? choices.filter((c) =>
          `${c.name} ${c.entity}`.toLowerCase().includes(search.toLowerCase()),
        )
      : [];
  const picked = rows.find((c) => c.id === selected);
  useEffect(
    () =>
      onState(
        JSON.stringify({
          instance,
          mode,
          revision,
          selected,
          rows: rows.map((r) => r.id),
        }),
        [
          "region:layout",
          "field:workspace",
          "field:selection",
          "data:entity",
          "data:role",
          "data:scope",
          "action:preview-selection",
        ].map((i) => `entry-scope:${i}`),
      ),
    [mode, revision, selected, onState],
  );
  return (
    <section className="entry-card entry-scope">
      <h1>{t(l, "Choose a workspace", "업무 공간 선택", "اختر مساحة عمل")}</h1>
      <p>
        {t(
          l,
          "Preview the workspace you want to open. These examples are not verified memberships.",
          "열어볼 업무 공간을 선택하세요. 아래 예제는 실제 가입이나 권한을 나타내지 않습니다.",
          "اختر مساحة العمل لمعاينتها. الأمثلة لا تمثل عضويات موثقة.",
        )}
      </p>
      <label>
        {t(l, "Find a workspace", "업무 공간 찾기", "البحث عن مساحة عمل")}
        <input
          id="workspace-search"
          type="search"
          disabled={mode !== "synthetic"}
          value={search}
          onChange={(e) => {
            setSearch(e.target.value);
            setSelected("");
            setRevision((n) => n + 1);
          }}
        />
      </label>
      <div className="entry-workspaces">
        {rows.map((c) => (
          <label
            className={`entry-workspace ${selected === c.id ? "selected" : ""}`}
            key={c.id}
          >
            <input
              name="workspace-choice"
              type="radio"
              value={c.id}
              checked={selected === c.id}
              onChange={() => setSelected(c.id)}
            />
            <span>
              <strong>
                <bdi>{c.name}</bdi>
              </strong>
              <span>
                <bdi>{c.entity}</bdi>
              </span>
              <span>
                {c.role} · {t(l, "Synthetic scope", "합성 범위", "نطاق تجريبي")}
              </span>
            </span>
          </label>
        ))}
      </div>
      {!rows.length && (
        <p role="status">
          {mode === "synthetic"
            ? t(
                l,
                "No matching workspace. Clear your search to try again.",
                "일치하는 업무 공간이 없습니다. 검색어를 지우고 다시 확인하세요.",
                "لا توجد مساحة مطابقة. امسح البحث وحاول مجدداً.",
              )
            : t(
                l,
                "Membership service not connected. No authorized workspaces have been loaded.",
                "가입 정보 서비스 미연결로 허용된 업무 공간을 불러오지 않았습니다.",
                "خدمة العضوية غير متصلة. لم تُحمّل مساحات عمل مصرح بها.",
              )}
        </p>
      )}
      <div className="entry-selection">
        <h2>
          {t(
            l,
            "Selected workspace",
            "선택한 업무 공간",
            "مساحة العمل المختارة",
          )}
        </h2>
        <p>
          {picked ? (
            <bdi>{picked.name}</bdi>
          ) : (
            t(
              l,
              "Select a workspace above to continue.",
              "계속하려면 위에서 업무 공간을 선택하세요.",
              "اختر مساحة عمل أعلاه للمتابعة.",
            )
          )}
        </p>
        <button
          className="primary"
          disabled={!picked}
          onClick={() => {
            if (picked)
              location.hash =
                entryUrl(
                  picked.id === "sample-b"
                    ? "core-customer-overview"
                    : "my-tasks",
                  l,
                  "synthetic",
                ) + `&scope=${picked.id}`;
          }}
        >
          {t(
            l,
            "Open workspace preview",
            "업무 공간 화면 미리보기",
            "فتح معاينة مساحة العمل",
          )}
        </button>
        {picked?.id === "sample-a" && (
          <a href={entryUrl("setup-progress", l, mode) + "&scope=sample-a"}>
            {t(
              l,
              "First setup preview · implementation pending",
              "최초 설정 미리보기 · 구현 대기",
              "معاينة الإعداد الأول · التنفيذ معلق",
            )}
          </a>
        )}
      </div>
    </section>
  );
}
