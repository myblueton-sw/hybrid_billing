import { useEffect, useRef, useState } from "react";
import type { Locale } from "./types";
import { text as t } from "./copy";
import {
  digest,
  readRecords,
  saveRecord,
  type ReviewRecord,
} from "./review-store";
export default function ReviewPanel({
  screen,
  locale: l,
  context,
  items,
}: {
  screen: string;
  locale: Locale;
  context: string;
  items: string[];
}) {
  const [width, setWidth] = useState(innerWidth),
    [records, setRecords] = useState<ReviewRecord[]>([]),
    [error, setError] = useState(false),
    [scope, setScope] = useState(""),
    [pin, setPin] = useState(""),
    [ack, setAck] = useState(false),
    [busy, setBusy] = useState(false);
  const dialog = useRef<HTMLDialogElement>(null),
    trigger = useRef<HTMLButtonElement>(null);
  const parsed = JSON.parse(context),
    state = JSON.parse(parsed.state || "{}");
  const variants = {
    locale: l,
    width,
    profile: parsed.profile,
    mode: parsed.mode,
    extensions: parsed.extensions,
    state: {
      phase: state.phase ?? "overview",
      workspace: state.scope ?? "none",
      filter: state.filter ?? "none",
      page: state.page ?? 0,
      scenario: state.scenario ?? "none",
      selectedCount: state.selected?.length ?? 0,
      detailOpen: !!state.detail,
    },
  };
  const ackLive = useRef(ack);
  ackLive.current = ack;
  const generation = useRef(0);
  const current = JSON.stringify({
    screen,
    revision: __SOURCE_REVISION__,
    locale: l,
    width,
    context,
    items,
  });
  const live = useRef(current);
  live.current = current;
  useEffect(() => {
    const resize = () => setWidth(innerWidth);
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, []);
  useEffect(() => {
    try {
      setRecords(readRecords(localStorage));
      setError(false);
    } catch {
      setError(true);
    }
  }, []);
  useEffect(() => {
    let active = true;
    setScope("");
    digest(current)
      .then((value) => {
        if (active) setScope(value);
      })
      .catch(() => {
        if (active) setError(true);
      });
    return () => {
      active = false;
    };
  }, [current]);
  const fixed =
    !error &&
    scope &&
    records.some(
      (r) =>
        r.screen === screen &&
        r.revision === __SOURCE_REVISION__ &&
        r.scope === scope,
    );
  const prior = records.some((r) => r.screen === screen);
  function close() {
    generation.current++;
    dialog.current?.close();
    setAck(false);
    trigger.current?.focus();
  }
  async function save() {
    if (!ack || busy || pin !== live.current) return;
    const attempt = generation.current;
    setBusy(true);
    try {
      const scope = await digest(pin);
      if (
        !ackLive.current ||
        attempt !== generation.current ||
        pin !== live.current ||
        innerWidth !== width ||
        !dialog.current?.open
      )
        return;
      const next = saveRecord(localStorage, {
        screen,
        revision: __SOURCE_REVISION__,
        scope,
        items: [...items],
        variants,
        confirmedAt: new Date().toISOString(),
        evidence: "local-owner-acknowledgement",
      });
      setRecords(next);
      setError(false);
      close();
    } catch {
      setError(true);
    } finally {
      setBusy(false);
    }
  }
  function exportRecords() {
    try {
      const records = readRecords(localStorage);
      const url = URL.createObjectURL(
        new Blob([JSON.stringify({ format: 1, records }, null, 2)], {
          type: "application/json",
        }),
      );
      const a = document.createElement("a");
      a.href = url;
      a.download = "screen-review-evidence.json";
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      setError(false);
    } catch {
      setError(true);
    }
  }
  return (
    <section
      className="review"
      aria-label={t(
        l,
        "Screen design review",
        "화면 설계 검수",
        "مراجعة تصميم الشاشة",
      )}
    >
      <div className="review-heading">
        <div>
          <strong>
            {t(
              l,
              "Screen design review",
              "화면 설계 검수",
              "مراجعة تصميم الشاشة",
            )}
          </strong>
          <p>
            <bdi>
              {screen} · {__SOURCE_REVISION__.slice(0, 12)}
            </bdi>
          </p>
        </div>
        <span id="review-status" className="tag">
          {fixed
            ? t(l, "FIX · local record", "FIX · 로컬 기록", "FIX · سجل محلي")
            : prior
              ? t(
                  l,
                  "Scope not confirmed",
                  "현재 범위 미확인",
                  "النطاق الحالي غير مؤكد",
                )
              : t(l, "Ready for review", "검수 대기", "جاهز للمراجعة")}
        </span>
        <button
          id="confirm-review"
          ref={trigger}
          disabled={!scope || items.length === 0}
          onClick={() => {
            generation.current++;
            setPin(current);
            setAck(false);
            dialog.current?.showModal();
          }}
        >
          {t(l, "Confirm screen review", "검수 확인", "تأكيد مراجعة الشاشة")}
        </button>
      </div>
      <p>
        {t(
          l,
          "Local browser evidence only. This is not authenticated project acceptance or financial approval.",
          "이 브라우저의 로컬 검수 기록입니다. 인증된 프로젝트 확정이나 금전 승인이 아닙니다.",
          "دليل محلي في هذا المتصفح فقط، وليس اعتمادًا موثقًا للمشروع أو موافقة مالية.",
        )}
      </p>
      <details>
        <summary>
          {t(
            l,
            "Implementation item IDs",
            "구현 항목 ID",
            "معرّفات عناصر التنفيذ",
          )}
        </summary>
        <ul>
          {items.map((id) => (
            <li key={id}>
              <bdi>{id}</bdi>
            </li>
          ))}
        </ul>
      </details>
      {error && (
        <p role="alert">
          {t(
            l,
            "Review storage unavailable or invalid. FIX has not been verified as saved.",
            "검수 저장소를 사용할 수 없거나 기록이 손상되었습니다. FIX 저장을 확인할 수 없습니다.",
            "تخزين المراجعة غير متاح أو غير صالح. لم يتم التحقق من حفظ FIX.",
          )}
        </p>
      )}
      <button onClick={exportRecords}>
        {t(
          l,
          "Export review evidence",
          "검수 기록 내보내기",
          "تصدير دليل المراجعة",
        )}
      </button>
      <dialog ref={dialog} aria-labelledby="review-title" onCancel={close}>
        <h2 id="review-title">
          {t(
            l,
            "Confirm this exact scope",
            "현재 범위 검수 확인",
            "تأكيد هذا النطاق المحدد",
          )}
        </h2>
        <p>
          <bdi>
            {screen} · {l} · {width}px
          </bdi>
        </p>
        <p>
          {t(
            l,
            "Only the current profile, data mode, visible state and listed items are included. Backend integration, other variants and native-language acceptance are deferred.",
            "현재 설치 구성·데이터 모드·표시 상태·항목만 포함됩니다. 백엔드 연동, 다른 변형 화면 및 원어민 검수는 제외됩니다.",
            "يشمل ملف التثبيت ونمط البيانات والحالة والعناصر الحالية فقط. الربط الخلفي والحالات الأخرى والمراجعة اللغوية مؤجلة.",
          )}
        </p>
        <ul>
          {items.map((id) => (
            <li key={id}>
              <bdi>{id}</bdi>
            </li>
          ))}
        </ul>
        <label className="check">
          <input
            id="owner-ack"
            type="checkbox"
            checked={ack}
            onChange={(e) => setAck(e.target.checked)}
          />
          {t(
            l,
            "I am the owner confirming this scope as a local record.",
            "소유자로서 이 범위를 확인하고 로컬 기록으로 남깁니다.",
            "أنا المالك وأؤكد هذا النطاق كسجل محلي.",
          )}
        </label>
        {pin !== current && (
          <p role="alert">
            {t(
              l,
              "Scope changed. Close and reopen review.",
              "범위가 변경되었습니다. 닫고 다시 검수하세요.",
              "تغير النطاق. أغلق المراجعة وافتحها مجددًا.",
            )}
          </p>
        )}
        {error && (
          <p role="alert">
            {t(
              l,
              "Save failed. No persisted FIX confirmed.",
              "저장 실패: FIX가 저장되었다고 확인할 수 없습니다.",
              "فشل الحفظ. لم يتم تأكيد حفظ FIX.",
            )}
          </p>
        )}
        <div className="actions">
          <button onClick={close}>{t(l, "Cancel", "취소", "إلغاء")}</button>
          <button
            id="save-review"
            className="primary"
            disabled={!ack || busy || pin !== current}
            onClick={save}
          >
            {t(l, "Save local FIX", "로컬 FIX 저장", "حفظ FIX محليًا")}
          </button>
        </div>
      </dialog>
    </section>
  );
}
