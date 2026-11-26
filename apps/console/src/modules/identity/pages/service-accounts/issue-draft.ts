import type { ServiceLimits } from "../../../../service/service-query-types";
import { admitsCommand } from "./command-intents";

export type IssueDraft = Readonly<{
  target: string;
  days: string;
  reason: string;
}>;

export function admitsIssueDraft(draft: IssueDraft, limits: ServiceLimits) {
  return (
    /^[A-Za-z0-9._:-]{1,128}$/.test(draft.target) &&
    draft.days.length <= 3 &&
    admitsCommand(
      {
        command: "service.issue_credential",
        target: draft.target,
        reason: draft.reason,
        fields: { days: draft.days },
        revision: "r".repeat(128),
        replay: 1,
      },
      limits,
      { request: "r".repeat(128), key: "k".repeat(128) },
    )
  );
}

export function admitsRotateDraft(draft: IssueDraft & { overlap: string }, limits: ServiceLimits) {
  return (
    /^[A-Za-z0-9._:-]{1,128}$/.test(draft.target) &&
    draft.days.length <= 3 &&
    draft.overlap.length <= 6 &&
    admitsCommand(
      { command: "service.rotate_credential", target: draft.target, reason: draft.reason, fields: { days: draft.days, overlap_minutes: draft.overlap }, revision: "r".repeat(128), replay: 1 },
      limits,
      { request: "r".repeat(128), key: "k".repeat(128) },
    )
  );
}

export function issueDrafts(current: () => boolean) {
  let value: IssueDraft | undefined,
    disposed = false,
    version = 0;
  const listeners = new Set<() => void>();
  const active = () => !disposed && current();
  const changed = () => {
    version++;
    for (const listener of listeners) listener();
  };
  return {
    subscribe(listener: () => void) {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    version: () => version,
    get: () => (active() ? value : undefined),
    guarded: () => active() && !!value,
    save(draft: IssueDraft, limits: ServiceLimits) {
      if (
        !active() ||
        !admitsIssueDraft(draft, limits) ||
        (value && value.target !== draft.target)
      )
        return false;
      value = Object.freeze({
        target: draft.target,
        days: draft.days,
        reason: draft.reason,
      });
      changed();
      return true;
    },
    clear(expected: IssueDraft) {
      if (
        !active() ||
        !value ||
        value !== expected ||
        value.target !== expected.target
      )
        return false;
      value = undefined;
      changed();
      return true;
    },
    dispose() {
      disposed = true;
      value = undefined;
      changed();
      listeners.clear();
    },
  };
}

export type IssueDrafts = ReturnType<typeof issueDrafts>;
