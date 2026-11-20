import type { ServiceLimits } from "../../../../service/service-query-types";
import { admitsCommand } from "./command-intents";

export type DraftAction = "suspend" | "reinstate" | "retire" | "revoke";
export type ActionDraft = Readonly<{
  action: DraftAction;
  target: string;
  reason: string;
}>;
export const actionCommand = (action: DraftAction) =>
  action === "revoke"
    ? "service.revoke_credential"
    : (`service.${action}` as const);
export function admitsActionDraft(draft: ActionDraft, limits: ServiceLimits) {
  return (
    ["suspend", "reinstate", "retire", "revoke"].includes(draft.action) &&
    /^[A-Za-z0-9._:-]{1,128}$/.test(draft.target) &&
    admitsCommand(
      {
        command: actionCommand(draft.action),
        target: draft.target,
        reason: draft.reason,
        fields: {},
        revision: "r".repeat(128),
        replay: 1,
      },
      limits,
      { request: "r".repeat(128), key: "k".repeat(128) },
    )
  );
}
export function actionDrafts(current: () => boolean) {
  let value: ActionDraft | undefined,
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
    save(draft: ActionDraft, limits: ServiceLimits) {
      if (
        !active() ||
        !admitsActionDraft(draft, limits) ||
        (value &&
          (value.action !== draft.action || value.target !== draft.target))
      )
        return false;
      value = Object.freeze({
        action: draft.action,
        target: draft.target,
        reason: draft.reason,
      });
      changed();
      return true;
    },
    clear(expected: ActionDraft) {
      if (
        !active() ||
        !value ||
        value !== expected ||
        value.action !== expected.action ||
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
export type ActionDrafts = ReturnType<typeof actionDrafts>;
