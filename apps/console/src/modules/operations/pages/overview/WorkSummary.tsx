import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { stages, type Closing, type Stage } from "./model";
import { stageName, statusName } from "./copy";

export default function WorkSummary({
  rows,
  known,
  expected,
  locale: l,
  onSelect,
}: {
  rows: Closing[];
  known: number;
  expected: number;
  locale: Locale;
  onSelect: (row: Closing, stage: Stage) => void;
}) {
  const attention = [
    {
      id: "blocked",
      label: t(l, "Blocked units", "처리 차단", "وحدات محظورة"),
      rows: rows.filter((r) => stages.some((s) => r.status[s] === "blocked")),
      stage: (r: Closing) => stages.find((s) => r.status[s] === "blocked")!,
    },
    {
      id: "approval",
      label: t(l, "Awaiting approval", "승인 대기", "بانتظار الموافقة"),
      rows: rows.filter((r) => r.status.approval === "pending"),
      stage: () => "approval" as Stage,
    },
    {
      id: "erp",
      label: t(l, "ERP unconfirmed", "ERP 미확인", "ERP غير مؤكد"),
      rows: rows.filter((r) => r.status.accounting === "unknown"),
      stage: () => "accounting" as Stage,
    },
  ];
  return (
    <section className="overview-work-summary">
      <header>
        <h2>{t(l, "Settlement work", "정산 업무", "أعمال التسوية")}</h2>
        <span>
          <bdi data-closing-count>{rows.length}</bdi>{" "}
          {t(l, "units in scope", "정산 건", "وحدات في النطاق")}
        </span>
      </header>
      <div className="overview-attention-counts">
        {attention.map((group) => (
          <div key={group.id} data-attention={group.id}>
            <span>{group.label}</span>
            <strong>{group.rows.length}</strong>
            {group.rows.length > 0 && (
              <button
                aria-label={`${group.label} · ${t(l, "Inspect first unit", "첫 정산 건 확인", "فحص الوحدة الأولى")}`}
                onClick={() =>
                  onSelect(group.rows[0], group.stage(group.rows[0]))
                }
              >
                {t(
                  l,
                  "Inspect first unit",
                  "첫 정산 건 확인",
                  "فحص الوحدة الأولى",
                )}
              </button>
            )}
          </div>
        ))}
      </div>
      <p className="overview-operational-note">
        {t(
          l,
          "Categories can overlap. Pending is not permission to approve. Owner and deadline are unknown.",
          "항목 간 중복이 있습니다. 승인 대기는 승인 권한을 뜻하지 않습니다. 담당자·납기는 미확인입니다.",
          "قد تتداخل الفئات. الانتظار لا يمنح صلاحية الموافقة. المسؤول والموعد غير معلومين.",
        )}
      </p>
      <details className="overview-independent-states">
        <summary>
          {t(
            l,
            "Issuance, ERP and receipt are independent",
            "발행·ERP·수납 상태는 각각 독립적입니다",
            "الإصدار وERP والتحصيل حالات مستقلة",
          )}
        </summary>
        <dl>
          {(["issuance", "accounting", "receipt"] as const).map((stage) => (
            <div key={stage}>
              <dt>{stageName(l, stage)}</dt>
              <dd>
                {(
                  ["complete", "pending", "blocked", "unknown", "na"] as const
                ).map((status) => (
                  <span key={status}>
                    {statusName(l, status)}{" "}
                    <bdi>
                      {rows.filter((r) => r.status[stage] === status).length}
                    </bdi>
                  </span>
                ))}
              </dd>
            </div>
          ))}
        </dl>
      </details>
      <p className="overview-coverage">
        {t(
          l,
          "Cost records available",
          "비용 자료 확인",
          "سجلات التكلفة المتاحة",
        )}{" "}
        <strong data-coverage>
          <bdi dir="ltr">
            {known} / {expected}
          </bdi>
        </strong>
        <span>
          {t(
            l,
            "Source × organization · not closing completion",
            "원천 × 조직 · 정산 완료율 아님",
            "المصدر × المؤسسة · ليست نسبة اكتمال الإقفال",
          )}
        </span>
      </p>
    </section>
  );
}
