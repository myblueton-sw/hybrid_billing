import { useEffect, useState } from "react";
import type { Locale } from "../shared/types";
import type { PlanningScreen } from "../planning/registry";
import { local, titles } from "../planning/copy";
import { planningUrl } from "../planning/routing";
import {
  businessSections,
  administrationSection,
  dataSection,
  parentFor,
  navigationScreenIds,
  navigationLabels,
  type ServiceSection,
} from "./service-ia";
export default function ServiceNavigation({
  locale,
  entries,
  active,
  query,
  onNavigate,
  developer,
  activeView,
}: {
  locale: Locale;
  entries: PlanningScreen[];
  active?: string;
  query: URLSearchParams;
  onNavigate: (path: string, changes?: Record<string, string>) => void;
  developer: boolean;
  activeView?: string;
}) {
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  useEffect(() => {
    const group = [
      ...businessSections,
      dataSection,
      administrationSection,
    ].find((s) => s.screens.includes(parentFor(active ?? "")));
    if (group)
      setExpanded((old) =>
        old[group.key] ? old : { ...old, [group.key]: true },
      );
  }, [active]);
  const render = (s: PlanningScreen, view?: string) => (
    <a
      key={s.id + (view ?? "")}
      href={planningUrl(s.path, query, view ? { view } : {})}
      aria-current={
        (s.id === active || s.id === parentFor(active ?? "")) &&
        (s.id !== "B11" ||
          (view === "cases" ? activeView === "cases" : activeView !== "cases"))
          ? "page"
          : undefined
      }
      onClick={(e) => {
        e.preventDefault();
        onNavigate(s.path, view ? { view } : {});
      }}
    >
      <span>
        {local(
          locale,
          view === "cases"
            ? ["청구 문의", "Billing inquiries", "استفسارات الفوترة"]
            : (navigationLabels[s.id] ?? titles[s.id]),
        )}
      </span>
      {developer && <small>{s.id}</small>}
    </a>
  );
  const section = (group: ServiceSection) => {
    const pages = navigationScreenIds(
      group,
      new Set(entries.map((s) => s.id)),
    ).flatMap((id) => {
      const found = entries.find((s) => s.id === id);
      return found ? [found] : [];
    });
    if (!pages.length) return null;
    if (group.direct)
      return (
        <div
          key={group.key}
          data-section={group.key}
          className="service-direct"
        >
          {pages.map((s) => render(s))}
        </div>
      );
    return (
      <details
        key={group.key}
        data-section={group.key}
        open={
          expanded[group.key] ??
          (developer || pages.some((s) => s.id === active))
        }
      >
        <summary
          onClick={(e) => {
            e.preventDefault();
            const current = e.currentTarget.parentElement as HTMLDetailsElement;
            setExpanded((old) => ({ ...old, [group.key]: !current.open }));
          }}
        >
          {local(locale, group.label)}
        </summary>
        {pages.flatMap((s) =>
          s.id === "B11"
            ? [render(s, "balances"), render(s, "cases")]
            : [render(s)],
        )}
      </details>
    );
  };
  return (
    <nav
      className="service-navigation"
      aria-label={local(locale, [
        "서비스 메뉴",
        "Service navigation",
        "تنقل الخدمة",
      ])}
    >
      <div className="service-business">{businessSections.map(section)}</div>
      <div className="service-administration">
        {section(dataSection)}
        {section(administrationSection)}
      </div>
    </nav>
  );
}
