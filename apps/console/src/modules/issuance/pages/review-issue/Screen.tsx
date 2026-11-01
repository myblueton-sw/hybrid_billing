import { useEffect, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation, routeUrl } from "../../../../shared/navigation";
import { findWorkRecord } from "../../../operations";
export default function Screen({ locale: l, mode, onState }: ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]),
    scope = params.get("scope") === "sample-b" ? "sample-b" : "sample-a";
  const record =
    mode === "synthetic"
      ? findWorkRecord(scope, params.get("case"), params.get("period"))
      : undefined;
  const [revision, setRevision] = useState("DEMO-r1"),
    [currency, setCurrency] = useState("USD"),
    [reviewer, setReviewer] = useState(""),
    [issuer, setIssuer] = useState(""),
    [checks, setChecks] = useState({
      source: false,
      recipient: false,
      payment: false,
    }),
    [prepared, setPrepared] = useState("");
  const context = JSON.stringify([scope, mode, record?.id]);
  useEffect(() => {
    setPrepared("");
    setReviewer("");
    setIssuer("");
    setChecks({ source: false, recipient: false, payment: false });
    setCurrency(record?.currency ?? "USD");
    setRevision("DEMO-r1");
  }, [context]);
  const key = JSON.stringify({
    scope,
    case: record?.id,
    period: record?.period,
    revision,
    currency,
    reviewer,
    issuer,
    checks,
  });
  useEffect(() => {
    setPrepared("");
  }, [key]);
  const valid =
      !!record &&
      !!reviewer &&
      !!issuer &&
      reviewer !== issuer &&
      Object.values(checks).every(Boolean),
    done = valid && prepared === key;
  const items = [
    "region:layout",
    "section:business-context",
    ...(record
      ? [
          "field:revision",
          "field:currency",
          "field:reviewer",
          "field:issuer",
          "field:source",
          "field:recipient",
          "field:payment",
          "action:prepare-review",
          "action:approve",
          "action:issue",
        ]
      : []),
  ].map((i) => "review-issue:" + i);
  useEffect(
    () => onState(JSON.stringify({ context, key, prepared: done }), items),
    [context, key, done, onState],
  );
  if (!record)
    return (
      <section className="panel">
        <h2>
          {t(
            l,
            "No valid billing record selected",
            "검토할 청구 업무가 선택되지 않았습니다",
            "لم يتم تحديد سجل فوترة صالح",
          )}
        </h2>
        <p>
          {t(
            l,
            "Choose a record within the current account.",
            "현재 계정의 업무를 선택하세요.",
            "اختر سجلاً ضمن الحساب الحالي.",
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
  const titles = {
    source: t(
      l,
      "Source evidence checked",
      "원본 근거 확인",
      "تم فحص أدلة المصدر",
    ),
    recipient: t(
      l,
      "Recipient details checked",
      "수신 대상 확인",
      "تم فحص بيانات المستلم",
    ),
    payment: t(
      l,
      "Payment terms checked",
      "지급 조건 확인",
      "تم فحص شروط الدفع",
    ),
  };
  return (
    <>
      <section className="panel">
        <p className="notice">
          {t(
            l,
            "Local preparation only. No approval, issuance or payment is performed.",
            "로컬 검토 준비입니다. 승인·발행·결제는 수행되지 않습니다.",
            "إعداد محلي فقط. لا تتم موافقة أو إصدار أو دفع.",
          )}
        </p>
        <dl className="facts">
          <div>
            <dt>{t(l, "Customer", "고객", "العميل")}</dt>
            <dd>
              <bdi>{record.customer}</bdi>
            </dd>
          </div>
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
        </dl>
      </section>
      <div className="sd-editor-layout">
        <section className="panel">
          <h2>
            {t(l, "Review prerequisites", "검수 선행 조건", "متطلبات المراجعة")}
          </h2>
          <div className="form-grid">
            <label>
              {t(l, "Revision", "리비전", "المراجعة")}
              <select
                name="revision"
                value={revision}
                onChange={(e) => setRevision(e.target.value)}
              >
                {["DEMO-r1", "DEMO-r2"].map((v) => (
                  <option key={v}>{v}</option>
                ))}
              </select>
            </label>
            <label>
              {t(l, "Currency", "통화", "العملة")}
              <select
                name="currency"
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
              >
                {["USD", "AED", "KRW"].map((v) => (
                  <option key={v}>{v}</option>
                ))}
              </select>
            </label>
            {(["reviewer", "issuer"] as const).map((role) => (
              <label key={role}>
                {role === "reviewer"
                  ? t(
                      l,
                      "Designated reviewer",
                      "지정 검수 담당자",
                      "المراجع المعين",
                    )
                  : t(
                      l,
                      "Designated issuer",
                      "지정 발행 담당자",
                      "مسؤول الإصدار المعين",
                    )}
                <select
                  name={role}
                  value={role === "reviewer" ? reviewer : issuer}
                  onChange={(e) =>
                    (role === "reviewer" ? setReviewer : setIssuer)(
                      e.target.value,
                    )
                  }
                >
                  <option value="">{t(l, "Choose", "선택", "اختر")}</option>
                  {["DEMO-PERSON-A", "DEMO-PERSON-B"].map((v) => (
                    <option key={v}>{v}</option>
                  ))}
                </select>
              </label>
            ))}
          </div>
          <p id="issue-assignment-error" role="status">
            {reviewer && reviewer === issuer
              ? t(
                  l,
                  "Reviewer and issuer must be different.",
                  "검수 담당자와 발행 담당자는 달라야 합니다.",
                  "يجب أن يختلف المراجع عن مسؤول الإصدار.",
                )
              : t(
                  l,
                  "Check all prerequisites before preparing the review.",
                  "선행 조건을 모두 확인한 후 검토를 준비하세요.",
                  "تحقق من جميع المتطلبات قبل إعداد المراجعة.",
                )}
          </p>
          <div className="sd-evidence-checks">
            {(["source", "recipient", "payment"] as const).map((c) => (
              <label className="inline" key={c}>
                <input
                  type="checkbox"
                  name={c}
                  checked={checks[c]}
                  onChange={(e) =>
                    setChecks({ ...checks, [c]: e.target.checked })
                  }
                />
                {titles[c]}
              </label>
            ))}
          </div>
          <div className="form-actions">
            <button
              id="workflow-review"
              className="primary"
              disabled={!valid || done}
              onClick={() => setPrepared(key)}
            >
              {t(
                l,
                "Prepare local review",
                "로컬 검토 준비",
                "إعداد مراجعة محلية",
              )}
            </button>
          </div>
          <p id="issue-review-result" role="status">
            {done
              ? t(
                  l,
                  "Local review prepared. Financial actions remain blocked.",
                  "로컬 검토를 준비했습니다. 금전 처리는 계속 차단됩니다.",
                  "تم إعداد المراجعة محلياً. تبقى الإجراءات المالية محظورة.",
                )
              : ""}
          </p>
        </section>
        <aside className="panel">
          <h2>
            <bdi>DEMO-DRAFT-{record.id}</bdi>
          </h2>
          <dl className="facts">
            <div>
              <dt>{t(l, "Revision", "리비전", "المراجعة")}</dt>
              <dd>{revision}</dd>
            </div>
            <div>
              <dt>{t(l, "Currency", "통화", "العملة")}</dt>
              <dd>{currency}</dd>
            </div>
            {[
              t(l, "Net amount", "순액", "المبلغ الصافي"),
              t(l, "Tax", "세금", "الضريبة"),
              t(l, "Total", "총액", "الإجمالي"),
            ].map((title) => (
              <div key={title}>
                <dt>{title}</dt>
                <dd>{t(l, "Not available", "확인 불가", "غير متاح")}</dd>
              </div>
            ))}
          </dl>
          <p className="notice">
            {t(
              l,
              "Verified amounts, server permissions and issuance services are not connected.",
              "검증된 금액·서버 권한·발행 서비스가 연결되지 않았습니다.",
              "المبالغ المتحقق منها وصلاحيات الخادم وخدمات الإصدار غير متصلة.",
            )}
          </p>
          <div className="form-actions">
            <button id="issue-approve" disabled>
              {t(l, "Approve — blocked", "승인 — 차단", "الموافقة — محظورة")}
            </button>
            <button id="issue-issue" disabled>
              {t(l, "Issue — blocked", "발행 — 차단", "الإصدار — محظور")}
            </button>
          </div>
        </aside>
      </div>
    </>
  );
}
