import { useState, useSyncExternalStore, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import type { AdminClient, CommandId } from "../../../../service/admin";
import type {
  AccountDetail,
  CredentialDetail,
} from "../../../../service/service-query-types";
import { reasonProblem } from "../users/Live";
import { w } from "../users/live-copy";
import { c } from "../../../../service/copy";
import { v } from "./live-copy";
import { qcopy as t } from "./query-copy";
import { Facts, bdi } from "./query-ui";
import {
  BoundedReason,
  IntentNotice,
  Reauthenticate,
  useIntents,
  usePresent,
} from "./command-ui";
import type { Recovery } from "./recovery-context";
import { wholeDays } from "./live-model";
import { utf8Length } from "../../../../service/helpers";

export type AccountAction = "suspend" | "reinstate" | "retire";
export function ActionForm({
  l,
  api,
  recovery,
  detail,
  action,
  initial = "",
  refresh,
  onDone,
  onClose,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  detail: AccountDetail | CredentialDetail;
  action: AccountAction | "revoke";
  initial?: string;
  refresh: () => void;
  onDone: () => void;
  onClose: () => void;
}) {
  useIntents(recovery);
  const present = usePresent();
  const kind = action === "revoke" ? "revoke" : "state",
    entry = recovery.commands.get(kind);
  const [reason, setReason] = useState(initial),
    [problem, setProblem] = useState("");
  const title =
    action === "suspend"
      ? v.suspendTitle(l)
      : action === "reinstate"
        ? v.reinstateTitle(l)
        : action === "retire"
          ? v.retireTitle(l)
          : v.revokeTitle(l);
  const allowed =
    "can_revoke" in detail.eligibility
      ? detail.eligibility.can_revoke
      : detail.eligibility[
          action === "suspend"
            ? "can_suspend"
            : action === "reinstate"
              ? "can_reinstate"
              : "can_retire"
        ];
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (entry && entry.phase !== "unknown") return;
    if (
      !entry &&
      (!allowed ||
        reasonProblem(l, reason) ||
        utf8Length(reason) > detail.limits.rationale_bytes)
    ) {
      setProblem(reasonProblem(l, reason) || t(l, "blocked"));
      return;
    }
    const done = await recovery.commands.submit(
      kind,
      entry
        ? undefined
        : {
            command: (action === "revoke"
              ? "service.revoke_credential"
              : "service." + action) as CommandId,
            target: detail.row.id,
            reason,
            fields: {},
            revision: detail.revision,
            replay: detail.limits.replay_seconds,
          },
      detail.limits,
      (draft, tried) =>
        api.command(
          draft.command,
          draft.target,
          draft.revision,
          draft.reason,
          { ...draft.fields },
          tried,
        ),
    );
    if (done && present.current && recovery.issues.current()) {
      recovery.commands.discard(kind);
      onDone();
    }
  };
  if (
    entry &&
    (entry.draft.target !== detail.row.id ||
      entry.draft.command !==
        (action === "revoke"
          ? "service.revoke_credential"
          : "service." + action))
  )
    return <IntentNotice l={l} recovery={recovery} kind={kind} />;
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 tabIndex={-1}>{title}</h2>
      <p className="live-mono">{bdi(entry?.draft.target ?? detail.row.id)}</p>
      <IntentNotice l={l} recovery={recovery} kind={kind} />
      <Reauthenticate
        l={l}
        state={detail.eligibility.step_up_state}
        refresh={refresh}
      />
      <p className="live-impact">
        {action === "revoke"
          ? v.revokeImpact(l, detail.row.id)
          : action === "suspend"
            ? v.suspendImpact(l)
            : action === "reinstate"
              ? v.reinstateImpact(l)
              : v.retireImpact(l)}
      </p>
      <fieldset disabled={!!entry}>
        <BoundedReason
          l={l}
          limit={detail.limits.rationale_bytes}
          value={entry?.draft.reason ?? reason}
          onChange={(value) => {
            if (utf8Length(value) <= detail.limits.rationale_bytes)
              setReason(value);
            else setProblem(t(l, "bounds"));
          }}
          error={problem}
        />
      </fieldset>
      <div className="live-buttons">
        <button
          className="live-primary"
          type="submit"
          disabled={entry ? entry.phase !== "unknown" : !allowed}
        >
          {entry?.phase === "unknown" ? c.tryAgain(l) : title}
        </button>
        <button className="live-secondary" type="button" onClick={onClose}>
          {w.cancel(l)}
        </button>
      </div>
    </form>
  );
}

export function IssueControl({
  l,
  api,
  recovery,
  detail,
  user,
  refresh,
  show,
}: {
  l: Locale;
  api: AdminClient;
  recovery: Recovery;
  detail: AccountDetail;
  user: string;
  refresh: () => void;
  show: () => void;
}) {
  const { issues } = recovery;
  useSyncExternalStore(issues.subscribe, issues.version);
  const present = usePresent();
  const entry = issues.get(user, detail.row.id),
    refusal = issues.refusal(user, detail.row.id);
  const [days, setDays] = useState(String(detail.limits.credential_days)),
    [reason, setReason] = useState(""),
    [problem, setProblem] = useState("");
  const locked = !!entry;
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (entry && entry.phase !== "unknown") return;
    if (
      !entry &&
      (!detail.eligibility.can_issue ||
        !wholeDays(days, detail.limits.credential_days) ||
        reasonProblem(l, reason) ||
        utf8Length(reason) > detail.limits.rationale_bytes)
    ) {
      setProblem(reasonProblem(l, reason) || t(l, "required"));
      return;
    }
    await issues.submit(
      user,
      detail.row.id,
      { days: entry?.days ?? days, reason: entry?.reason ?? reason },
      detail.revision,
      detail.limits.replay_seconds,
      (tried, expected, frozen) =>
        api.command(
          "service.issue_credential",
          detail.row.id,
          expected,
          frozen.reason,
          { days: frozen.days },
          tried,
        ),
    );
    if (present.current && issues.current()) refresh();
  };
  return (
    <form className="live-step" onSubmit={submit} noValidate>
      <h2 id="svc-issue-title" tabIndex={-1}>
        {v.issueTitle(l)}
      </h2>
      <p className="live-mono">{bdi(detail.row.id)}</p>
      <Facts
        items={[
          [t(l, "state"), detail.eligibility.state],
          [
            v.credentialsFact(l),
            detail.eligibility.live_credentials +
              " / " +
              detail.eligibility.credential_limit,
          ],
        ]}
      />
      <Reauthenticate
        l={l}
        state={detail.eligibility.step_up_state}
        refresh={refresh}
      />
      {!entry && !detail.eligibility.can_issue && <p>{t(l, "blocked")}</p>}
      {problem && <p role="alert">{problem}</p>}
      <fieldset disabled={locked}>
        <label>
          {v.validFor(l)}
          <input
            id="svc-days"
            inputMode="numeric"
            value={entry?.days ?? days}
            onChange={(e) => {
              if (e.target.value.length <= 3) setDays(e.target.value);
            }}
          />
        </label>
        <p>{v.daysHint(l, detail.limits.credential_days)}</p>
        <BoundedReason
          l={l}
          limit={detail.limits.rationale_bytes}
          value={entry?.reason ?? reason}
          onChange={(value) => {
            if (utf8Length(value) <= detail.limits.rationale_bytes)
              setReason(value);
            else setProblem(t(l, "bounds"));
          }}
          error=""
        />
      </fieldset>
      {entry && (
        <p role="status">
          {entry.phase === "unknown"
            ? t(l, "unknown")
            : entry.phase === "issued"
              ? v.attemptIssued(l, entry.issued!.credential, entry.service)
              : entry.phase}
        </p>
      )}
      {(entry?.error || refusal) && <p role="alert">{t(l, "blocked")}</p>}
      <div className="live-buttons">
        <button
          className="live-primary"
          type="submit"
          disabled={
            entry ? entry.phase !== "unknown" : !detail.eligibility.can_issue
          }
        >
          {entry?.phase === "unknown" ? c.tryAgain(l) : v.issueTitle(l)}
        </button>
        {entry?.phase === "issued" && (
          <button className="live-secondary" type="button" onClick={show}>
            {v.showSecret(l)}
          </button>
        )}
        {entry && entry.phase !== "sending" && entry.phase !== "issued" && (
          <button
            className="live-secondary"
            type="button"
            onClick={() => issues.discard(user, detail.row.id)}
          >
            {v.discardAttempt(l)}
          </button>
        )}
      </div>
    </form>
  );
}
