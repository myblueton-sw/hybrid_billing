import type { KeyboardEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { dateCounts, type Target } from "./model";
export default function Calendar({
  rows,
  month,
  period,
  locale: l,
  onDate,
}: {
  rows: Target[];
  month: string;
  period: string;
  locale: Locale;
  onDate: (date: string) => void;
}) {
  const counts = dateCounts(rows),
    year = Number(month.slice(0, 4)),
    m = Number(month.slice(5));
  const days = new Date(Date.UTC(year, m, 0)).getUTCDate(),
    offset = new Date(Date.UTC(year, m - 1, 1)).getUTCDay();
  const move = (e: KeyboardEvent<HTMLDivElement>) => {
    const delta =
      e.key === "ArrowDown"
        ? 7
        : e.key === "ArrowUp"
          ? -7
          : e.key === "ArrowRight"
            ? l === "ar"
              ? -1
              : 1
            : e.key === "ArrowLeft"
              ? l === "ar"
                ? 1
                : -1
              : 0;
    if (!delta) return;
    const buttons = [
        ...e.currentTarget.querySelectorAll<HTMLButtonElement>("button"),
      ],
      at = buttons.indexOf(document.activeElement as HTMLButtonElement);
    if (at < 0) return;
    e.preventDefault();
    buttons[Math.max(0, Math.min(buttons.length - 1, at + delta))]?.focus();
  };
  return (
    <section className="closing-calendar">
      <h2>
        <bdi>{period}</bdi> {t(l, "cycle", "주기", "دورة")} · <bdi>{month}</bdi>{" "}
        ·{" "}
        {t(
          l,
          "Current stage deadlines · UTC",
          "현재 단계 기한 · UTC",
          "مواعيد المرحلة الحالية · UTC",
        )}
      </h2>
      <p>
        {t(
          l,
          "One current stage schedule per closing target. Counts cover all filtered targets. Arrow keys move dates; Enter opens that date’s list. Mobile shows a date agenda.",
          "마감 대상당 현재 단계 일정 1건입니다. 건수는 필터 전체 대상 기준입니다. 방향키로 날짜 이동, Enter로 날짜별 목록을 엽니다. 모바일은 날짜순 일정입니다.",
          "جدول مرحلة حالي واحد لكل هدف. الأعداد لكل الأهداف المصفاة. الأسهم للتنقل وEnter لقائمة التاريخ. الهاتف يعرض جدولاً زمنياً.",
        )}
      </p>
      <div className="closing-weekdays" aria-hidden="true">
        {Array.from({ length: 7 }, (_, i) => (
          <span key={i}>
            {new Intl.DateTimeFormat(l, {
              weekday: "short",
              timeZone: "UTC",
            }).format(new Date(Date.UTC(2026, 10, i + 1)))}
          </span>
        ))}
      </div>
      <div
        className="closing-days"
        onKeyDown={move}
        role="group"
        aria-label={t(l, "Deadline dates", "마감 날짜", "تواريخ المواعيد")}
      >
        <span
          className="closing-calendar-offset"
          style={{
            gridColumn: `span ${offset || 1}`,
            display: offset ? "block" : "none",
          }}
        />
        {Array.from({ length: days }, (_, i) => {
          const date = `${month}-${String(i + 1).padStart(2, "0")}`;
          return (
            <button
              key={date}
              onClick={() => onDate(date)}
              aria-label={`${date} · ${counts.get(date) ?? 0} ${t(l, "targets", "대상", "أهداف")}`}
            >
              <bdi>{date}</bdi>
              <strong>
                {counts.get(date) ?? 0} {t(l, "targets", "대상", "أهداف")}
              </strong>
            </button>
          );
        })}
      </div>
    </section>
  );
}
