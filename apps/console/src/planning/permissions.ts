import { planningScreens } from "./registry";
import type { Audience } from "./routing";
export type Permission = {
  key: string;
  screenId: string;
  kind: "view" | "action";
  allowedClasses: Audience[];
  delegable: boolean;
  incompatible: string[];
  sensitivity: "none" | "unresolved";
  executable: boolean;
  evidence: string;
};
export const viewPermissions: Permission[] = planningScreens.map((s) => ({
  key: `screen.${s.id.toLowerCase()}.view`,
  screenId: s.id,
  kind: "view",
  allowedClasses:
    s.audience === "admin+portal"
      ? ["admin", "customer"]
      : s.audience === "portal"
        ? ["customer"]
        : s.audience === "prelogin"
          ? ["unauthenticated"]
          : ["admin"],
  delegable: false,
  incompatible: [],
  sensitivity: "none",
  executable: false,
  evidence: s.spec,
}));
export const unresolvedActionEvidence = planningScreens.map((s) => ({
  screenId: s.id,
  references: s.commandReferences,
  source: s.spec,
  summary: s.actionEvidence,
  status: "unbound" as const,
}));
export function canHold(permission: Permission, audience: Audience) {
  return permission.allowedClasses.includes(audience);
}
export function canDelegate(
  permission: Permission,
  audience: Audience,
  heldDelegableKeys: ReadonlySet<string>,
  withinScope: boolean,
) {
  return (
    permission.executable &&
    permission.delegable &&
    withinScope &&
    canHold(permission, audience) &&
    heldDelegableKeys.has(permission.key)
  );
}
