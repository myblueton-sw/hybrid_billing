import { useEffect, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation, routeUrl } from "../../../../shared/navigation";
import { findWorkRecord } from "../../../operations";
import { screens } from "../../../../shared/catalog";
const spec = screens.find((s) => s.id === "customer-detail")!;
export default function Screen({ locale: l, mode, onState }: ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]),
    scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const record =
    mode === "synthetic"
      ? findWorkRecord(scope, params.get("case"), params.get("period"))
      : undefined;
  const [tab, setTab] = useState("overview"),
    [inspected, setInspected] = useState(false);
  const context = JSON.stringify([scope, mode, record?.id]);
  useEffect(() => {
    setTab("overview");
    setInspected(false);
  }, [context]);
  const items = [
    "region:layout",
    "section:context",
    ...(record
      ? [
          "tab:overview",
          "tab:evidence",
          "tab:history",
          "action:inspect",
          "navigate:index-intake",
          "navigate:review-issue",
          ...spec.columns.map((c) => "data:" + c.key),
        ]
      : []),
  ].map((i) => "customer-detail:" + i);
  useEffect(
    () =>
      onState(
        JSON.stringify({
          scope,
          mode,
          case: record?.id ?? null,
          tab,
          inspected,
        }),
        items,
      ),
    [context, tab, inspected, onState],
  );
  if (!record)
    return (
      <section className="panel">
        <h2>
          {t(
            l,
            "No customer record selected",
            "선택한 고객 업무를 확인할 수 없습니다",
            "لا يوجد سجل عميل محدد",
          )}
        </h2>
        <p>
          {t(
            l,
            "Choose a record from the current account work queue. No other record has been substituted.",
            "현재 계정의 청구 업무에서 고객을 선택하세요. 다른 고객 정보로 대체하지 않습니다.",
            "اختر سجلاً من قائمة أعمال الحساب الحالي. لم يتم استبداله بسجل آخر.",
          )}
        </p>
        <a href={routeUrl("operations", { case: "" })}>
          {t(
            l,
            "Open billing operations",
            "청구 운영 열기",
            "فتح عمليات الفوترة",
          )}
        </a>
      </section>
    );
  const values: Record<string, string> = {
    account: record.account,
    customer: record.customer,
    contract: record.contract,
    effective: "2026-10-01",
    status:
      record.status === "missing"
        ? t(l, "Source missing", "원본 누락", "المصدر مفقود")
        : t(l, "Review pending", "검수 대기", "بانتظار المراجعة"),
  };
  const title = (key: string) =>
    key === "overview"
      ? t(l, "Overview", "개요", "نظرة عامة")
      : key === "evidence"
        ? t(l, "Evidence", "근거", "الأدلة")
        : t(l, "History", "이력", "السجل");
  return (
    <>
      <section className="panel sd-context">
        <div>
          <h2>
            <bdi>{record.customer}</bdi>
          </h2>
          <p>
            {t(
              l,
              "Selected customer, account and contract references",
              "선택한 고객·계정·계약 참조",
              "مراجع العميل والحساب والعقد المحدد",
            )}
          </p>
        </div>
        <dl className="facts">
          <div>
            <dt>ID</dt>
            <dd>
              <bdi>{record.id}</bdi>
            </dd>
          </div>
          <div>
            <dt>{t(l, "Billing period", "청구 기간", "فترة الفوترة")}</dt>
            <dd>
              <bdi>{record.period}</bdi>
            </dd>
          </div>
          <div>
            <dt>{t(l, "Revision", "리비전", "المراجعة")}</dt>
            <dd>DEMO-r1</dd>
          </div>
        </dl>
      </section>
      <section className="panel">
        <div
          className="sd-tabs"
          role="tablist"
          aria-label={t(
            l,
            "Customer information",
            "고객 정보",
            "معلومات العميل",
          )}
          onKeyDown={(e) => {
            if (["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)) {
              e.preventDefault();
              const all = ["overview", "evidence", "history"],
                i = all.indexOf(tab),
                next =
                  e.key === "Home"
                    ? 0
                    : e.key === "End"
                      ? 2
                      : (i + (e.key === "ArrowRight" ? 1 : -1) + 3) % 3;
              setTab(all[next]);
              document.getElementById("screen-tab-" + all[next])?.focus();
            }
          }}
        >
          {["overview", "evidence", "history"].map((key) => (
            <button
              key={key}
              id={"screen-tab-" + key}
              role="tab"
              aria-selected={tab === key}
              aria-controls="screen-detail-panel"
              tabIndex={tab === key ? 0 : -1}
              onClick={() => setTab(key)}
            >
              {title(key)}
            </button>
          ))}
        </div>
        <div
          id="screen-detail-panel"
          role="tabpanel"
          aria-labelledby={"screen-tab-" + tab}
        >
          {tab === "overview" ? (
            <dl className="facts">
              {spec.columns.map((c) => (
                <div key={c.key}>
                  <dt>{c.label[l]}</dt>
                  <dd>
                    <bdi>{values[c.key]}</bdi>
                  </dd>
                </div>
              ))}
            </dl>
          ) : tab === "evidence" ? (
            <>
              <h3>{title("evidence")}</h3>
              <dl className="facts">
                <div>
                  <dt>
                    {t(l, "Source reference", "원본 참조", "مرجع المصدر")}
                  </dt>
                  <dd>
                    <bdi>{record.source}</bdi>
                  </dd>
                </div>
                <div>
                  <dt>{t(l, "Provider", "공급자", "المورد")}</dt>
                  <dd>{record.provider}</dd>
                </div>
                <div>
                  <dt>
                    {t(
                      l,
                      "Verified amount",
                      "검증된 금액",
                      "المبلغ المتحقق منه",
                    )}
                  </dt>
                  <dd>{t(l, "Not available", "확인 불가", "غير متاح")}</dd>
                </div>
              </dl>
              <a
                className="button"
                href={routeUrl("index-intake", {
                  q: record.source,
                  provider: "",
                  filter: "all",
                  sort: "asc",
                  size: "25",
                  case: record.id,
                  page: "1",
                })}
              >
                {t(
                  l,
                  "Inspect source evidence",
                  "연결된 원본 검토",
                  "فحص أدلة المصدر",
                )}
              </a>
            </>
          ) : (
            <>
              <h3>{title("history")}</h3>
              <p>
                {t(
                  l,
                  "No production history is connected.",
                  "운영 이력은 연결되지 않았습니다.",
                  "سجل الإنتاج غير متصل.",
                )}
              </p>
              {inspected && (
                <p>
                  {t(
                    l,
                    "Local evidence inspection completed in this preview.",
                    "이 시안에서 로컬 근거 확인을 수행했습니다.",
                    "تم فحص الأدلة محلياً في هذه المعاينة.",
                  )}
                </p>
              )}
            </>
          )}
        </div>
        <div className="form-actions">
          <button id="screen-design-inspect" onClick={() => setInspected(true)}>
            {t(l, "Review evidence", "근거 검토", "مراجعة الأدلة")}
          </button>
          <a
            className="button primary"
            href={routeUrl("review-issue", {
              case: record.id,
              period: record.period,
            })}
          >
            {t(
              l,
              "Review issuance prerequisites",
              "검수·발행 선행 조건 열기",
              "مراجعة متطلبات الإصدار",
            )}
          </a>
        </div>
        {inspected && (
          <p role="status">
            {t(
              l,
              "Local inspection recorded. No financial approval.",
              "로컬 확인 상태입니다. 금전 승인이 아닙니다.",
              "تم تسجيل الفحص محلياً دون موافقة مالية.",
            )}
          </p>
        )}
      </section>
    </>
  );
}
