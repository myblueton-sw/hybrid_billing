import type { Connection } from "../connections/api";
import { actionUrl, uuid } from "../connections/api";
export const stages = [
  "path",
  "permissions",
  "first-data",
  "reconciliation",
  "activation",
] as const;
export type Stage = (typeof stages)[number];
export const stageIndex = (stage: string) =>
  Math.max(0, stages.indexOf(stage as Stage));
export function stageUrl(stage: Stage, locale: string, connection?: string) {
  return (
    "#/onboarding?" +
    new URLSearchParams({
      lang: locale,
      service: "local",
      view: stage,
      ...(connection ? { connection } : {}),
    })
  );
}
export function destination(
  data: Connection,
  action: Connection["actions"][number],
  locale: string,
) {
  return data.actions.includes(action) ? actionUrl(action, data, locale) : null;
}

export const validContext = (value: string | undefined) =>
  value === undefined || uuid.test(value);
