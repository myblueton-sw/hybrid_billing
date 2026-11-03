import type { Locale } from "../shared/types";
import type { PlanningScreen } from "../planning/registry";
import { local, titles, menuNames } from "../planning/copy";
import { planningUrl } from "../planning/routing";
export default function ServiceNavigation({
  locale,
  entries,
  active,
  query,
  onNavigate,
  developer,
}: {
  locale: Locale;
  entries: PlanningScreen[];
  active?: string;
  query: URLSearchParams;
  onNavigate: (path: string) => void;
  developer: boolean;
}) {
  const groups = Object.keys(menuNames).filter(
    (k) => !["pre", "portal", "me"].includes(k),
  );
  const render = (s: PlanningScreen) => (
    <a
      key={s.id}
      href={planningUrl(s.path, query)}
      aria-current={s.id === active ? "page" : undefined}
      onClick={(e) => {
        e.preventDefault();
        onNavigate(s.path);
      }}
    >
      <span>
        {s.id === "U01"
          ? local(locale, [
              "통합 빌링 현황",
              "Billing overview",
              "نظرة عامة على الفوترة",
            ])
          : local(locale, titles[s.id])}
      </span>
      {developer && <small>{s.id}</small>}
    </a>
  );
  return (
    <nav
      className="service-navigation"
      aria-label={local(locale, [
        "서비스 메뉴",
        "Service navigation",
        "تنقل الخدمة",
      ])}
    >
      <div className="service-primary">
        {entries.filter((s) => s.menu === "home").map(render)}
      </div>
      {groups
        .filter((k) => k !== "home")
        .map((key) => {
          const pages = entries.filter((s) => s.menu === key);
          return pages.length ? (
            <details
              key={key + ":" + active}
              open={developer || pages.some((s) => s.id === active)}
            >
              <summary>{local(locale, menuNames[key])}</summary>
              {pages.map(render)}
            </details>
          ) : null;
        })}
    </nav>
  );
}
