import { useEffect, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { routeUrl } from "../../../../shared/navigation";
import { screens, groups } from "../../../../shared/catalog";
import { readRecords } from "../../../../shared/review-store";
export default function Screen({ locale: l, onState }: ScreenProps) {
  const [search, setSearch] = useState(""),
    [group, setGroup] = useState(""),
    [status, setStatus] = useState(""),
    [page, setPage] = useState(0);
  const [, refreshReviews] = useState(0);
  useEffect(() => {
    const update = () => refreshReviews((n) => n + 1);
    window.addEventListener("screen-review-saved", update);
    window.addEventListener("storage", update);
    return () => {
      window.removeEventListener("screen-review-saved", update);
      window.removeEventListener("storage", update);
    };
  }, []);
  let records: ReturnType<typeof readRecords> = [],
    error = false;
  try {
    records = readRecords(localStorage);
  } catch {
    error = true;
  }
  function state(id: string) {
    const found = records.filter((r) => r.screen === id);
    return error
      ? "error"
      : !found.length
        ? "pending"
        : found.some((r) => r.revision === __SOURCE_REVISION__)
          ? "reviewed"
          : "changed";
  }
  const matches = screens.filter(
    (s) =>
      (!search ||
        `${s.title[l]} ${s.id} ${s.registryIds.join(" ")}`
          .toLowerCase()
          .includes(search.toLowerCase())) &&
      (!group || s.group === group) &&
      (!status || state(s.id) === status),
  );
  const items = [
    "region:layout",
    "field:design-directory-search",
    "field:design-directory-group",
    "field:design-directory-status",
    "action:design-directory-prev",
    "action:design-directory-next",
    ...matches.slice(page * 24, (page + 1) * 24).map((s) => `navigate:${s.id}`),
  ].map((i) => `contents:${i}`);
  useEffect(
    () => onState(JSON.stringify({ search, group, status, page }), items),
    [search, group, status, page, items.join("|"), onState],
  );
  const labels = {
    pending: t(
      l,
      "Unreviewed current variant",
      "현재 표시 조건 미검수",
      "نسخة العرض الحالية غير مراجعة",
    ),
    reviewed: t(
      l,
      "Local review records",
      "로컬 검수 기록",
      "سجلات مراجعة محلية",
    ),
    changed: t(
      l,
      "Impact assessment pending",
      "변경 영향 검토 대기",
      "تقييم الأثر معلق",
    ),
    error: t(
      l,
      "Review storage unavailable",
      "검수 기록 확인 불가",
      "تخزين المراجعة غير متاح",
    ),
  };
  return (
    <>
      <div className="stats">
        <article className="stat">
          {t(l, "Screen designs", "화면 시안", "تصاميم الشاشات")}
          <strong>{screens.length}</strong>
        </article>
        <article className="stat">
          {t(
            l,
            "Screens with local review evidence",
            "로컬 검수 기록이 있는 화면",
            "شاشات لها أدلة مراجعة محلية",
          )}
          <strong>
            {error
              ? "—"
              : screens.filter((s) => records.some((r) => r.screen === s.id))
                  .length}
          </strong>
        </article>
      </div>
      <section className="panel">
        <h2>
          {t(
            l,
            "Screen design library",
            "화면 설계 목록",
            "مكتبة تصميم الشاشات",
          )}
        </h2>
        <p>
          {t(
            l,
            "Choose one screen, inspect its item IDs and visible states, then confirm only the scope you reviewed. No screen has been approved by the assistant.",
            "화면을 선택해 구현 항목 ID와 상태를 검토한 뒤, 확인한 범위만 검수하세요. 어시스턴트가 대신 확정한 화면은 없습니다.",
            "اختر شاشة وراجع معرّفات عناصرها وحالاتها ثم أكد النطاق الذي راجعته فقط. لم يعتمد المساعد أي شاشة.",
          )}
        </p>
        <div className="review-index-toolbar">
          <label className="grow">
            {t(
              l,
              "Find a screen by name or ID",
              "이름·ID로 화면 찾기",
              "البحث بالاسم أو المعرّف",
            )}
            <input
              id="design-directory-search"
              type="search"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(0);
              }}
            />
          </label>
          <label>
            {t(l, "Product area", "업무 영역", "مجال المنتج")}
            <select
              id="design-directory-group"
              value={group}
              onChange={(e) => {
                setGroup(e.target.value);
                setPage(0);
              }}
            >
              <option value="">{t(l, "All", "전체", "الكل")}</option>
              {groups.map((g) => (
                <option key={g.id} value={g.id}>
                  {g.label[l]}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t(l, "Review status", "검수 상태", "حالة المراجعة")}
            <select
              id="design-directory-status"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(0);
              }}
            >
              <option value="">{t(l, "All", "전체", "الكل")}</option>
              {(["pending", "reviewed", "changed"] as const).map((s) => (
                <option key={s} value={s}>
                  {labels[s]}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div id="design-directory-results" aria-live="polite">
          <div className="screen-index">
            {matches.slice(page * 24, (page + 1) * 24).map((s) => (
              <a key={s.id} href={routeUrl(s.id)}>
                <strong>{s.title[l]}</strong>
                <small>
                  <bdi>{s.id}</bdi> ·{" "}
                  {groups.find((g) => g.id === s.group)?.label[l]}
                </small>
                <small className="review-status">{labels[state(s.id)]}</small>
              </a>
            ))}
          </div>
          {!matches.length && (
            <p>
              {t(
                l,
                "No screen matches these filters.",
                "조건에 맞는 화면이 없습니다.",
                "لا توجد شاشة تطابق المرشحات.",
              )}
            </p>
          )}
        </div>
        <div className="pager">
          <span>
            {matches.length} · {page + 1} /{" "}
            {Math.max(1, Math.ceil(matches.length / 24))}
          </span>
          <div>
            <button
              id="design-directory-prev"
              disabled={!page}
              onClick={() => setPage(page - 1)}
            >
              {t(l, "Previous", "이전", "السابق")}
            </button>
            <button
              id="design-directory-next"
              disabled={(page + 1) * 24 >= matches.length}
              onClick={() => setPage(page + 1)}
            >
              {t(l, "Next", "다음", "التالي")}
            </button>
          </div>
        </div>
      </section>
    </>
  );
}
