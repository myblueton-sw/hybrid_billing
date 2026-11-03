import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { stages, type Target } from "./model";
import { stageLabel, stateLabel } from "./copy";
export default function Detail({
  row: r,
  locale: l,
  onClose,
}: {
  row: Target;
  locale: Locale;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    ref.current?.showModal();
    return () => ref.current?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className="closing-detail"
      aria-labelledby="closing-detail-title"
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      <header>
        <div>
          <h2 id="closing-detail-title">
            <bdi>{r.contract}</bdi>
          </h2>
          <p>
            <bdi>{r.period}</bdi> · {stateLabel(l, r.state)} ·{" "}
            {stageLabel(l, r.stage)}
          </p>
        </div>
        <button onClick={onClose} autoFocus>
          {t(l, "Back to results", "목록으로", "العودة للنتائج")}
        </button>
      </header>
      <p>
        {t(
          l,
          "Synthetic read-only closing detail",
          "합성 데이터 · 마감 상세 조회",
          "تفاصيل إقفال تجريبية للقراءة فقط",
        )}
      </p>
      <dl>
        <dt>
          {t(l, "Organization / entity", "조직 / 법인", "المؤسسة / الكيان")}
        </dt>
        <dd>
          <bdi>
            {r.organization} / {r.entity}
          </bdi>
        </dd>
        <dt>{t(l, "Owner", "담당자", "المسؤول")}</dt>
        <dd>
          <bdi>{r.owner}</bdi>
        </dd>
        <dt>
          {t(l, "Schedule / revision", "일정 / 개정", "الجدول / المراجعة")}
        </dt>
        <dd>
          <bdi>
            {r.schedule} / {r.revision}
          </bdi>
        </dd>
        <dt>{t(l, "Calendar policy", "달력 정책", "سياسة التقويم")}</dt>
        <dd>
          {t(
            l,
            "Not configured · fixture timestamps only",
            "미설정 · 예제에 지정된 시각만 표시",
            "غير مهيأة · أوقات تجريبية فقط",
          )}
        </dd>
      </dl>
      <h3>{t(l, "Completion conditions", "완료 조건", "شروط الإكمال")}</h3>
      <table>
        <thead>
          <tr>
            <th>{t(l, "Stage", "단계", "المرحلة")}</th>
            <th>{t(l, "Evidence state", "근거 상태", "حالة الأدلة")}</th>
          </tr>
        </thead>
        <tbody>
          {stages.map((s) => {
            const na = r.internal && (s === "issuance" || s === "delivery");
            const status = na
              ? t(l, "Not applicable", "해당 없음", "غير منطبق")
              : r.state === "exempt"
                ? t(
                    l,
                    "Not evaluated · exempt",
                    "미평가 · 면제",
                    "غير مقيم · معفى",
                  )
                : s === "accounting" && r.erp === "unknown"
                  ? t(
                      l,
                      "Unconfirmed · blocks closing",
                      "미확인 · 마감 차단",
                      "غير مؤكد · يمنع الإقفال",
                    )
                  : s === "accounting" && r.erp === "na"
                    ? t(l, "Not applicable", "해당 없음", "غير منطبق")
                    : r.state === "complete" ||
                        s === "collection" ||
                        (s === "accounting" && r.erp === "confirmed")
                      ? t(
                          l,
                          "Fixture evidence confirmed",
                          "예제 근거 확인",
                          "تم تأكيد دليل تجريبي",
                        )
                      : t(
                          l,
                          "Pending verification",
                          "검증 대기",
                          "بانتظار التحقق",
                        );
            return (
              <tr key={s}>
                <th>{stageLabel(l, s)}</th>
                <td>{status}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <h3>
        {t(l, "Blockers and evidence", "차단 사유·근거", "العوائق والأدلة")}
      </h3>
      <p>
        {r.state === "blocked"
          ? t(
              l,
              "ERP posting outcome is unknown; completion is blocked for this contract, including internal paths.",
              "ERP 반영 결과 미확인으로 마감이 차단되었습니다. 내부 정산도 해당 계약의 확인 조건을 따릅니다.",
              "نتيجة الترحيل غير مؤكدة؛ الإقفال محظور بما في ذلك المسارات الداخلية.",
            )
          : r.state === "exempt"
            ? t(
                l,
                "Evidenced exemption, counted separately from completion.",
                "근거에 따른 면제로 완료와 별도 집계합니다.",
                "إعفاء موثق يحتسب منفصلاً عن الإكمال.",
              )
            : r.state === "held"
              ? t(
                  l,
                  "Held by a fixture review decision; not completed.",
                  "예제 검토 결정에 따른 보류이며 완료가 아닙니다.",
                  "معلق بموجب قرار مراجعة تجريبي؛ ليس مكتملاً.",
                )
              : t(
                  l,
                  "See independent condition evidence above.",
                  "위 단계별 근거 상태를 확인하세요.",
                  "راجع أدلة الشروط المستقلة أعلاه.",
                )}
      </p>
      <p>
        <bdi>
          {r.id}:EVIDENCE:{r.state}
        </bdi>
      </p>
      <h3>{t(l, "Deadline history", "기한 변경 이력", "سجل المواعيد")}</h3>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>{t(l, "Revision", "개정", "المراجعة")}</th>
              <th>
                {t(l, "Event time UTC", "기록 시각 UTC", "وقت الحدث UTC")}
              </th>
              <th>
                {t(
                  l,
                  "Stage deadline UTC",
                  "단계 기한 UTC",
                  "موعد المرحلة UTC",
                )}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr>
              <th>r1</th>
              <td>
                <bdi>{r.period}-01T00:00:00Z</bdi>
              </td>
              <td>
                <bdi>{r.original}</bdi>
              </td>
            </tr>
            {r.revision === "r2" && (
              <tr>
                <th>r2</th>
                <td>
                  <bdi>{r.period}-15T00:00:00Z</bdi>
                </td>
                <td>
                  <bdi>{r.current}</bdi>
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <p>
        {r.revision === "r2"
          ? t(
              l,
              "Fixture postponement: source delivery delay. Original deadline retained. Only this stage schedule changed.",
              "예제 연기 사유: 원천 전달 지연. 원 기한을 보존하며 이 단계 일정만 변경되었습니다.",
              "تأجيل تجريبي بسبب تأخر المصدر. الموعد الأصلي محفوظ وتغير جدول هذه المرحلة فقط.",
            )
          : t(
              l,
              "No deadline change in this fixture.",
              "이 예제의 기한 변경 이력이 없습니다.",
              "لا تغيير للموعد في هذا المثال.",
            )}
      </p>
      <h3>
        {t(
          l,
          "Monthly ERP recognition",
          "월 ERP 인식",
          "الاعتراف الشهري في ERP",
        )}
      </h3>
      <p>
        {r.erp === "unknown"
          ? t(l, "Posting unconfirmed", "반영 미확인", "الترحيل غير مؤكد")
          : r.erp === "confirmed"
            ? t(
                l,
                "Fixture posting evidence confirmed",
                "예제 반영 근거 확인",
                "دليل ترحيل تجريبي مؤكد",
              )
            : t(
                l,
                "Not applicable to this fixture contract",
                "이 예제 계약은 해당 없음",
                "غير منطبق على هذا العقد التجريبي",
              )}
      </p>
      <p>
        {t(
          l,
          "Quarterly reconciliation and monetary amounts: not connected. Posting and billing remain independent.",
          "분기 대사·금액 정보: 미연결. ERP 반영과 청구는 독립 상태입니다.",
          "التسوية الفصلية والمبالغ غير متصلة. الترحيل والفوترة مستقلان.",
        )}
      </p>
      <p className="closing-note">
        {t(
          l,
          "Read-only candidate. Deadline changes, reassignment, exemptions, approvals and notifications require the pending policy, permission and service integration.",
          "조회 시안입니다. 기한 변경·담당 지정·면제·승인·알림 실행은 정책·권한·서비스 연동 후 제공됩니다.",
          "نموذج للقراءة فقط. تغييرات المواعيد والتعيين والإعفاء والموافقة والإشعارات تتطلب تكامل السياسة والصلاحيات والخدمة.",
        )}
      </p>
    </dialog>
  );
}
