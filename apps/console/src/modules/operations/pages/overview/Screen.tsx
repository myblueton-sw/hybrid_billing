import { useEffect } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { routeUrl, useLocation } from "../../../../shared/navigation";
import { workRecords } from "../../index";
import "./overview.css";

export default function Overview({ locale: l, mode, onState }: ScreenProps) {
  const hash = useLocation();
  const params = new URLSearchParams(hash.split("?")[1]);
  const scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const period = params.get("overview-period") ?? "2026-10";
  const rows =
    mode === "synthetic"
      ? workRecords(scope).filter((r) => r.period === period)
      : [];
  const missing = rows.filter((r) => r.status === "missing");
  const mine = rows.filter((r) => r.owner === "DEMO-PERSON-A");
  const metrics = [
    {
      key: "total",
      status: "",
      count: rows.length,
      label: t(l, "Total records", "전체 업무", "إجمالي السجلات"),
    },
    {
      key: "missing",
      status: "missing",
      count: missing.length,
      label: t(l, "Source missing", "원본 누락", "المصدر مفقود"),
    },
    {
      key: "processing",
      status: "processing",
      count: rows.filter((r) => r.status === "processing").length,
      label: t(l, "Review pending", "검수 대기", "بانتظار المراجعة"),
    },
    {
      key: "ready",
      status: "ready",
      count: rows.filter((r) => r.status === "ready").length,
      label: t(l, "Evidence ready", "근거 확인 가능", "الأدلة جاهزة"),
    },
  ];
  function queue(id: string, status = "") {
    return routeUrl(id, {
      [id + "-period"]: period,
      [id + "-status"]: status,
      [id + "-q"]: "",
      [id + "-provider"]: "",
      [id + "-page"]: "1",
    });
  }
  useEffect(() => {
    onState(JSON.stringify({ mode, scope, period }), [
      "overview:region:layout",
      "overview:field:period",
      ...(mode === "synthetic"
        ? [
            "overview:data:counts",
            "overview:section:missing",
            "overview:section:my-work",
            "overview:action:drill-down",
          ]
        : ["overview:state:unconnected"]),
    ]);
  }, [mode, scope, period, onState]);
  return (
    <div className="overview-dashboard" data-design-id="overview:region:layout">
      <div className="overview-scope">
        <label>
          {t(l, "Settlement period", "정산 기간", "فترة التسوية")}
          <input
            id="overview-period"
            type="month"
            value={period}
            onChange={(e) => {
              location.hash = routeUrl("overview", {
                "overview-period": e.target.value,
              });
            }}
          />
        </label>
        <p>
          {mode === "synthetic"
            ? t(
                l,
                "Synthetic snapshot · DEMO-S1 · Not live operational data",
                "합성 스냅샷 · DEMO-S1 · 실제 운영 데이터 아님",
                "لقطة تجريبية · DEMO-S1 · ليست بيانات تشغيلية حية",
              )
            : t(
                l,
                "Snapshot unavailable",
                "스냅샷 확인 불가",
                "اللقطة غير متاحة",
              )}
        </p>
      </div>
      {mode === "unconnected" ? (
        <section className="state" role="status">
          <h2>
            {t(
              l,
              "Overview service not connected",
              "전체 현황 서비스 미연결",
              "خدمة النظرة العامة غير متصلة",
            )}
          </h2>
          <p>
            {t(
              l,
              "Current work and amounts cannot be verified.",
              "현재 업무 현황과 금액을 확인할 수 없습니다.",
              "لا يمكن التحقق من الأعمال والمبالغ الحالية.",
            )}
          </p>
        </section>
      ) : (
        <>
          <section
            className="overview-metrics"
            aria-label={t(l, "Work summary", "업무 요약", "ملخص الأعمال")}
          >
            {metrics.map((m) => (
              <article key={m.key} className="overview-metric">
                <h2>{m.label}</h2>
                <strong data-count={m.key}>{m.count}</strong>
                <a href={queue("operations", m.status)}>
                  {t(l, "View records", "업무 보기", "عرض السجلات")}
                </a>
                {m.key === "ready" && (
                  <small>
                    {t(
                      l,
                      "Not a completion count",
                      "완료 건수가 아닙니다",
                      "ليس عدد الأعمال المكتملة",
                    )}
                  </small>
                )}
              </article>
            ))}
          </section>
          <p className="overview-amounts">
            {t(
              l,
              "Cost · Sales · Allocation: amounts are unavailable in this synthetic snapshot.",
              "원가 · 매출 · 배분: 이 합성 스냅샷에는 금액 정보가 없습니다.",
              "التكلفة · المبيعات · التخصيص: المبالغ غير متاحة في هذه اللقطة التجريبية.",
            )}
          </p>
          <div className="overview-lists">
            {[
              {
                key: "missing",
                title: t(
                  l,
                  "Sources to check",
                  "확인이 필요한 원본",
                  "مصادر تحتاج إلى مراجعة",
                ),
                records: missing,
                target: "operations",
                status: "missing",
              },
              {
                key: "mine",
                title: t(
                  l,
                  "My next actions",
                  "내가 확인할 작업",
                  "إجراءاتي التالية",
                ),
                records: mine,
                target: "my-tasks",
                status: "",
              },
            ].map((section) => (
              <section className="overview-list" key={section.key}>
                <header>
                  <h2>{section.title}</h2>
                  <a href={queue(section.target, section.status)}>
                    {t(l, "View all", "전체 보기", "عرض الكل")} (
                    {section.records.length})
                  </a>
                </header>
                {section.key === "mine" && (
                  <p>
                    {t(l, "Example assignee", "예제 담당자", "مسؤول تجريبي")}:{" "}
                    <bdi>DEMO-PERSON-A</bdi>
                  </p>
                )}
                {section.records.length === 0 ? (
                  <p role="status">
                    {t(
                      l,
                      "No matching work in this period.",
                      "이 기간에 해당하는 업무가 없습니다.",
                      "لا توجد أعمال مطابقة في هذه الفترة.",
                    )}
                  </p>
                ) : (
                  <ul>
                    {section.records.slice(0, 5).map((r) => (
                      <li key={r.id}>
                        <div>
                          <a
                            href={routeUrl("customer-detail", {
                              case: r.id,
                              period,
                              from: "overview",
                            })}
                          >
                            <bdi>{r.id}</bdi>
                          </a>
                          <span>
                            {t(l, "Due", "기한", "الاستحقاق")}{" "}
                            <bdi>{r.due}</bdi>
                          </span>
                        </div>
                        <p>
                          <bdi>{r.customer}</bdi> ·{" "}
                          {section.key === "missing"
                            ? r.provider
                            : r.status === "missing"
                              ? t(
                                  l,
                                  "Check source",
                                  "원본 확인",
                                  "مراجعة المصدر",
                                )
                              : r.status === "processing"
                                ? t(
                                    l,
                                    "Assign reviewer",
                                    "검수 담당자 확인",
                                    "تعيين مراجع",
                                  )
                                : t(
                                    l,
                                    "Review prerequisites",
                                    "선행 조건 검토",
                                    "مراجعة المتطلبات",
                                  )}
                        </p>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
