import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import { useService } from "../../../../service/context";
import { CommandFailure } from "./command-failure";
import { c } from "../../../../service/copy";
import { w } from "../users/live-copy";
import { qcopy as t } from "./query-copy";
import { Facts, bdi } from "./query-ui";
import type { Recovery } from "./recovery-context";
import type { IntentKind } from "./command-intents";
import type { StepUpState } from "../../../../service/service-query-types";

export function usePresent() {
  const active = useRef(false);
  useLayoutEffect(() => {
    active.current = true;
    return () => {
      active.current = false;
    };
  }, []);
  return active;
}
export function useIntents(recovery: Recovery) {
  useSyncExternalStore(recovery.commands.subscribe, recovery.commands.version);
}
export function settledAfterSending(before: string | undefined, now: string | undefined): boolean {
  return before === "sending" && (now === "refused" || now === "unknown" || now === "lapsed");
}
export function IntentNotice({
  l,
  recovery,
  kind,
}: {
  l: Locale;
  recovery: Recovery;
  kind: IntentKind;
}) {
  useIntents(recovery);
  const entry = recovery.commands.get(kind);
  const title = useRef<HTMLHeadingElement>(null);
  const seen = useRef(entry?.phase);
  const phase = entry?.phase;
  useEffect(() => {
    if (settledAfterSending(seen.current, phase)) title.current?.focus();
    seen.current = phase;
  }, [phase]);
  if (!entry) return null;
  return (
    <section className="live-held" aria-live="polite">
      <h3 ref={title} tabIndex={-1}>
        {entry.phase === "done"
          ? t(l, "done")
          : entry.phase === "unknown"
            ? t(l, "unknown")
            : entry.phase === "lapsed"
              ? vLapsed(l)
              : entry.phase === "refused"
                ? t(l, entry.hadUnknown ? "unknown" : "refused")
                : w.saving(l)}
      </h3>
      <p>{t(l, "held")}</p>
      <Facts
        items={[
          [t(l, "action"), bdi(entry.draft.command)],
          [t(l, "target"), bdi(entry.draft.target)],
          [t(l, "expectedRevision"), bdi(entry.draft.revision)],
          [w.reason(l), entry.draft.reason],
        ]}
      />
      {entry.error != null && (
        <CommandFailure
          l={l}
          error={entry.error}
          variant="inline"
          outcome={entry.hadUnknown ? "unknown" : "refused"}
        />
      )}
      {entry.phase !== "sending" && (
        <>
          <p>{t(l, "discardNote")}</p>
          <button
            type="button"
            className="live-secondary"
            onClick={() => recovery.commands.discard(kind)}
          >
            {t(l, "discard")}
          </button>
        </>
      )}
    </section>
  );
}
function vLapsed(l: Locale) {
  return l === "ko"
    ? "원본 재시도 기간 지남"
    : l === "ar"
      ? "انتهت فترة إعادة محاولة الطلب الأصلي"
      : "Original retry window elapsed";
}
export function Reauthenticate({
  l,
  state,
  refresh,
}: {
  l: Locale;
  state: StepUpState;
  refresh: () => void;
}) {
  const service = useService(),
    present = usePresent();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null);
  if (state === "ready") return null;
  const run = async () => {
    if (busy || !service?.stepUp) return;
    setBusy(true);
    setError(null);
    try {
      if (await service.stepUp()) {
        if (present.current) refresh();
      }
    } catch (err) {
      if (present.current) setError(err);
    } finally {
      if (present.current) setBusy(false);
    }
  };
  return (
    <div className="live-note-rule">
      <p>
        {state === "reauthentication_required"
          ? t(l, "reviewAgain")
          : state === "prerequisite_failed"
            ? t(l, "authenticationPrerequisite")
            : t(l, "authenticationUnavailable")}
      </p>
      {error != null && (
        <CommandFailure l={l} error={error} outcome="confirmation" />
      )}
      {state === "reauthentication_required" && (
        <button
          type="button"
          className="live-secondary"
          disabled={busy || !service?.stepUp}
          onClick={run}
        >
          {t(l, "reauth")}
        </button>
      )}
      {state !== "reauthentication_required" && (
        <button type="button" className="live-secondary" onClick={refresh}>
          {w.reload(l)}
        </button>
      )}
    </div>
  );
}
export function RetainedCommands({
  l,
  api,
  recovery,
  children,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  children?: ReactNode;
}) {
  useIntents(recovery);
  return (
    <>
      {(["create", "declare", "state", "revoke"] as const).map((kind) => {
        const entry = recovery.commands.get(kind);
        return entry ? (
          <div key={kind}>
            <IntentNotice l={l} recovery={recovery} kind={kind} />
            {entry.phase === "unknown" && (
              <button
                type="button"
                className="live-secondary"
                onClick={() =>
                  recovery.commands.submit(
                    kind,
                    undefined,
                    undefined,
                    (d, tried) =>
                      api.command(
                        d.command,
                        d.target,
                        d.revision,
                        d.reason,
                        { ...d.fields },
                        tried,
                      ),
                  )
                }
              >
                {c.tryAgain(l)}
              </button>
            )}
          </div>
        ) : null;
      })}
      {children}
    </>
  );
}

export function BoundedReason({
  l,
  value,
  limit,
  onChange,
  error,
}: {
  l: Locale;
  value: string;
  limit: number;
  onChange: (value: string) => void;
  error: string;
}) {
  const bytes = new TextEncoder().encode(value).length;
  const hint =
    l === "ko"
      ? `이 변경과 함께 기록됩니다. 최대 ${limit}바이트입니다.`
      : l === "ar"
        ? `يُسجّل مع هذا التغيير. الحد الأقصى ${limit} بايت.`
        : `Recorded with this change. Up to ${limit} bytes.`;
  return (
    <div className="service-field live-reason">
      <label htmlFor="live-reason">{w.reason(l)}</label>
      <p className="service-hint" id="live-reason-hint">
        {hint}
      </p>
      {error && (
        <p className="service-field-error" id="live-reason-error">
          {error}
        </p>
      )}
      <textarea
        id="live-reason"
        rows={3}
        value={value}
        aria-invalid={error ? true : undefined}
        aria-describedby={
          error ? "live-reason-hint live-reason-error" : "live-reason-hint"
        }
        onChange={(e) => onChange(e.target.value)}
      />
      <p className="live-counter">
        <bdi dir="ltr">
          {bytes} / {limit}
        </bdi>{" "}
        {l === "ko" ? "바이트" : l === "ar" ? "بايت" : "bytes"}
      </p>
    </div>
  );
}
