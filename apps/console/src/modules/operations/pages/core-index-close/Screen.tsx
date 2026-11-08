import { useEffect, useMemo, useRef, useState } from "react";
import type { ScreenProps } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { useLocation, routeUrl } from "../../../../shared/navigation";
import Paging, { usePage } from "../overview/Paging";
import {
  asOf,
  cyclePeriods,
  states,
  stages,
  fixtureTargets,
  filterTargets,
  isOverdue,
  deadlineMonth,
  type Filter,
  type Target,
} from "./model";
import { stateLabel, stageLabel } from "./copy";
import Calendar from "./Calendar";
import Detail from "./Detail";
import "./closing.css";
export default function Screen({ locale: l, mode, onState }: ScreenProps) {
  const hash = useLocation(),
    params = new URLSearchParams(hash.split("?")[1]);
  const workspace =
      params.get("scope") === "sample-b" ? "sample-b" : "sample-a",
    period = params.get("period") ?? params.get("close-period") ?? "2026-10",
    view = params.get("close-view") === "calendar" ? "calendar" : "list",
    date = params.get("close-date") ?? "",
    scenario = params.get("close-state") ?? "normal";
  const filter: Filter = {
    query: params.get("close-q") ?? "",
    owner: params.get("close-owner") ?? "",
    stage: params.get("close-stage") ?? "",
    state: params.get("close-status") ?? "",
    path: params.get("close-path") ?? "",
    overdue: params.get("close-overdue") === "yes",
  };
  const key = JSON.stringify([workspace, period, filter, date, scenario, mode]);
  const rows = useMemo(
    () =>
      mode === "synthetic" && scenario === "normal"
        ? fixtureTargets(workspace, period)
        : [],
    [workspace, period, scenario, mode],
  );
  const filtered = filterTargets(rows, filter),
    visible = date
      ? filtered.filter((r) => r.current.startsWith(date))
      : filtered;
  const page = usePage(visible, key);
  const [selection, setSelection] = useState<{
      key: string;
      id: string;
    } | null>(null),
    trigger = useRef<HTMLElement | null>(null);
  const selected =
    selection?.key === key
      ? visible.find((r) => r.id === selection.id)
      : undefined;
  function change(values: Record<string, string>) {
    setSelection(null);
    location.hash = routeUrl("core-index-close", {
      "close-date": "",
      ...values,
    });
  }
  function inspect(row: Target) {
    trigger.current = document.activeElement as HTMLElement;
    setSelection({ key, id: row.id });
  }
  function close() {
    setSelection(null);
    queueMicrotask(() => {
      if (trigger.current?.isConnected) trigger.current.focus();
    });
  }
  useEffect(() => {
    setSelection(null);
  }, [key]);
  useEffect(() => {
    onState(JSON.stringify({ key, view, selected: selected?.id }), [
      "core-index-close:region:layout",
      "U02:field:scope",
      "U02:field:filters",
      "U02:action:view",
      "U02:data:state-counts",
      "U02:data:targets",
      "U02:action:page",
      ...(view === "calendar" ? ["U02:data:calendar"] : []),
      ...(selected
        ? [
            "U02:section:conditions",
            "U02:section:history",
            "U02:section:recognition",
          ]
        : []),
      "U02:state:service",
    ]);
  }, [key, view, selected?.id, onState]);
  const reset = () =>
    change({
      "close-q": "",
      "close-owner": "",
      "close-stage": "",
      "close-status": "",
      "close-path": "",
      "close-overdue": "",
    });
  const ready = mode === "synthetic" && scenario === "normal";
  return (
    <div className="closing-screen">
      <div className="closing-scope">
        <label>
          {t(l, "Settlement cycle", "정산 주기", "دورة التسوية")}
          <select
            id="closing-period"
            value={period}
            onChange={(e) => change({ period: e.target.value })}
          >
            {!cyclePeriods.includes(
              period as (typeof cyclePeriods)[number],
            ) && <option>{period}</option>}
            {cyclePeriods.map((p) => (
              <option key={p}>{p}</option>
            ))}
          </select>
        </label>
        <div
          className="closing-views"
          role="group"
          aria-label={t(l, "Presentation", "보기 방식", "طريقة العرض")}
        >
          {(["list", "calendar"] as const).map((v) => (
            <button
              key={v}
              aria-pressed={view === v}
              onClick={() => {
                location.hash = routeUrl("core-index-close", {
                  "close-view": v,
                });
              }}
            >
              {v === "list"
                ? t(l, "List", "목록", "قائمة")
                : t(l, "Calendar", "달력", "تقويم")}
            </button>
          ))}
        </div>
        <p>
          {t(
            l,
            "Synthetic fixed snapshot",
            "합성 고정 스냅샷",
            "لقطة تجريبية ثابتة",
          )}{" "}
          <bdi>{asOf}</bdi> · UTC
        </p>
      </div>
      {ready && (
        <p className="closing-note">
          {t(
            l,
            "Read-only candidate · 1,200 example targets · no live connection. Original and current deadlines are supplied fixture timestamps, not an approved business calendar.",
            "조회 시안 · 예제 마감 대상 1,200건 · 실제 연동 없음. 원 기한·현재 기한은 예제 지정 시각이며 승인된 영업일 달력이 아닙니다.",
            "نموذج قراءة · 1200 هدف تجريبي · دون اتصال حي. المواعيد أوقات تجريبية وليست تقويماً معتمداً.",
          )}
        </p>
      )}
      {ready ? (
        <>
          <details className="closing-filter-panel" open>
            <summary>{t(l, "Filters", "조회 조건", "عوامل التصفية")}</summary>
            <div className="closing-filters">
              <label>
                {t(
                  l,
                  "Organization, contract or entity code",
                  "조직·계약·법인 코드",
                  "رمز المؤسسة أو العقد أو الكيان",
                )}
                <input
                  id="closing-query"
                  type="search"
                  value={filter.query}
                  onChange={(e) => change({ "close-q": e.target.value })}
                />
              </label>
              <label>
                {t(l, "Owner", "담당자", "المسؤول")}
                <select
                  id="closing-owner"
                  value={filter.owner}
                  onChange={(e) => change({ "close-owner": e.target.value })}
                >
                  <option value="">{t(l, "All", "전체", "الكل")}</option>
                  {Array.from({ length: 8 }, (_, i) => (
                    <option key={i}>OWNER-{i + 1}</option>
                  ))}
                </select>
              </label>
              <label>
                {t(l, "Stage", "단계", "المرحلة")}
                <select
                  id="closing-stage"
                  value={filter.stage}
                  onChange={(e) => change({ "close-stage": e.target.value })}
                >
                  <option value="">{t(l, "All", "전체", "الكل")}</option>
                  {stages.map((s) => (
                    <option key={s} value={s}>
                      {stageLabel(l, s)}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                {t(l, "State", "마감 상태", "الحالة")}
                <select
                  id="closing-status"
                  value={filter.state}
                  onChange={(e) => change({ "close-status": e.target.value })}
                >
                  <option value="">{t(l, "All", "전체", "الكل")}</option>
                  {states.map((s) => (
                    <option key={s} value={s}>
                      {stateLabel(l, s)}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                {t(l, "Contract path", "계약 경로", "مسار العقد")}
                <select
                  id="closing-path"
                  value={filter.path}
                  onChange={(e) => change({ "close-path": e.target.value })}
                >
                  <option value="">{t(l, "All", "전체", "الكل")}</option>
                  <option value="internal">
                    {t(l, "Internal", "내부 정산", "داخلي")}
                  </option>
                  <option value="external">
                    {t(l, "External", "외부 청구", "خارجي")}
                  </option>
                </select>
              </label>
              <label className="closing-check">
                <input
                  type="checkbox"
                  checked={filter.overdue}
                  onChange={(e) =>
                    change({ "close-overdue": e.target.checked ? "yes" : "" })
                  }
                />
                {t(l, "Overdue only", "기한 초과만", "المتأخر فقط")}
              </label>
              <button onClick={reset}>
                {t(l, "Reset filters", "필터 초기화", "إعادة تعيين")}
              </button>
            </div>
          </details>
          <section
            className="closing-counts"
            aria-label={t(
              l,
              "Filtered target states",
              "조회 대상 상태",
              "حالات الأهداف المصفاة",
            )}
          >
            {states.map((s) => (
              <div key={s}>
                <span>{stateLabel(l, s)}</span>
                <strong data-state-count={s}>
                  {filtered.filter((r) => r.state === s).length}
                </strong>
              </div>
            ))}
            <div>
              <span>
                {t(
                  l,
                  "Overdue · unfinished",
                  "기한 초과 · 미완료 대상",
                  "متأخر · غير منته",
                )}
              </span>
              <strong>{filtered.filter(isOverdue).length}</strong>
            </div>
          </section>
          <p className="closing-note">
            {t(
              l,
              "Counts use all filtered targets, before a date is selected. Exempt, held and complete are separate. Amounts: Not available.",
              "건수는 날짜 선택 전 필터 전체 대상 기준입니다. 면제·보류·완료를 별도로 집계합니다. 금액 정보는 미연결입니다.",
              "الأعداد لكل الأهداف المصفاة قبل اختيار التاريخ. الإعفاء والتعليق والإكمال منفصلة. المبالغ غير متاحة.",
            )}
          </p>
          {date && (
            <p>
              <bdi>{date}</bdi> · UTC{" "}
              <button onClick={() => change({ "close-date": "" })}>
                {t(l, "Clear date", "날짜 선택 해제", "مسح التاريخ")}
              </button>
            </p>
          )}
          {view === "calendar" ? (
            <Calendar
              rows={filtered}
              month={deadlineMonth(period)}
              period={period}
              locale={l}
              onDate={(d) => change({ "close-date": d, "close-view": "list" })}
            />
          ) : (
            <section className="closing-list">
              <header>
                <h2>{t(l, "Closing targets", "마감 대상", "أهداف الإقفال")}</h2>
                <span data-closing-total>{visible.length}</span>
              </header>
              {!visible.length ? (
                <p role="status">
                  {t(
                    l,
                    "No closing targets match the filters.",
                    "조건에 맞는 마감 대상이 없습니다.",
                    "لا توجد أهداف مطابقة.",
                  )}
                </p>
              ) : (
                <div className="table-wrap">
                  <table>
                    <thead>
                      <tr>
                        {[
                          t(
                            l,
                            "Contract / cycle",
                            "계약 / 주기",
                            "العقد / الدورة",
                          ),
                          t(l, "Stage", "단계", "المرحلة"),
                          t(
                            l,
                            "Original deadline · UTC",
                            "원 기한 · UTC",
                            "الموعد الأصلي · UTC",
                          ),
                          t(
                            l,
                            "Current deadline · UTC",
                            "현재 기한 · UTC",
                            "الموعد الحالي · UTC",
                          ),
                          t(l, "Deadline status", "기한 상태", "حالة الموعد"),
                          t(l, "Closing state", "마감 상태", "حالة الإقفال"),
                          t(
                            l,
                            "Blocker / owner",
                            "차단 / 담당",
                            "العائق / المسؤول",
                          ),
                        ].map((h) => (
                          <th key={h}>{h}</th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {page.rows.map((r) => (
                        <tr key={r.id}>
                          <th>
                            <button
                              className="closing-open"
                              onClick={() => inspect(r)}
                            >
                              <bdi>{r.contract}</bdi>
                            </button>
                            <small>
                              <bdi>
                                {r.organization} · {r.period}
                              </bdi>
                            </small>
                            <small>
                              {r.internal
                                ? t(l, "Internal", "내부 정산", "داخلي")
                                : t(l, "External", "외부 청구", "خارجي")}
                            </small>
                          </th>
                          <td>{stageLabel(l, r.stage)}</td>
                          <td>
                            <bdi>
                              {r.original.slice(0, 16).replace("T", " ")}
                            </bdi>
                          </td>
                          <td>
                            <bdi>
                              {r.current.slice(0, 16).replace("T", " ")}
                            </bdi>
                          </td>
                          <td>
                            {r.state === "complete" || r.state === "exempt"
                              ? t(
                                  l,
                                  "Not open",
                                  "미결 대상 아님",
                                  "ليس مفتوحاً",
                                )
                              : isOverdue(r)
                                ? t(l, "Overdue", "기한 초과", "متأخر")
                                : t(l, "Upcoming", "예정", "قادم")}
                          </td>
                          <td>{stateLabel(l, r.state)}</td>
                          <td>
                            {r.erp === "unknown"
                              ? t(
                                  l,
                                  "ERP unconfirmed",
                                  "ERP 미확인",
                                  "ERP غير مؤكد",
                                )
                              : "—"}
                            <small>
                              <bdi>{r.owner}</bdi>
                            </small>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <Paging value={page} locale={l} />
            </section>
          )}
          {selected && <Detail row={selected} locale={l} onClose={close} />}
        </>
      ) : (
        <section className="state" role="status">
          <h2>
            {mode === "unconnected"
              ? t(
                  l,
                  "Closing service not connected",
                  "마감 서비스 미연결",
                  "خدمة الإقفال غير متصلة",
                )
              : scenario === "revoked"
                ? t(
                    l,
                    "Preview access revoked",
                    "예제 접근 철회",
                    "إلغاء الوصول التجريبي",
                  )
                : scenario === "error"
                  ? t(
                      l,
                      "Closing conditions could not be verified",
                      "마감 조건을 확인할 수 없습니다",
                      "تعذر التحقق من شروط الإقفال",
                    )
                  : t(
                      l,
                      "No closing targets",
                      "마감 대상 없음",
                      "لا توجد أهداف",
                    )}
          </h2>
          <p>
            {t(
              l,
              "No target data is shown in this state. Production permissions and service behavior are not connected.",
              "이 상태에서는 대상 정보를 표시하지 않습니다. 실제 권한·서비스는 미연결입니다.",
              "لا تعرض بيانات الأهداف هنا. الصلاحيات والخدمة الفعلية غير متصلة.",
            )}
          </p>
          {mode === "synthetic" && (
            <button onClick={() => change({ "close-state": "normal" })}>
              {t(l, "Restore example", "예제 다시 보기", "استعادة المثال")}
            </button>
          )}
        </section>
      )}
      <details className="closing-scenarios">
        <summary>
          {t(l, "Preview state checks", "예제 상태 검증", "فحص حالات المعاينة")}
        </summary>
        <p>
          {t(
            l,
            "Local simulations only; no real permission change.",
            "로컬 예제 전환이며 실제 권한을 변경하지 않습니다.",
            "محاكاة محلية فقط دون تغيير صلاحيات فعلية.",
          )}
        </p>
        {["normal", "empty", "error", "revoked"].map((s) => (
          <button key={s} onClick={() => change({ "close-state": s })}>
            {t(
              l,
              ...(
                {
                  normal: ["Normal", "정상", "عادي"],
                  empty: ["Empty", "빈 결과", "فارغ"],
                  error: ["Error", "오류", "خطأ"],
                  revoked: ["Revoked", "접근 철회", "إلغاء الوصول"],
                } as Record<string, [string, string, string]>
              )[s],
            )}
          </button>
        ))}
      </details>
    </div>
  );
}
