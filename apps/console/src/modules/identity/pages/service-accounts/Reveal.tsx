import { useCallback, useEffect, useRef, useState } from "react";
import { flushSync } from "react-dom";
import type { Locale } from "../../../../shared/types";
import type { AdminClient } from "../../../../service/admin";
import { c } from "../../../../service/copy";
import { Summary } from "../../../../service/Service";
import { guardUnload } from "../../../../shared/unload-intent";
import { when } from "../authentication/live-model";
import { v } from "./live-copy";
import { qcopy as t } from "./query-copy";
import type { IssueAttempts } from "./issue-attempts";
import { Facts, bdi as ltr } from "./query-ui";
export function Reveal({
  l,
  api,
  issues,
  user,
  service,
  refId,
  credential,
  until,
  untilEstimated = false,
  onDone,
  onRevoke,
  onLeave,
}: {
  l: Locale;
  api: AdminClient;
  issues: IssueAttempts;
  user: string;
  service: string;
  refId: string;
  credential: string;
  until: string;
  untilEstimated?: boolean;
  onDone: () => void;
  onRevoke: (notStored: boolean) => void;
  onLeave: () => void;
}) {
  const [secret, setSecret] = useState("");
  const [returned, setReturned] = useState(false);
  const [state, setState] = useState<"loading" | "shown" | "retry" | "gone">(
    "loading",
  );
  const [ack, setAck] = useState(false);
  const [problem, setProblem] = useState("");
  const [focusKey, setFocusKey] = useState(0);
  const retried = useRef(false);
  const observer = useRef(0);
  const heading = useRef<HTMLHeadingElement>(null);
  const fetchSecret = useCallback(
    (retry = false) => {
      if (!issues.current() || document.visibilityState !== "visible") {
        observer.current++;
        setState("gone");
        return;
      }
      const generation = ++observer.current;
      setState("loading");
      const visible = () =>
        observer.current === generation &&
        issues.current() &&
        document.visibilityState === "visible";
      issues
        .collect(
          user,
          service,
          { credential, ref: refId },
          (reference) => api.secret(reference),
          retry,
        )
        .then(
          (got) => {
            if (!visible()) return;
            setSecret(got.secret);
            setReturned(true);
            setState("shown");
          },
          () => {
            if (visible()) setState("retry");
          },
        );
    },
    [api, issues, user, service, refId, credential],
  );
  useEffect(() => {
    if (refId) fetchSecret();
    else setState("gone");
    return () => {
      observer.current++;
    };
  }, [fetchSecret, refId]);
  useEffect(() => heading.current?.focus(), []);
  useEffect(() => {
    if (retried.current && state === "shown") heading.current?.focus();
  }, [state]);
  useEffect(() => {
    if (state !== "shown") return;
    const warn = guardUnload(() => true);
    addEventListener("beforeunload", warn);
    return () => removeEventListener("beforeunload", warn);
  }, [state]);
  useEffect(() => {
    const drop = () => {
      observer.current++;
      flushSync(() => {
        setSecret("");
        setState("gone");
      });
    };
    addEventListener("pagehide", drop);
    return () => removeEventListener("pagehide", drop);
  }, []);
  const finish = () => {
    if (state === "shown" && !ack) {
      setProblem(v.ackError(l));
      setFocusKey((k) => k + 1);
      return;
    }
    setSecret("");
    onDone();
  };
  return (
    <section className="live-step" aria-labelledby="svc-reveal-title">
      <p className="service-status">{v.doneIssued(l, credential)}</p>
      <h2 id="svc-reveal-title" tabIndex={-1} ref={heading}>
        {v.revealTitle(l)}
      </h2>
      <Facts
        items={[
          [v.colService(l), ltr(service)],
          [v.colCred(l), <span className="live-mono">{ltr(credential)}</span>],
          [
            untilEstimated ? t(l, "estimatedExpiry") : v.colValid(l),
            v.until(l, when(l, until)),
          ],
        ]}
      />
      {problem && (
        <Summary
          locale={l}
          title={c.check(l)}
          problems={[{ field: "svc-ack", message: problem }]}
          focusKey={focusKey}
        />
      )}
      {problem && (
        <p className="visually-hidden" id="svc-ack-error">
          {problem}
        </p>
      )}
      {state === "loading" && <p role="status">{c.loading(l)}</p>}
      {state === "shown" && (
        <>
          <div className="service-field">
            <label htmlFor="svc-secret">{v.secretLabel(l)}</label>
            <p className="service-hint" id="svc-secret-hint">
              {v.secretHint(l)}
            </p>
            <textarea
              id="svc-secret"
              className="live-link live-secret"
              readOnly
              rows={2}
              dir="ltr"
              spellCheck={false}
              autoComplete="off"
              autoCapitalize="off"
              autoCorrect="off"
              translate="no"
              aria-describedby="svc-secret-hint svc-secret-once"
              value={secret}
            />
          </div>
          <p className="live-impact" id="svc-secret-once">
            <strong>{v.shownOnce(l)}</strong>
          </p>
          <p className="live-impact">
            {v.prelaunchLead(l)} {v.prelaunchNote(l)}
          </p>
          <label className="live-check">
            <input
              id="svc-ack"
              type="checkbox"
              checked={ack}
              aria-invalid={problem ? true : undefined}
              aria-describedby={problem ? "svc-ack-error" : undefined}
              onChange={(e) => {
                setAck(e.target.checked);
                setProblem("");
              }}
            />
            {v.ackLabel(l)}
          </label>
        </>
      )}
      {state === "retry" && (
        <div className="service-summary" role="alert">
          <p>{v.secretRetry(l)}</p>
          <button
            type="button"
            className="live-secondary"
            onClick={() => {
              retried.current = true;
              fetchSecret(true);
            }}
          >
            {v.showSecret(l)}
          </button>
        </div>
      )}
      {state === "gone" && (
        <p className="live-impact">
          {returned ? v.secretReturned(l) : v.secretGone(l)}
        </p>
      )}
      <div className="live-buttons">
        {state === "shown" ? (
          <button type="button" className="live-primary" onClick={finish}>
            {v.done(l)}
          </button>
        ) : (
          <button type="button" className="live-secondary" onClick={onLeave}>
            {v.close(l)}
          </button>
        )}
        <button
          type="button"
          className={state === "gone" ? "live-primary" : "live-secondary"}
          onClick={() => {
            setSecret("");
            onRevoke(state === "shown");
          }}
        >
          {state === "shown" ? v.notStored(l) : v.revokeTitle(l)}
        </button>
      </div>
    </section>
  );
}
