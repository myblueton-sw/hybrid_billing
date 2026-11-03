import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { stages, type Closing, type Cost, type Stage } from "./model";
import { organizationName, stageName } from "./copy";
import Paging, { usePage } from "./Paging";

type WorkItem = {
  key: string;
  group: "blocked" | "missing" | "erp" | "approval";
  kind: "stage" | "record";
  id: string;
  organization: string;
  stage?: Stage;
  source: string;
};
export default function WorkSummary({
  rows,
  missing,
  scope,
  locale: l,
  onInspect,
}: {
  rows: Closing[];
  missing: Cost[];
  scope: string;
  locale: Locale;
  onInspect: (kind: "stage" | "record", id: string, stage?: Stage) => void;
}) {
  const groups = [
    {
      id: "blocked" as const,
      label: t(l, "Blocked", "차단", "محظور"),
      rows: rows.filter((r) => stages.some((s) => r.status[s] === "blocked")),
      stage: (r: Closing) => stages.find((s) => r.status[s] === "blocked")!,
    },
    {
      id: "erp" as const,
      label: t(l, "ERP unconfirmed", "ERP 미확인", "ERP غير مؤكد"),
      rows: rows.filter((r) => r.status.accounting === "unknown"),
      stage: () => "accounting" as Stage,
    },
    {
      id: "approval" as const,
      label: t(l, "Approval pending", "승인 대기", "الموافقة معلقة"),
      rows: rows.filter((r) => r.status.approval === "pending"),
      stage: () => "approval" as Stage,
    },
  ];
  const items: WorkItem[] = [
    ...groups[0].rows.map((r) => ({
      key: `blocked:${r.id}`,
      group: "blocked" as const,
      kind: "stage" as const,
      id: r.id,
      organization: r.organization,
      stage: groups[0].stage(r),
      source: r.internal
        ? t(l, "Internal", "내부 정산", "داخلي")
        : t(l, "External", "외부 청구", "خارجي"),
    })),
    ...missing.map((r) => ({
      key: `missing:${r.id}`,
      group: "missing" as const,
      kind: "record" as const,
      id: r.id,
      organization: r.organization,
      source: r.source,
    })),
    ...groups.slice(1).flatMap((g) =>
      g.rows.map((r) => ({
        key: `${g.id}:${r.id}`,
        group: g.id,
        kind: "stage" as const,
        id: r.id,
        organization: r.organization,
        stage: g.stage(r),
        source: r.internal
          ? t(l, "Internal", "내부 정산", "داخلي")
          : t(l, "External", "외부 청구", "خارجي"),
      })),
    ),
  ];
  const page = usePage(items, scope, 6);
  const labels = {
    blocked: groups[0].label,
    missing: t(l, "Missing data", "자료 누락", "بيانات مفقودة"),
    erp: groups[1].label,
    approval: groups[2].label,
  };
  return (
    <section
      className="overview-workboard unified-exceptions"
      data-design-id="U01:work:attention"
    >
      <header>
        <h2>{t(l, "Next work", "다음 작업", "العمل التالي")}</h2>
        <span>
          <bdi data-closing-count>{rows.length}</bdi>{" "}
          {t(l, "settlement units", "정산 건", "وحدات تسوية")}
        </span>
      </header>
      <div className="overview-work-counts">
        {groups.map((g) => (
          <span key={g.id} data-attention={g.id}>
            {g.label} <strong>{g.rows.length}</strong>
          </span>
        ))}
        <span>
          {t(l, "Missing records", "자료 누락", "السجلات المفقودة")}{" "}
          <strong>{missing.length}</strong>
        </span>
      </div>
      <div className="overview-work-head" aria-hidden="true">
        <span>{t(l, "State", "상태", "الحالة")}</span>
        <span>{t(l, "Target", "대상", "الهدف")}</span>
        <span>{t(l, "Stage", "단계", "المرحلة")}</span>
        <span>{t(l, "Action", "동작", "الإجراء")}</span>
      </div>
      <ul className="overview-work-list">
        {page.rows.map((item) => (
          <li
            key={item.key}
            data-work-group={item.group}
            data-work-kind={item.kind}
          >
            <span
              className={`overview-work-state ${item.group === "blocked" ? "is-blocked" : ""}`}
            >
              {labels[item.group]}
            </span>
            <div className="overview-work-target">
              {organizationName(l, item.organization)}
              <small>{item.source}</small>
            </div>
            <span className="overview-work-stage">
              {item.stage
                ? stageName(l, item.stage)
                : t(l, "Source cost", "원천 비용", "تكلفة المصادر")}
            </span>
            <button
              aria-label={`${labels[item.group]} · ${organizationName(l, item.organization)} · ${item.source} · ${item.stage ? stageName(l, item.stage) : t(l, "Source cost", "원천 비용", "تكلفة المصادر")} · ${t(l, "Inspect", "확인", "فحص")}`}
              onClick={() => onInspect(item.kind, item.id, item.stage)}
            >
              {t(l, "Inspect", "확인", "فحص")}
            </button>
          </li>
        ))}
      </ul>
      {!items.length && (
        <p role="status">
          {t(
            l,
            "No blocked units, missing cost records, unknown ERP outcomes or pending approvals in this selection.",
            "선택 범위에 차단·자료 누락·ERP 미확인·승인 대기 항목이 없습니다.",
            "لا توجد وحدات محظورة أو سجلات تكلفة مفقودة أو ERP غير مؤكد أو موافقات معلقة في هذا النطاق.",
          )}
        </p>
      )}
      <p className="overview-work-footnote">
        {t(
          l,
          "Categories overlap. Owners and deadlines are not connected.",
          "항목 간 중복 가능 · 담당자·납기 미연결",
          "قد تتداخل الفئات · المسؤولون والمواعيد غير متصلة",
        )}
      </p>
      <Paging value={page} locale={l} jump={items.length > 60} />
    </section>
  );
}
